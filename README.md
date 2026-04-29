# CADENZA

AI-powered MIDI generator for progressive house and melodic techno. Give it a BPM and a musical key; get three professional MIDI files (bassline, arpeggio, melody) back.

## How it works

```
BPM + Key
  ↓
Chord Progression (4 chords, harmonic contract)
  ↓                    ↓                    ↓
Bass Generator    Arp Generator    Melody Generator   ← parallel (LLM or offline)
  ↓                    ↓                    ↓
Validator ──────────────────────────────────────── (scale, range, density, chord coherence)
  ↓                    ↓                    ↓
Style Profile    Style Profile    Style Profile      ← DynamicCurve + evolution
  ↓                    ↓                    ↓
Renderer ───────────────────────────────────────── (velocity, timing, gate, CC)
  ↓                    ↓                    ↓
bassline.mid      arpeggio.mid      melody.mid
```

Each generator receives the same chord progression, so the three MIDI files are harmonically coherent out of the box. The LLM owns musical creativity; the renderer applies deterministic timing, velocity grids, portamento, and filter automation.

If the LLM fails (network error, exhausted retries), the system automatically falls back to offline generation — it never crashes.

## Requirements

- Go 1.24+
- `ANTHROPIC_API_KEY` set (Claude mode) **or** Ollama running locally (Ollama mode) **or** `--no-llm` (no requirements)
- `make` (optional, for shortcuts)

## Quickstart

```bash
# Clone and build
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza ./cmd/cadenza/

# Offline (no LLM required — hypnotic algorithmic patterns)
./bin/cadenza --bpm 122 --key Am --no-llm

# With Claude (tool_use structured output)
export ANTHROPIC_API_KEY=sk-ant-...
./bin/cadenza --bpm 122 --key Am

# With Ollama (JSON schema mode)
ollama pull qwen2.5:7b
./bin/cadenza --bpm 126 --key Fm --provider ollama --model qwen2.5:7b
```

Output goes to `./output/` by default.

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

All builds are static (`CGO_ENABLED=0`) — no dependencies, fully portable.

## Usage

```
cadenza --bpm <tempo> --key <key> [options]

Flags:
  --bpm float       Tempo in BPM (80-150)
  --key string      Musical key: Am, D, F#m, Bb, Em, ...
  --output string   Output directory (default: output)
  --no-llm          Offline deterministic generation, no API calls
  --provider string LLM provider: claude (default), ollama
  --model string    Model name (default: claude-opus-4-7 / qwen2.5:7b)
  --version         Print version and exit
```

### Key formats

| Format | Meaning |
|--------|---------|
| `Am`   | A natural minor |
| `A`    | A major |
| `F#m`  | F# minor |
| `Bb`   | Bb major |
| `Em`   | E minor |

## Output files

Each run produces three MIDI files named `output_<type>_<key>_<bpm>.mid`:

| File | Range | Notes |
|------|-------|-------|
| `bassline` | A1–G3 (MIDI 33–55) | 8–13 active steps, portamento CC5/CC65, filter CC74 |
| `arpeggio` | C3–C6 (MIDI 48–84) | 12–16 active steps, filter CC74 |
| `melody`   | C4–C7 (MIDI 60–96) | 4–10 active steps, space and legato phrasing |

All three files: 16 bars, 16th-note grid, 480 ticks/beat, MIDI Type-0, channel 1.

## Docker

```bash
# Build image
make docker

# Run offline
make docker-run

# With docker-compose (supports claude + ollama services)
docker compose up cadenza
```

## Makefile

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

## Architecture

```
cmd/cadenza/         CLI entry point + interactive mode
internal/
  theory/               Key parser, scales, note↔MIDI, chords, progressions
  schema/               PatternSpec types + musical validator (with chord coherence)
  llm/                  Provider interface, Claude (tool_use), Ollama (JSON schema), Mock, retry
  generator/            Single + multi-pattern generators, offline templates, cache integration
  renderer/             MIDI rendering — velocity, timing, gate, portamento, sweep, evolution
  renderer/styleprofile/ Deterministic style profiles with DynamicCurve
  midi/                 MIDI Type-0 writer with priority-based event ordering
  cache/                SHA256-keyed disk cache (30-day TTL)
prompts/                LLM prompt templates
```

