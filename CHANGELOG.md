
## 0.0.436 - 2026-07-24

### Documentation
- Update README (#139)


### Fixed
- Correct query-filter BSON marshaling and bump vulnerable deps (#138)
- Bump hardcoded semconv import to v1.41.0 (#143)


## 0.0.360 - 2024-11-05

### Fixed
- Convert start option to correct type


## 0.0.1 - 2021-10-06

# Changelog

All notable changes to this project will be documented in this file.

## 0.0.436 - 2026-07-23

### Documentation
- Update README (#139)

### Fixed
- Correct query-filter BSON marshaling and bump vulnerable deps (#138)

## [Unreleased]

### Added

- CONTRIBUTING.md, CODE_OF_CONDUCT.md, AGENTS.md/CLAUDE.md repo scaffolding.
- Regression test covering `ConvertQueryParams` filter-operator BSON marshaling.

### Fixed

- `ConvertQueryParams` filters with an operator (anything beyond a bare equality match)
  produced a malformed BSON document (`{"key": "$op", "value": ...}` instead of
  `{"$op": ...}`), so any query using a filter operator silently matched nothing.
- Removed an `unsafe.Pointer`-based bool-to-int conversion in the same function that read
  past the bounds of a 1-byte `bool` as an 8-byte `int`.
- Fixed a slice pre-allocation bug in `tracing.BuildSpanWithParams` that prefixed trace
  attributes with empty-string entries.
- Bumped `golang.org/x/crypto`, `golang.org/x/net`, `go.opentelemetry.io/otel*`, and
  `google.golang.org/grpc` to resolve 18 open Dependabot alerts (10 critical, 3 high).
