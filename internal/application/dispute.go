package application

import (
	"fmt"
	"strings"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/analysis"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func (s *Service) ResolveDisputesBatch(caseID string, input BatchResolveDisputesInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	if len(input.Items) < 1 || len(input.Items) > 100 {
		return domain.CaseRecord{}, domain.Invalid("items", "数量必须在 1 到 100 之间")
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("batch_dispute_stale：批量处置基于过期档案版本")
	}
	if record.Case.Status != domain.StatusNeedsCorrection && record.Case.Status != domain.StatusAnalyzed {
		return domain.CaseRecord{}, domain.Conflict("当前状态不能批量处置争议")
	}
	if input.RunID == "" || input.RunID != record.ActiveRunID {
		return domain.CaseRecord{}, domain.Conflict("active_run_mismatch：所有提示必须属于当前分析")
	}
	existing := map[string]bool{}
	for _, r := range domain.CurrentResolutions(record) {
		existing[r.FlagCode+"|"+r.AffectedRange] = true
	}
	seen := map[string]bool{}
	refs := make([]string, 0, len(input.EvidenceRefs))
	refSeen := map[string]bool{}
	for _, ref := range input.EvidenceRefs {
		ref = strings.TrimSpace(ref)
		if ref != "" && !refSeen[ref] {
			refSeen[ref] = true
			refs = append(refs, ref)
		}
	}
	now := s.now().UTC()
	pending := make([]domain.DisputeResolution, 0, len(input.Items))
	for _, item := range input.Items {
		k := item.FlagCode + "|" + item.AffectedRange
		if seen[k] {
			return domain.CaseRecord{}, domain.Invalid("items", "包含重复提示 "+k)
		}
		seen[k] = true
		if existing[k] {
			return domain.CaseRecord{}, domain.Conflict("dispute_already_resolved：提示已被其他命令处置 " + k)
		}
		if _, ok := domain.FindFlag(record, item.FlagCode, item.AffectedRange); !ok {
			return domain.CaseRecord{}, domain.Conflict("flag_mismatch：提示代码与影响范围不匹配 " + k)
		}
		if strings.TrimSpace(item.Rationale) == "" {
			return domain.CaseRecord{}, domain.Invalid("rationale", "每项必须填写独立理由")
		}
		pending = append(pending, domain.DisputeResolution{ResolutionID: domain.NewID("resolution"), CaseID: caseID, RunID: input.RunID, FlagCode: item.FlagCode, AffectedRange: item.AffectedRange, Disposition: "explain", Rationale: strings.TrimSpace(item.Rationale), EvidenceRefs: append([]string(nil), refs...), ResolvedBy: input.Actor, ResolvedAt: now})
	}
	record.Resolutions = append(record.Resolutions, pending...)
	if len(domain.UnresolvedFlags(record)) == 0 && record.Case.Status == domain.StatusNeedsCorrection {
		record.Case.Status = domain.StatusAnalyzed
	}
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "DisputesBatchResolved", input.Actor, fmt.Sprintf("批量解释 %d 项争议，共享证据 %d 项", len(pending), len(refs)), now)
	return result, err
}

func (s *Service) ResolveDispute(caseID string, input ResolveDisputeInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("争议处置基于过期档案版本")
	}
	if record.Case.Status != domain.StatusNeedsCorrection && record.Case.Status != domain.StatusAnalyzed {
		return domain.CaseRecord{}, domain.Conflict("当前状态不能处置争议")
	}
	if _, ok := domain.FindFlag(record, input.FlagCode, input.AffectedRange); !ok {
		return domain.CaseRecord{}, domain.NotFound("当前分析中不存在该提示")
	}
	for _, existing := range domain.CurrentResolutions(record) {
		if existing.FlagCode == input.FlagCode && existing.AffectedRange == input.AffectedRange {
			return domain.CaseRecord{}, domain.Conflict("该争议已经处置")
		}
	}
	if input.Disposition != "explain" && input.Disposition != "replace" {
		return domain.CaseRecord{}, domain.Invalid("disposition", "必须为 explain 或 replace")
	}
	if strings.TrimSpace(input.Rationale) == "" {
		return domain.CaseRecord{}, domain.Invalid("rationale", "必须给出证据化说明")
	}
	resolution := domain.DisputeResolution{ResolutionID: domain.NewID("resolution"), CaseID: caseID, RunID: record.ActiveRunID, FlagCode: input.FlagCode, AffectedRange: input.AffectedRange, Disposition: input.Disposition, Rationale: strings.TrimSpace(input.Rationale), EvidenceRefs: append([]string(nil), input.EvidenceRefs...), ResolvedBy: input.Actor, ResolvedAt: s.now().UTC()}
	if input.Disposition == "replace" {
		if input.Replacement == nil {
			return domain.CaseRecord{}, domain.Invalid("replacement", "替代处置必须提供序列")
		}
		replacement := domain.RingSeriesRevision{RevisionID: domain.NewID("rev"), CaseID: caseID, ParentRevisionID: record.ActiveRevisionID, Unit: input.Replacement.Unit, MeasurementDirection: input.Replacement.MeasurementDirection, Widths: append([]float64(nil), input.Replacement.Widths...), ChangeReason: input.Replacement.ChangeReason, SubmittedBy: input.Actor, SubmittedAt: s.now().UTC()}
		if err := domain.ValidateRevision(replacement, record.Case.MinimumOverlap); err != nil {
			return domain.CaseRecord{}, err
		}
		replacement.ContentDigest = domain.MustDigest(struct {
			Unit      string    `json:"unit"`
			Direction string    `json:"direction"`
			Widths    []float64 `json:"widths"`
		}{replacement.Unit, replacement.MeasurementDirection, replacement.Widths})
		resolution.ReplacementRevisionID = replacement.RevisionID
		record.Resolutions = append(record.Resolutions, resolution)
		record.Revisions = append(record.Revisions, replacement)
		record.ActiveRevisionID = replacement.RevisionID
		record.LastSubmitter = input.Actor
		run, err := s.engine.Analyze(caseID, replacement, analysis.DefaultParameters(record.Case.MinimumOverlap), s.now())
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
		record.Case.Status = target
	} else {
		record.Resolutions = append(record.Resolutions, resolution)
		if len(domain.UnresolvedFlags(record)) == 0 {
			record.Case.Status = domain.StatusAnalyzed
		}
	}
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "DisputeResolved", input.Actor, "处置交叉定年争议", s.now())
	return result, err
}