### Renderer pipeline (per note)

1. **Evolution** — `introduce`/`build`/`peak`/`release`/`octave_up`/`octave_down`/`density_up`/`density_down` with velocity scaling and density changes (min floor: 4 steps)
2. **Timing** — downbeat on-grid (invariant); offbeats shifted by profile's `OffbeatOffset` array
3. **Velocity** — ghost: 35–55 (clamped 60); normal/accent from `AccentGrid`, scaled by evolution + DynamicCurve (crescendo/arch)
4. **Gate** — ratio × `ticksPerStep`; slide extends to next active note tick + 5 (true portamento)
5. **Portamento** — CC5 at tick 0; CC65=127 fires 5 ticks before slide notes; skipped when tick < 10
6. **Filter sweep** — CC74 synced to evolution arc; S-curve/exponential; FNV-1a jitter with variation seed
7. **Event ordering** — same-tick: CC < NoteOff < NoteOn (DAW-safe)

### LLM integration

- **Claude** — `tool_use` with `generate_pattern` tool; JSON Schema forces valid structure; retry for musical violations only
- **Ollama** — full JSON schema object in `format` field; retry handles both structural and musical errors
- **System prompt** — rules and constraints separated from the generation task
- **Retry** — max 3 attempts; classifies errors (structural vs musical); targeted correction prompts
- **Cache** — SHA256(provider+type+key+mode+seed), 30-day TTL; skips API call on cache hit
- **Graceful fallback** — on total LLM failure, automatically uses offline template (never crashes)
- **Chord coherence** — validator ensures at least one chord tone per 4-step section

## Offline mode

The `--no-llm` flag generates professional-quality patterns algorithmically:

- **Bassline**: 6 rhythmic patterns per chord section (driving, syncopated, octave jump, minimal, rolling, pulsing) — octave jump range-checked
- **Arpeggio**: 5 shapes per chord (ascending, descending, pendulum, skip, pulse) — anti-monotony forces at least 2 different patterns across 4 sections
- **Melody**: 5 contours × 4 rhythms — chord-tone gravity uses closest chord tone (not forced root); MIDI range validated
- **Style direction**: minor + high BPM → techno intensity; major → progressive feel
- **Profile selection**: seed-hash-based selection prepares for future profile expansion

Every run uses a unique UUID seed → different patterns every time.

## Musical rules (enforced by validator)

- Notes must be in the declared scale
- Notes must be within the pattern type's MIDI range
- Density must fall within the pattern-type window
- Each 4-step section must contain at least one chord tone (chord coherence)
- BPM must be 80–150
- Style profile must be a known registered name
- Bars must be exactly 16

## Extending

### Add a new style profile

1. Create `internal/renderer/styleprofile/bass_techno.go` with a `var BassTechno = StyleProfile{...}`
2. Register it in `registry.go`: `r.Register(&BassTechno)`
3. Add the name to `validStyleProfiles` in `internal/schema/validator.go`

### Add a new LLM provider

Implement `llm.Provider`:

```go
type Provider interface {
    Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
    Name() string
}
```

Then wire it in `cmd/cadenza/main.go`'s `buildProvider`.

### Add a new evolution action

Add a case to `applyEvolution` in `internal/renderer/evolution.go` and document the semantics in `SPECS.md §3`.

## Testing

```bash
go test ./...                          # all unit tests
go test ./... -race                    # race detector
go test ./internal/renderer/ -v        # renderer (evolution density, velocity, timing)
go test ./internal/generator/ -v       # multi-generator pipeline
go test ./internal/schema/ -v          # validator (chord coherence)
make test-coverage                     # HTML coverage report
```

## License

MIT
