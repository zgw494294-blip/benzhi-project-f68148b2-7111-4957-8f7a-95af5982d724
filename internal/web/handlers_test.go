package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/analysis"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/application"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/persistence"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	repo, err := persistence.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return NewServer(application.NewService(repo, analysis.NewEngine(), "test")).Handler()
}
func TestIndexAndStrictJSON(t *testing.T) {
	handler := testHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "<body>") {
		t.Fatal("工作台 HTML 未提供")
	}
	bad := httptest.NewRequest(http.MethodPost, "/api/cases", strings.NewReader(`{"caseID":"x","unknown":true}`))
	bad.Header.Set("Content-Type", "application/json")
	badResponse := httptest.NewRecorder()
	handler.ServeHTTP(badResponse, bad)
	if badResponse.Code != http.StatusBadRequest {
		t.Fatalf("未知字段状态码=%d", badResponse.Code)
	}
	if badResponse.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("缺少安全响应头")
	}
}
