package application

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func (s *Service) PreflightSeries(caseID string, input SeriesPreflightInput) (SeriesPreflightResult, error) {
	record, err := s.repo.Get(caseID)
	if err != nil {
		return SeriesPreflightResult{}, err
	}
	if input.CaseVersion != record.Case.Version {
		return SeriesPreflightResult{}, domain.Conflict("preflight_version_conflict：预检基于过期档案版本")
	}
	if !domain.CanSubmitSeries(record.Case.Status) {
		return SeriesPreflightResult{}, domain.Conflict("当前状态禁止预检序列")
	}
	result := SeriesPreflightResult{CaseVersion: record.Case.Version, ParentRevisionID: record.ActiveRevisionID, FirstImport: record.ActiveRevisionID == "", Warnings: []string{}, Errors: []SeriesFieldError{}, Differences: []SeriesDifference{}}
	if input.Unit != "mm" && input.Unit != "0.01mm" {
		result.Errors = append(result.Errors, SeriesFieldError{Code: "invalid_unit", Message: "测量单位仅支持 mm 或 0.01mm"})
	}
	if input.MeasurementDirection != "pith-to-bark" && input.MeasurementDirection != "bark-to-pith" {
		result.Errors = append(result.Errors, SeriesFieldError{Code: "invalid_direction", Message: "起始方向不受支持"})
	}
	values, parseErrors := parseWidths(input.RawText)
	result.Widths, result.Errors = values, append(result.Errors, parseErrors...)
	if len(values) > 2000 {
		result.Errors = append(result.Errors, SeriesFieldError{Position: 2001, Code: "too_long", Message: "序列最多 2000 轮"})
	}
	if len(values) < record.Case.MinimumOverlap {
		result.Errors = append(result.Errors, SeriesFieldError{Position: len(values) + 1, Code: "insufficient_overlap", Message: "轮数小于档案最小重叠年数"})
	}
	physical := toMillimeters(values, input.Unit)
	result.Statistics = seriesStatistics(physical)
	if parent, ok := domain.ActiveRevision(record); ok {
		if parent.MeasurementDirection != input.MeasurementDirection {
			result.Warnings = append(result.Warnings, "测量方向相对父修订发生变化，请确认轮位解释")
		}
		result.Differences = seriesDifferences(toMillimeters(parent.Widths, parent.Unit), physical)
		denominator := len(parent.Widths)
		if len(values) > denominator {
			denominator = len(values)
		}
		if denominator > 0 {
			result.ChangeRatio = math.Round(float64(len(result.Differences))/float64(denominator)*1e6) / 1e6
		}
	} else {
		result.Warnings = append(result.Warnings, "首版导入：不存在父修订")
	}
	result.ContentDigest = seriesContentDigest(input.Unit, input.MeasurementDirection, values)
	result.PreflightDigest = domain.MustDigest(struct {
		CaseVersion      int64     `json:"caseVersion"`
		ParentRevisionID string    `json:"parentRevisionID"`
		Unit             string    `json:"unit"`
		Direction        string    `json:"direction"`
		Widths           []float64 `json:"widths"`
		ContentDigest    string    `json:"contentDigest"`
	}{result.CaseVersion, result.ParentRevisionID, input.Unit, input.MeasurementDirection, values, result.ContentDigest})
	return result, nil
}

func parseWidths(raw string) ([]float64, []SeriesFieldError) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "，", ",")
	segments := make([]string, 0)
	start := 0
	for index, r := range normalized {
		if r == ',' || r == '\n' {
			segments = append(segments, normalized[start:index])
			start = index + len(string(r))
		}
	}
	segments = append(segments, normalized[start:])
	trimmed := strings.TrimSpace(normalized)
	errors := []SeriesFieldError{}
	if trimmed == "" {
		return nil, []SeriesFieldError{{Position: 1, Code: "empty", Message: "第 1 轮为空"}}
	}
	values := make([]float64, 0)
	position := 1
	for _, segment := range segments {
		fields := strings.Fields(segment)
		if len(fields) == 0 {
			errors = append(errors, SeriesFieldError{Position: position, Code: "empty", Message: "该轮位为空"})
			position++
			continue
		}
		for _, token := range fields {
			value, err := strconv.ParseFloat(token, 64)
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
				errors = append(errors, SeriesFieldError{Position: position, Code: "not_numeric", Message: "不是有限数值"})
				position++
				continue
			}
			if value <= 0 {
				errors = append(errors, SeriesFieldError{Position: position, Code: "not_positive", Message: "宽度必须为正数"})
			}
			values = append(values, value)
			position++
		}
	}
	return values, errors
}

