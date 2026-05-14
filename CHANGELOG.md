# Changelog

All notable changes to this project will be documented in this file.

## [1.3.0]

### Added
- Custom request headers per monitor. Set them as `Key: Value` lines from
  the monitor detail page, or via the JSON API on create.
- Outbound auth per monitor. Choose Basic (`user:pass`) or Bearer; pingtower
  attaches the matching `Authorization` header on every poll.
- Dashboard routes `POST /dashboard/checks/{id}/headers` and
  `POST /dashboard/checks/{id}/auth` for managing headers and auth.
- New `headers`, `auth_type`, and `auth_value` fields on the check model and
  on `POST /checks`.
- New `SetCheckHeaders` and `SetCheckAuth` store methods.

### Notes
- Auth values are stored in the data file as plain text. Restrict
  permissions on `data/pingtower.json` to trusted users.
- The dashboard reflects the masked state of an existing auth value rather
  than the secret itself. Submitting the auth form with the value field
  blank preserves the existing secret.
- Unlocks monitoring of authenticated APIs that were previously impossible
  to check (anything behind an API key, OAuth bearer, or HTTP Basic).

## [1.2.0]

### Added
- Webhook alerts on status transitions. Configure a webhook URL per monitor
  from its detail page. Pingtower POSTs a JSON payload whenever a check goes
  from up to down or back again.
- New `webhook_url` field on the check model and `SetCheckWebhook` store
  method.
- Dashboard route `POST /dashboard/checks/{id}/webhook` to set or clear the
  webhook URL.

### Notes
- Webhooks fire only on transitions never on the very first poll, so adding
  a monitor doesn't trigger a false alert.
- Delivery is fire-and-forget with a 10-second timeout. Failures are logged
  but never block the polling loop.
- Payload includes `check_id`, `name`, `url`, `status`, `previous_status`,
  `status_code`, `response_ms`, and `checked_at`.
- Works with anything that accepts a JSON POST. Use [webhook.site](https://webhook.site)
  to inspect payloads while testing.

## [1.1.0]

### Added
- "Check now" button on each monitor's detail page — runs the check
  immediately and refreshes the result history without waiting for the next
  polling interval.
- `POST /checks/{id}/trigger` API endpoint that returns the result as JSON,
  for scripting and integrations.

### Changed
- CI workflow updated to use Node.js 24.

## [1.0.0]

### Added
- First public release of pingtower — a lightweight self-hosted uptime
  monitor for websites and APIs.
- Built-in web dashboard for viewing monitors, adding new ones, and
  inspecting per-monitor history.
- Per-check polling intervals and request timeouts.
- Status history with pause, resume, and delete controls per monitor.
- Configurable expected status code per monitor.
- Local JSON-backed storage (no database required).
- Docker support via `docker compose up --build`.
- Unit tests and GitHub Actions CI.

[1.2.0]: https://github.com/crleonard/pingtower/releases/tag/v1.2.0
[1.1.0]: https://github.com/crleonard/pingtower/releases/tag/v1.1.0
[1.0.0]: https://github.com/crleonard/pingtower/releases/tag/v1.0.0
