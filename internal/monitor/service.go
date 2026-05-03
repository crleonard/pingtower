package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/crleonard/pingtower/internal/model"
	"github.com/crleonard/pingtower/internal/store"
)

type Service struct {
	store      store.Store
	logger     *log.Logger
	httpClient *http.Client
	userAgent  string
	maxHistory int
	stopCh     chan struct{}
	doneCh     chan struct{}
}

func NewService(dataStore store.Store, logger *log.Logger, userAgent string, maxHistory int) *Service {
	return &Service{
		store:      dataStore,
		logger:     logger,
		httpClient: &http.Client{},
		userAgent:  userAgent,
		maxHistory: maxHistory,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}
}

func (s *Service) Start() {
	go s.run()
}

func (s *Service) Stop() {
	close(s.stopCh)
	<-s.doneCh
}

func (s *Service) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer close(s.doneCh)

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Service) tick() {
	checks, err := s.store.ListChecks()
	if err != nil {
		s.logger.Printf("failed to list checks error=%v", err)
		return
	}

	now := time.Now().UTC()
	for _, check := range checks {
		if check.Paused {
			continue
		}
		interval := time.Duration(check.IntervalSeconds) * time.Second
		if !check.LastCheckedAt.IsZero() && now.Sub(check.LastCheckedAt) < interval {
			continue
		}
		s.evaluate(check)
	}
}

// RunNow immediately evaluates a check by ID and returns the result.
// Returns an error only for store failures (e.g. store.ErrNotFound); a
// network failure to the monitored URL is captured as a "down" Result.
func (s *Service) RunNow(checkID string) (model.Result, error) {
	check, err := s.store.GetCheck(checkID)
	if err != nil {
		return model.Result{}, err
	}
	return s.evaluate(check), nil
}

func (s *Service) evaluate(check model.Check) model.Result {
	timeout := time.Duration(check.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, check.URL, nil)
	if err != nil {
		result := model.Result{
			CheckID:      check.ID,
			CheckedAt:    time.Now().UTC(),
			Status:       "down",
			ResponseMS:   time.Since(start).Milliseconds(),
			ErrorMessage: err.Error(),
		}
		s.persistResult(check, result)
		return result
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		result := model.Result{
			CheckID:      check.ID,
			CheckedAt:    time.Now().UTC(),
			Status:       "down",
			ResponseMS:   time.Since(start).Milliseconds(),
			ErrorMessage: err.Error(),
		}
		s.persistResult(check, result)
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	status := "healthy"
	if resp.StatusCode != check.ExpectedStatusCode {
		status = "down"
	}

	result := model.Result{
		CheckID:        check.ID,
		CheckedAt:      time.Now().UTC(),
		Status:         status,
		StatusCode:     resp.StatusCode,
		ResponseMS:     time.Since(start).Milliseconds(),
		ResponseSample: strings.TrimSpace(string(body)),
	}
	s.persistResult(check, result)
	return result
}

func (s *Service) persistResult(check model.Check, result model.Result) {
	if err := s.store.UpdateCheckStatus(check.ID, result, s.maxHistory); err != nil {
		s.logger.Printf("failed to save result check_id=%s error=%v", check.ID, err)
		return
	}

	s.logger.Printf(
		"check evaluated check_id=%s name=%q url=%s status=%s status_code=%d latency_ms=%d",
		check.ID,
		check.Name,
		check.URL,
		result.Status,
		result.StatusCode,
		result.ResponseMS,
	)

	if check.WebhookURL != "" && check.LastStatus != "" && result.Status != check.LastStatus {
		go s.fireWebhook(check, result)
	}
}

func (s *Service) fireWebhook(check model.Check, result model.Result) {
	payload, err := json.Marshal(map[string]any{
		"check_id":        check.ID,
		"name":            check.Name,
		"url":             check.URL,
		"status":          result.Status,
		"previous_status": check.LastStatus,
		"status_code":     result.StatusCode,
		"response_ms":     result.ResponseMS,
		"checked_at":      result.CheckedAt,
	})
	if err != nil {
		s.logger.Printf("webhook marshal failed check_id=%s error=%v", check.ID, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, check.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		s.logger.Printf("webhook request build failed check_id=%s error=%v", check.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Printf("webhook delivery failed check_id=%s error=%v", check.ID, err)
		return
	}
	defer resp.Body.Close()

	s.logger.Printf("webhook delivered check_id=%s status=%d", check.ID, resp.StatusCode)
}
