package shareddatalogcorruption_test

import (
	"testing"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/persistence"
)

func TestTwoRepositoriesMustNotCorruptSharedLog(t *testing.T) {
	dir := t.TempDir()
	first, err := persistence.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := persistence.Open(dir)
	if err != nil {
		return // 拒绝第二个所有者也是安全行为。
	}

	firstRecord := domain.CaseRecord{Case: domain.DatingCase{CaseID: "case-first", Status: domain.StatusDraft}}
	if _, _, err := first.Save(firstRecord, 0, "first", "CaseCreated", "u1", "建档一", time.Unix(1, 0)); err != nil {
		t.Fatal(err)
	}
	secondRecord := domain.CaseRecord{Case: domain.DatingCase{CaseID: "case-second", Status: domain.StatusDraft}}
	if _, _, err := second.Save(secondRecord, 0, "second", "CaseCreated", "u2", "建档二", time.Unix(2, 0)); err != nil {
		return // 在追加事件前拒绝已经过期的写入者也是安全行为。
	}

	reopened, err := persistence.Open(dir)
	if err != nil {
		t.Fatalf("TestTwoRepositoriesMustNotCorruptSharedLog: 两个实例成功写入后事件日志无法重放: %v", err)
	}
	if _, err := reopened.Get("case-first"); err != nil {
		t.Fatalf("TestTwoRepositoriesMustNotCorruptSharedLog: 首个已确认写入丢失: %v", err)
	}
	if _, err := reopened.Get("case-second"); err != nil {
		t.Fatalf("TestTwoRepositoriesMustNotCorruptSharedLog: 第二个已确认写入丢失: %v", err)
	}
}