func toMillimeters(values []float64, unit string) []float64 {
	factor := 1.0
	if unit == "0.01mm" {
		factor = .01
	}
	out := make([]float64, len(values))
	for i, v := range values {
		out[i] = v * factor
	}
	return out
}
func seriesStatistics(values []float64) SeriesStatistics {
	st := SeriesStatistics{Count: len(values), AbnormalPositions: []int{}}
	if len(values) == 0 {
		return st
	}
	st.Minimum, st.Maximum = values[0], values[0]
	for i, v := range values {
		if v < st.Minimum {
			st.Minimum = v
		}
		if v > st.Maximum {
			st.Maximum = v
		}
		if v < .01 || v > 20 {
			st.AbnormalPositions = append(st.AbnormalPositions, i+1)
		}
	}
	return st
}
func seriesDifferences(before, after []float64) []SeriesDifference {
	n := len(before)
	if len(after) > n {
		n = len(after)
	}
	out := []SeriesDifference{}
	for i := 0; i < n; i++ {
		switch {
		case i >= len(before):
			v := after[i]
			out = append(out, SeriesDifference{Position: i + 1, Kind: "added", After: &v})
		case i >= len(after):
			v := before[i]
			out = append(out, SeriesDifference{Position: i + 1, Kind: "removed", Before: &v})
		case math.Abs(before[i]-after[i]) > 1e-9:
			b, a := before[i], after[i]
			out = append(out, SeriesDifference{Position: i + 1, Kind: "changed", Before: &b, After: &a})
		}
	}
	return out
}
func seriesContentDigest(unit, direction string, widths []float64) string {
	return domain.MustDigest(struct {
		Unit      string    `json:"unit"`
		Direction string    `json:"direction"`
		Widths    []float64 `json:"widths"`
	}{unit, direction, widths})
}

func (s *Service) RunHistory(caseID string) ([]RunHistoryItem, error) {
	record, err := s.repo.Get(caseID)
	if err != nil {
		return nil, err
	}
	result := make([]RunHistoryItem, 0, len(record.Runs))
	for i := len(record.Runs) - 1; i >= 0; i-- {
		run := record.Runs[i]
		result = append(result, RunHistoryItem{Run: run, Current: run.RunID == record.ActiveRunID, Stale: run.RevisionID != record.ActiveRevisionID})
	}
	return result, nil
}

