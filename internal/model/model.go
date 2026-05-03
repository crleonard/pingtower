package model

import "time"

type Check struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	URL                string    `json:"url"`
	IntervalSeconds    int       `json:"interval_seconds"`
	TimeoutSeconds     int       `json:"timeout_seconds"`
	ExpectedStatusCode int       `json:"expected_status_code"`
	Paused             bool      `json:"paused"`
	WebhookURL         string    `json:"webhook_url,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	LastCheckedAt      time.Time `json:"last_checked_at,omitempty"`
	LastStatus         string    `json:"last_status,omitempty"`
	LastStatusCode     int       `json:"last_status_code,omitempty"`
	LastResponseMS     int64     `json:"last_response_ms,omitempty"`
	LastError          string    `json:"last_error,omitempty"`
}

type Result struct {
	CheckID        string    `json:"check_id"`
	CheckedAt      time.Time `json:"checked_at"`
	Status         string    `json:"status"`
	StatusCode     int       `json:"status_code,omitempty"`
	ResponseMS     int64     `json:"response_ms"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	ResponseSample string    `json:"response_sample,omitempty"`
}

type Snapshot struct {
	Checks  []Check             `json:"checks"`
	History map[string][]Result `json:"history"`
}
