# CADENZA

> AI-powered MIDI generator for progressive house and melodic techno.  
> Give it a BPM and a musical key — get three harmonically coherent MIDI files back.

[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![CGO_ENABLED=0](https://img.shields.io/badge/CGO-disabled-lightgrey)](https://pkg.go.dev/cmd/cgo)

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

```
BPM + Key
  ↓
Chord Progression (4 chords, harmonic contract)       ← Step 0
  ↓                    ↓                    ↓
Bass Generator    Arp Generator    Melody Generator    ← Step 1: parallel (LLM or offline)
  ↓                    ↓                    ↓
Validator ──────────────────────────────────────────── (scale, range, density, chord coherence)
  ↓                    ↓                    ↓
Style Profile    Style Profile    Style Profile        ← DynamicCurve + evolution
  ↓                    ↓                    ↓
Renderer ──────────────────────────────────────────── (velocity, timing, gate, CC)
  ↓                    ↓                    ↓
bassline.mid      arpeggio.mid      melody.mid
```

All three generators receive the same chord progression, so the MIDI files play together out of the box. The LLM owns musical creativity; the renderer owns professional sound — micro-timing, velocity grids, portamento, and filter automation are all deterministic.

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

## Requirements

| Mode | Requirement |
|------|-------------|
| `--no-llm` | Go 1.26+ only |
| Claude | `ANTHROPIC_API_KEY` env var |
| Ollama | Ollama running locally (`ollama serve`) |

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
