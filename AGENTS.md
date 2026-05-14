# AGENTS.md — LLMIDI-Gen

AI-powered MIDI generator: Go 1.25, LLM-driven, progressive house / melodic techno.
**IMPORTANT:** Always read TODO.md first for the most current project status and work priorities.
This file provides architectural overview; TODO.md contains actionable tasks.

**Core Focus:** The primary goal is musical quality — generating hypnotic, varied, and musically interesting patterns, especially in offline mode (`--no-llm`). All architectural decisions serve this goal.

## Project Overview
Takes **BPM + Key** as input, produces **3 MIDI files** (bassline, arpeggio, melody). 
A shared chord progression ensures harmonic coherence. The LLM creates musical motifs; 
the renderer applies professional timing, velocity, and automation deterministically via style profiles. 
Offline mode (`--no-llm`) generates hypnotic, varied patterns algorithmically.


## Architecture

```
User (BPM + Key)
  → KeyParser
  → Step 0: Chord Progression (4 chords, shared)
  → Step 1a: MusicalPlan (style card + tension curve + motif intent)
  → Step 1: 3x parallel generators (bass, arp, melody) — all receive same chord progression + plan
  → Critic + one targeted revision round when musical scoring is weak
  → Validator (scale + range + density + chord coherence + soft musical scoring)
  → StyleProfile → Renderer
  → 3 MIDI Type-0 files
```

- **Step 0 (Chord Progression)** owns: harmonic contract — 4 chords, one per 4 bars
- **MusicalPlan** owns: producer intent — style card, tension curve, motif concept, density target, and section intent
- **LLM** owns: motif creativity, note choice, evolution arc (within chord and plan constraints)
- **Critic** owns: soft quality judgment — repetition, motif clarity, chord-tone strength, contour, density, tension arc, track separation; at most one targeted revision
- **Renderer** owns: timing offsets, velocity grids, gate lengths, CC automation, portamento
- **Validator** enforces: note range, scale membership, density, chord coherence, BPM bounds; it also reports soft musical scores
- **Offline mode** owns: seed-based algorithmic pattern generation (no API calls)

## Implementation Order

```
P0-binary
  → P0-0a (seed entropy)
  → P0-0b (key differentiation)
  → P0-0c (modal support)
  → P0-0d (LLM prompt quality)
  → P0-0e (offline hypnotic patterns)
  → P0-0f (LLM planner + critic quality)
    → P1 (README + demo — only after music sounds good)
      → P2-Phase1
      → P2-Phase2
      → P2-Phase3 + Phase4
        → P3 (coverage gaps from modal work)
```

> **Rule:** Do not start P1 (README / demo audio) until P0 music quality is solved.
> A bad demo hurts more than no demo.

## Key Directories

| Path | Purpose |
|------|---------|
| `backend/cmd/cadenza/` | CLI entry point, interactive mode |
| `backend/cmd/desktop/` | Wails desktop app entry point and AppService bindings |
| `backend/cmd/desktop/frontend/` | Vite + React + TypeScript desktop UI embedded into the Wails binary |
| `scripts/` | Local automation scripts for packaging and release builds |
| `backend/internal/theory/` | Key parsing, scales, note↔MIDI, chords, progressions |
| `backend/internal/schema/` | PatternSpec types + musical validator (with chord coherence check and soft musical scoring) |
| `backend/internal/llm/` | Provider interface, Claude (`tool_use`), Ollama (JSON schema mode), mock, retry with error classification |
| `backend/internal/renderer/` | MIDI rendering: velocity, timing, gate, sweep, evolution, portamento |
| `backend/internal/renderer/styleprofile/` | Deterministic style profiles with DynamicCurve (crescendo/arch) |
| `backend/internal/generator/` | Chord progression gen + MusicalPlan/style cards + single/multi-pattern generation + offline templates + LLM cache integration |
| `backend/internal/midi/` | MIDI Type-0 file writer with priority-based event ordering |
| `backend/internal/cache/` | SHA256-keyed disk cache (30-day TTL) |
| `backend/internal/config/` | Viper-based config loading (cadenza.yaml + env vars) |
| `backend/internal/session/` | File-based session persistence with checkpoint/eviction |
| `backend/internal/metrics/` | Prometheus counters for LLM calls, tokens, errors |
| `backend/internal/logger/` | slog setup with JSON handler for production |
| `backend/internal/models/` | Shared domain types (GenerateRequest, GenerateResult, etc.) |
| `backend/internal/prompts/` | Embedded LLM prompt templates via `//go:embed` |
| `backend/internal/service/` | Business logic layer callable from CLI and desktop |

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

- **Claude:** `tool_use` with `generate_pattern` tool forces structurally valid JSON — retry only for musical violations
- **Ollama:** JSON schema format object (full schema in `format` field) — retry handles both structural and musical errors
- **System prompt:** Persistent rules and constraints sent via `System` field; user message contains only the specific generation task
- **Retry:** max 3 attempts; classifies errors as structural (JSON parse) vs musical (validation); different correction prompts for each
- **Temperature:** 0.3 for consistency
- **MusicalPlan:** generated before PatternSpec prompts; injects style card, tension curve, motif concept, section intent, and revision priorities
- **Critic/revision:** one critic pass may request one targeted revision; never loop endlessly
- **Cache:** SHA256(provider+type+key+mode+seed+prompt hash+planner version+style-card version+style-card name+critic version+revision policy), 30-day TTL on disk — skip API call if cached
- **Graceful fallback:** if LLM fails after 3 retries, falls back to offline template (never fails completely)
- **Chord coherence:** validator checks that each 4-step section contains at least one chord tone

## Running

```bash
# Build
make build

# Cross-compile all platforms
make build-all

# Desktop app
make desktop         # Wails Windows build
make desktop-dev     # Wails dev mode with HMR
make desktop-manual  # npm install + npm run build + Go production build
# If wails is installed outside PATH, pass WAILS=/path/to/wails.

# With Claude
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

1. **Read TODO.md first** when touching musical logic — it defines the exact behavior
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
