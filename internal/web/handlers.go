package web

import (
	"net/http"
	"strconv"
	"strings"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/application"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	writeData(w, http.StatusOK, s.app.Health())
}
func (s *Server) ListCasesHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		s.QueueHandler(w, r)
		return
	}
	records, err := s.app.ListCases()
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, records)
}
func (s *Server) QueueHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := application.QueueQuery{Site: q.Get("site"), StructureRef: q.Get("structureRef"), UpdatedFrom: q.Get("updatedFrom"), UpdatedTo: q.Get("updatedTo"), Cursor: q.Get("cursor")}
	for _, statuses := range q["status"] {
		for _, status := range strings.Split(strings.TrimSpace(statuses), ",") {
			if strings.TrimSpace(status) != "" {
				query.Statuses = append(query.Statuses, domain.Status(strings.TrimSpace(status)))
			}
		}
	}
	if size := q.Get("pageSize"); size != "" {
		n, err := strconv.Atoi(size)
		if err != nil {
			writeProblem(w, domain.Invalid("pageSize", "必须为整数"))
			return
		}
		query.PageSize = n
	}
	if raw := q.Get("hasUnresolvedBlocking"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			writeProblem(w, domain.Invalid("hasUnresolvedBlocking", "必须为 true 或 false"))
			return
		}
		query.HasUnresolvedBlocking = &value
	}
	result, err := s.app.QueryQueue(query)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) CreateCaseHandler(w http.ResponseWriter, r *http.Request) {
	var in application.CreateCaseInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	record, err := s.app.CreateCase(in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusCreated, record)
}
func (s *Server) GetCaseHandler(w http.ResponseWriter, r *http.Request) {
	record, err := s.app.GetCase(r.PathValue("caseID"))
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, record)
}
func (s *Server) SubmitSeriesHandler(w http.ResponseWriter, r *http.Request) {
	var in application.SubmitSeriesInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	record, err := s.app.SubmitSeries(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, record)
}
func (s *Server) SeriesPreflightHandler(w http.ResponseWriter, r *http.Request) {
	var in application.SeriesPreflightInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	result, err := s.app.PreflightSeries(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) MetadataDuplicatesHandler(w http.ResponseWriter, r *http.Request) {
	var in application.ReviseMetadataInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	result, err := s.app.MetadataDuplicates(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) ReviseMetadataHandler(w http.ResponseWriter, r *http.Request) {
	var in application.ReviseMetadataInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	result, err := s.app.ReviseMetadata(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	var in application.AnalyzeInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	record, err := s.app.Analyze(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, record)
}
func (s *Server) ResolveDisputeHandler(w http.ResponseWriter, r *http.Request) {
	var in application.ResolveDisputeInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	record, err := s.app.ResolveDispute(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, record)
}
func (s *Server) ResolveDisputesBatchHandler(w http.ResponseWriter, r *http.Request) {
	var in application.BatchResolveDisputesInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	result, err := s.app.ResolveDisputesBatch(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) RunHistoryHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.RunHistory(r.PathValue("caseID"))
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) CompareRunsHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.CompareRuns(r.PathValue("caseID"), r.URL.Query().Get("firstRunID"), r.URL.Query().Get("secondRunID"))
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) StartReviewHandler(w http.ResponseWriter, r *http.Request) {
	var in application.StartReviewInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	record, err := s.app.StartReview(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, record)
}
func (s *Server) DecideReviewHandler(w http.ResponseWriter, r *http.Request) {
	var in application.ReviewChecklistInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	record, err := s.app.DecideReviewChecklist(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, record)
}
func (s *Server) ReviewChecklistHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.CurrentReviewChecklist(r.PathValue("caseID"))
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) RemediationResponseHandler(w http.ResponseWriter, r *http.Request) {
	var in application.RemediationResponseInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	result, err := s.app.RespondRemediation(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) ManifestPreviewHandler(w http.ResponseWriter, r *http.Request) {
	manifest, err := s.app.ManifestReadiness(r.PathValue("caseID"))
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, manifest)
}
func (s *Server) FreezeHandler(w http.ResponseWriter, r *http.Request) {
	var in application.ConfirmedFreezeInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	record, err := s.app.FreezeConfirmed(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, record)
}
func (s *Server) VerifyCredentialBatchHandler(w http.ResponseWriter, r *http.Request) {
	var in application.BatchVerifyInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	result, err := s.app.VerifyBatch(in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
func (s *Server) PublishHandler(w http.ResponseWriter, r *http.Request) {
	var in application.PublishInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	record, err := s.app.Publish(r.PathValue("caseID"), in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, record)
}
func (s *Server) VerifyCredentialHandler(w http.ResponseWriter, r *http.Request) {
	var in application.VerifyInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}
	result, err := s.app.Verify(in)
	if err != nil {
		writeProblem(w, err)
		return
	}
	writeData(w, http.StatusOK, result)
}
