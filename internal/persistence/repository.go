package persistence

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

type Repository struct {
	mu              sync.RWMutex
	dir             string
	logPath         string
	snapshotPath    string
	lockPath        string
	cases           map[string]domain.CaseRecord
	idempotent      map[string]domain.CaseRecord
	credentialIndex map[string]string
	sequence        int64
	lastDigest      string
}

func Open(dir string) (*Repository, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("数据目录不能为空")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	repo := &Repository{dir: dir, logPath: filepath.Join(dir, "events.jsonl"), snapshotPath: filepath.Join(dir, "projection.json"), lockPath: filepath.Join(dir, ".write.lock")}
	events, err := readEvents(repo.logPath)
	if err != nil {
		return nil, err
	}
	state, err := replay(events)
	if err != nil {
		return nil, err
	}
	repo.cases, repo.idempotent = state.cases, state.idempotent
	repo.sequence, repo.lastDigest = state.sequence, state.digest
	repo.rebuildCredentialIndex()
	if snapshot, snapErr := readSnapshot(repo.snapshotPath); snapErr == nil {
		if snapshot.Sequence != repo.sequence || snapshot.LastDigest != repo.lastDigest {
			return nil, fmt.Errorf("投影快照与事件日志不一致")
		}
	} else if !os.IsNotExist(snapErr) {
		return nil, snapErr
	}
	if err := repo.persistSnapshot(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *Repository) Save(record domain.CaseRecord, expectedVersion int64, idempotencyKey, kind, actor, summary string, now time.Time) (domain.CaseRecord, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if strings.TrimSpace(idempotencyKey) == "" {
		return domain.CaseRecord{}, false, domain.Invalid("idempotencyKey", "不能为空")
	}
	release, err := exclusiveLock(r.lockPath)
	if err != nil {
		return domain.CaseRecord{}, false, err
	}
	defer release()
	// 多实例共享同一数据目录时，本实例在打开后到写入前可能已有其他实例追加事件。
	// 因此在持有跨进程锁后重新读取日志并重放，使序号、摘要链与幂等索引回到最新状态。
	if err := r.resyncFromLog(); err != nil {
		return domain.CaseRecord{}, false, err
	}
	index := idempotencyIndex(record.Case.CaseID, idempotencyKey)
	if prior, ok := r.idempotent[index]; ok {
		clone, err := domain.CloneRecord(prior)
		return clone, true, err
	}
	current, exists := r.cases[record.Case.CaseID]
	actualVersion := int64(0)
	if exists {
		actualVersion = current.Case.Version
	}
	if actualVersion != expectedVersion {
		return domain.CaseRecord{}, false, domain.Conflict(fmt.Sprintf("版本冲突：期待 %d，当前 %d", expectedVersion, actualVersion))
	}
	if !exists && expectedVersion != 0 {
		return domain.CaseRecord{}, false, domain.Conflict("新档案版本必须从 0 开始")
	}
	record.Case.Version = expectedVersion + 1
	record.Case.UpdatedAt = now.UTC()
	record.Timeline = append(record.Timeline, domain.TimelineEntry{Sequence: r.sequence + 1, Kind: kind, Actor: actor, At: now.UTC(), Summary: summary})
	event := Event{Sequence: r.sequence + 1, CaseID: record.Case.CaseID, CaseVersion: record.Case.Version, Kind: kind, Actor: actor, Summary: summary, IdempotencyKey: idempotencyKey, OccurredAt: now.UTC(), PreviousDigest: r.lastDigest, Record: record}
	digest, err := eventDigest(event)
	if err != nil {
		return domain.CaseRecord{}, false, err
	}
	event.Digest = digest
	if err := appendEvent(r.logPath, event); err != nil {
		return domain.CaseRecord{}, false, err
	}
	r.sequence, r.lastDigest = event.Sequence, event.Digest
	r.cases[event.CaseID] = record
	r.idempotent[index] = record
	if record.Credential != nil {
		r.credentialIndex[record.Credential.CredentialID] = record.Case.CaseID
	}
	if err := r.persistSnapshot(); err != nil {
		return domain.CaseRecord{}, false, err
	}
	clone, err := domain.CloneRecord(record)
	return clone, false, err
}

// resyncFromLog 重新读取事件日志并重放，用最新的权威状态覆盖本实例的内存状态。
// 调用方必须已经持有跨进程锁 r.lockPath，避免重放期间其他实例继续追加事件。
func (r *Repository) resyncFromLog() error {
	events, err := readEvents(r.logPath)
	if err != nil {
		return err
	}
	state, err := replay(events)
	if err != nil {
		return err
	}
	r.cases, r.idempotent = state.cases, state.idempotent
	r.sequence, r.lastDigest = state.sequence, state.digest
	r.rebuildCredentialIndex()
	return nil
}

func (r *Repository) Get(caseID string) (domain.CaseRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.cases[caseID]
	if !ok {
		return domain.CaseRecord{}, domain.NotFound("档案不存在")
	}
	return domain.CloneRecord(record)
}

func (r *Repository) List() ([]domain.CaseRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.cases))
	for id := range r.cases {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]domain.CaseRecord, 0, len(ids))
	for _, id := range ids {
		clone, err := domain.CloneRecord(r.cases[id])
		if err != nil {
			return nil, err
		}
		result = append(result, clone)
	}
	return result, nil
}

func (r *Repository) FindCredential(credentialID string) (domain.CaseRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	caseID, ok := r.credentialIndex[credentialID]
	if !ok {
		return domain.CaseRecord{}, domain.NotFound("凭据编号不存在")
	}
	return domain.CloneRecord(r.cases[caseID])
}

func (r *Repository) RunOwner(runID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for caseID, record := range r.cases {
		for _, run := range record.Runs {
			if run.RunID == runID {
				return caseID, true
			}
		}
	}
	return "", false
}

func (r *Repository) persistSnapshot() error {
	copied := make(map[string]domain.CaseRecord, len(r.cases))
	for id, record := range r.cases {
		copied[id] = record
	}
	return writeSnapshot(r.snapshotPath, snapshotPayload{Sequence: r.sequence, LastDigest: r.lastDigest, Cases: copied})
}

func (r *Repository) rebuildCredentialIndex() {
	r.credentialIndex = map[string]string{}
	for caseID, record := range r.cases {
		if record.Credential != nil {
			r.credentialIndex[record.Credential.CredentialID] = caseID
		}
	}
}

func idempotencyIndex(caseID, key string) string { return caseID + "\x00" + key }

func (r *Repository) Sequence() int64 { r.mu.RLock(); defer r.mu.RUnlock(); return r.sequence }
