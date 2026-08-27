package domain

import "testing"

func TestTransitionAndUnresolvedFlags(t *testing.T) {
	record := CaseRecord{Case: DatingCase{Status: StatusDraft}}
	if err := SetStatus(&record, StatusSeriesSubmitted); err != nil {
		t.Fatal(err)
	}
	if err := SetStatus(&record, StatusUnderReview); err == nil {
		t.Fatal("应拒绝跳过分析的状态转换")
	}
	record.Case.Status = StatusNeedsCorrection
	record.ActiveRunID = "run-1"
	record.Runs = []CrossdatingRun{{RunID: "run-1", QualityFlags: []QualityFlag{{Code: "LOW", AffectedRange: "1-20", Blocking: true}, {Code: "INFO", AffectedRange: "8", Blocking: false}}}}
	if got := len(UnresolvedFlags(record)); got != 1 {
		t.Fatalf("未解决阻断项=%d", got)
	}
	record.Resolutions = []DisputeResolution{{RunID: "run-1", FlagCode: "LOW", AffectedRange: "1-20"}}
	if got := len(UnresolvedFlags(record)); got != 0 {
		t.Fatalf("处置后仍有阻断项=%d", got)
	}
}

func TestRevisionValidation(t *testing.T) {
	revision := RingSeriesRevision{Unit: "mm", MeasurementDirection: "pith-to-bark", Widths: []float64{1, 2, 3, 4, 5}, SubmittedBy: "u", ChangeReason: "首版"}
	if err := ValidateRevision(revision, 5); err != nil {
		t.Fatal(err)
	}
	revision.Widths[2] = 0
	if err := ValidateRevision(revision, 5); err == nil {
		t.Fatal("应拒绝零宽度")
	}
}
