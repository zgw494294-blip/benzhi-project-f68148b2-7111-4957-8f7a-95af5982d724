package application

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func (s *Service) PreviewManifest(caseID string) (domain.EvidenceManifest, error) {
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.EvidenceManifest{}, err
	}
	if record.Manifest != nil {
		return *record.Manifest, nil
	}
	return s.buildManifest(record, "preview")
}

func (s *Service) ManifestReadiness(caseID string) (domain.ManifestReadiness, error) {
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.ManifestReadiness{}, err
	}
	if record.Manifest != nil && (record.Case.Status == domain.StatusFrozen || record.Case.Status == domain.StatusPublished) {
		sections := manifestSectionsFromManifest(*record.Manifest, record.Revisions)
		return domain.ManifestReadiness{Ready: true, Frozen: true, Sections: sections, Blockers: []string{}, Confirmation: manifestConfirmation(record, record.Manifest.ManifestDigest), Manifest: record.Manifest}, nil
	}
	report := domain.ManifestReadiness{Ready: true, Frozen: false, Blockers: []string{}, Sections: manifestSections(record)}
	if record.Case.Status != domain.StatusApproved {
		report.Blockers = append(report.Blockers, "档案状态不是 Approved")
	}
	if _, ok := domain.ActiveRevision(record); !ok {
		report.Blockers = append(report.Blockers, "缺少当前序列修订")
	}
	if _, ok := domain.ActiveRun(record); !ok {
		report.Blockers = append(report.Blockers, "缺少当前交叉定年分析")
	}
	if len(domain.UnresolvedFlags(record)) > 0 {
		report.Blockers = append(report.Blockers, "仍有未解决阻断争议")
	}
	if err := domain.EligibleForFreeze(record); err != nil {
		report.Blockers = append(report.Blockers, err.Error())
	}
	if err := domain.ValidateApprovedChecklist(record); err != nil {
		report.Blockers = append(report.Blockers, err.Error())
	}
	report.Ready = len(report.Blockers) == 0
	manifest, buildErr := s.buildManifest(record, "preview")
	if buildErr == nil {
		report.Manifest = &manifest
		report.Confirmation = manifestConfirmation(record, manifest.ManifestDigest)
	}
	for i := range report.Sections {
		report.Sections[i].Ready = report.Ready
		if !report.Ready {
			report.Sections[i].Message = "存在就绪阻断项"
		}
	}
	return report, nil
}

func (s *Service) Freeze(caseID string, input FreezeInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("冻结请求基于过期版本")
	}
	if err := domain.EligibleForFreeze(record); err != nil {
		return domain.CaseRecord{}, err
	}
	manifest, err := s.buildManifest(record, input.Actor)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if err := domain.SetStatus(&record, domain.StatusFrozen); err != nil {
		return domain.CaseRecord{}, err
	}
	record.Manifest = &manifest
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "EvidenceFrozen", input.Actor, "冻结确定性证据清单", s.now())
	if err == nil && result.Manifest != nil && result.Manifest.ManifestDigest != manifest.ManifestDigest {
		return domain.CaseRecord{}, domain.Conflict("persisted_manifest_mismatch：持久化清单摘要核对失败")
	}
	return result, err
}

func (s *Service) FreezeConfirmed(caseID string, input ConfirmedFreezeInput) (domain.CaseRecord, error) {
	if input.Confirmation == nil {
		return domain.CaseRecord{}, domain.Invalid("confirmation", "必须携带刚确认的冻结预览摘要")
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("冻结请求基于过期版本")
	}
	if err := domain.ValidateApprovedChecklist(record); err != nil {
		return domain.CaseRecord{}, err
	}
	manifest, err := s.buildManifest(record, input.Actor)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	expected := manifestConfirmation(record, manifest.ManifestDigest)
	if *input.Confirmation != expected {
		return domain.CaseRecord{}, domain.Conflict("manifest_preview_stale：确认摘要与当前投影不一致，请重新预览")
	}
	return s.Freeze(caseID, FreezeInput{CommandMeta: input.CommandMeta})
}

func (s *Service) buildManifest(record domain.CaseRecord, actor string) (domain.EvidenceManifest, error) {
	revision, ok := domain.ActiveRevision(record)
	if !ok {
		return domain.EvidenceManifest{}, domain.Conflict("缺少有效序列")
	}
	run, ok := domain.ActiveRun(record)
	if !ok {
		return domain.EvidenceManifest{}, domain.Conflict("缺少有效分析")
	}
	if len(record.Reviews) == 0 {
		return domain.EvidenceManifest{}, domain.Conflict("缺少复核决定")
	}
	checklist := domain.ReviewChecklist{}
	for i := len(record.ReviewChecklists) - 1; i >= 0; i-- {
		if record.ReviewChecklists[i].Status == "decided" {
			checklist = record.ReviewChecklists[i]
			break
		}
	}
	manifest := domain.EvidenceManifest{CaseSnapshot: record.Case, ActiveRevision: revision, ActiveRun: run, Resolutions: domain.CurrentResolutions(record), Review: record.Reviews[len(record.Reviews)-1], ReviewChecklist: checklist, FrozenBy: actor, FrozenAt: s.now().UTC()}
	sections := manifestSections(record)
	manifest.ManifestDigest = domain.MustDigest(sections)
	return manifest, nil
}

