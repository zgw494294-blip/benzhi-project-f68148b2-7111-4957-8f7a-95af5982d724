package analysis

import (
	"fmt"
	"math"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func DetectFlags(widths []float64, best domain.Candidate, params domain.AnalysisParameters) []domain.QualityFlag {
	flags := make([]domain.QualityFlag, 0)
	if best.Correlation < params.LowCorrelation {
		flags = append(flags, domain.QualityFlag{Code: "LOW_CORRELATION", AffectedRange: fmt.Sprintf("%d-%d", best.StartYear, best.EndYear), Severity: "high", Message: "最佳候选相关得分低于阈值", Blocking: true})
	}
	for i := 1; i < len(widths); i++ {
		ratio := widths[i] / widths[i-1]
		year := best.StartYear + i
		if ratio >= 2.8 {
			flags = append(flags, domain.QualityFlag{Code: "POSSIBLE_MISSING_RING", AffectedRange: fmt.Sprintf("%d", year), Severity: "medium", Message: "相邻宽度突增，需核查可能缺轮", Blocking: true})
		} else if ratio <= 0.25 {
			flags = append(flags, domain.QualityFlag{Code: "POSSIBLE_FALSE_RING", AffectedRange: fmt.Sprintf("%d", year), Severity: "medium", Message: "相邻宽度突降，需核查可能伪轮", Blocking: true})
		}
	}
	indices := RingIndex(widths)
	for i, value := range indices {
		if math.Abs(value) > params.OutlierZScore {
			flags = append(flags, domain.QualityFlag{Code: "WIDTH_OUTLIER", AffectedRange: fmt.Sprintf("%d", best.StartYear+i), Severity: "low", Message: "局部标准化宽度异常", Blocking: false})
		}
	}
	return stableFlags(flags)
}

func stableFlags(flags []domain.QualityFlag) []domain.QualityFlag {
	result := make([]domain.QualityFlag, 0, len(flags))
	seen := map[string]bool{}
	for _, flag := range flags {
		key := flag.Code + "|" + flag.AffectedRange
		if !seen[key] {
			result = append(result, flag)
			seen[key] = true
		}
	}
	return result
}
