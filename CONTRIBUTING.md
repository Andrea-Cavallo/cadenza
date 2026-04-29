# Contributing

Contributions are welcome. This document explains how to get started.

## Development Setup

```bash
# Prerequisites
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Clone
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza

# Verify
make test
make lint
```

## Project Structure

```
cmd/cadenza/     CLI entry point
internal/
  theory/           Music theory (keys, scales, chords, progressions)
  schema/           PatternSpec types + musical validator
  llm/              LLM provider interface + implementations
  generator/        Pattern generation (single, multi, offline)
  renderer/         MIDI rendering pipeline
  midi/             MIDI file writer
  cache/            LLM response cache
prompts/            LLM prompt templates
```

## Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make changes
4. Run checks: `make test && make lint`
5. Commit with conventional format: `feat: add new style profile`
6. Open a pull request against `main`

## Commit Messages

```
<type>: <description>

Types: feat, fix, refactor, docs, test, chore, perf, ci
```

## Adding a Style Profile

1. Create `internal/renderer/styleprofile/<name>.go`
2. Define a `var` with type `StyleProfile`
3. Register in `registry.go`
4. Add the name to `validStyleProfiles` in `internal/schema/validator.go`
5. Add tests verifying the profile renders without panics

## Adding an LLM Provider

1. Implement `llm.Provider` interface in `internal/llm/<name>.go`
2. Wire it in `internal/generator/multi_generator.go`'s `buildProvider`
3. Add integration test with `//go:build integration` tag

## Musical Rules

These are invariants — code must never violate them:

- Max velocity: 120 (never 127)
- Downbeat (step 0) always on-grid, never deactivated
- Ghost velocity: 35-55 (never above 60)
- Notes must be in declared scale
- Notes must be within pattern type MIDI range
- Filter sweep: S-curve or exponential only (never linear)

## Testing

```bash
make test               # unit tests
make test-coverage      # with coverage report
make test-integration   # integration tests (requires LLM access)
make listening-test     # generate patterns for DAW comparison
```

Target: 80%+ coverage on all packages with logic.

## Code Style

- `gofmt` + `goimports` are mandatory
- Accept interfaces, return structs
- Wrap errors with context: `fmt.Errorf("...: %w", err)`
- Table-driven tests with AAA pattern
- Files: 200-400 lines typical, 800 max
