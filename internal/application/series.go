package application

import (
	"strings"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func (s *Service) SubmitSeries(caseID string, input SubmitSeriesInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("提交基于过期档案版本")
	}
	if input.PreflightDigest != "" {
		expected := domain.MustDigest(struct {
			CaseVersion      int64     `json:"caseVersion"`
			ParentRevisionID string    `json:"parentRevisionID"`
			Unit             string    `json:"unit"`
			Direction        string    `json:"direction"`
			Widths           []float64 `json:"widths"`
			ContentDigest    string    `json:"contentDigest"`
		}{record.Case.Version, record.ActiveRevisionID, input.Unit, input.MeasurementDirection, input.Widths, seriesContentDigest(input.Unit, input.MeasurementDirection, input.Widths)})
		if expected != input.PreflightDigest {
			return domain.CaseRecord{}, domain.Conflict("preflight_mismatch：输入、父修订或档案版本已变化，请重新预检")
		}
	}
	if !domain.CanSubmitSeries(record.Case.Status) {
		return domain.CaseRecord{}, domain.Conflict("当前状态禁止提交序列")
	}
	if record.Case.Status == domain.StatusFrozen || record.Case.Status == domain.StatusPublished {
		return domain.CaseRecord{}, domain.Conflict("冻结后禁止业务内容变更")
	}
	revision := domain.RingSeriesRevision{RevisionID: domain.NewID("rev"), CaseID: caseID, ParentRevisionID: record.ActiveRevisionID, Unit: input.Unit, MeasurementDirection: input.MeasurementDirection, Widths: append([]float64(nil), input.Widths...), ChangeReason: strings.TrimSpace(input.ChangeReason), SubmittedBy: input.Actor, SubmittedAt: s.now().UTC()}
	if err := domain.ValidateRevision(revision, record.Case.MinimumOverlap); err != nil {
		return domain.CaseRecord{}, err
	}
	revision.ContentDigest = domain.MustDigest(struct {
		Unit      string    `json:"unit"`
		Direction string    `json:"direction"`
		Widths    []float64 `json:"widths"`
	}{revision.Unit, revision.MeasurementDirection, revision.Widths})
	record.Revisions = append(record.Revisions, revision)
	record.ActiveRevisionID = revision.RevisionID
	record.ActiveRunID = ""
	record.LastSubmitter = input.Actor
	if record.Case.Status != domain.StatusSeriesSubmitted {
		if err := domain.SetStatus(&record, domain.StatusSeriesSubmitted); err != nil {
			return domain.CaseRecord{}, err
		}
	}
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "SeriesSubmitted", input.Actor, "提交不可变年轮序列修订", s.now())
	return result, err
}
