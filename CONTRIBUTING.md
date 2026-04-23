# Contributing to Pingtower

Thanks for your interest in contributing! Pingtower is a lightweight, self-hosted uptime monitor written in Go with no external dependencies.

## Getting Started

**Prerequisites:** Go 1.25+, Docker (optional)

```bash
git clone https://github.com/crleonard/pingtower.git
cd pingtower
make run        # starts the server at http://localhost:8080
```

Or with Docker:

```bash
docker compose up --build
```

## Development Workflow

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Run tests: `make test`
4. Open a pull request against `main`

Keep the zero-external-dependencies rule intact - the `go.mod` should only reference the standard library.

## Project Structure

```
cmd/server/       entry point — wires components together
internal/config/  environment variable parsing
internal/httpapi/ HTTP handlers and dashboard
internal/model/   core data types (Check, Result, Snapshot)
internal/monitor/ background polling loop
internal/store/   JSON file persistence
```

## Configuration (for testing locally)

All env vars are optional:

| Variable                     | Default               |
|------------------------------|-----------------------|
| `PINGTOWER_ADDR`             | `:8080`               |
| `PINGTOWER_DATA_FILE`        | `data/pingtower.json` |
| `PINGTOWER_DEFAULT_INTERVAL` | `60s`                 |
| `PINGTOWER_DEFAULT_TIMEOUT`  | `10s`                 |
| `PINGTOWER_MAX_HISTORY`      | `100`                 |
| `PINGTOWER_USER_AGENT`       | `pingtower/1.0`       |

## Guidelines

- **No external dependencies.** Standard library only.
- **Tests required** for new functionality — run `go test ./...` or `make test`.
- **Keep it simple.**
- Follow standard Go conventions (`gofmt`, meaningful names, short functions).
- Write clear commit messages that explain *why*, not just *what*.

## Reporting Issues

Please open a GitHub issue with:

- What you expected to happen
- What actually happened
- Steps to reproduce
- Go version and Browser

## API Reference

| Method | Path                  | Description         |
|--------|-----------------------|---------------------|
| GET    | `/health`             | Health check        |
| GET    | `/checks`             | List all monitors   |
| POST   | `/checks`             | Create a monitor    |
| GET    | `/checks/:id`         | Get a monitor       |
| GET    | `/checks/:id/history` | Get check history   |

---

All contributions are welcome - bug fixes, features, docs, and tests.
