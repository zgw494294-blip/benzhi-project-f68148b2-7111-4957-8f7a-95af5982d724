package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/analysis"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/application"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
	webdelivery "benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/web"
)

type apiEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func runSelfcheck(cfg config) error {
	dataDir, err := os.MkdirTemp("", "dendro-selfcheck-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dataDir)
	delivery, err := build(dataDir)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("自检监听失败: %w", err)
	}
	server := webdelivery.NewHTTPServer(cfg.Addr, delivery.Handler())
	serverError := make(chan error, 1)
	go func() { serverError <- server.Serve(listener) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	base := "http://" + cfg.Addr
	client := &http.Client{Timeout: 5 * time.Second}
	if err := waitReady(client, base); err != nil {
		return err
	}
	caseID := "case-selfcheck-001"
	record, err := postRecord(client, base+"/api/cases", map[string]any{"caseID": caseID, "siteName": "自检古建", "structureRef": "正殿-梁架-01", "samplingPosition": "构件端部", "speciesAssessment": "松属", "sampledAt": "2026-08-27", "minimumOverlap": 20, "createdBy": "测量员-自检", "idempotencyKey": "self-create"})
	if err != nil {
		return err
	}
	record, err = postRecord(client, base+"/api/cases/"+caseID+"/metadata/revise", map[string]any{"expectedVersion": record.Case.Version, "idempotencyKey": "self-metadata", "actor": "测量员-自检", "siteName": "自检古建", "structureRef": "正殿-梁架-01", "samplingPosition": "构件端部距榫卯 20cm", "speciesAssessment": "松属", "sampledAt": "2026-08-27", "minimumOverlap": 20, "reason": "补充精确采样位置"})
	if err != nil {
		return err
	}
	chronology := analysis.BuiltinChronology()
	widths := append([]float64(nil), chronology.Widths[20:65]...)
	rawWidths := strings.Trim(strings.ReplaceAll(fmt.Sprint(widths), " ", ","), "[]")
	var preflight application.SeriesPreflightResult
	if err := postJSON(client, base+"/api/cases/"+caseID+"/series/preflight", map[string]any{"caseVersion": record.Case.Version, "rawText": rawWidths, "unit": "mm", "measurementDirection": "pith-to-bark"}, &preflight); err != nil {
		return err
	}
	if len(preflight.Errors) > 0 {
		return fmt.Errorf("序列预检未通过: %v", preflight.Errors)
	}
	record, err = postRecord(client, base+"/api/cases/"+caseID+"/series", map[string]any{"expectedVersion": record.Case.Version, "idempotencyKey": "self-series", "actor": "测量员-自检", "unit": "mm", "measurementDirection": "pith-to-bark", "widths": preflight.Widths, "preflightDigest": preflight.PreflightDigest, "changeReason": "自检参考片段"})
	if err != nil {
		return err
	}
	record, err = postRecord(client, base+"/api/cases/"+caseID+"/analyze", map[string]any{"expectedVersion": record.Case.Version, "idempotencyKey": "self-analysis", "actor": "分析员-自检"})
	if err != nil {
		return err
	}
	runID := record.ActiveRunID
	var comparison application.RunComparison
	if err := getJSON(client, base+"/api/cases/"+caseID+"/runs/compare?firstRunID="+runID+"&secondRunID="+runID, &comparison); err != nil {
		return err
	}
	if comparison.FirstRunID != runID {
		return fmt.Errorf("运行历史对比失败")
	}
	if flags := domain.UnresolvedFlags(record); len(flags) > 0 {
		items := make([]map[string]any, 0, len(flags))
		for _, flag := range flags {
			items = append(items, map[string]any{"flagCode": flag.Code, "affectedRange": flag.AffectedRange, "rationale": "自检中对显微图像与原始测量轨迹完成复核"})
		}
		record, err = postRecord(client, base+"/api/cases/"+caseID+"/disputes/batch", map[string]any{"expectedVersion": record.Case.Version, "idempotencyKey": "self-resolve-batch", "actor": "分析员-自检", "runID": record.ActiveRunID, "items": items, "evidenceRefs": []string{"SELF-EVIDENCE-01", "SELF-EVIDENCE-01"}})
		if err != nil {
			return err
		}
	}
	record, err = postRecord(client, base+"/api/cases/"+caseID+"/review/start", map[string]any{"expectedVersion": record.Case.Version, "idempotencyKey": "self-review-start", "actor": "分析员-自检"})
	if err != nil {
		return err
	}
	checklist := record.ReviewChecklists[len(record.ReviewChecklists)-1]
	items := append([]domain.ReviewChecklistItem(nil), checklist.Items...)
	for i := range items {
		items[i].Outcome = "pass"
	}
	record, err = postRecord(client, base+"/api/cases/"+caseID+"/review/decide", map[string]any{"expectedVersion": record.Case.Version, "idempotencyKey": "self-review-approve", "actor": "独立复核员-自检", "outcome": "approve", "checklistID": checklist.ChecklistID, "items": items})
	if err != nil {
		return err
	}
	var readiness domain.ManifestReadiness
	if err := getJSON(client, base+"/api/cases/"+caseID+"/manifest", &readiness); err != nil {
		return err
	}
	if !readiness.Ready {
		return fmt.Errorf("冻结就绪核验失败: %v", readiness.Blockers)
	}
	record, err = postRecord(client, base+"/api/cases/"+caseID+"/freeze", map[string]any{"expectedVersion": record.Case.Version, "idempotencyKey": "self-freeze", "actor": "档案负责人-自检", "confirmation": readiness.Confirmation})
	if err != nil {
		return err
	}
	record, err = postRecord(client, base+"/api/cases/"+caseID+"/publish", map[string]any{"expectedVersion": record.Case.Version, "idempotencyKey": "self-publish", "actor": "档案负责人-自检", "confidenceStatement": "自检凭据：完整流程通过"})
	if err != nil {
		return err
	}
	if record.Case.Status != domain.StatusPublished || record.Credential == nil {
		return fmt.Errorf("自检未推进至 Published")
	}
	var verification struct {
		Valid   bool   `json:"valid"`
		Current bool   `json:"current"`
		Message string `json:"message"`
	}
	if err := postJSON(client, base+"/api/verify", map[string]any{"payload": record.Credential}, &verification); err != nil {
		return err
	}
	if !verification.Valid || !verification.Current {
		return fmt.Errorf("凭据验证失败: %s", verification.Message)
	}
	payloadJSON, _ := json.Marshal(record.Credential)
	var batch application.BatchVerificationResult
	if err := postJSON(client, base+"/api/verify/batch", map[string]any{"entries": []string{record.Credential.CredentialID, string(payloadJSON), "DENDRO-UNKNOWN", "{"}}, &batch); err != nil {
		return err
	}
	if len(batch.Items) != 4 || batch.Counts["current_valid"] != 2 || batch.Counts["not_found"] != 1 || batch.Counts["format_error"] != 1 {
		return fmt.Errorf("混合批量验证分类异常: %v", batch.Counts)
	}
	var queue application.QueueResult
	if err := getJSON(client, base+"/api/cases/queue?status=Published&hasUnresolvedBlocking=false", &queue); err != nil {
		return err
	}
	if len(queue.Rows) != 1 || queue.Rows[0].Record.Case.CaseID != caseID {
		return fmt.Errorf("发布队列筛选异常")
	}
	fmt.Printf("自检通过：%s，凭据 %s，结论 %s\n", record.Case.Status, record.Credential.CredentialID, record.Credential.CalendarConclusion)
	return nil
}

func getJSON(client *http.Client, url string, target any) error {
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var envelope apiEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if envelope.Error != nil {
			return fmt.Errorf("HTTP %d %s: %s", response.StatusCode, envelope.Error.Code, envelope.Error.Message)
		}
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	return json.Unmarshal(envelope.Data, target)
}

func waitReady(client *http.Client, base string) error {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(base + "/api/health")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("自检服务未就绪")
}

func postRecord(client *http.Client, url string, input any) (domain.CaseRecord, error) {
	var record domain.CaseRecord
	err := postJSON(client, url, input, &record)
	return record, err
}
func postJSON(client *http.Client, url string, input, target any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var envelope apiEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if envelope.Error != nil {
			return fmt.Errorf("HTTP %d %s: %s", response.StatusCode, envelope.Error.Code, envelope.Error.Message)
		}
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	return json.Unmarshal(envelope.Data, target)
}
