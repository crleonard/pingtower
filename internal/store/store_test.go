package store

import (
	"path/filepath"
	"testing"

	"github.com/crleonard/pingtower/internal/model"
)

func TestFileStorePersistsChecks(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "pingtower.json")
	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	check, err := fs.CreateCheck(model.Check{
		Name:               "Example",
		URL:                "https://example.com",
		IntervalSeconds:    60,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("CreateCheck() error = %v", err)
	}

	reloaded, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore(reload) error = %v", err)
	}

	got, err := reloaded.GetCheck(check.ID)
	if err != nil {
		t.Fatalf("GetCheck() error = %v", err)
	}

	if got.Name != check.Name {
		t.Fatalf("GetCheck().Name = %q, want %q", got.Name, check.Name)
	}
}

func TestSetCheckWebhook(t *testing.T) {
	t.Parallel()

	fs, err := NewFileStore(filepath.Join(t.TempDir(), "pingtower.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	check, err := fs.CreateCheck(model.Check{
		Name:               "Webhook Test",
		URL:                "https://example.com",
		IntervalSeconds:    60,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("CreateCheck() error = %v", err)
	}

	updated, err := fs.SetCheckWebhook(check.ID, "https://hooks.example.com/notify")
	if err != nil {
		t.Fatalf("SetCheckWebhook() error = %v", err)
	}
	if updated.WebhookURL != "https://hooks.example.com/notify" {
		t.Fatalf("WebhookURL = %q, want set value", updated.WebhookURL)
	}

	// clear it
	cleared, err := fs.SetCheckWebhook(check.ID, "")
	if err != nil {
		t.Fatalf("SetCheckWebhook(clear) error = %v", err)
	}
	if cleared.WebhookURL != "" {
		t.Fatalf("WebhookURL = %q after clear, want empty", cleared.WebhookURL)
	}

	_, err = fs.SetCheckWebhook("nonexistent", "https://example.com")
	if err == nil {
		t.Fatal("SetCheckWebhook() expected error for unknown ID, got nil")
	}
}
