package monitor

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/crleonard/pingtower/internal/model"
	"github.com/crleonard/pingtower/internal/store"
)

func newTestService(t *testing.T) (*Service, store.Store) {
	t.Helper()
	dataStore, err := store.NewFileStore(filepath.Join(t.TempDir(), "pingtower.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	return NewService(dataStore, log.New(io.Discard, "", 0), "pingtower-test", 10), dataStore
}

func TestRunNow_Healthy(t *testing.T) {
	t.Parallel()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	svc, dataStore := newTestService(t)

	check, err := dataStore.CreateCheck(model.Check{
		Name:               "RunNow healthy",
		URL:                target.URL,
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: http.StatusOK,
	})
	if err != nil {
		t.Fatalf("CreateCheck() error = %v", err)
	}

	result, err := svc.RunNow(check.ID)
	if err != nil {
		t.Fatalf("RunNow() error = %v", err)
	}
	if result.Status != "healthy" {
		t.Fatalf("Status = %q, want %q", result.Status, "healthy")
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", result.StatusCode, http.StatusOK)
	}

	// result should also be persisted in the store
	stored, err := dataStore.GetCheck(check.ID)
	if err != nil {
		t.Fatalf("GetCheck() error = %v", err)
	}
	if stored.LastStatus != "healthy" {
		t.Fatalf("LastStatus = %q, want %q", stored.LastStatus, "healthy")
	}
}

func TestRunNow_Down(t *testing.T) {
	t.Parallel()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer target.Close()

	svc, dataStore := newTestService(t)

	check, err := dataStore.CreateCheck(model.Check{
		Name:               "RunNow down",
		URL:                target.URL,
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: http.StatusOK,
	})
	if err != nil {
		t.Fatalf("CreateCheck() error = %v", err)
	}

	result, err := svc.RunNow(check.ID)
	if err != nil {
		t.Fatalf("RunNow() error = %v", err)
	}
	if result.Status != "down" {
		t.Fatalf("Status = %q, want %q", result.Status, "down")
	}
}

func TestRunNow_NotFound(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	_, err := svc.RunNow("nonexistent-id")
	if err == nil {
		t.Fatal("RunNow() expected error for unknown ID, got nil")
	}
}