func manifestSections(record domain.CaseRecord) []domain.ManifestSection {
	sections := make([]domain.ManifestSection, 0, 6)
	appendSection := func(name string, count int, value any) {
		sections = append(sections, domain.ManifestSection{Name: name, Count: count, Digest: domain.MustDigest(value), Ready: true, Message: "就绪"})
	}
	appendSection("sampleMetadata", 1, record.Case)
	revision, revisionOK := domain.ActiveRevision(record)
	if revisionOK {
		appendSection("activeRevision", 1, revision)
	} else {
		appendSection("activeRevision", 0, nil)
	}
	run, runOK := domain.ActiveRun(record)
	if runOK {
		appendSection("activeAnalysis", 1, run)
	} else {
		appendSection("activeAnalysis", 0, nil)
	}
	resolutions := domain.CurrentResolutions(record)
	appendSection("disputeResolutions", len(resolutions), resolutions)
	if len(record.Reviews) > 0 {
		checklist := domain.ReviewChecklist{}
		for i := len(record.ReviewChecklists) - 1; i >= 0; i-- {
			if record.ReviewChecklists[i].Status == "decided" {
				checklist = record.ReviewChecklists[i]
				break
			}
		}
		appendSection("reviewDecision", 1, struct {
			Decision  domain.ReviewDecision  `json:"decision"`
			Checklist domain.ReviewChecklist `json:"checklist"`
		}{record.Reviews[len(record.Reviews)-1], checklist})
	} else {
		appendSection("reviewDecision", 0, nil)
	}
	appendSection("revisionLineage", len(record.Revisions), record.Revisions)
	return sections
}

func manifestSectionsFromManifest(manifest domain.EvidenceManifest, revisions []domain.RingSeriesRevision) []domain.ManifestSection {
	record := domain.CaseRecord{Case: manifest.CaseSnapshot, Revisions: revisions, Runs: []domain.CrossdatingRun{manifest.ActiveRun}, Resolutions: manifest.Resolutions, Reviews: []domain.ReviewDecision{manifest.Review}, ReviewChecklists: []domain.ReviewChecklist{manifest.ReviewChecklist}, ActiveRevisionID: manifest.ActiveRevision.RevisionID, ActiveRunID: manifest.ActiveRun.RunID}
	return manifestSections(record)
}

func manifestConfirmation(record domain.CaseRecord, manifestDigest string) domain.ManifestConfirmation {
	reviewID := ""
	if len(record.Reviews) > 0 {
		reviewID = record.Reviews[len(record.Reviews)-1].DecisionID
	}
	c := domain.ManifestConfirmation{CaseVersion: record.Case.Version, ActiveRevisionID: record.ActiveRevisionID, ActiveRunID: record.ActiveRunID, ReviewDecisionID: reviewID, ManifestDigest: manifestDigest}
	c.ConfirmationDigest = domain.MustDigest(struct {
		CaseVersion      int64  `json:"caseVersion"`
		ActiveRevisionID string `json:"activeRevisionID"`
		ActiveRunID      string `json:"activeRunID"`
		ReviewDecisionID string `json:"reviewDecisionID"`
		ManifestDigest   string `json:"manifestDigest"`
	}{c.CaseVersion, c.ActiveRevisionID, c.ActiveRunID, c.ReviewDecisionID, c.ManifestDigest})
	return c
}

func (s *Service) Publish(caseID string, input PublishInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("发布请求基于过期版本")
	}
	if record.Case.Status != domain.StatusFrozen || record.Manifest == nil {
		return domain.CaseRecord{}, domain.Conflict("仅已冻结档案可发布")
	}
	confidence := strings.TrimSpace(input.ConfidenceStatement)
	if confidence == "" {
		confidence = confidenceFor(record.Manifest.ActiveRun.CorrelationScore)
	}
	now := s.now().UTC()
	credential := domain.DatingCredential{CredentialID: domain.NewID("DENDRO"), CaseID: caseID, ManifestDigest: record.Manifest.ManifestDigest, CalendarConclusion: record.Manifest.ActiveRun.BestCalendarRange, ConfidenceStatement: confidence, IssuedBy: input.Actor, IssuedAt: now, Status: "valid"}
	credential.PayloadDigest = credentialPayloadDigest(credential)
	credential.SignatureDigest = s.sign(credential.PayloadDigest)
	record.Credential = &credential
	if err := domain.SetStatus(&record, domain.StatusPublished); err != nil {
		return domain.CaseRecord{}, err
	}
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "CredentialPublished", input.Actor, "发布定年结论凭据", now)
	return result, err
}

