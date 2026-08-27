package analysis

import (
	"testing"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func TestAnalyzeDeterministicDigestAndBestYear(t *testing.T) {
	engine := NewEngine()
	chronology := BuiltinChronology()
	widths := append([]float64(nil), chronology.Widths[20:65]...)
	revision := domain.RingSeriesRevision{RevisionID: "rev-1", CaseID: "case-1", Unit: "mm", MeasurementDirection: "pith-to-bark", Widths: widths, ContentDigest: domain.MustDigest(widths)}
	params := DefaultParameters(20)
	one, err := engine.Analyze("case-1", revision, params, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	two, err := engine.Analyze("case-1", revision, params, time.Unix(200, 0))
	if err != nil {
		t.Fatal(err)
	}
	if one.ResultDigest != two.ResultDigest {
		t.Fatal("相同输入的结果摘要不稳定")
	}
	if one.RankedCandidates[0].StartYear != 1870 {
		t.Fatalf("最佳年份=%d", one.RankedCandidates[0].StartYear)
	}
	if one.RankedCandidates[0].Correlation < 0.999 {
		t.Fatalf("完全匹配得分=%f", one.RankedCandidates[0].Correlation)
	}
}

func TestDirectionReversal(t *testing.T) {
	engine := NewEngine()
	c := BuiltinChronology()
	widths := append([]float64(nil), c.Widths[10:40]...)
	reverse(widths)
	revision := domain.RingSeriesRevision{RevisionID: "r", Unit: "mm", MeasurementDirection: "bark-to-pith", Widths: widths, ContentDigest: "x"}
	run, err := engine.Analyze("c", revision, DefaultParameters(20), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if run.RankedCandidates[0].StartYear != 1860 {
		t.Fatalf("反向序列匹配年份=%d", run.RankedCandidates[0].StartYear)
	}
}
