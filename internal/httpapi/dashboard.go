package httpapi

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/crleonard/pingtower/internal/model"
	"github.com/crleonard/pingtower/internal/store"
)

//go:embed templates/*.html
var templateFS embed.FS

type dashboardStats struct {
	Total   int
	Healthy int
	Down    int
	Pending int
}

type dashboardCheckView struct {
	ID                 string
	Name               string
	URL                string
	ExpectedStatusCode int
	IntervalSeconds    int
	TimeoutSeconds     int
	Paused             bool
	LastStatus         string
	LastStatusCode     int
	LastResponseMS     int64
	LastCheckedLabel   string
	LastError          string
	StatusClass        string
}

type dashboardPageData struct {
	Stats       dashboardStats
	Checks      []dashboardCheckView
	FormValues  map[string]string
	FormError   string
	CurrentTime string
	RefreshURL  string
}

type checkDetailPageData struct {
	Check       dashboardCheckView
	Results     []dashboardResultView
	CurrentTime string
	RefreshURL  string
}

type dashboardResultView struct {
	Status         string
	StatusClass    string
	StatusCode     int
	ResponseMS     int64
	CheckedAtLabel string
	ErrorMessage   string
	ResponseSample string
}

var dashboardTemplates = template.Must(template.New("").
	Funcs(template.FuncMap{
		"formatStatus": prettyStatus,
	}).
	ParseFS(templateFS, "templates/*.html"))

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.renderDashboard(w, r, dashboardPageData{})
}

func (s *Server) handleCreateCheckForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderDashboard(w, r, dashboardPageData{
			FormError: "Could not read the form submission.",
		})
		return
	}

	req := createCheckRequest{
		Name:               strings.TrimSpace(r.FormValue("name")),
		URL:                strings.TrimSpace(r.FormValue("url")),
		IntervalSeconds:    parseIntOrZero(r.FormValue("interval_seconds")),
		TimeoutSeconds:     parseIntOrZero(r.FormValue("timeout_seconds")),
		ExpectedStatusCode: parseIntOrZero(r.FormValue("expected_status_code")),
	}

	if err := s.validateCreateCheck(req); err != nil {
		s.renderDashboard(w, r, dashboardPageData{
			FormError: err.Error(),
			FormValues: map[string]string{
				"name":                 req.Name,
				"url":                  req.URL,
				"interval_seconds":     strconv.Itoa(req.IntervalSeconds),
				"timeout_seconds":      strconv.Itoa(req.TimeoutSeconds),
				"expected_status_code": strconv.Itoa(req.ExpectedStatusCode),
			},
		})
		return
	}

	check, err := s.buildCheck(req)
	if err != nil {
		s.renderDashboard(w, r, dashboardPageData{
			FormError: err.Error(),
		})
		return
	}

	created, err := s.store.CreateCheck(check)
	if err != nil {
		s.renderDashboard(w, r, dashboardPageData{
			FormError: "Could not create the monitor.",
		})
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/checks/%s/view", created.ID), http.StatusSeeOther)
}

