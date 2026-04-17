package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/crleonard/pingtower/internal/config"
	"github.com/crleonard/pingtower/internal/model"
	"github.com/crleonard/pingtower/internal/store"
)

func TestCreateAndListChecks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataStore, err := store.NewFileStore(filepath.Join(dir, "pingtower.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	server := NewServer(config.Load(), log.New(io.Discard, "", 0), dataStore)

	body := map[string]any{
		"name": "Example",
		"url":  "https://example.com/health",
	}
	payload, _ := json.Marshal(body)

	createReq := httptest.NewRequest(http.MethodPost, "/checks", bytes.NewReader(payload))
	createRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRes, createReq)

	if createRes.Code != http.StatusCreated {
		t.Fatalf("POST /checks status = %d, want %d", createRes.Code, http.StatusCreated)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/checks", nil)
	listRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRes, listReq)

	if listRes.Code != http.StatusOK {
		t.Fatalf("GET /checks status = %d, want %d", listRes.Code, http.StatusOK)
	}
}

func TestDashboardRendersHTML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataStore, err := store.NewFileStore(filepath.Join(dir, "pingtower.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	server := NewServer(config.Load(), log.New(io.Discard, "", 0), dataStore)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", res.Code, http.StatusOK)
	}

	if got := res.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
}

func TestDashboardFormRedirectsToDetailPage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataStore, err := store.NewFileStore(filepath.Join(dir, "pingtower.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	server := NewServer(config.Load(), log.New(io.Discard, "", 0), dataStore)

	form := "name=Home+Page&url=https%3A%2F%2Fexample.com%2Fhealth&interval_seconds=60&timeout_seconds=10&expected_status_code=200"
	req := httptest.NewRequest(http.MethodPost, "/dashboard/checks", bytes.NewBufferString(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("POST /dashboard/checks status = %d, want %d", res.Code, http.StatusSeeOther)
	}

	location := res.Header().Get("Location")
	if location == "" {
		t.Fatalf("Location header is empty")
	}
}

func TestPauseAndResumeCheckViaDashboard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataStore, err := store.NewFileStore(filepath.Join(dir, "pingtower.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	server := NewServer(config.Load(), log.New(io.Discard, "", 0), dataStore)

	created, err := dataStore.CreateCheck(model.Check{
		Name:               "Example",
		URL:                "https://example.com",
		IntervalSeconds:    60,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("CreateCheck() error = %v", err)
	}

	pauseReq := httptest.NewRequest(http.MethodPost, "/dashboard/checks/"+created.ID+"/pause", bytes.NewBufferString(url.Values{
		"redirect_to": []string{"/"},
	}.Encode()))
	pauseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	pauseRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(pauseRes, pauseReq)

	if pauseRes.Code != http.StatusSeeOther {
		t.Fatalf("pause status = %d, want %d", pauseRes.Code, http.StatusSeeOther)
	}

	check, err := dataStore.GetCheck(created.ID)
	if err != nil {
		t.Fatalf("GetCheck() error = %v", err)
	}
	if !check.Paused {
		t.Fatalf("check.Paused = false, want true")
	}

	resumeReq := httptest.NewRequest(http.MethodPost, "/dashboard/checks/"+created.ID+"/resume", bytes.NewBufferString(url.Values{
		"redirect_to": []string{"/"},
	}.Encode()))
	resumeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resumeRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(resumeRes, resumeReq)

	if resumeRes.Code != http.StatusSeeOther {
		t.Fatalf("resume status = %d, want %d", resumeRes.Code, http.StatusSeeOther)
	}

	check, err = dataStore.GetCheck(created.ID)
	if err != nil {
		t.Fatalf("GetCheck() error = %v", err)
	}
	if check.Paused {
		t.Fatalf("check.Paused = true, want false")
	}
}

func TestDeleteCheckViaDashboard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataStore, err := store.NewFileStore(filepath.Join(dir, "pingtower.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	server := NewServer(config.Load(), log.New(io.Discard, "", 0), dataStore)

	created, err := dataStore.CreateCheck(model.Check{
		Name:               "Delete Me",
		URL:                "https://example.com",
		IntervalSeconds:    60,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("CreateCheck() error = %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/dashboard/checks/"+created.ID+"/delete", bytes.NewBufferString(url.Values{
		"redirect_to": []string{"/"},
	}.Encode()))
	deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRes, deleteReq)

	if deleteRes.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want %d", deleteRes.Code, http.StatusSeeOther)
	}

	_, err = dataStore.GetCheck(created.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetCheck() error = %v, want ErrNotFound", err)
	}
}
