package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/crleonard/pingtower/internal/config"
	"github.com/crleonard/pingtower/internal/model"
	"github.com/crleonard/pingtower/internal/store"
)

// Triggerer runs an on-demand check evaluation.
type Triggerer interface {
	RunNow(checkID string) (model.Result, error)
}

type Server struct {
	cfg       config.Config
	logger    *log.Logger
	store     store.Store
	triggerer Triggerer
	mux       *http.ServeMux
}

// SetTriggerer wires the monitor service so the trigger endpoint works.
func (s *Server) SetTriggerer(t Triggerer) {
	s.triggerer = t
}

type createCheckRequest struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	IntervalSeconds    int    `json:"interval_seconds"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	ExpectedStatusCode int    `json:"expected_status_code"`
}

func NewServer(cfg config.Config, logger *log.Logger, dataStore store.Store) *Server {
	s := &Server{
		cfg:    cfg,
		logger: logger,
		store:  dataStore,
		mux:    http.NewServeMux(),
	}

	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.loggingMiddleware(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleDashboard)
	s.mux.HandleFunc("POST /dashboard/checks", s.handleCreateCheckForm)
	s.mux.HandleFunc("POST /dashboard/checks/", s.handleDashboardCheckAction)
	s.mux.HandleFunc("GET /checks/", s.handleCheckSubresource)
	s.mux.HandleFunc("POST /checks/", s.handleCheckPOSTSubresource)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /checks", s.handleListChecks)
	s.mux.HandleFunc("POST /checks", s.handleCreateCheck)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (s *Server) handleListChecks(w http.ResponseWriter, _ *http.Request) {
	checks, err := s.store.ListChecks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list checks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"checks": checks,
	})
}

func (s *Server) handleCreateCheck(w http.ResponseWriter, r *http.Request) {
	var req createCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	check, err := s.buildCheck(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := s.store.CreateCheck(check)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create check")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleCheckSubresource(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/checks/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	checkID := parts[0]
	if len(parts) == 1 {
		s.handleGetCheck(w, r, checkID)
		return
	}

	if len(parts) == 2 && parts[1] == "view" {
		s.handleCheckDetailPage(w, r, checkID)
		return
	}

	if len(parts) == 2 && parts[1] == "history" {
		s.handleCheckHistory(w, r, checkID)
		return
	}

	if len(parts) == 2 && parts[1] == "pause" && r.Method == http.MethodPost {
		s.handleSetCheckPaused(w, r, checkID, true)
		return
	}

	if len(parts) == 2 && parts[1] == "resume" && r.Method == http.MethodPost {
		s.handleSetCheckPaused(w, r, checkID, false)
		return
	}

	if len(parts) == 2 && parts[1] == "delete" && r.Method == http.MethodDelete {
		s.handleDeleteCheck(w, r, checkID)
		return
	}

	http.NotFound(w, r)
}

func (s *Server) handleGetCheck(w http.ResponseWriter, _ *http.Request, checkID string) {
	check, err := s.store.GetCheck(checkID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get check")
		return
	}
	writeJSON(w, http.StatusOK, check)
}

func (s *Server) handleCheckHistory(w http.ResponseWriter, _ *http.Request, checkID string) {
	results, err := s.store.ListResults(checkID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get history")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"check_id": checkID,
		"results":  results,
	})
}

func (s *Server) handleSetCheckPaused(w http.ResponseWriter, _ *http.Request, checkID string, paused bool) {
	check, err := s.store.SetCheckPaused(checkID, paused)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update check")
		return
	}
	writeJSON(w, http.StatusOK, check)
}

func (s *Server) handleDeleteCheck(w http.ResponseWriter, _ *http.Request, checkID string) {
	if err := s.store.DeleteCheck(checkID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete check")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCheckPOSTSubresource(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/checks/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if parts[1] == "trigger" {
		s.handleTriggerCheck(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleTriggerCheck(w http.ResponseWriter, _ *http.Request, checkID string) {
	if s.triggerer == nil {
		writeError(w, http.StatusNotImplemented, "trigger not available")
		return
	}
	result, err := s.triggerer.RunNow(checkID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to trigger check")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.logger.Printf(
			"request completed method=%s path=%s status=%d content_length=%s",
			r.Method,
			r.URL.Path,
			rec.statusCode,
			strconv.FormatInt(r.ContentLength, 10),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
