# api-core.go

[![CI](https://github.com/sweetrpg/api-core.go/actions/workflows/ci.yaml/badge.svg)](https://github.com/sweetrpg/api-core.go/actions/workflows/ci.yaml)
[![License](https://img.shields.io/github/license/sweetrpg/api-core.go.svg)](https://img.shields.io/github/license/sweetrpg/api-core.go.svg)
[![Issues](https://img.shields.io/github/issues/sweetrpg/api-core.go.svg)](https://img.shields.io/github/issues/sweetrpg/api-core.go.svg)
[![PRs](https://img.shields.io/github/issues-pr/sweetrpg/api-core.go.svg)](https://img.shields.io/github/issues-pr/sweetrpg/api-core.go.svg)
[![Dependabot](https://badgen.net/github/dependabot/sweetrpg/api-core.go)](https://badgen.net/github/dependabot/sweetrpg/api-core.go)

Shared HTTP API building blocks for sweetrpg's Go services: query-string parsing into
paging/sort/filter/projection params, conversion of those params into MongoDB BSON filters,
OpenTelemetry tracing setup, health/ping handlers, and common response value objects.

## Install

```bash
go get github.com/sweetrpg/api-core.go
```

## Packages

- `util` - `GetQueryParams` (parses a JSON:API-style query string) and `ConvertQueryParams`
  (turns those params into MongoDB filter/sort/projection `bson.D` documents)
- `tracing` - `SetupTracing`/`TeardownTracing` and `BuildSpanWithParams` (OpenTelemetry)
- `server` - `HealthHandler`/`PingHandler` for status endpoints
- `vo` - shared response value objects (`ErrorVO`, `PingResponseVO`, `HealthResponseVO`)
- `constants` - environment variable names used by the packages above

## Documentation

Package documentation: [pkg.go.dev/github.com/sweetrpg/api-core.go](https://pkg.go.dev/github.com/sweetrpg/api-core.go).
Test coverage reports are published to [sweetrpg.github.io/api-core.go](https://sweetrpg.github.io/api-core.go)
on every merge to `develop`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and
[RELEASE.md](RELEASE.md) for how versions get cut.
