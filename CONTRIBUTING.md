# Contributing

Contributions are welcome. This guide is here to make the first steps easy and to help contributors start from the current project structure instead of stale assumptions.

## Contributors wanted

Cadenza is an AI-powered MIDI generator for progressive house and melodic techno, written in Go.

We are especially looking for contributors interested in:

- MIDI generation
- Go backend development
- music theory
- AI-assisted composition
- CLI tools
- desktop app UX
- documentation and examples

Good first issues are available here:
[https://github.com/Andrea-Cavallo/cadenza/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22](https://github.com/Andrea-Cavallo/cadenza/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)

If you are not sure where to begin, documentation, contributor onboarding, examples, and small UX improvements are all valuable contributions too.

## Development Setup

```bash
# Prerequisites
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.5.0

# Clone
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza/backend

# Verify
go build ./...
make test
make ci
```

The active Go application lives in `backend/`. Most contributor work for the CLI, generator, renderer, LLM integrations, and tests starts there.

## Project Structure

```
backend/cmd/cadenza/              CLI entry point
backend/cmd/desktop/              Wails desktop shell
backend/cmd/desktop/frontend/     React + TypeScript desktop UI
backend/internal/theory/          Music theory (keys, scales, chords, progressions)
backend/internal/schema/          PatternSpec types + musical validator
backend/internal/llm/             LLM provider interface + implementations
backend/internal/generator/       Pattern generation (single, multi, offline)
backend/internal/renderer/        MIDI rendering pipeline
backend/internal/midi/            MIDI file writer
backend/internal/cache/           LLM response cache
backend/prompts/                  LLM prompt templates
scripts/                          Packaging and release helpers
```

## Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make changes
4. Run checks from `backend/`: `make ci`
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

1. Implement `llm.Provider` interface in `backend/internal/llm/<name>.go`
2. Wire it into the provider selection flow used by the current CLI/generator path
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

When updating docs or examples, please verify that commands still match the current `backend/` layout and current flags. A lot of friction for new contributors comes from stale copy more than from hard code.

## Code Style

- `gofmt` + `goimports` are mandatory
- Accept interfaces, return structs
- Wrap errors with context: `fmt.Errorf("...: %w", err)`
- Table-driven tests with AAA pattern
- Files: 200-400 lines typical, 800 max
