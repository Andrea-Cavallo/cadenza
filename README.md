# CADENZA

<p align="center">
  <img src="cadenza.png" alt="Cadenza" width="480" />
</p>

> AI-powered MIDI generator for progressive house and melodic techno.  
> Give it a BPM and a musical key — get three harmonically coherent MIDI files back.

<p align="center">

[![Go 1.25](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://golang.org)
[![CI](https://github.com/Andrea-Cavallo/cadenza/actions/workflows/ci.yml/badge.svg)](https://github.com/Andrea-Cavallo/cadenza/actions/workflows/ci.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=coverage)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![CGO_ENABLED=0](https://img.shields.io/badge/CGO-disabled-lightgrey)](https://pkg.go.dev/cmd/cgo)

</p>

---

## What is Cadenza?

Cadenza turns two numbers into a complete musical sketch. You provide a **BPM** and a **key** (e.g. `122 Am`). In seconds you get three MIDI files — bassline, arpeggio, melody — that are musically coherent and ready to drop into your DAW.

The LLM handles creative decisions: which notes, which motifs, how the pattern evolves over 16 bars. The renderer enforces professional musical rules that no LLM gets reliably right on its own: micro-timing offsets, velocity grids, portamento, S-curve filter sweeps, evolution arcs. Every run produces a different result even with the same input parameters.

**No LLM? No problem.** The `--no-llm` flag generates hypnotic algorithmic patterns offline, with zero API calls and zero dependencies.

---

## Features

- **One command, three MIDI files** — bassline, arpeggio, and melody generated in parallel
- **Harmonic coherence** — a shared chord progression (Step 0) binds all three generators
- **Professional renderer** — deterministic velocity grids, portamento CC5/CC65, S-curve filter sweeps, DynamicCurve evolution arcs (crescendo, arch, plateau, tension)
- **Multiple LLM backends** — Claude (`tool_use`), Ollama (JSON schema), OpenAI (structured output), Gemini (JSON mode), or no LLM at all
- **Graceful fallback** — if the LLM fails after 3 retries, offline templates take over; it never crashes
- **LLM response cache** — SHA256-keyed, 30-day TTL; skip redundant API calls during iteration
- **Observability** — expvar metrics for generation counts, cache hit rate, LLM calls, and errors
- **Fully static binaries** — `CGO_ENABLED=0`, no runtime dependencies, works everywhere
- **Cross-platform** — Linux, macOS, and Windows (amd64 + arm64)

---

## How it works

Cadenza works in two modes — with an LLM or fully offline — but the pipeline is the same in both cases.

**Step 0** generates a shared chord progression of 4 chords (one per group of 4 bars). This is the harmonic contract that ties the three instruments together.

**Step 1** runs three generators in parallel — bassline, arpeggio, and melody — each receiving the same chord progression. In LLM mode the model decides which notes to use, what motifs to build, and how the pattern evolves across 16 bars. In offline mode (`--no-llm`) a deterministic algorithm does the same job using a random UUID seed, so you still get different output every run.

Either way, every generated pattern goes through the same **validator** (scale membership, MIDI range, density, chord coherence) and then through the same **renderer** (micro-timing offsets, velocity grids, portamento, filter sweeps, evolution arcs). The renderer is where the professional sound comes from — those details are hardcoded as style profiles, not left to the LLM.

The result is always three MIDI files that are harmonically coherent and ready to use together in a DAW.

---

## Quickstart

```bash
# Clone and build
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza ./cmd/cadenza/

# Offline — no API key needed, hypnotic algorithmic patterns
./bin/cadenza --bpm 122 --key Am --no-llm

# With Claude (tool_use structured output)
export ANTHROPIC_API_KEY=sk-ant-...
./bin/cadenza --bpm 122 --key Am

# With Ollama (JSON schema mode)
ollama pull qwen2.5:7b
./bin/cadenza --bpm 126 --key Fm --provider ollama --model qwen2.5:7b

# With OpenAI (structured output)
export OPENAI_API_KEY=sk-...
./bin/cadenza --bpm 124 --key Em --provider openai

# With Gemini (JSON mode)
export GEMINI_API_KEY=...
./bin/cadenza --bpm 128 --key Dm --provider gemini
```

Output goes to `./output/` by default. Each run prints the generated files and seed.

---

## Setup & Configuration

Cadenza runs in five modes. Pick the one that fits you.

### Mode 1 — Offline (no API key, no internet)

The easiest option. Works out of the box.

```bash
./cadenza --bpm 122 --key Am --no-llm
```

The pattern generator is entirely algorithmic. Every run produces different output because it uses a random seed internally. The musical rules (scale, velocity, timing, chord coherence) are identical to LLM mode — you're not getting a worse product, just a different creative process.

**Use this if:** you want to experiment quickly, you don't have an API key, or you need reproducible offline generation.

---

### Mode 2 — Claude (Anthropic API)

Claude is the default provider. You need an Anthropic API key.

**Step 1** — Get your API key from [console.anthropic.com](https://console.anthropic.com). It looks like `sk-ant-api03-...`.

**Step 2** — Set it as an environment variable:

```bash
# macOS / Linux
export ANTHROPIC_API_KEY=sk-ant-api03-...

# Windows (PowerShell)
$env:ANTHROPIC_API_KEY="sk-ant-api03-..."

# Windows (Command Prompt)
set ANTHROPIC_API_KEY=sk-ant-api03-...
```

To make it permanent, add the `export` line to your `~/.zshrc`, `~/.bashrc`, or Windows environment variables.

**Step 3** — Run:

```bash
./cadenza --bpm 122 --key Am
# or explicitly:
./cadenza --bpm 122 --key Am --provider claude
```

Each run costs roughly $0.01–$0.03 depending on the model. The result is cached on disk (30-day TTL) so the same input won't hit the API twice.

---

### Mode 3 — Ollama (local LLM, no API costs)

Run a local model on your machine. No internet required after the model is downloaded.

**Step 1** — Install Ollama from [ollama.com](https://ollama.com).

**Step 2** — Pull a model (run this once):

```bash
ollama pull qwen2.5:7b        # recommended — fast and good quality
# or
ollama pull llama3.2:3b       # lighter, less VRAM
# or
ollama pull mistral:7b        # good alternative
```

**Step 3** — Start the Ollama server (keep this running in a separate terminal):

```bash
ollama serve
```

**Step 4** — Run Cadenza pointing to Ollama:

```bash
./cadenza --bpm 126 --key Fm --provider ollama --model qwen2.5:7b
```

**Minimum hardware:** 8 GB RAM for `qwen2.5:7b`. A GPU is not required but makes generation faster.

---

### Mode 4 — OpenAI

Uses GPT-4o structured output mode with JSON Schema.

```bash
export OPENAI_API_KEY=sk-...
./cadenza --bpm 124 --key Em --provider openai --model gpt-4o
```

---

### Mode 5 — Gemini

Uses Gemini 2.0 Flash in JSON mode. API key is sent via `x-goog-api-key` header (never in the URL).

```bash
export GEMINI_API_KEY=...
./cadenza --bpm 128 --key Dm --provider gemini --model gemini-2.0-flash-exp
```

---

### Choosing a mode at a glance

| | Offline | Claude | Ollama | OpenAI | Gemini |
|---|---|---|---|---|---|
| API key needed | No | Yes | No | Yes | Yes |
| Internet needed | No | Yes | No (after download) | Yes | Yes |
| Cost per run | Free | ~$0.01–0.03 | Free | ~$0.01–0.05 | ~$0.01 |
| Creative variation | Algorithmic | LLM-driven | LLM-driven | LLM-driven | LLM-driven |
| Speed | Instant | 5–15 s | 10–60 s | 5–15 s | 3–10 s |

---

## Requirements

| Mode | Requirement |
|------|-------------|
| `--no-llm` | None (static binary) |
| Claude | `ANTHROPIC_API_KEY` environment variable |
| Ollama | Ollama installed + `ollama serve` running |
| OpenAI | `OPENAI_API_KEY` environment variable |
| Gemini | `GEMINI_API_KEY` environment variable |

`make` is optional but recommended for the shortcuts below.

---

## Configuration

Cadenza reads configuration from `cadenza.yaml` (current dir, `~/.cadenza/`, or `$XDG_CONFIG_HOME/cadenza/`). Environment variables override file values.

```yaml
app:
  env: development        # development, production

audio:
  bpm: 122
  key: Am
  bars: 16
  variations: 1
  groove: straight
  max_velocity: 120

llm:
  provider: claude
  model: claude-opus-4-7
  temperature: 0.3
  max_retries: 3
  ollama_url: http://localhost:11434

output:
  dir: output

logging:
  level: info             # debug, info, warn, error
  format: text            # text, json
  file: cadenza.log

cache:
  enabled: true
  ttl_days: 30
  dir: .cache
```

Environment variable format: `CADENZA_<SECTION>_<KEY>` (e.g. `CADENZA_LLM_PROVIDER=ollama`).

Configuration is validated at startup — invalid values produce clear error messages.

---

## Usage

```
cadenza --bpm <tempo> --key <key> [options]

Core Flags:
  --bpm float       Tempo in BPM (80-150)
  --key string      Musical key: Am, D, F#m, Bb, Em, ...
  --output string   Output directory (default: output)
  --no-llm          Offline deterministic generation, no API calls
  --provider string LLM provider: claude, ollama, openai, gemini (default: claude)
  --model string    Model name (default per provider)
  --version         Print version and exit

Advanced Flags:
  --seed uint64           Deterministic seed (0 = random, printed to stdout)
  --single-file           Output MIDI Type-1 with 3 tracks in one file
  --bars int              Number of bars: 16, 32, 64, 128 (default: 16)
  --progression string    Custom chord progression (e.g. "Am-F-C-G")
  --drums                 Add drum pattern (kick/clap/hihat on CH10)
  --variations int        Generate N versions with incremental seeds (default: 1)
  --groove string         Timing preset: straight, mpc60, linndrum, humanize
  --dump-spec dir         Dump PatternSpec YAML to directory (for inspection)
  --from-spec file        Re-render from PatternSpec file (bypasses LLM)
  --watch                 Stay in loop: Enter generates new variation, 'q' exits
  --dry-run               Execute pipeline without writing files, print summary
  --dev                   Interactive dev mode REPL
```

### Advanced flag examples

**Reproducible generation:**
```bash
# Use a specific seed
./cadenza --bpm 122 --key Am --seed 42

# The seed is printed — save it to reproduce later
# Output: seed: 1234567890
./cadenza --bpm 122 --key Am --no-llm
```

**Custom chord progressions:**
```bash
# Classic i-VI-III-VII
./cadenza --bpm 124 --key Am --progression "Am-F-C-G"

# Minor with borrowed chords
./cadenza --bpm 126 --key Dm --progression "Dm-Bb-F-C"
```

**Longer arrangements:**
```bash
# 32-bar breakdown section
./cadenza --bpm 122 --key Am --bars 32

# 64-bar full track structure
./cadenza --bpm 128 --key Fm --bars 64 --drums
```

**A/B variations:**
```bash
# Generate 5 versions with incremental seeds
./cadenza --bpm 122 --key Am --variations 5

# Files: output_bassline_Am_122_v1.mid ... v5.mid
```

**Groove & timing:**
```bash
# MPC60 swing (54%)
./cadenza --bpm 122 --key Am --groove mpc60

# LinnDrum shuffle (58%)
./cadenza --bpm 126 --key Fm --groove linndrum

# Subtle humanization (+/-5 ticks)
./cadenza --bpm 124 --key Em --groove humanize
```

**Inspection & re-rendering workflow:**
```bash
# 1. Generate and dump the raw spec
./cadenza --bpm 122 --key Am --dump-spec ./specs

# 2. Inspect specs/spec_bassline.yaml in editor

# 3. Modify by hand (optional)

# 4. Re-render without calling LLM
./cadenza --from-spec ./specs/spec_bassline.yaml
```

**Dry run (validate pipeline without writing files):**
```bash
./cadenza --bpm 122 --key Am --dry-run
```

### Key formats

| Input | Meaning |
|-------|---------|
| `Am`  | A natural minor |
| `A`   | A major |
| `F#m` | F-sharp minor |
| `Bb`  | B-flat major |
| `Em`  | E minor |

All standard English major and minor key names are accepted.

---

## Output files

Each run produces three MIDI files named `output_<type>_<key>_<bpm>_<timestamp>.mid`:

| File | MIDI range | Active steps | CC channels |
|------|-----------|--------------|-------------|
| `bassline` | A1-G3 (33-55) | 8-13 / 16 | CC5 + CC65 (portamento), CC74 (filter) |
| `arpeggio` | C3-C6 (48-84) | 12-16 / 16 | CC74 (filter sweep) |
| `melody`   | C4-C7 (60-96) | 4-10 / 16  | CC1 (mod wheel) |

All files: 480 ticks/beat, 16th-note grid, MIDI Type-0, channel 1. Bar count configurable via `--bars` (16, 32, 64, 128).

---

## DAW workflow

1. Generate the three files with `cadenza`
2. Import into your DAW as three separate MIDI tracks
3. Assign synths by role:
   - **Bassline** -> sub bass synth (Diva, TAL-BassLine, Sub 37, Massive)
   - **Arpeggio** -> pad or pluck (Serum, Vital, Omnisphere)
   - **Melody** -> lead (Repro-1, Sylenth1, Analog Lab)
4. Route CC74 to filter cutoff on bass and arp; CC1 to expression on melody
5. Add drums, effects, and arrange

The three files are intentionally on the same MIDI channel so you can assign synths freely in the DAW without channel routing.

---

## Renderer pipeline (per note)

The renderer turns a `PatternSpec` from the LLM into DAW-safe MIDI events. Steps in order:

1. **Evolution** — `introduce / build / peak / release / octave_up / density_up / ...` with proportional velocity and density changes (minimum active floor: 4 steps)
2. **Timing** — downbeat always on-grid (invariant); offbeats shifted by the profile's per-step `OffbeatOffset` array
3. **Velocity** — ghost: 35-55 (clamped 60); normal/accent driven by `AccentGrid`; scaled by evolution + DynamicCurve (bass/melody: crescendo 0.7->1.0; arp: arch 0.75->1.0->0.85; bass_driving: plateau; arp_epic: tension)
4. **Gate** — ratio x ticksPerStep; slide notes extend to next active note tick + 5 (true portamento feel)
5. **Portamento** — CC5 at tick 0; CC65=127 fires 5 ticks before slide notes; skipped when tick < 10
6. **Filter sweep** — CC74 synced to evolution arc; S-curve or exponential; FNV-1a jitter for non-robotic automation
7. **Event ordering** — same-tick priority: CC < NoteOff < NoteOn (DAW-safe)

---

## LLM integration

| Provider | Structured output method | Retry strategy |
|----------|--------------------------|----------------|
| Claude | `tool_use` with `generate_pattern` tool + JSON Schema | Retry on musical violations only (structural errors eliminated by tool_use) |
| Ollama | Full JSON schema object in `format` field | Retry handles both structural and musical errors |
| OpenAI | Structured output with JSON Schema | Retry on musical violations |
| Gemini | JSON mode with `x-goog-api-key` header auth | Retry handles both structural and musical errors |

All providers share the same retry logic: up to 3 attempts with exponential back-off, context-cancellable. Each failed attempt appends a correction message with the invalid output and validation errors. Error classification distinguishes structural (JSON parse) from musical (validation) failures and tailors the correction prompt accordingly.

**Cache:** SHA256(provider + type + key + mode + seed + prompt_hash), 30-day TTL on disk. Prompt template changes automatically invalidate the cache.

---

## Style profiles

Each pattern type has dedicated deterministic style profiles. The renderer selects one automatically based on the spec, or you can pin a profile in the spec.

| Type | Profiles | DynamicCurve |
|------|---------|--------------|
| Bassline | `bass_progressive`, `bass_driving`, `bass_sub` | crescendo, plateau |
| Arpeggio | `arp_flowing`, `arp_epic`, `arp_staccato` | arch, tension |
| Melody   | `melody_expressive`, `melody_hypnotic` | crescendo |

Each profile encodes: per-step timing offsets, accent velocity grids, gate ratios for normal/accent/ghost/slide/staccato/legato notes, portamento settings, filter sweep curve + range, and mod wheel configuration.

---

## Offline mode (`--no-llm`)

Generates professional-quality patterns algorithmically with a unique UUID seed each run:

- **Bassline** — 6 rhythmic patterns per chord section (driving, syncopated, octave jump, minimal, rolling, pulsing); chromatic approach notes (true semitone below)
- **Arpeggio** — 5 shapes per chord (ascending, descending, pendulum, skip, pulse); anti-monotony forces at least 2 different shapes across 4 sections
- **Melody** — 5 contours x 4 rhythms; chord-tone gravity toward the closest chord tone; validated MIDI range

Every offline run produces different patterns because the UUID seed varies. The offline path passes through the same validator and renderer as the LLM path — the musical rules are identical.

---

## Musical rules (enforced by the validator)

These are invariants. The validator enforces them regardless of what the LLM outputs:

- Notes must be in the declared scale
- Notes must be within the pattern type's MIDI range
- Density must fall within the pattern-type window
- Chord coherence: per-section chord tone ratios (bassline 75%, arpeggio 80%, melody 30%)
- BPM must be 80-150
- Style profile must be a registered name in the profile registry
- Bars must be 16, 32, 64, or 128
- Velocity max: 120 — 127 is never used (it clips on hardware)
- Downbeat always on-grid — no timing offset on step 0
- Ghost velocity: 35-55, never above 60
- Filter sweep: S-curve or exponential, never linear

---

## Architecture

```
cmd/cadenza/                CLI entry point, interactive mode, provider setup
  main.go                   Flag parsing, generation orchestration
  provider.go               LLM provider construction + model defaults
  logger.go                 Structured logging setup (slog)
  cli.go                    Interactive CLI prompts
  dev.go                    Dev mode REPL
internal/
  theory/                   Key parser, scales, note<->MIDI, chords, progressions
  schema/                   PatternSpec types + musical validator (chord coherence)
  llm/                      Provider interface, Claude, Ollama, OpenAI, Gemini, Mock, retry
  generator/                Single + multi-pattern generators, offline templates, cache integration
  renderer/                 MIDI rendering: velocity, timing, gate, portamento, sweep, evolution
  renderer/styleprofile/    Deterministic style profiles with DynamicCurve
  midi/                     MIDI Type-0 writer with priority-based event ordering
  cache/                    SHA256-keyed disk cache (30-day TTL)
  config/                   Viper-based configuration with env var override + validation
  metrics/                  expvar counters (generations, errors, cache hits, LLM calls)
  logger/                   Structured logger initialization
prompts/                    LLM prompt templates (bassline, arpeggio, melody)
```

---

## Cross-platform builds

```bash
make build-all
# Produces:
#   bin/cadenza-linux-amd64
#   bin/cadenza-linux-arm64
#   bin/cadenza-darwin-amd64
#   bin/cadenza-darwin-arm64
#   bin/cadenza-windows-amd64.exe
#   bin/cadenza-windows-arm64.exe
```

All builds are fully static (`CGO_ENABLED=0`) — no shared libraries, no runtime dependencies.

---

## Docker

```bash
# Build image (runs as non-root user)
make docker

# Run offline
make docker-run

# With docker-compose (supports claude + ollama services)
docker compose up cadenza

# With Ollama via docker-compose (includes healthcheck)
docker compose up cadenza-ollama
```

The Docker image runs as a non-root `cadenza` user for security.

---

## Makefile reference

```bash
make build              # compile to bin/cadenza
make build-all          # cross-compile all 6 platforms
make test               # all unit tests
make test-race          # with race detector
make test-coverage      # coverage report
make lint               # run golangci-lint
make vet                # go vet ./...
make vuln               # govulncheck vulnerability scan
make ci                 # full local CI pipeline: build -> fmt -> vet -> lint -> vuln -> coverage
make install-tools      # install golangci-lint, govulncheck, goimports
make run-offline        # quick run: 122 BPM, Am, no-llm
make listening-test     # generate patterns across multiple keys/BPMs for DAW comparison
make release            # build + zip all platforms into dist/
make release-snapshot   # build all platforms without packaging (smoke test)
make docker             # build Docker image
make docker-run         # run container (offline mode)
make sonar              # run SonarScanner locally (requires SONAR_TOKEN)
make clean              # remove bin/, output/, coverage
make help               # show all targets
```

---

## Testing

```bash
go test ./...                          # all unit tests
go test ./... -race                    # race detector
go test ./internal/renderer/ -v        # renderer (evolution, velocity, timing)
go test ./internal/generator/ -v       # multi-generator pipeline
go test ./internal/schema/ -v          # validator (chord coherence, density, range)
go test ./internal/cache/ -v           # cache TTL, hit/miss, expiry
go test ./internal/config/ -v          # config loading, validation, env override
make test-coverage                     # HTML coverage report
make listening-test                    # generate files for A/B comparison in DAW
```

---

## Extending

### Add a new style profile

1. Create `internal/renderer/styleprofile/bass_techno.go` with a `var BassTechno = StyleProfile{...}`
2. Register it in `registry.go`: `r.Register(&BassTechno)`
3. The validator automatically accepts all profiles registered in the Registry (single source of truth)

### Add a new LLM provider

1. Implement the `llm.Provider` interface:

```go
type Provider interface {
    Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
    Name() string
}
```

2. Wire it in `cmd/cadenza/provider.go` inside the `buildProvider` switch

### Add a new evolution action

Add a case to `applyEvolution` in `internal/renderer/evolution.go` and document the semantics in `SPECS.md`.

---

## License

MIT