func (s *Service) CompareRuns(caseID, firstID, secondID string) (RunComparison, error) {
	record, err := s.repo.Get(caseID)
	if err != nil {
		return RunComparison{}, err
	}
	first, ok := findRun(record, firstID)
	if !ok {
		if owner, exists := s.repo.RunOwner(firstID); exists && owner != caseID {
			return RunComparison{}, domain.NewError("run_ownership_mismatch", "运行不属于当前档案")
		}
		return RunComparison{}, domain.NewError("run_not_found", "第一个运行不存在")
	}
	second, ok := findRun(record, secondID)
	if !ok {
		if owner, exists := s.repo.RunOwner(secondID); exists && owner != caseID {
			return RunComparison{}, domain.NewError("run_ownership_mismatch", "运行不属于当前档案")
		}
		return RunComparison{}, domain.NewError("run_not_found", "第二个运行不存在")
	}
	if first.CaseID != caseID || second.CaseID != caseID {
		return RunComparison{}, domain.NewError("run_ownership_mismatch", "运行不属于当前档案")
	}
	if !validRunDigest(record, first) || !validRunDigest(record, second) {
		return RunComparison{}, domain.NewError("run_digest_mismatch", "运行输入摘要损坏")
	}
	result := RunComparison{FirstRunID: firstID, SecondRunID: secondID, BestYearChange: first.BestCalendarRange + " → " + second.BestCalendarRange, CorrelationDifference: math.Round((second.CorrelationScore-first.CorrelationScore)*1e6) / 1e6, FirstStale: first.RevisionID != record.ActiveRevisionID, SecondStale: second.RevisionID != record.ActiveRevisionID, AddedFlags: []domain.QualityFlag{}, RemovedFlags: []domain.QualityFlag{}, PersistentFlags: []domain.QualityFlag{}}
	before := map[int]int{}
	after := map[int]int{}
	for _, c := range first.RankedCandidates {
		before[c.StartYear] = c.Rank
	}
	for _, c := range second.RankedCandidates {
		after[c.StartYear] = c.Rank
	}
	years := map[int]bool{}
	for y := range before {
		years[y] = true
	}
	for y := range after {
		years[y] = true
	}
	ordered := make([]int, 0, len(years))
	for y := range years {
		ordered = append(ordered, y)
	}
	sort.Ints(ordered)
	for _, y := range ordered {
		b, a := before[y], after[y]
		if b != a {
			result.CandidateChanges = append(result.CandidateChanges, CandidateRankChange{StartYear: y, BeforeRank: b, AfterRank: a, Change: b - a})
		}
	}
	fm := map[string]domain.QualityFlag{}
	sm := map[string]domain.QualityFlag{}
	for _, f := range first.QualityFlags {
		fm[f.Code+"|"+f.AffectedRange] = f
	}
	for _, f := range second.QualityFlags {
		sm[f.Code+"|"+f.AffectedRange] = f
	}
	for k, f := range fm {
		if _, ok := sm[k]; ok {
			result.PersistentFlags = append(result.PersistentFlags, f)
		} else {
			result.RemovedFlags = append(result.RemovedFlags, f)
		}
	}
	for k, f := range sm {
		if _, ok := fm[k]; !ok {
			result.AddedFlags = append(result.AddedFlags, f)
		}
	}
	result.AddedFlags = domain.StableFlags(result.AddedFlags)
	result.RemovedFlags = domain.StableFlags(result.RemovedFlags)
	result.PersistentFlags = domain.StableFlags(result.PersistentFlags)
	return result, nil
}

func findRun(record domain.CaseRecord, id string) (domain.CrossdatingRun, bool) {
	for _, run := range record.Runs {
		if run.RunID == id {
			return run, true
		}
	}
	return domain.CrossdatingRun{}, false
}
func validRunDigest(record domain.CaseRecord, run domain.CrossdatingRun) bool {
	var revision domain.RingSeriesRevision
	found := false
	for _, r := range record.Revisions {
		if r.RevisionID == run.RevisionID {
			revision = r
			found = true
			break
		}
	}
	if !found {
		return false
	}
	if len(run.RankedCandidates) == 0 {
		return false
	}
	best := run.RankedCandidates[0]
	if run.BestCalendarRange != strconv.Itoa(best.StartYear)+"-"+strconv.Itoa(best.EndYear) || run.CorrelationScore != best.Correlation {
		return false
	}
	for index, candidate := range run.RankedCandidates {
		if candidate.Rank != index+1 {
			return false
		}
	}
	expected := domain.MustDigest(struct {
		RevisionDigest string                    `json:"revisionDigest"`
		ReferenceID    string                    `json:"referenceID"`
		Parameters     domain.AnalysisParameters `json:"parameters"`
		Candidates     []domain.Candidate        `json:"candidates"`
		Flags          []domain.QualityFlag      `json:"flags"`
	}{revision.ContentDigest, run.ReferenceChronologyID, run.ParameterSet, run.RankedCandidates, run.QualityFlags})
	return expected == run.ResultDigest
}
