# AGENTS.md

This file provides guidance to Claude Code, Codex, GitHub Copilot, and other AI coding agents
working in this repository.

## About This Project

`api-core.go` provides shared HTTP API building blocks for sweetrpg's Go services: query-string
parsing into paging/sort/filter/projection params (`util`), conversion of those params into
MongoDB BSON filters (`util.ConvertQueryParams`), OpenTelemetry tracing setup (`tracing`),
health/ping handlers (`server`), and common response value objects (`vo`).

## Dependencies

Depends on `common.go` (logging) and `mongodb.go` (query constants, health-check database
access). Depended on by `catalog-data.go`, `catalog-api`, and other sweetrpg API services.

## Committing Code

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>
```

## Branches and Workflow

* `develop` - integration branch, default branch, target for all PRs.
* `master` - latest released state, nothing committed directly.
* `feature/*`, `fix/*` branched from `develop`; `hotfix/*` branched from `master`.

See `CONTRIBUTING.md` for the full workflow.

## Running Checks Locally

```bash
go build -v ./...
go vet ./...
go test -v -coverprofile coverage.out ./...
```

## Releases

See `RELEASE.md`. Summary: trigger `prepare-release.yaml` (`workflow_dispatch` against
`develop`), which computes the next version from conventional commits via git-cliff and opens
a `release/<version>` PR into `master`. Merging that PR tags the release
(`tag-release.yaml`), which triggers `release.yaml` - re-runs tests, creates a GitHub
Release, and merges `master` back into `develop`.