func (s *Server) handleDashboardCheckAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/dashboard/checks/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	checkID := parts[0]
	action := parts[1]

	var paused bool
	switch action {
	case "pause":
		paused = true
	case "resume":
		paused = false
	case "delete":
		if err := s.store.DeleteCheck(checkID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "failed to delete check", http.StatusInternalServerError)
			return
		}
		redirectTo := strings.TrimSpace(r.FormValue("redirect_to"))
		if redirectTo == "" || strings.Contains(redirectTo, checkID) {
			redirectTo = "/"
		}
		http.Redirect(w, r, redirectTo, http.StatusSeeOther)
		return
	default:
		http.NotFound(w, r)
		return
	}

	if _, err := s.store.SetCheckPaused(checkID, paused); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to update check", http.StatusInternalServerError)
		return
	}

	redirectTo := strings.TrimSpace(r.FormValue("redirect_to"))
	if redirectTo == "" {
		redirectTo = "/"
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

func (s *Server) handleCheckDetailPage(w http.ResponseWriter, _ *http.Request, checkID string) {
	check, err := s.store.GetCheck(checkID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, nil)
			return
		}
		http.Error(w, "failed to load check", http.StatusInternalServerError)
		return
	}

	results, err := s.store.ListResults(checkID)
	if err != nil {
		http.Error(w, "failed to load history", http.StatusInternalServerError)
		return
	}

	data := checkDetailPageData{
		Check:       newDashboardCheckView(check),
		Results:     make([]dashboardResultView, 0, len(results)),
		CurrentTime: time.Now().UTC().Format(time.RFC1123),
		RefreshURL:  fmt.Sprintf("/checks/%s/view", checkID),
	}

	for _, result := range results {
		data.Results = append(data.Results, dashboardResultView{
			Status:         result.Status,
			StatusClass:    statusClass(result.Status),
			StatusCode:     result.StatusCode,
			ResponseMS:     result.ResponseMS,
			CheckedAtLabel: humanizeTime(result.CheckedAt),
			ErrorMessage:   result.ErrorMessage,
			ResponseSample: result.ResponseSample,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTemplates.ExecuteTemplate(w, "detail.html", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) renderDashboard(w http.ResponseWriter, r *http.Request, state dashboardPageData) {
	checks, err := s.store.ListChecks()
	if err != nil {
		http.Error(w, "failed to load dashboard", http.StatusInternalServerError)
		return
	}

	data := dashboardPageData{
		Stats:       summarizeChecks(checks),
		Checks:      make([]dashboardCheckView, 0, len(checks)),
		FormValues:  withFormDefaults(state.FormValues, s),
		FormError:   state.FormError,
		CurrentTime: time.Now().UTC().Format(time.RFC1123),
		RefreshURL:  r.URL.RequestURI(),
	}

	for _, check := range checks {
		data.Checks = append(data.Checks, newDashboardCheckView(check))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTemplates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func summarizeChecks(checks []model.Check) dashboardStats {
	stats := dashboardStats{Total: len(checks)}
	for _, check := range checks {
		switch check.LastStatus {
		case "healthy":
			stats.Healthy++
		case "down":
			stats.Down++
		default:
			stats.Pending++
		}
	}
	return stats
}

func newDashboardCheckView(check model.Check) dashboardCheckView {
	return dashboardCheckView{
		ID:                 check.ID,
		Name:               check.Name,
		URL:                check.URL,
		ExpectedStatusCode: check.ExpectedStatusCode,
		IntervalSeconds:    check.IntervalSeconds,
		TimeoutSeconds:     check.TimeoutSeconds,
		Paused:             check.Paused,
		LastStatus:         effectiveStatus(check),
		LastStatusCode:     check.LastStatusCode,
		LastResponseMS:     check.LastResponseMS,
		LastCheckedLabel:   humanizeTime(check.LastCheckedAt),
		LastError:          check.LastError,
		StatusClass:        statusClass(effectiveStatus(check)),
	}
}

func prettyStatus(status string) string {
	switch status {
	case "healthy":
		return "Healthy"
	case "down":
		return "Down"
	case "paused":
		return "Paused"
	default:
		return "Pending"
	}
}

func statusClass(status string) string {
	switch status {
	case "healthy":
		return "status-healthy"
	case "down":
		return "status-down"
	case "paused":
		return "status-paused"
	default:
		return "status-pending"
	}
}

func effectiveStatus(check model.Check) string {
	if check.Paused {
		return "paused"
	}
	return firstNonEmpty(check.LastStatus, "pending")
}

func humanizeTime(ts time.Time) string {
	if ts.IsZero() {
		return "Never"
	}
	return ts.Local().Format("2006-01-02 15:04:05 MST")
}

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func withFormDefaults(values map[string]string, s *Server) map[string]string {
	if values == nil {
		values = map[string]string{}
	}
	if values["interval_seconds"] == "" {
		values["interval_seconds"] = strconv.Itoa(int(s.cfg.DefaultInterval.Seconds()))
	}
	if values["timeout_seconds"] == "" {
		values["timeout_seconds"] = strconv.Itoa(int(s.cfg.DefaultTimeout.Seconds()))
	}
	if values["expected_status_code"] == "" {
		values["expected_status_code"] = strconv.Itoa(http.StatusOK)
	}
	return values
}

func parseIntOrZero(value string) int {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
