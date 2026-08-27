package application

import (
	"fmt"
	"strings"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func (s *Service) StartReview(caseID string, input StartReviewInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("进入复核请求已过期")
	}
	if err := domain.EligibleForReview(record); err != nil {
		return domain.CaseRecord{}, err
	}
	if len(domain.OpenRemediationItems(record)) > 0 {
		return domain.CaseRecord{}, domain.Conflict("remediation_incomplete：仍有复核整改项未回应")
	}
	for i := range record.ReviewChecklists {
		if record.ReviewChecklists[i].Status == "decided" {
			record.ReviewChecklists[i].Status = "superseded"
		}
	}
	if err := domain.SetStatus(&record, domain.StatusUnderReview); err != nil {
		return domain.CaseRecord{}, err
	}
	record.ReviewChecklists = append(record.ReviewChecklists, domain.NewReviewChecklist(record, s.now()))
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "ReviewStarted", input.Actor, "提交独立复核", s.now())
	return result, err
}

func (s *Service) CurrentReviewChecklist(caseID string) (domain.ReviewChecklist, error) {
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.ReviewChecklist{}, err
	}
	for i := len(record.ReviewChecklists) - 1; i >= 0; i-- {
		if record.ReviewChecklists[i].Status == "current" {
			return record.ReviewChecklists[i], nil
		}
	}
	return domain.ReviewChecklist{}, domain.NotFound("当前档案没有待决复核清单")
}

func (s *Service) DecideReviewChecklist(caseID string, input ReviewChecklistInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("review_version_conflict：复核决定针对过期版本")
	}
	items, err := domain.ValidateChecklistDecision(record, input.ChecklistID, input.Outcome, input.Actor, input.Items)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	now := s.now().UTC()
	findings := make([]string, 0)
	for _, item := range items {
		if item.Outcome == "return" {
			findings = append(findings, item.Label+"："+strings.TrimSpace(item.Finding)+"；整改目标："+strings.TrimSpace(item.RemediationTarget))
		}
	}
	if input.Outcome == "approve" {
		findings = append(findings, "全部结构化必查项通过")
	}
	decision := domain.ReviewDecision{DecisionID: domain.NewID("review"), CaseID: caseID, CaseVersion: input.ExpectedVersion, RevisionID: record.ActiveRevisionID, RunID: record.ActiveRunID, Outcome: input.Outcome, Findings: strings.Join(findings, "；"), ReviewerID: input.Actor, DecidedAt: now}
	record.Reviews = append(record.Reviews, decision)
	for i := range record.ReviewChecklists {
		if record.ReviewChecklists[i].ChecklistID == input.ChecklistID {
			record.ReviewChecklists[i].Items = items
			record.ReviewChecklists[i].Status = "decided"
			record.ReviewChecklists[i].ReviewerID = input.Actor
			record.ReviewChecklists[i].DecidedAt = now
		}
	}
	target := domain.StatusApproved
	if input.Outcome == "return" {
		target = domain.StatusNeedsCorrection
	}
	if err := domain.SetStatus(&record, target); err != nil {
		return domain.CaseRecord{}, err
	}
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "ReviewChecklistDecided", input.Actor, fmt.Sprintf("结构化独立复核：%s，核对 %d 项", input.Outcome, len(items)), now)
	return result, err
}

func (s *Service) RespondRemediation(caseID string, input RemediationResponseInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("整改回应基于过期档案版本")
	}
	if record.Case.Status != domain.StatusNeedsCorrection && record.Case.Status != domain.StatusAnalyzed {
		return domain.CaseRecord{}, domain.Conflict("当前状态不能回应整改项")
	}
	if strings.TrimSpace(input.Response) == "" {
		return domain.CaseRecord{}, domain.Invalid("response", "不能为空")
	}
	if len(input.EvidenceEvents) == 0 {
		return domain.CaseRecord{}, domain.Invalid("evidenceEvents", "至少引用一个退回后业务事件")
	}
	var checklist *domain.ReviewChecklist
	for i := range record.ReviewChecklists {
		if record.ReviewChecklists[i].ChecklistID == input.ChecklistID {
			checklist = &record.ReviewChecklists[i]
			break
		}
	}
	if checklist == nil {
		return domain.CaseRecord{}, domain.NotFound("复核清单不存在")
	}
	foundItem := false
	for _, item := range checklist.Items {
		if item.ItemCode == input.ItemCode && item.Outcome == "return" {
			foundItem = true
		}
	}
	if !foundItem {
		return domain.CaseRecord{}, domain.Invalid("itemCode", "不是该清单的整改项")
	}
	for _, r := range record.RemediationResponses {
		if r.ChecklistID == input.ChecklistID && r.ItemCode == input.ItemCode {
			return domain.CaseRecord{}, domain.Conflict("该整改项已经回应")
		}
	}
	seen := map[int64]bool{}
	events := make([]int64, 0, len(input.EvidenceEvents))
	for _, sequence := range input.EvidenceEvents {
		if seen[sequence] {
			continue
		}
		seen[sequence] = true
		valid := false
		for _, event := range record.Timeline {
			validKind := event.Kind == "SeriesSubmitted" || event.Kind == "CrossdatingAnalyzed" || event.Kind == "DisputeResolved" || event.Kind == "DisputesBatchResolved"
			if event.Sequence == sequence && event.At.After(checklist.DecidedAt) && validKind {
				valid = true
				break
			}
		}
		if !valid {
			return domain.CaseRecord{}, domain.Invalid("evidenceEvents", "必须引用退回之后产生的有效业务事件")
		}
		events = append(events, sequence)
	}
	now := s.now().UTC()
	record.RemediationResponses = append(record.RemediationResponses, domain.RemediationResponse{ChecklistID: input.ChecklistID, ItemCode: input.ItemCode, Response: strings.TrimSpace(input.Response), EvidenceEvents: events, RespondedBy: input.Actor, RespondedAt: now})
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "RemediationResponded", input.Actor, "回应复核整改项 "+input.ItemCode, now)
	return result, err
}

func (s *Service) DecideReview(caseID string, input ReviewInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("复核决定针对过期版本")
	}
	if record.Case.Status != domain.StatusUnderReview {
		return domain.CaseRecord{}, domain.Conflict("档案不在复核中")
	}
	if input.Actor == record.LastSubmitter {
		return domain.CaseRecord{}, domain.Forbidden("复核员不得与最近序列提交者相同")
	}
	if input.Outcome != "approve" && input.Outcome != "return" {
		return domain.CaseRecord{}, domain.Invalid("outcome", "必须为 approve 或 return")
	}
	if strings.TrimSpace(input.Findings) == "" {
		return domain.CaseRecord{}, domain.Invalid("findings", "不能为空")
	}
	decision := domain.ReviewDecision{DecisionID: domain.NewID("review"), CaseID: caseID, CaseVersion: input.ExpectedVersion, RevisionID: record.ActiveRevisionID, RunID: record.ActiveRunID, Outcome: input.Outcome, Findings: strings.TrimSpace(input.Findings), ReviewerID: input.Actor, DecidedAt: s.now().UTC()}
	record.Reviews = append(record.Reviews, decision)
	target := domain.StatusApproved
	if input.Outcome == "return" {
		target = domain.StatusNeedsCorrection
	}
	if err := domain.SetStatus(&record, target); err != nil {
		return domain.CaseRecord{}, err
	}
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "ReviewDecided", input.Actor, "记录独立复核决定", s.now())
	return result, err
}
