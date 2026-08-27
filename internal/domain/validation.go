package domain

import (
	"math"
	"strings"
	"time"
)

func ValidateNewCase(c DatingCase) error {
	if strings.TrimSpace(c.SiteName) == "" {
		return Invalid("siteName", "不能为空")
	}
	if strings.TrimSpace(c.StructureRef) == "" {
		return Invalid("structureRef", "不能为空")
	}
	if strings.TrimSpace(c.SamplingPosition) == "" {
		return Invalid("samplingPosition", "不能为空")
	}
	if strings.TrimSpace(c.SpeciesAssessment) == "" {
		return Invalid("speciesAssessment", "不能为空")
	}
	if _, err := time.Parse("2006-01-02", c.SampledAt); err != nil {
		return Invalid("sampledAt", "必须为 YYYY-MM-DD")
	}
	if c.MinimumOverlap < 5 || c.MinimumOverlap > 500 {
		return Invalid("minimumOverlap", "必须在 5 到 500 之间")
	}
	if strings.TrimSpace(c.CreatedBy) == "" {
		return Invalid("createdBy", "不能为空")
	}
	return nil
}

func ValidateRevision(r RingSeriesRevision, minimum int) error {
	if r.Unit != "mm" && r.Unit != "0.01mm" {
		return Invalid("unit", "仅支持 mm 或 0.01mm")
	}
	if r.MeasurementDirection != "pith-to-bark" && r.MeasurementDirection != "bark-to-pith" {
		return Invalid("measurementDirection", "方向不受支持")
	}
	if len(r.Widths) < minimum {
		return Invalid("widths", "长度小于最小重叠年数")
	}
	if len(r.Widths) > 2000 {
		return Invalid("widths", "最多 2000 个测量值")
	}
	for _, w := range r.Widths {
		if w <= 0 || math.IsNaN(w) || math.IsInf(w, 0) {
			return Invalid("widths", "宽度必须为有限正数")
		}
	}
	if strings.TrimSpace(r.SubmittedBy) == "" {
		return Invalid("submittedBy", "不能为空")
	}
	if strings.TrimSpace(r.ChangeReason) == "" {
		return Invalid("changeReason", "不能为空")
	}
	return nil
}

func CanSubmitSeries(status Status) bool {
	return status == StatusDraft || status == StatusSeriesSubmitted || status == StatusAnalyzed || status == StatusNeedsCorrection
}

func CanAnalyze(status Status) bool {
	return status == StatusSeriesSubmitted || status == StatusNeedsCorrection || status == StatusAnalyzed
}

func ActiveRevision(record CaseRecord) (RingSeriesRevision, bool) {
	for i := len(record.Revisions) - 1; i >= 0; i-- {
		if record.Revisions[i].RevisionID == record.ActiveRevisionID {
			return record.Revisions[i], true
		}
	}
	return RingSeriesRevision{}, false
}

func ActiveRun(record CaseRecord) (CrossdatingRun, bool) {
	for i := len(record.Runs) - 1; i >= 0; i-- {
		if record.Runs[i].RunID == record.ActiveRunID {
			return record.Runs[i], true
		}
	}
	return CrossdatingRun{}, false
}

func CurrentResolutions(record CaseRecord) []DisputeResolution {
	out := make([]DisputeResolution, 0)
	for _, resolution := range record.Resolutions {
		if resolution.RunID == record.ActiveRunID {
			out = append(out, resolution)
		}
	}
	return out
}

func UnresolvedFlags(record CaseRecord) []QualityFlag {
	run, ok := ActiveRun(record)
	if !ok {
		return nil
	}
	resolved := map[string]bool{}
	for _, resolution := range CurrentResolutions(record) {
		resolved[resolution.FlagCode+"|"+resolution.AffectedRange] = true
	}
	var remaining []QualityFlag
	for _, flag := range run.QualityFlags {
		if flag.Blocking && !resolved[flag.Code+"|"+flag.AffectedRange] {
			remaining = append(remaining, flag)
		}
	}
	return remaining
}

func EligibleForReview(record CaseRecord) error {
	if record.Case.Status != StatusAnalyzed && record.Case.Status != StatusNeedsCorrection {
		return Conflict("当前状态不能进入复核")
	}
	if len(UnresolvedFlags(record)) > 0 {
		return Conflict("仍有未解决的阻断争议")
	}
	run, runOK := ActiveRun(record)
	revision, revisionOK := ActiveRevision(record)
	if !runOK || !revisionOK || run.RevisionID != revision.RevisionID {
		return Conflict("当前修订缺少有效分析")
	}
	return nil
}

func EligibleForFreeze(record CaseRecord) error {
	if record.Case.Status != StatusApproved {
		return Conflict("仅 Approved 档案可冻结")
	}
	if len(record.Reviews) == 0 {
		return Conflict("缺少复核决定")
	}
	review := record.Reviews[len(record.Reviews)-1]
	if review.Outcome != "approve" || review.RevisionID != record.ActiveRevisionID || review.RunID != record.ActiveRunID || review.CaseVersion != record.Case.Version-1 {
		return Conflict("批准决定与当前证据不匹配")
	}
	return nil
}

func FindFlag(record CaseRecord, code, affectedRange string) (QualityFlag, bool) {
	run, ok := ActiveRun(record)
	if !ok {
		return QualityFlag{}, false
	}
	for _, flag := range run.QualityFlags {
		if flag.Code == code && flag.AffectedRange == affectedRange {
			return flag, true
		}
	}
	return QualityFlag{}, false
}