func confidenceFor(score float64) string {
	if score >= 0.8 {
		return "高置信：参考年表匹配稳定且已通过独立复核"
	}
	if score >= 0.55 {
		return "中等置信：候选匹配已完成争议处置并通过独立复核"
	}
	return "审慎置信：低相关证据已解释并经独立复核批准"
}

func credentialPayloadDigest(c domain.DatingCredential) string {
	return domain.MustDigest(struct {
		CredentialID        string `json:"credentialID"`
		CaseID              string `json:"caseID"`
		ManifestDigest      string `json:"manifestDigest"`
		CalendarConclusion  string `json:"calendarConclusion"`
		ConfidenceStatement string `json:"confidenceStatement"`
		IssuedBy            string `json:"issuedBy"`
		IssuedAt            any    `json:"issuedAt"`
		Status              string `json:"status"`
	}{c.CredentialID, c.CaseID, c.ManifestDigest, c.CalendarConclusion, c.ConfidenceStatement, c.IssuedBy, c.IssuedAt, c.Status})
}
func (s *Service) sign(digest string) string {
	mac := hmac.New(sha256.New, []byte(s.signingKey))
	mac.Write([]byte(digest))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) Verify(input VerifyInput) (VerificationResult, error) {
	var supplied domain.DatingCredential
	var record domain.CaseRecord
	var err error
	if input.Payload != nil {
		supplied = *input.Payload
		record, err = s.repo.FindCredential(supplied.CredentialID)
	} else if strings.TrimSpace(input.CredentialID) != "" {
		record, err = s.repo.FindCredential(strings.TrimSpace(input.CredentialID))
		if err == nil && record.Credential != nil {
			supplied = *record.Credential
		}
	} else {
		return VerificationResult{}, domain.Invalid("credentialID", "编号或完整载荷至少提供一项")
	}
	if err != nil {
		return VerificationResult{Valid: false, Current: false, CredentialID: supplied.CredentialID, Message: "编号不存在", Category: "not_found"}, nil
	}
	result := VerificationResult{CredentialID: supplied.CredentialID, CaseID: record.Case.CaseID}
	if supplied.PayloadDigest != credentialPayloadDigest(supplied) {
		result.Message = "载荷摘要不匹配"
		result.Category = "payload_tampered"
		return result, nil
	}
	if !hmac.Equal([]byte(supplied.SignatureDigest), []byte(s.sign(supplied.PayloadDigest))) {
		result.Message = "签发摘要不匹配"
		result.Category = "signature_mismatch"
		return result, nil
	}
	if record.Manifest == nil || record.Manifest.ManifestDigest != supplied.ManifestDigest {
		result.Message = "冻结清单摘要不匹配"
		result.Category = "payload_tampered"
		return result, nil
	}
	if record.Credential == nil {
		result.Message = "当前档案没有已发布凭据"
		return result, nil
	}
	stored, _ := json.Marshal(record.Credential)
	given, _ := json.Marshal(supplied)
	if !hmac.Equal(stored, given) {
		result.Valid = true
		result.Message = "载荷真实但不是当前登记载荷"
		result.Category = "authentic_not_current"
		return result, nil
	}
	result.Valid = true
	result.Current = record.Case.Status == domain.StatusPublished && record.Credential.Status == "valid"
	if result.Current {
		result.Message = "凭据真实、完整且当前有效"
		result.Category = "current_valid"
	} else {
		result.Message = "凭据真实但当前无效"
		result.Category = "authentic_not_current"
	}
	return result, nil
}

func (s *Service) VerifyBatch(input BatchVerifyInput) (BatchVerificationResult, error) {
	if len(input.Entries) > 50 {
		return BatchVerificationResult{}, domain.Invalid("entries", "一次最多验证 50 项")
	}
	if len(input.Entries) == 0 {
		return BatchVerificationResult{}, domain.Invalid("entries", "至少提供一项")
	}
	total := 0
	for _, entry := range input.Entries {
		total += len(entry)
		if len(entry) > 16*1024 {
			return BatchVerificationResult{}, domain.Invalid("entries", "单项不能超过 16 KiB")
		}
	}
	if total > 256*1024 {
		return BatchVerificationResult{}, domain.Invalid("entries", "总内容不能超过 256 KiB")
	}
	result := BatchVerificationResult{Items: make([]BatchVerificationItem, 0, len(input.Entries)), Counts: map[string]int{}}
	for i, entry := range input.Entries {
		trimmed := strings.TrimSpace(entry)
		var verification VerificationResult
		if strings.HasPrefix(trimmed, "{") {
			var payload domain.DatingCredential
			if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
				verification = VerificationResult{Message: "该项 JSON 格式无效", Category: "format_error"}
			} else {
				verification, _ = s.Verify(VerifyInput{Payload: &payload})
			}
		} else if trimmed == "" {
			verification = VerificationResult{Message: "该项为空", Category: "format_error"}
		} else {
			verification, _ = s.Verify(VerifyInput{CredentialID: trimmed})
		}
		result.Counts[verification.Category]++
		result.Items = append(result.Items, BatchVerificationItem{Index: i, Input: entry, VerificationResult: verification})
	}
	return result, nil
}
