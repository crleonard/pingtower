package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/crleonard/pingtower/internal/model"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	CreateCheck(check model.Check) (model.Check, error)
	ListChecks() ([]model.Check, error)
	GetCheck(id string) (model.Check, error)
	ListResults(id string) ([]model.Result, error)
	UpdateCheckStatus(id string, result model.Result, maxHistory int) error
	SetCheckPaused(id string, paused bool) (model.Check, error)
	DeleteCheck(id string) error
}

type FileStore struct {
	path string

	mu      sync.RWMutex
	checks  map[string]model.Check
	history map[string][]model.Result
}

func NewFileStore(path string) (*FileStore, error) {
	fs := &FileStore{
		path:    path,
		checks:  map[string]model.Check{},
		history: map[string][]model.Result{},
	}

	if err := fs.load(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (fs *FileStore) CreateCheck(check model.Check) (model.Check, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	now := time.Now().UTC()
	check.ID = newID()
	check.CreatedAt = now
	check.UpdatedAt = now
	fs.checks[check.ID] = check
	fs.history[check.ID] = []model.Result{}

	if err := fs.saveLocked(); err != nil {
		return model.Check{}, err
	}
	return check, nil
}

func (fs *FileStore) ListChecks() ([]model.Check, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	items := make([]model.Check, 0, len(fs.checks))
	for _, check := range fs.checks {
		items = append(items, check)
	}
	slices.SortFunc(items, func(a, b model.Check) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return items, nil
}

func (fs *FileStore) GetCheck(id string) (model.Check, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	check, ok := fs.checks[id]
	if !ok {
		return model.Check{}, ErrNotFound
	}
	return check, nil
}

func (fs *FileStore) ListResults(id string) ([]model.Result, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if _, ok := fs.checks[id]; !ok {
		return nil, ErrNotFound
	}

	results := append([]model.Result(nil), fs.history[id]...)
	slices.SortFunc(results, func(a, b model.Result) int {
		return b.CheckedAt.Compare(a.CheckedAt)
	})
	return results, nil
}

func (fs *FileStore) UpdateCheckStatus(id string, result model.Result, maxHistory int) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	check, ok := fs.checks[id]
	if !ok {
		return ErrNotFound
	}

	check.LastCheckedAt = result.CheckedAt
	check.LastStatus = result.Status
	check.LastStatusCode = result.StatusCode
	check.LastResponseMS = result.ResponseMS
	check.LastError = result.ErrorMessage
	check.UpdatedAt = time.Now().UTC()
	fs.checks[id] = check

	fs.history[id] = append([]model.Result{result}, fs.history[id]...)
	if len(fs.history[id]) > maxHistory {
		fs.history[id] = fs.history[id][:maxHistory]
	}

	return fs.saveLocked()
}

func (fs *FileStore) SetCheckPaused(id string, paused bool) (model.Check, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	check, ok := fs.checks[id]
	if !ok {
		return model.Check{}, ErrNotFound
	}

	check.Paused = paused
	check.UpdatedAt = time.Now().UTC()
	if paused {
		check.LastError = ""
	}
	fs.checks[id] = check

	if err := fs.saveLocked(); err != nil {
		return model.Check{}, err
	}

	return check, nil
}

func (fs *FileStore) DeleteCheck(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, ok := fs.checks[id]; !ok {
		return ErrNotFound
	}

	delete(fs.checks, id)
	delete(fs.history, id)

	return fs.saveLocked()
}

func (fs *FileStore) load() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(fs.path), 0o755); err != nil {
		return err
	}

	bytes, err := os.ReadFile(fs.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var snapshot model.Snapshot
	if err := json.Unmarshal(bytes, &snapshot); err != nil {
		return err
	}

	for _, check := range snapshot.Checks {
		fs.checks[check.ID] = check
	}
	for id, results := range snapshot.History {
		fs.history[id] = results
	}

	return nil
}

func (fs *FileStore) saveLocked() error {
	snapshot := model.Snapshot{
		Checks:  make([]model.Check, 0, len(fs.checks)),
		History: map[string][]model.Result{},
	}

	for _, check := range fs.checks {
		snapshot.Checks = append(snapshot.Checks, check)
	}
	slices.SortFunc(snapshot.Checks, func(a, b model.Check) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	for id, results := range fs.history {
		snapshot.History[id] = append([]model.Result(nil), results...)
	}

	bytes, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.path, bytes, 0o644)
}

func newID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(raw[:])
}
