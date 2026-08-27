package analysis

import (
	"fmt"
	"sort"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

type Engine struct{ chronology Chronology }

func NewEngine() *Engine { return &Engine{chronology: BuiltinChronology()} }

func (e *Engine) Chronology() Chronology { return e.chronology }

func (e *Engine) Analyze(caseID string, revision domain.RingSeriesRevision, params domain.AnalysisParameters, now time.Time) (domain.CrossdatingRun, error) {
	if params.MinimumOverlap < 5 {
		return domain.CrossdatingRun{}, domain.Invalid("minimumOverlap", "至少为 5")
	}
	if params.MaxCandidates < 1 || params.MaxCandidates > 20 {
		return domain.CrossdatingRun{}, domain.Invalid("maxCandidates", "必须在 1 到 20 之间")
	}
	sample := append([]float64(nil), revision.Widths...)
	if revision.MeasurementDirection == "bark-to-pith" {
		reverse(sample)
	}
	if len(sample) < params.MinimumOverlap {
		return domain.CrossdatingRun{}, domain.Invalid("widths", "不足最小重叠长度")
	}
	candidates := e.scoreCandidates(sample, params.MinimumOverlap)
	if len(candidates) == 0 {
		return domain.CrossdatingRun{}, domain.Conflict("参考年表不存在足够重叠")
	}
	if len(candidates) > params.MaxCandidates {
		candidates = candidates[:params.MaxCandidates]
	}
	for i := range candidates {
		candidates[i].Rank = i + 1
	}
	best := candidates[0]
	flags := DetectFlags(sample, best, params)
	digestInput := struct {
		RevisionDigest string                    `json:"revisionDigest"`
		ReferenceID    string                    `json:"referenceID"`
		Parameters     domain.AnalysisParameters `json:"parameters"`
		Candidates     []domain.Candidate        `json:"candidates"`
		Flags          []domain.QualityFlag      `json:"flags"`
	}{revision.ContentDigest, e.chronology.ID, params, candidates, flags}
	run := domain.CrossdatingRun{
		RunID: domain.NewID("run"), CaseID: caseID, RevisionID: revision.RevisionID,
		ReferenceChronologyID: e.chronology.ID, ParameterSet: params,
		RankedCandidates: candidates, BestCalendarRange: fmt.Sprintf("%d-%d", best.StartYear, best.EndYear),
		CorrelationScore: best.Correlation, QualityFlags: flags, ExecutedAt: now.UTC(),
	}
	run.ResultDigest = domain.MustDigest(digestInput)
	return run, nil
}

func (e *Engine) scoreCandidates(sample []float64, minimum int) []domain.Candidate {
	reference := e.chronology.Widths
	var candidates []domain.Candidate
	for start := 0; start < len(reference); start++ {
		overlap := len(sample)
		if len(reference)-start < overlap {
			overlap = len(reference) - start
		}
		if overlap < minimum {
			continue
		}
		a := RingIndex(sample[:overlap])
		b := RingIndex(reference[start : start+overlap])
		score := roundScore(Pearson(a, b))
		candidates = append(candidates, domain.Candidate{StartYear: e.chronology.StartYear + start, EndYear: e.chronology.StartYear + start + overlap - 1, Overlap: overlap, Correlation: score, ReferenceFrom: start})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Correlation == candidates[j].Correlation {
			if candidates[i].Overlap == candidates[j].Overlap {
				return candidates[i].StartYear < candidates[j].StartYear
			}
			return candidates[i].Overlap > candidates[j].Overlap
		}
		return candidates[i].Correlation > candidates[j].Correlation
	})
	return candidates
}

func reverse(values []float64) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
