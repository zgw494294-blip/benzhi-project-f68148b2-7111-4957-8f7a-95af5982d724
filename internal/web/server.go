package web

import (
	"net/http"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/application"
)

type Server struct {
	app *application.Service
	mux *http.ServeMux
}

func NewServer(app *application.Service) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return securityHeaders(s.mux) }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.IndexHandler)
	s.mux.HandleFunc("GET /static/{name}", s.AssetHandler)
	s.mux.HandleFunc("GET /api/health", s.HealthHandler)
	s.mux.HandleFunc("GET /api/cases", s.ListCasesHandler)
	s.mux.HandleFunc("GET /api/cases/queue", s.QueueHandler)
	s.mux.HandleFunc("POST /api/cases", s.CreateCaseHandler)
	s.mux.HandleFunc("GET /api/cases/{caseID}", s.GetCaseHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/metadata/duplicates", s.MetadataDuplicatesHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/metadata/revise", s.ReviseMetadataHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/series/preflight", s.SeriesPreflightHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/series", s.SubmitSeriesHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/analyze", s.AnalyzeHandler)
	s.mux.HandleFunc("GET /api/cases/{caseID}/runs", s.RunHistoryHandler)
	s.mux.HandleFunc("GET /api/cases/{caseID}/runs/compare", s.CompareRunsHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/disputes", s.ResolveDisputeHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/disputes/batch", s.ResolveDisputesBatchHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/review/start", s.StartReviewHandler)
	s.mux.HandleFunc("GET /api/cases/{caseID}/review/checklist", s.ReviewChecklistHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/review/decide", s.DecideReviewHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/review/remediations", s.RemediationResponseHandler)
	s.mux.HandleFunc("GET /api/cases/{caseID}/manifest", s.ManifestPreviewHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/freeze", s.FreezeHandler)
	s.mux.HandleFunc("POST /api/cases/{caseID}/publish", s.PublishHandler)
	s.mux.HandleFunc("POST /api/verify", s.VerifyCredentialHandler)
	s.mux.HandleFunc("POST /api/verify/batch", s.VerifyCredentialBatchHandler)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	data, err := assets.ReadFile("static/index.html")
	if err != nil {
		writeProblem(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
func (s *Server) AssetHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name != "app.js" && name != "style.css" {
		http.NotFound(w, r)
		return
	}
	data, err := assets.ReadFile("static/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if name == "app.js" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(data)
}

func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
}
