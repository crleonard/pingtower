package httpapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/crleonard/pingtower/internal/model"
)

func (s *Server) validateCreateCheck(req createCheckRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if _, err := url.ParseRequestURI(req.URL); err != nil {
		return fmt.Errorf("url must be a valid absolute URL")
	}

	switch req.AuthType {
	case "", "none", "basic", "bearer":
	default:
		return fmt.Errorf("auth_type must be one of: none, basic, bearer")
	}

	return nil
}

func (s *Server) buildCheck(req createCheckRequest) (model.Check, error) {
	if err := s.validateCreateCheck(req); err != nil {
		return model.Check{}, err
	}

	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = int(s.cfg.DefaultInterval.Seconds())
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = int(s.cfg.DefaultTimeout.Seconds())
	}
	if req.ExpectedStatusCode <= 0 {
		req.ExpectedStatusCode = http.StatusOK
	}

	authType := req.AuthType
	authValue := req.AuthValue
	if authType == "" || authType == "none" {
		authType = ""
		authValue = ""
	}

	return model.Check{
		Name:               strings.TrimSpace(req.Name),
		URL:                req.URL,
		IntervalSeconds:    req.IntervalSeconds,
		TimeoutSeconds:     req.TimeoutSeconds,
		ExpectedStatusCode: req.ExpectedStatusCode,
		Headers:            req.Headers,
		AuthType:           authType,
		AuthValue:          authValue,
	}, nil
}
