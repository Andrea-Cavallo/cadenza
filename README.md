# CADENZA

<p align="center">
  <img src="cadenza.png" alt="Cadenza" width="480" />
</p>

> AI-powered MIDI generator for progressive house and melodic techno.  
> Give it a BPM and a musical key — get three harmonically coherent MIDI files back.

<p align="center">

[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://golang.org)
[![CI](https://github.com/Andrea-Cavallo/cadenza/actions/workflows/ci.yml/badge.svg)](https://github.com/Andrea-Cavallo/cadenza/actions/workflows/ci.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=coverage)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![CGO_ENABLED=0](https://img.shields.io/badge/CGO-disabled-lightgrey)](https://pkg.go.dev/cmd/cgo)

</p>

> **Backend component.** Cadenza is the core generation engine — a Go backend exposing musical generation logic as public service interfaces. A frontend and HTTP API layer are planned on top of this foundation.

---

## What is Cadenza?

Cadenza turns two numbers into a complete musical sketch. You provide a **BPM** and a **key** (e.g. `122 Am`). In seconds you get three MIDI files — bassline, arpeggio, melody — that are musically coherent and ready to drop into your DAW.

The LLM handles creative decisions: which notes, which motifs, how the pattern evolves over 16 bars. The renderer enforces professional musical rules that no LLM gets reliably right on its own: micro-timing offsets, velocity grids, portamento, S-curve filter sweeps, evolution arcs. Every run produces a different result even with the same input parameters.

**No LLM? No problem.** The `--no-llm` flag generates hypnotic algorithmic patterns offline, with zero API calls and zero dependencies.

---

## Features

- **One command, three MIDI files** — bassline, arpeggio, and melody generated in parallel
- **Harmonic coherence** — a shared chord progression (Step 0) binds all three generators
- **Professional renderer** — deterministic velocity grids, portamento CC5/CC65, S-curve filter sweeps, DynamicCurve evolution arcs
- **Multiple LLM backends** — Claude (`tool_use` structured output), Ollama (JSON schema mode), or no LLM at all
- **Graceful fallback** — if the LLM fails after 3 retries, offline templates take over; it never crashes
- **LLM response cache** — SHA256-keyed, 30-day TTL; skip redundant API calls during iteration
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
```

Output goes to `./output/` by default. Each run prints the generated files and total LLM cost.

---

## Setup & Configuration

Cadenza runs in three modes. Pick the one that fits you.

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

### Choosing a mode at a glance

| | Offline | Claude | Ollama |
|---|---|---|---|
| API key needed | No | Yes | No |
| Internet needed | No | Yes | No (after download) |
| Cost per run | Free | ~$0.01–0.03 | Free |
| Creative variation | Algorithmic | LLM-driven | LLM-driven |
| Speed | Instant | 5–15 s | 10–60 s |

---

## Requirements

| Mode | Requirement |
|------|-------------|
| `--no-llm` | None (static binary) |
| Claude | `ANTHROPIC_API_KEY` environment variable |
| Ollama | Ollama installed + `ollama serve` running |

`make` is optional but recommended for the shortcuts below.

---

## Usage

```
cadenza --bpm <tempo> --key <key> [options]

Flags:
  --bpm float       Tempo in BPM (80–150)
  --key string      Musical key: Am, D, F#m, Bb, Em, ...
  --output string   Output directory (default: output)
  --no-llm          Offline deterministic generation, no API calls
  --provider string LLM provider: claude (default), ollama
  --model string    Model name (default: claude-opus-4-7 / qwen2.5:7b)
  --version         Print version and exit
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

Each run produces three MIDI files named `<timestamp>_<type>_<key>_<bpm>.mid`:

| File | MIDI range | Active steps | CC channels |
|------|-----------|--------------|-------------|
| `bassline` | A1–G3 (33–55) | 8–13 / 16 | CC5 + CC65 (portamento), CC74 (filter) |
| `arpeggio` | C3–C6 (48–84) | 12–16 / 16 | CC74 (filter sweep) |
| `melody`   | C4–C7 (60–96) | 4–10 / 16  | CC1 (mod wheel) |

All files: 16 bars · 16th-note grid · 480 ticks/beat · MIDI Type-0 · channel 1.

---

## DAW workflow

1. Generate the three files with `cadenza`
2. Import into your DAW as three separate MIDI tracks
3. Assign synths by role:
   - **Bassline** → sub bass synth (Diva, TAL-BassLine, Sub 37, Massive)
   - **Arpeggio** → pad or pluck (Serum, Vital, Omnisphere)
   - **Melody** → lead (Repro-1, Sylenth1, Analog Lab)
4. Route CC74 to filter cutoff on bass and arp; CC1 to expression on melody
5. Add drums, effects, and arrange

The three files are intentionally on the same MIDI channel so you can assign synths freely in the DAW without channel routing.

---

## Renderer pipeline (per note)

The renderer turns a `PatternSpec` from the LLM into DAW-safe MIDI events. Steps in order:

1. **Evolution** — `introduce / build / peak / release / octave_up / density_up / ...` with proportional velocity and density changes (minimum active floor: 4 steps)
2. **Timing** — downbeat always on-grid (invariant); offbeats shifted by the profile's per-step `OffbeatOffset` array
3. **Velocity** — ghost: 35–55 (clamped 60); normal/accent driven by `AccentGrid`; scaled by evolution + DynamicCurve (bass/melody: crescendo 0.7→1.0; arp: arch 0.75→1.0→0.85)
4. **Gate** — ratio × ticksPerStep; slide notes extend to next active note tick + 5 (true portamento feel)
5. **Portamento** — CC5 at tick 0; CC65=127 fires 5 ticks before slide notes; skipped when tick < 10
6. **Filter sweep** — CC74 synced to evolution arc; S-curve or exponential; FNV-1a jitter for non-robotic automation
7. **Event ordering** — same-tick priority: CC < NoteOff < NoteOn (DAW-safe)

---

## LLM integration

| Provider | Structured output method | Retry strategy |
|----------|--------------------------|----------------|
| Claude | `tool_use` with `generate_pattern` tool + JSON Schema | Retry on musical violations only (structural errors eliminated by tool_use) |
| Ollama | Full JSON schema object in `format` field | Retry handles both structural parse errors and musical violations |

Both providers share the same retry logic: up to 3 attempts with exponential back-off, context-cancellable. Each failed attempt appends a correction message with the invalid output and validation errors delimited by rigid XML tags to prevent prompt injection.

**Cache:** SHA256(provider + type + key + mode + seed), 30-day TTL on disk. Use `--no-cache` to force a fresh LLM call.

---

## Style profiles

Each pattern type has dedicated deterministic style profiles. The renderer selects one automatically based on BPM and key mode, or you can pin a profile in the spec.

| Type | Profiles |
|------|---------|
| Bassline | `bass_progressive`, `bass_techno_driving`, `bass_melodic_dark`, `bass_deep` |
| Arpeggio | `arp_flowing`, `arp_rhythmic`, `arp_ambient`, `arp_plucky` |
| Melody   | `melody_expressive`, `melody_minimal`, `melody_soaring`, `melody_rhythmic` |

Each profile encodes: per-step timing offsets, accent velocity grids, gate ratios for normal/accent/ghost/slide notes, portamento settings, and filter sweep curve + range.

---

## Offline mode (`--no-llm`)

Generates professional-quality patterns algorithmically with a unique UUID seed each run:

- **Bassline** — 6 rhythmic patterns per chord section (driving, syncopated, octave jump, minimal, rolling, pulsing)
- **Arpeggio** — 5 shapes per chord (ascending, descending, pendulum, skip, pulse); anti-monotony forces at least 2 different shapes across 4 sections
- **Melody** — 5 contours × 4 rhythms; chord-tone gravity toward the closest chord tone; validated MIDI range

Every offline run produces different patterns because the UUID seed varies. The offline path passes through the same validator and renderer as the LLM path — the musical rules are identical.

---

## Musical rules (enforced by the validator)

These are invariants. The renderer enforces them regardless of what the LLM outputs:

- Notes must be in the declared scale
- Notes must be within the pattern type's MIDI range
- Density must fall within the pattern-type window
- Each 4-step section must contain at least one chord tone (chord coherence)
- BPM must be 80–150
- Style profile must be a registered name
- Bars must be exactly 16
- Velocity max: 120 — 127 is never used (it clips on hardware)
- Downbeat always on-grid — no timing offset on step 0
- Ghost velocity: 35–55, never above 60
- Filter sweep: S-curve or exponential, never linear

---

## Architecture

```
cmd/cadenza/                CLI entry point + interactive mode
internal/
  theory/                   Key parser, scales, note↔MIDI, chords, progressions
  schema/                   PatternSpec types + musical validator (chord coherence)
  llm/                      Provider interface, Claude (tool_use), Ollama (JSON schema), Mock, retry
  generator/                Single + multi-pattern generators, offline templates, cache integration
  renderer/                 MIDI rendering — velocity, timing, gate, portamento, sweep, evolution
  renderer/styleprofile/    Deterministic style profiles with DynamicCurve
  midi/                     MIDI Type-0 writer with priority-based event ordering
  cache/                    SHA256-keyed disk cache (30-day TTL)
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
# Build image
make docker

# Run offline
make docker-run

# With docker-compose (supports claude + ollama services)
docker compose up cadenza
```

---

## Makefile reference

```bash
make build          # compile to bin/cadenza
make build-all      # cross-compile all 6 platforms
make test           # all unit tests
make test-race      # with race detector
make test-coverage  # coverage report with HTML
make run-offline    # quick run: 122 BPM, Am, no-llm
make listening-test # generate patterns across multiple keys/BPMs for DAW comparison
make docker         # build Docker image
make lint           # run golangci-lint
make clean          # remove bin/, output/, coverage
make help           # show all targets
```

---

## Testing

```bash
go test ./...                          # all unit tests
go test ./... -race                    # race detector
go test ./internal/renderer/ -v        # renderer (evolution, velocity, timing)
go test ./internal/generator/ -v       # multi-generator pipeline
go test ./internal/schema/ -v          # validator (chord coherence)
make test-coverage                     # HTML coverage report
make listening-test                    # generate files for A/B comparison in DAW
```

---

## Extending

### Add a new style profile

1. Create `internal/renderer/styleprofile/bass_techno.go` with a `var BassTechno = StyleProfile{...}`
2. Register it in `registry.go`: `r.Register(&BassTechno)`
3. Add the name to `validStyleProfiles` in `internal/schema/validator.go`

### Add a new LLM provider

Implement the `llm.Provider` interface and wire it in `cmd/cadenza/main.go`:

```go
type Provider interface {
    Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
    Name() string
}
```

### Add a new evolution action

Add a case to `applyEvolution` in `internal/renderer/evolution.go` and document the semantics in `SPECS.md §3`.

---

## License

MIT
