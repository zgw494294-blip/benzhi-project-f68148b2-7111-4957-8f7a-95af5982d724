package snapshotfailurevisible_test

import (
	"os"
	"path/filepath"
	"testing"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/analysis"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/application"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/persistence"
)

func TestFailedCreateMustNotBecomeVisible(t *testing.T) {
	dir := t.TempDir()
	repo, err := persistence.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repo, analysis.NewEngine(), "test-key")

	snapshotPath := filepath.Join(dir, "projection.json")
	if err := os.Remove(snapshotPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(snapshotPath, 0o700); err != nil {
		t.Fatal(err)
	}

	input := application.CreateCaseInput{
		CaseID: "case-snapshot-failure", SiteName: "遗址", StructureRef: "梁架",
		SamplingPosition: "端部", SpeciesAssessment: "松属", SampledAt: "2026-08-27",
		MinimumOverlap: 5, CreatedBy: "测量员", IdempotencyKey: "create-once",
	}
	if _, err := service.CreateCase(input); err == nil {
		t.Fatal("测试前提失败：快照目标为目录时写入应报错")
	}
	if record, err := service.GetCase(input.CaseID); err == nil {
		t.Fatalf("TestFailedCreateMustNotBecomeVisible: 失败的 CreateCase 仍暴露 version=%d 的档案", record.Case.Version)
	}
}
