package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

const maxRequestBody int64 = 2 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return domain.Invalid("body", "JSON 无效或字段不受支持："+err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return domain.Invalid("body", "只能包含一个 JSON 对象")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeData(w http.ResponseWriter, status int, value any) {
	writeJSON(w, status, map[string]any{"data": value})
}
func writeProblem(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := domain.ErrorCode(err)
	switch code {
	case "invalid_argument":
		status = http.StatusBadRequest
	case "not_found":
		status = http.StatusNotFound
	case "conflict":
		status = http.StatusConflict
	case "forbidden":
		status = http.StatusForbidden
	case "duplicate_confirmation_required":
		status = http.StatusConflict
	case "run_not_found":
		status = http.StatusNotFound
	case "run_ownership_mismatch":
		status = http.StatusForbidden
	case "run_digest_mismatch":
		status = http.StatusConflict
	}
	message := "服务内部错误"
	var details any
	var de *domain.Error
	if errors.As(err, &de) {
		message = de.Message
		details = de.Details
	}
	problem := map[string]any{"code": code, "message": message}
	if details != nil {
		problem["details"] = details
	}
	writeJSON(w, status, map[string]any{"error": problem})
}
