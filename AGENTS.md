# AGENTS.md — LLMIDI-Gen

AI-powered MIDI generator: Go 1.25, LLM-driven, progressive house / melodic techno.

## Project Overview

Takes **BPM + Key** as input, produces **3 MIDI files** (bassline, arpeggio, melody). A shared chord progression ensures harmonic coherence. The LLM creates musical motifs; the renderer applies professional timing, velocity, and automation deterministically via style profiles. Offline mode (`--no-llm`) generates hypnotic, varied patterns algorithmically.

**Specs:** `"C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\docs\superpowers\specs\2026-04-29-midi-examples-design.md"` is the source of truth for architecture, PatternSpec schema, style profiles, and musical rules.

## Architecture

```
User (BPM + Key)
  → KeyParser
  → Step 0: Chord Progression (4 chords, shared)
  → Step 1: 3x parallel generators (bass, arp, melody) — all receive same chord progression
  → Validator (scale + range + density + chord coherence)
  → StyleProfile → Renderer
  → 3 MIDI Type-0 files
```

- **Step 0 (Chord Progression)** owns: harmonic contract — 4 chords, one per 4 bars
- **LLM** owns: motif creativity, note choice, evolution arc (within chord constraints)
- **Renderer** owns: timing offsets, velocity grids, gate lengths, CC automation, portamento
- **Validator** enforces: note range, scale membership, density, chord coherence, BPM bounds
- **Offline mode** owns: seed-based algorithmic pattern generation (no API calls)

## Key Directories

| Path | Purpose |
|------|---------|
| `cmd/cadenza/` | CLI entry point, interactive mode |
| `internal/theory/` | Key parsing, scales, note↔MIDI, chords, progressions |
| `internal/schema/` | PatternSpec types + musical validator (with chord coherence check) |
| `internal/llm/` | Provider interface, Codex (`tool_use`), Ollama (JSON schema mode), mock, retry with error classification |
| `internal/renderer/` | MIDI rendering: velocity, timing, gate, sweep, evolution, portamento |
| `internal/renderer/styleprofile/` | Deterministic style profiles with DynamicCurve (crescendo/arch) |
| `internal/generator/` | Chord progression gen + single/multi-pattern generation + offline templates + LLM cache integration |
| `internal/midi/` | MIDI Type-0 file writer with priority-based event ordering |
| `internal/cache/` | SHA256-keyed disk cache (30-day TTL) |
| `prompts/` | LLM prompt templates (bassline, arpeggio, melody) |

## Conventions

- **Go 1.25** — use stdlib `log/slog` for logging, `context` for cancellation
- **Module path:** `github.com/Andrea-Cavallo/cadenza`
- **Tests:** `_test.go` next to source, table-driven, AAA pattern
- **MIDI:** Type-0, 480 ticks/beat, 120 ticks/step, CH 1 (zero-indexed 0)
- **Velocity max: 120** — never 127, it clips
- **Downbeat always on-grid** — no timing offset on step 0
- **Ghost velocity: 35-55** — never above 60
- **Portamento:** skip CC65 when tick < 10
- **Cross-platform:** `filepath.Join` for all paths, `CGO_ENABLED=0` for builds

## Go Version — Critical Rule

**`go.mod` and `.golangci.yml` must always declare the same Go version.**

```
go.mod          → go 1.25
.golangci.yml   → run: go: "1.25"
```

Whenever `go.mod` is updated to a newer Go version, `.golangci.yml` must be updated in the same commit. A mismatch causes `golangci-lint` to run the typecheck linter with the wrong toolchain, producing false `package requires newer Go version` errors and `undefined: <symbol>` errors for all SDK types. This has caused CI failures before and must not happen again.

## Musical Domain Rules

These are **invariants** the renderer enforces regardless of LLM output:

1. Notes must be in the declared scale
2. Notes must be within the pattern type's MIDI range (bass: 33-55, arp: 48-84, melody: 60-96)
3. Density must match pattern type constraints
4. Bass uses chord root as primary note per section; arp breaks chord notes; melody gravitates toward chord tones
5. Filter sweep uses S-curve or exponential — never linear
6. Portamento CC65 fires 5 ticks before slide notes, skipped entirely when tick < 10
7. Accent grid from style profile overrides LLM velocity on beats 1, 5, 9, 13
8. Evolution actions have exact semantics (see SPECS.md §3 catalog) — `intensity` scales effects proportionally
9. Evolution `introduce` cannot deactivate below `minActiveFloor` (4 steps)
10. DynamicCurve scales velocity across 16 bars: bass/melody = crescendo (0.7→1.0), arp = arch (0.75→1.0→0.85)

## LLM Integration

- **Codex:** `tool_use` with `generate_pattern` tool forces structurally valid JSON — retry only for musical violations
- **Ollama:** JSON schema format object (full schema in `format` field) — retry handles both structural and musical errors
- **System prompt:** Persistent rules and constraints sent via `System` field; user message contains only the specific generation task
- **Retry:** max 3 attempts; classifies errors as structural (JSON parse) vs musical (validation); different correction prompts for each
- **Temperature:** 0.3 for consistency
- **Cache:** SHA256(provider+type+key+mode+seed), 30-day TTL on disk — skip API call if cached
- **Graceful fallback:** if LLM fails after 3 retries, falls back to offline template (never fails completely)
- **Chord coherence:** validator checks that each 4-step section contains at least one chord tone

