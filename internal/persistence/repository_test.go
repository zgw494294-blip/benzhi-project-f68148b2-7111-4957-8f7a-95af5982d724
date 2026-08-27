package persistence

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func TestRepositoryRecoveryAndIdempotency(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	record := domain.CaseRecord{Case: domain.DatingCase{CaseID: "case-1", Status: domain.StatusDraft}}
	saved, duplicate, err := repo.Save(record, 0, "idem-1", "CaseCreated", "u", "建档", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if duplicate || saved.Case.Version != 1 {
		t.Fatalf("首次保存异常 duplicate=%v version=%d", duplicate, saved.Case.Version)
	}
	retry, duplicate, err := repo.Save(record, 0, "idem-1", "CaseCreated", "u", "建档", time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !duplicate || retry.Case.Version != 1 {
		t.Fatal("幂等重试未复用首次结果")
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Get("case-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Case.Version != 1 || reopened.Sequence() != 1 {
		t.Fatal("事件重放结果错误")
	}
}

func TestRepositoryDetectsTruncatedLog(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	record := domain.CaseRecord{Case: domain.DatingCase{CaseID: "case-2", Status: domain.StatusDraft}}
	if _, _, err := repo.Save(record, 0, "idem", "CaseCreated", "u", "建档", time.Now()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "events.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data[:len(data)-1], 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("应识别截断日志")
	}
}
