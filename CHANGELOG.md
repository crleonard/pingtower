# Changelog

All notable changes to this project will be documented in this file.

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

[1.1.0]: https://github.com/crleonard/pingtower/releases/tag/v1.1.0
[1.0.0]: https://github.com/crleonard/pingtower/releases/tag/v1.0.0
