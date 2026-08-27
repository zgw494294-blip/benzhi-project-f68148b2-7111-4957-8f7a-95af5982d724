package application

import (
	"testing"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/analysis"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/persistence"
)

func TestCompleteWorkflowAndIndependentReview(t *testing.T) {
	repo, err := persistence.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, analysis.NewEngine(), "test-key")
	record, err := service.CreateCase(CreateCaseInput{CaseID: "case-flow", SiteName: "遗址", StructureRef: "构件", SamplingPosition: "端部", SpeciesAssessment: "松属", SampledAt: "2026-08-27", MinimumOverlap: 20, CreatedBy: "测量员", IdempotencyKey: "create"})
	if err != nil {
		t.Fatal(err)
	}
	widths := analysis.BuiltinChronology().Widths[30:75]
	record, err = service.SubmitSeries("case-flow", SubmitSeriesInput{CommandMeta: CommandMeta{ExpectedVersion: record.Case.Version, IdempotencyKey: "series", Actor: "测量员"}, Unit: "mm", MeasurementDirection: "pith-to-bark", Widths: widths, ChangeReason: "首版"})
	if err != nil {
		t.Fatal(err)
	}
	record, err = service.Analyze("case-flow", AnalyzeInput{CommandMeta: CommandMeta{ExpectedVersion: record.Case.Version, IdempotencyKey: "analyze", Actor: "分析员"}})
	if err != nil {
		t.Fatal(err)
	}
	for i, flag := range domain.UnresolvedFlags(record) {
		record, err = service.ResolveDispute("case-flow", ResolveDisputeInput{CommandMeta: CommandMeta{ExpectedVersion: record.Case.Version, IdempotencyKey: string(rune('a' + i)), Actor: "分析员"}, FlagCode: flag.Code, AffectedRange: flag.AffectedRange, Disposition: "explain", Rationale: "显微证据已核对"})
		if err != nil {
			t.Fatal(err)
		}
	}
	record, err = service.StartReview("case-flow", StartReviewInput{CommandMeta{ExpectedVersion: record.Case.Version, IdempotencyKey: "start", Actor: "分析员"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.DecideReview("case-flow", ReviewInput{CommandMeta: CommandMeta{ExpectedVersion: record.Case.Version, IdempotencyKey: "bad-review", Actor: "测量员"}, Outcome: "approve", Findings: "不应成功"})
	if domain.ErrorCode(err) != "forbidden" {
		t.Fatalf("同一提交者复核错误=%v", err)
	}
	record, err = service.DecideReview("case-flow", ReviewInput{CommandMeta: CommandMeta{ExpectedVersion: record.Case.Version, IdempotencyKey: "approve", Actor: "复核员"}, Outcome: "approve", Findings: "证据一致"})
	if err != nil {
		t.Fatal(err)
	}
	record, err = service.Freeze("case-flow", FreezeInput{CommandMeta{ExpectedVersion: record.Case.Version, IdempotencyKey: "freeze", Actor: "档案负责人"}})
	if err != nil {
		t.Fatal(err)
	}
	record, err = service.Publish("case-flow", PublishInput{CommandMeta: CommandMeta{ExpectedVersion: record.Case.Version, IdempotencyKey: "publish", Actor: "档案负责人"}})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := service.Verify(VerifyInput{Payload: record.Credential})
	if err != nil {
		t.Fatal(err)
	}
	if record.Case.Status != domain.StatusPublished || !verified.Valid || !verified.Current {
		t.Fatalf("发布验证失败: %+v", verified)
	}
}

func TestStaleVersionRejected(t *testing.T) {
	repo, _ := persistence.Open(t.TempDir())
	service := NewService(repo, analysis.NewEngine(), "")
	record, err := service.CreateCase(CreateCaseInput{CaseID: "stale", SiteName: "遗址", StructureRef: "构件", SamplingPosition: "端部", SpeciesAssessment: "松属", SampledAt: "2026-08-27", MinimumOverlap: 5, CreatedBy: "u", IdempotencyKey: "c"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.SubmitSeries("stale", SubmitSeriesInput{CommandMeta: CommandMeta{ExpectedVersion: record.Case.Version + 1, IdempotencyKey: "s", Actor: "u"}, Unit: "mm", MeasurementDirection: "pith-to-bark", Widths: []float64{1, 1, 1, 1, 1}, ChangeReason: "x"})
	if domain.ErrorCode(err) != "conflict" {
		t.Fatalf("过期版本错误=%v", err)
	}
}
