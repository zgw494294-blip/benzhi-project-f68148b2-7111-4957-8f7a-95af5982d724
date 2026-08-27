package application

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func (s *Service) MetadataDuplicates(caseID string, input ReviseMetadataInput) ([]DuplicateCandidate, error) {
	record, err := s.repo.Get(caseID)
	if err != nil {
		return nil, err
	}
	revised := metadataCase(record, input)
	if record.Case.Status != domain.StatusDraft || len(record.Revisions) != 0 {
		return nil, domain.Conflict("metadata_locked：档案已有序列修订或已离开 Draft")
	}
	if err := domain.ValidateNewCase(revised); err != nil {
		return nil, err
	}
	all, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	result := make([]DuplicateCandidate, 0)
	for _, candidate := range all {
		if candidate.Case.CaseID == caseID {
			continue
		}
		fields := matchingDuplicateFields(revised, candidate.Case)
		if len(fields) == 4 {
			result = append(result, DuplicateCandidate{CaseID: candidate.Case.CaseID, MatchingFields: fields})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CaseID < result[j].CaseID })
	return result, nil
}

func matchingDuplicateFields(a, b domain.DatingCase) []string {
	fields := make([]string, 0, 4)
	if domain.NormalizeMetadata(a.SiteName) == domain.NormalizeMetadata(b.SiteName) {
		fields = append(fields, "siteName")
	}
	if domain.NormalizeMetadata(a.StructureRef) == domain.NormalizeMetadata(b.StructureRef) {
		fields = append(fields, "structureRef")
	}
	if domain.NormalizeMetadata(a.SamplingPosition) == domain.NormalizeMetadata(b.SamplingPosition) {
		fields = append(fields, "samplingPosition")
	}
	if strings.TrimSpace(a.SampledAt) == strings.TrimSpace(b.SampledAt) {
		fields = append(fields, "sampledAt")
	}
	return fields
}

func metadataCase(record domain.CaseRecord, input ReviseMetadataInput) domain.DatingCase {
	c := record.Case
	c.SiteName = strings.TrimSpace(input.SiteName)
	c.StructureRef = strings.TrimSpace(input.StructureRef)
	c.SamplingPosition = strings.TrimSpace(input.SamplingPosition)
	c.SpeciesAssessment = strings.TrimSpace(input.SpeciesAssessment)
	c.SampledAt = strings.TrimSpace(input.SampledAt)
	c.MinimumOverlap = input.MinimumOverlap
	return c
}

func (s *Service) ReviseMetadata(caseID string, input ReviseMetadataInput) (domain.CaseRecord, error) {
	if err := validateMeta(input.CommandMeta); err != nil {
		return domain.CaseRecord{}, err
	}
	record, err := s.repo.Get(caseID)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if record.Case.Version != input.ExpectedVersion {
		return domain.CaseRecord{}, domain.Conflict("metadata_version_conflict：校订基于过期档案版本")
	}
	revised := metadataCase(record, input)
	if err := domain.ValidateMetadataRevision(record, revised, input.Reason); err != nil {
		return domain.CaseRecord{}, err
	}
	candidates, err := s.MetadataDuplicates(caseID, input)
	if err != nil {
		return domain.CaseRecord{}, err
	}
	if len(candidates) > 0 && strings.TrimSpace(input.DuplicateConfirmationReason) == "" {
		ids := make([]string, len(candidates))
		for i := range candidates {
			ids[i] = candidates[i].CaseID
		}
		return domain.CaseRecord{}, domain.NewDetailedError("duplicate_confirmation_required", "发现疑似重复档案："+strings.Join(ids, "、")+"；请填写确认理由", map[string]any{"candidates": candidates})
	}
	now := s.now().UTC()
	before := record.Case
	revised.Version = before.Version
	revised.UpdatedAt = before.UpdatedAt
	record.Case = revised
	record.MetadataRevisions = append(record.MetadataRevisions, domain.MetadataRevision{RevisionID: domain.NewID("metadata"), Before: before, After: revised, Reason: strings.TrimSpace(input.Reason), RevisedBy: input.Actor, RevisedAt: now})
	summary := fmt.Sprintf("校订档案元数据：%s；%s → %s", strings.TrimSpace(input.Reason), metadataSummary(before), metadataSummary(revised))
	result, _, err := s.repo.Save(record, input.ExpectedVersion, input.IdempotencyKey, "MetadataRevised", input.Actor, summary, now)
	return result, err
}

func metadataSummary(c domain.DatingCase) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%d", c.SiteName, c.StructureRef, c.SamplingPosition, c.SpeciesAssessment, c.SampledAt, c.MinimumOverlap)
}

var allStatuses = []domain.Status{domain.StatusDraft, domain.StatusSeriesSubmitted, domain.StatusAnalyzed, domain.StatusNeedsCorrection, domain.StatusUnderReview, domain.StatusApproved, domain.StatusFrozen, domain.StatusPublished}