## Running

```bash
# Build
make build

# Cross-compile all platforms
make build-all

# With Codex
export ANTHROPIC_API_KEY=sk-...
go run ./cmd/cadenza/ --bpm 122 --key Am

# With Ollama
go run ./cmd/cadenza/ --bpm 122 --key Am --provider ollama --model qwen2.5:7b

# Offline (deterministic, no LLM)
go run ./cmd/cadenza/ --bpm 122 --key Am --no-llm

# Docker
make docker && make docker-run
```

## Testing

```bash
make test                                        # all unit tests
make test-race                                   # with race detector
make test-integration                            # include integration tests
make test-coverage                               # coverage report
go test ./internal/renderer/ -v -run TestRender  # renderer tests
make listening-test                              # generate files for A/B test in DAW
```

## Quality Gate — Mandatory Before Any Completion

The full local CI pipeline is `make ci`. It runs in order: `fmt → vet → lint → vuln → coverage`. All steps must pass clean.

```bash
make ci
```

Individual tools:

```bash
golangci-lint run ./...     # static analysis — zero errors required
govulncheck ./...           # vulnerability scan — zero findings required
go test ./... -race -count=1 -coverprofile=coverage.out -covermode=atomic
```

### Known failure mode: Go version mismatch

If you see errors like:

```
package requires newer Go version go1.25 (application built with go1.23)
undefined: anthropic
```

The cause is always `.golangci.yml` declaring a lower `go:` version than `go.mod`. Fix: align both to the same version (currently `1.25`). Never set `.golangci.yml` `go:` below the version in `go.mod`.

### Linter warnings vs errors

- `level=warning msg="no need to enable check..."` — harmless, ignored
- Any `Error:` line — must be fixed before the work is considered done

## SonarCloud

Project: `Andrea-Cavallo_cadenza` / organization: `andrea-cavallo`  
Dashboard: `https://sonarcloud.io/project/overview?id=Andrea-Cavallo_cadenza`

SonarCloud runs automatically on every push via `.github/workflows/ci.yml` (the `sonar` job). It requires `SONAR_TOKEN` in the repo's GitHub Secrets.

Configuration lives in `sonar-project.properties`. Coverage is fed from `coverage.out` (generated by the `test` job and passed as an artifact).

**Quality gate thresholds** (enforced on SonarCloud, fail PR if not met):
- Coverage: ≥ 80%
- Duplications: ≤ 3%
- Maintainability rating: A
- Reliability rating: A
- Security rating: A

To run SonarCloud locally (requires `sonar-scanner` on PATH):
```bash
SONAR_TOKEN=<your-token> make sonar
```

## Guidelines

1. **Read SPECS.md first** when touching musical logic — it defines the exact behavior
2. **Style profiles are deterministic** — don't add randomness to velocity/timing/gate
3. **Chord progression is the harmonic contract** — all 3 generators must respect it
4. **Validator errors become correction prompts** — keep error messages human-readable
5. **Three pattern types have different constraints** — don't generalize bass rules to melody
6. **Evolution actions have a defined catalog** — don't invent new ones without adding to SPECS.md
7. When editing existing code: match the style, don't "improve" adjacent code
8. Every changed line should trace to the task at hand
9. **Offline patterns must be musically hypnotic** — never boring deterministic loops; use seed variation, chord awareness, contour diversity
10. **Services must be public methods** — no business logic locked inside CLI handlers; all core logic must be callable independently (see REFACTOR.md §21)
11. **JSON logging** — use `slog.NewJSONHandler` in non-dev environments; never `fmt.Println` in business logic (see REFACTOR.md §22)

## Post-Modification Checklist

After **every** code change, autonomously perform **all** of these steps in order. Do not skip any step. Do not mark work as done until all steps pass.

### Build and correctness

1. `go build ./...` — must pass with zero errors
2. `go vet ./...` — must pass clean
3. `GOOS=linux go build ./...` — cross-compilation must still pass
4. `go test ./...` — all tests must pass
5. `go test -race ./...` — no race conditions

### Quality gate — non-negotiable

6. `golangci-lint run ./...` — must pass with zero `Error:` lines
   - If you see `package requires newer Go version`: check that `.golangci.yml` `run.go` matches `go.mod`
   - If you see `undefined: anthropic` or similar SDK symbols: same root cause — fix the Go version mismatch
7. `govulncheck ./...` — must return zero vulnerability findings

### Documentation — always update, no exceptions

8. **`CHANGELOG.md`** — add an entry under `[Unreleased]` for every change, no matter how small. Format:
   ```
   ### Added / Changed / Fixed / Removed
   - <one line describing what changed and why>
   ```
9. **`README.md`** — update whenever: a flag is added/removed, output format changes, a new command or mode is introduced, installation steps change, examples become stale
10. **`AGENTS.md`** (this file) — update whenever: architecture changes, a new invariant is introduced, a convention changes, a new directory is added
11. **`REFACTOR.md`** — remove completed items, add newly discovered improvements
12. **`SPECS.md`** — update whenever musical rules, evolution catalog, or PatternSpec schema change

### Tests

13. If a new feature was added: add tests covering the happy path and at least one error case
14. If a bug was fixed: add a regression test that would have caught the bug
15. Target minimum 80% coverage on changed packages: `go test -cover ./...`

### Version file sync

16. If `go.mod` Go version changes: update `.golangci.yml` `run.go` to match **in the same change**
