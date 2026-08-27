package application

import (
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/analysis"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func (s *Service) Analyze(caseID string, input AnalyzeInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("分析基于过期档案版本")
	}
	if !domain.CanAnalyze(record.Case.Status) {
		return domain.CaseRecord{}, domain.Conflict("当前状态不能执行分析")
	}
	revision, ok := domain.ActiveRevision(record)
	if !ok {
		return domain.CaseRecord{}, domain.Conflict("没有当前序列修订")
	}
	params := analysis.DefaultParameters(record.Case.MinimumOverlap)
	if input.Parameters != nil {
		params = *input.Parameters
	}
	run, err := s.engine.Analyze(caseID, revision, params, s.now())
	if err != nil {
		return domain.CaseRecord{}, err
	}
	record.Runs = append(record.Runs, run)
	record.ActiveRunID = run.RunID
	target := domain.StatusAnalyzed
	for _, flag := range run.QualityFlags {
		if flag.Blocking {
			target = domain.StatusNeedsCorrection
			break
		}
	}
	if err := domain.SetStatus(&record, target); err != nil {
		return domain.CaseRecord{}, err
	}
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "CrossdatingAnalyzed", input.Actor, "执行确定性交叉定年", s.now())
	return result, err
}
