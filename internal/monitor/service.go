package monitor

import (
	"context"
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

func (s *Service) evaluate(check model.Check) {
	timeout := time.Duration(check.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, check.URL, nil)
	if err != nil {
		s.persistResult(check, model.Result{
			CheckID:      check.ID,
			CheckedAt:    time.Now().UTC(),
			Status:       "down",
			ResponseMS:   time.Since(start).Milliseconds(),
			ErrorMessage: err.Error(),
		})
		return
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.persistResult(check, model.Result{
			CheckID:      check.ID,
			CheckedAt:    time.Now().UTC(),
			Status:       "down",
			ResponseMS:   time.Since(start).Milliseconds(),
			ErrorMessage: err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	status := "healthy"
	if resp.StatusCode != check.ExpectedStatusCode {
		status = "down"
	}

	s.persistResult(check, model.Result{
		CheckID:        check.ID,
		CheckedAt:      time.Now().UTC(),
		Status:         status,
		StatusCode:     resp.StatusCode,
		ResponseMS:     time.Since(start).Milliseconds(),
		ResponseSample: strings.TrimSpace(string(body)),
	})
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
}
