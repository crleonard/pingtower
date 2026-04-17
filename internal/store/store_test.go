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