func (s *Service) QueryQueue(query QueueQuery) (QueueResult, error) {
	if query.PageSize == 0 {
		query.PageSize = 20
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		return QueueResult{}, domain.Invalid("pageSize", "必须在 1 到 100 之间")
	}
	valid := map[domain.Status]bool{}
	for _, status := range allStatuses {
		valid[status] = true
	}
	selected := map[domain.Status]bool{}
	for _, status := range query.Statuses {
		if !valid[status] {
			return QueueResult{}, domain.Invalid("status", "包含非法状态 "+string(status))
		}
		selected[status] = true
	}
	from, to, err := parseQueueTimes(query.UpdatedFrom, query.UpdatedTo)
	if err != nil {
		return QueueResult{}, err
	}
	records, err := s.repo.List()
	if err != nil {
		return QueueResult{}, err
	}
	filtered := make([]domain.CaseRecord, 0)
	for _, record := range records {
		if len(selected) > 0 && !selected[record.Case.Status] {
			continue
		}
		if domain.NormalizeMetadata(query.Site) != "" && !strings.Contains(domain.NormalizeMetadata(record.Case.SiteName), domain.NormalizeMetadata(query.Site)) {
			continue
		}
		if domain.NormalizeMetadata(query.StructureRef) != "" && domain.NormalizeMetadata(record.Case.StructureRef) != domain.NormalizeMetadata(query.StructureRef) {
			continue
		}
		if !from.IsZero() && record.Case.UpdatedAt.Before(from) {
			continue
		}
		if !to.IsZero() && record.Case.UpdatedAt.After(to) {
			continue
		}
		blocking := len(domain.UnresolvedFlags(record))
		if query.HasUnresolvedBlocking != nil && (blocking > 0) != *query.HasUnresolvedBlocking {
			continue
		}
		filtered = append(filtered, record)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Case.UpdatedAt.Equal(filtered[j].Case.UpdatedAt) {
			return filtered[i].Case.CaseID < filtered[j].Case.CaseID
		}
		return filtered[i].Case.UpdatedAt.After(filtered[j].Case.UpdatedAt)
	})
	stats := QueueStatistics{StatusCounts: map[domain.Status]int{}, StagnationGroups: map[string]int{"0-2天": 0, "3-7天": 0, "8-30天": 0, "31天以上": 0}}
	for _, status := range allStatuses {
		stats.StatusCounts[status] = 0
	}
	for _, record := range filtered {
		stats.StatusCounts[record.Case.Status]++
		n := len(domain.UnresolvedFlags(record))
		stats.BlockingTotal += n
		stats.StagnationGroups[stagnationGroup(s.now().UTC(), record.Case.UpdatedAt)]++
	}
	start := 0
	if query.Cursor != "" {
		start, err = locateCursor(filtered, query.Cursor)
		if err != nil {
			return QueueResult{}, err
		}
	}
	end := start + query.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	rows := make([]QueueRow, 0, end-start)
	for _, record := range filtered[start:end] {
		summary := ""
		if len(record.Timeline) > 0 {
			summary = record.Timeline[len(record.Timeline)-1].Summary
		}
		rows = append(rows, QueueRow{Record: record, LastEventSummary: summary, NextAction: domain.NextLegalAction(record), UnresolvedBlocking: len(domain.UnresolvedFlags(record))})
	}
	result := QueueResult{Rows: rows, Statistics: stats}
	if end < len(filtered) {
		last := filtered[end-1]
		result.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(last.Case.UpdatedAt.UnixNano(), 10) + "|" + last.Case.CaseID))
	}
	return result, nil
}

func parseQueueTimes(fromText, toText string) (time.Time, time.Time, error) {
	var from, to time.Time
	if strings.TrimSpace(fromText) != "" {
		var err error
		from, err = parseQueueTime(fromText, false)
		if err != nil {
			return from, to, domain.Invalid("updatedFrom", "必须为 YYYY-MM-DD 或 RFC3339")
		}
	}
	if strings.TrimSpace(toText) != "" {
		var err error
		to, err = parseQueueTime(toText, true)
		if err != nil {
			return from, to, domain.Invalid("updatedTo", "必须为 YYYY-MM-DD 或 RFC3339")
		}
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return from, to, domain.Invalid("updatedRange", "起始时间不得晚于结束时间")
	}
	return from, to, nil
}

func parseQueueTime(text string, endOfDay bool) (time.Time, error) {
	text = strings.TrimSpace(text)
	if parsed, err := time.Parse(time.RFC3339, text); err == nil {
		return parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", text)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return parsed, nil
}

func locateCursor(records []domain.CaseRecord, cursor string) (int, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, domain.Invalid("cursor", "格式无效")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return 0, domain.Invalid("cursor", "格式无效")
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, domain.Invalid("cursor", "格式无效")
	}
	for i, r := range records {
		if r.Case.CaseID == parts[1] && r.Case.UpdatedAt.UnixNano() == nanos {
			return i + 1, nil
		}
	}
	return 0, domain.Invalid("cursor", "游标已过期")
}

func stagnationGroup(now, updated time.Time) string {
	days := int(now.Sub(updated).Hours() / 24)
	if days < 3 {
		return "0-2天"
	}
	if days < 8 {
		return "3-7天"
	}
	if days < 31 {
		return "8-30天"
	}
	return "31天以上"
}
