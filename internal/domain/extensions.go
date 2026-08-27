package domain

import (
	"sort"
	"strings"
	"time"
)

var requiredReviewItems = []ReviewChecklistItem{
	{ItemCode: "revision", Label: "当前序列修订与内容摘要"},
	{ItemCode: "parameters", Label: "交叉定年参数与结果摘要"},
	{ItemCode: "candidate", Label: "最佳候选及排序结论"},
	{ItemCode: "disputes", Label: "阻断争议闭合情况"},
	{ItemCode: "lineage", Label: "不可变修订谱系"},
}

func NormalizeMetadata(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func ValidateMetadataRevision(record CaseRecord, revised DatingCase, reason string) error {
	if record.Case.Status != StatusDraft || len(record.Revisions) != 0 {
		return Conflict("metadata_locked：档案已有序列修订或已离开 Draft")
	}
	revised.CaseID = record.Case.CaseID
	revised.CreatedBy = record.Case.CreatedBy
	revised.CreatedAt = record.Case.CreatedAt
	if err := ValidateNewCase(revised); err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		return Invalid("reason", "不能为空")
	}
	return nil
}

func NewReviewChecklist(record CaseRecord, now time.Time) ReviewChecklist {
	items := make([]ReviewChecklistItem, len(requiredReviewItems))
	copy(items, requiredReviewItems)
	return ReviewChecklist{ChecklistID: NewID("checklist"), CaseID: record.Case.CaseID, CaseVersion: record.Case.Version + 1, RevisionID: record.ActiveRevisionID, RunID: record.ActiveRunID, Items: items, Status: "current", CreatedAt: now.UTC()}
}

func ValidateChecklistDecision(record CaseRecord, checklistID, outcome, reviewer string, submitted []ReviewChecklistItem) ([]ReviewChecklistItem, error) {
	if record.Case.Status != StatusUnderReview {
		return nil, Conflict("档案不在复核中")
	}
	if reviewer == record.LastSubmitter {
		return nil, Forbidden("复核员不得与最近序列提交者相同")
	}
	var current *ReviewChecklist
	for i := range record.ReviewChecklists {
		if record.ReviewChecklists[i].ChecklistID == checklistID && record.ReviewChecklists[i].Status == "current" {
			current = &record.ReviewChecklists[i]
			break
		}
	}
	if current == nil {
		return nil, Conflict("review_checklist_stale：清单不存在或已被取代")
	}
	if current.CaseVersion != record.Case.Version || current.RevisionID != record.ActiveRevisionID || current.RunID != record.ActiveRunID {
		return nil, Conflict("review_checklist_stale：清单证据绑定已过期")
	}
	if outcome != "approve" && outcome != "return" {
		return nil, Invalid("outcome", "必须为 approve 或 return")
	}
	byCode := make(map[string]ReviewChecklistItem, len(submitted))
	for _, item := range submitted {
		if _, exists := byCode[item.ItemCode]; exists {
			return nil, Invalid("items", "包含重复必查项")
		}
		byCode[item.ItemCode] = item
	}
	if len(byCode) != len(current.Items) {
		return nil, Invalid("items", "必须逐项提交全部必查项")
	}
	result := make([]ReviewChecklistItem, 0, len(current.Items))
	for _, required := range current.Items {
		item, ok := byCode[required.ItemCode]
		if !ok {
			return nil, Invalid("items", "缺少必查项 "+required.ItemCode)
		}
		item.Label = required.Label
		if item.Outcome != "pass" && item.Outcome != "return" {
			return nil, Invalid("items", required.ItemCode+" 的结论必须为 pass 或 return")
		}
		if item.Outcome == "return" && (strings.TrimSpace(item.Finding) == "" || strings.TrimSpace(item.RemediationTarget) == "") {
			return nil, Invalid("items", required.ItemCode+" 的退回发现和整改目标不能为空")
		}
		if outcome == "approve" && item.Outcome != "pass" {
			return nil, Conflict("批准要求所有必查项通过")
		}
		result = append(result, item)
	}
	if outcome == "return" {
		hasReturn := false
		for _, item := range result {
			hasReturn = hasReturn || item.Outcome == "return"
		}
		if !hasReturn {
			return nil, Invalid("items", "退回决定至少包含一个具体整改项")
		}
	}
	return result, nil
}

func OpenRemediationItems(record CaseRecord) []ReviewChecklistItem {
	responded := map[string]bool{}
	for _, response := range record.RemediationResponses {
		responded[response.ChecklistID+"|"+response.ItemCode] = true
	}
	var result []ReviewChecklistItem
	for _, checklist := range record.ReviewChecklists {
		if checklist.Status != "decided" {
			continue
		}
		for _, item := range checklist.Items {
			if item.Outcome == "return" && !responded[checklist.ChecklistID+"|"+item.ItemCode] {
				result = append(result, item)
			}
		}
	}
	return result
}

func ValidateApprovedChecklist(record CaseRecord) error {
	if len(record.Reviews) == 0 {
		return Conflict("review_checklist_missing：缺少批准决定")
	}
	review := record.Reviews[len(record.Reviews)-1]
	for i := len(record.ReviewChecklists) - 1; i >= 0; i-- {
		checklist := record.ReviewChecklists[i]
		if checklist.Status != "decided" {
			continue
		}
		if checklist.RevisionID != review.RevisionID || checklist.RunID != review.RunID || checklist.CaseVersion != review.CaseVersion || checklist.ReviewerID != review.ReviewerID {
			return Conflict("review_checklist_stale：批准清单与决定绑定不一致")
		}
		if len(checklist.Items) != len(requiredReviewItems) {
			return Conflict("review_checklist_incomplete：批准清单缺项")
		}
		for _, item := range checklist.Items {
			if item.Outcome != "pass" {
				return Conflict("review_checklist_incomplete：批准清单存在未通过项")
			}
		}
		return nil
	}
	return Conflict("review_checklist_missing：缺少结构化批准清单")
}

func NextLegalAction(record CaseRecord) string {
	switch record.Case.Status {
	case StatusDraft:
		return "校订元数据或预检并提交序列"
	case StatusSeriesSubmitted:
		return "执行交叉定年"
	case StatusNeedsCorrection:
		if len(UnresolvedFlags(record)) > 0 {
			return "处置阻断争议"
		}
		if len(OpenRemediationItems(record)) > 0 {
			return "回应复核整改项"
		}
		return "重新提交复核"
	case StatusAnalyzed:
		return "提交独立复核"
	case StatusUnderReview:
		return "完成结构化复核清单"
	case StatusApproved:
		return "预览并冻结证据清单"
	case StatusFrozen:
		return "签发定年凭据"
	case StatusPublished:
		return "验证已发布凭据"
	default:
		return "查看档案"
	}
}

func StableFlags(flags []QualityFlag) []QualityFlag {
	out := append([]QualityFlag(nil), flags...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Code == out[j].Code {
			return out[i].AffectedRange < out[j].AffectedRange
		}
		return out[i].Code < out[j].Code
	})
	return out
}
