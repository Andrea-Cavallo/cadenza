# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed
- **Dead code cleanup** — Removed 480+ lines of unused code identified through static analysis
  - `internal/examples/types.go` — Unused example bank types (never referenced anywhere)
  - `internal/cache/session.go` — Unused in-memory session cache (CLI uses only disk cache)
  - `internal/renderer/groove/groove.go` — Unused groove profiles (logic duplicated in service layer)
  - `internal/midi/writer_type1.go` — Unused Type-1 MIDI writer (logic duplicated in renderer service)
  - `formatChords()` function in `internal/service/cadenza.go` — Duplicate of `formatProgression()`

### Added
- **REFACTOR.md Priority 1-3 Improvements** — Musical quality and renderer enhancements
  - **Priority 1: Offline Pattern Quality**
    - Arpeggio: Eric Prydz/Opus-style open voicings (root in low octave, 3rd/5th spread high), quasi-legato gate (0.85-0.92), 4 inversion patterns (wide open, 1st/2nd inversion, drop-2), 6 rhythmic variations per section
    - Melody: Hypnotic motifs (3-5 notes) with micro-variations, wide intervals (4ths/5ths), note lengths 2-4 steps, 5 rhythm patterns (quarter feel, dotted 8th, sparse, syncopated, call-response)
    - Bassline: 8 pattern variations per 4-bar section (driving 16ths, syncopated, octave bounce, long sub, rolling triplet, off-beat pump, chromatic approach, minimal break), section 4 always simplified for release
  - **Priority 2: Renderer Improvements**
    - Multi-cycle filter sweep: 4-bar cycles instead of 16-bar monolithic, dramatic mode with up-down-up-peak pattern (2 complete sweeps over 16 bars)
    - Dynamic curves: added `plateau` (0.7→1.0 rise, hold, 0.85 decay) and `tension` (0.6→0.8→1.1→0.7 spike) to existing crescendo/arch
    - Evolution reale: `build` adds ghost fill notes on off-beat, `peak` applies octave-up to 30% of notes, `release` removes fills, bar-to-bar mutation every 4 bars (1-2 steps toggle ghost)
  - **Priority 3: Profile System**
    - New profiles: `bass_sub` (gate 0.95, no portamento, flat curve, root notes only), `arp_staccato` (gate 0.20, ping-pong timing, flat curve)
    - BPM-based profile selection: <118 prefers deep/ambient, 118-128 progressive/epic, >128 driving/staccato
    - Key-mode influence: minor keys lean toward darker profiles (hypnotic melody, driving bass)

### Changed
- **Go version downgrade** — Changed from Go 1.26 to Go 1.25 across all configuration files
  - `go.mod`: `go 1.26` → `go 1.25`
  - `.golangci.yml`: `go: "1.26"` → `go: "1.25"`
  - `Dockerfile`: `golang:1.26-alpine` → `golang:1.25-alpine`
  - `CLAUDE.md`: All references updated to Go 1.25
  - Reason: golangci-lint v1.64.8 requires Go >= 1.23 but is compiled with Go 1.25, and refuses to analyze code declaring go 1.26

### Added
- **Service Layer (REFACTOR.md point 21)** — Complete API-ready business logic layer with zero CLI dependencies
  - `internal/service/cadenza.go` — Main orchestration service implementing `CadenzaService` interface
  - `internal/service/renderer.go` — Rendering service with MIDI Type-0/Type-1 support and groove timing
  - `internal/service/generator.go` — Pattern generation service with LLM and offline template support
  - `internal/service/validator.go` — Validation service with structured error parsing
  - `internal/service/spec.go` — PatternSpec disk I/O service (YAML/JSON dump/load)
  - `internal/service/interfaces.go` — Public interfaces for all services
  - `internal/service/types.go` — Request/response types (no JSON strings, pure Go types)
  - All services accept `context.Context` as first parameter for cancellation and tracing
  - Services return structured Go types, not raw JSON strings
  - Zero business logic remains in `cmd/cadenza/` (CLI is thin adapter only)
- **Structured Logging (REFACTOR.md point 22)** — JSON logging with run correlation
  - `internal/logger/logger.go` — Configurable logger with environment-aware format
  - `--log-level` flag: debug, info, warn, error (default: info)
  - `--dev` flag: enables text format + debug level for development
  - JSON format in production/CI for parsing and correlation
  - `run_id` UUID on every log entry for correlation across services
  - Structured fields per REFACTOR.md point 22 table: provider, key, bpm, bars, seed, pattern_type, duration_ms, tokens_used, error_type, cache_key, etc.
  - Debug-level logging for LLM calls, cache hits/misses, rendering duration
- **Groove Timing Integration (--groove flag)** — Timing offsets applied in renderer service
  - `straight` (default): no offset
  - `mpc60`: Akai MPC swing 54% (±8 ticks on offbeats)
  - `linndrum`: LinnDrum shuffle 58% (±12 ticks)
  - `humanize`: micro-variations ±5 ticks per step
  - Applied in `RendererService.renderType0()` and `renderType1()` before rendering
- **--dump-spec Integration** — PatternSpec YAML dumping wired to `CadenzaService.Run()`
  - Specs dumped to `<outputDir>/specs/` when `RunRequest.DumpSpec = true`
  - All three patterns (bassline, arpeggio, melody) dumped automatically
  - Uses `SpecService.DumpSpec()` for format-aware writes
- **--single-file Foundation** — MIDI Type-1 writer integrated in `RendererService`
  - `RendererService.renderType1()` combines all tracks into one file
  - Proper MTrk chunk building with track names
  - Format 1 header with correct track count
  - Foundation ready for full CLI wiring

### Changed
- **CLI Layer Refactored** — `cmd/cadenza/main.go` now uses service layer exclusively
  - No direct calls to `generator`, `renderer`, `llm`, or `midi` packages
  - All business logic moved to `internal/service/`
  - CLI handles only: flag parsing, I/O formatting, ANSI colors, user prompts
  - Prepared for future `cmd/server/` HTTP API with zero duplication
- **Cache Construction** — Fixed `cache.New()` signature (now `cache.New(ttlDays, dir)`)
  - Updated `generator.NewGenerator()` and `multi_generator.go` to pass 30-day TTL
- **Service Type Alignment** — Fixed type mismatches between service and domain layers
  - `GenerateRequest.ChordProgression` now `theory.ChordProgression` (not `[]theory.Chord`)
  - `RendererService.RenderPattern()` uses `*styleprofile.StyleProfile` (concrete type)
  - `ValidatorService.ValidateWithChords()` accepts `theory.ChordProgression`
  - All service methods use context-first signatures

### Fixed
- Groove timing offset field corrected to `TimingProfile.OffbeatOffset` (was incorrectly `OffsetTicks`)
- MIDI Type-1 track struct definition for inline anonymous types in `renderType1()`
- Custom progression parsing now correctly joins `[]string` to `-`-separated string for logging

### Added
- **Schema inference from Go structs** — Use `invopop/jsonschema` library to generate complete JSON Schema from `PatternSpec` struct, replacing fragile manual inference that could miss optional fields (REFACTOR.md point 1)
- **Differentiated chord coherence thresholds** — Pattern-type-specific validation: bassline 75%, arpeggio 80%, melody 30% instead of uniform "at least 1 chord tone" check (REFACTOR.md point 2)
- **Retry correction prompts with positive examples** — LLM retry messages now include correct step format examples for the failing section, improving fix success rate (REFACTOR.md point 3)
- **Filter sweep phase offset per pattern type** — Bassline/arpeggio/melody now have staggered sweep curves (0/0.25/0.5 phase offset) to prevent simultaneous filter peaks when all three tracks are used together (REFACTOR.md point 5)
- **Cache invalidation on prompt template changes** — Prompt template content hash included in cache key; schema or prompt changes now automatically invalidate stale cached responses (REFACTOR.md point 6)
- **CLI Flag: `--seed uint64`** — Deterministic seed for reproducible generation; prints to stdout if random
- **CLI Flag: `--single-file`** — MIDI Type-1 output with 3 tracks in one file (foundation implemented)
- **CLI Flag: `--bars int`** — Variable bar count (16, 32, 64, 128); default 16
- **CLI Flag: `--progression string`** — Custom chord progression parser (e.g., "Am-F-C-G")
- **CLI Flag: `--drums`** — Drum pattern generator (kick/clap/hihat on CH10)
- **CLI Flag: `--variations int`** — Generate N versions with incremental seeds (default 1)
- **CLI Flag: `--groove string`** — Timing presets: straight, mpc60, linndrum, humanize
- **CLI Flag: `--dump-spec dir`** — Dump PatternSpec to YAML (foundation for inspection workflow)
- **CLI Flag: `--from-spec file`** — Re-render from PatternSpec YAML file (bypasses LLM)
- **CLI Flag: `--watch`** — Watch mode: stay in loop on stdin, each Enter generates new variation with incremental seed, 'q' exits (REFACTOR.md point 15)
- **CLI Flag: `--dry-run`** — Execute full pipeline (LLM, validation, rendering) without writing files; print summary (tokens, cost, seed) (REFACTOR.md point 19)
- **CLI Flag: `--dev`** — Interactive dev mode REPL with commands: generate, render, validate, inspect, chord-progression, cache-info, help, exit (REFACTOR.md point 20)
- **Provider: OpenAI** — gpt-4o structured output with JSON Schema support (REFACTOR.md point 16)
- **Provider: Gemini** — Gemini 2.0 Flash JSON mode with schema validation (REFACTOR.md point 16)
- **Benchmark tests** — Added BenchmarkRenderer, BenchmarkRendererArpeggio, BenchmarkValidator, BenchmarkValidatorArpeggio, BenchmarkChordCoherenceCheck for performance regression tracking (REFACTOR.md point 18)
- Cache statistics tracking (`Cache.Stats()`) for hit/miss rate and key count inspection in dev mode
- Validator support for variable bars (16/32/64/128) instead of hardcoded 16
- Groove profile system (`internal/renderer/groove/`) with MPC60, Linndrum, humanize presets
- MIDI Type-1 writer (`internal/midi/writer_type1.go`) with multi-track support
- Drum pattern generator (`internal/generator/drums.go`) with 4-on-floor kick, clap, hi-hat
- PatternSpec YAML I/O (`internal/schema/spec_io.go`) for dump/load workflow
- Custom chord progression parser in main.go supporting major/minor/dim/aug/sus2/sus4
- CLI flag validation for bars (powers of 2), variations (1-100), groove (valid presets)
- Claude `tool_use` structured output with `generate_pattern` tool and forced tool choice
- System prompt separation (rules in system, task in user message)
- LLM response cache integration (SHA256 keyed, 30-day TTL, automatic hit/miss)
- Ollama JSON schema format (full schema object instead of bare `"json"`)
- Retry error classification: structural (JSON parse) vs musical (validation) with different correction prompts
- Enriched chord progression in prompts (includes chord notes for each section)
- Token usage logging per LLM call (provider, tokens, latency, attempt)
- Graceful LLM fallback — on failure, automatically uses offline template instead of crashing
- Chord coherence validation — validator checks at least one chord tone per section
- `Provider.Name()` method on all LLM providers
- Offline melody: MIDI range validation (clamp octave if note out of 60-96)
- Offline arpeggio: anti-monotony (forces at least 2 different patterns across 4 sections)
- Offline bassline: octave jump range check (clamp to octave 2 if root+"3" exceeds MIDI 55)
- Offline melody: closest chord tone selection (respects contour direction, not forced root)
- Evolution `introduce` floor — never deactivates below 4 steps minimum
- DynamicCurve assigned to all profiles (bass=crescendo, arp=arch, melody=crescendo)
- Filter sweep receives variation seed for unique jitter per generation
- Portamento CC65 skipped when tick < 10 (no sense at bar 0 start)
- Evolution density tests (introduce/peak/release coverage)
- Profile selection logic using seed hash (prepares for future profiles)
- Windows ARM64 target in Makefile and goreleaser
- Cross-platform path handling (`filepath.Join` for prompt paths)


### Changed
- Claude provider uses `tool_use` API instead of raw text parsing
- Ollama provider sends full JSON schema object in `format` field (was: `"json"` string)
- `SingleGenerator` accepts and uses `*cache.Cache`
- `MultiGenerator` creates cache instance automatically
- Retry `appendCorrection` uses targeted prompts based on error type
- Validator split into `Validate()` and `ValidateWithChords()` for chord-aware validation
- `progressionString` now includes chord notes (e.g., `Am (notes: A, C, E)`) in LLM prompts
- Offline `chooseBassProfile`/`chooseArpProfile`/`chooseMelodyProfile` now use candidate lists + seed hash
- `deterministicJitter` signature includes seed string for per-generation uniqueness
- `filterSweepEvents` signature includes `variationSeed` parameter

### Fixed
- **Offline melody legato boundary bug** — Legato no longer applied to first note of new chord section (4-step boundaries), preventing incorrect portamento across chord changes (REFACTOR.md point 4)
- Claude responses with markdown fences or extra text (tool_use eliminates this class of errors)
- Ollama small models producing wrong field names (JSON schema format fixes this)
- LLM failure causing entire session to crash (now gracefully falls back to offline)
- Same seed producing identical filter sweep jitter across different generations
- Melody offline forced chord root breaking contour direction
- Arpeggio offline could produce 4 identical patterns (monotony)
- Bassline octave jump could exceed MIDI range for high root notes
- Melody notes could fall outside 60-96 range for edge-case keys
- Evolution `introduce` could over-deactivate below minimum density
- Portamento CC65 at tick 0 could conflict with portamento time CC5

## [0.3.0] - 2026-04-27

### Added
- Multi-pattern parallel generation (3 goroutines)
- Shared chord progression (Step 0) ensuring harmonic coherence
- Claude provider with structured text output
- Ollama provider with JSON mode
- Retry mechanism with correction prompts (max 3 attempts, exponential backoff)
- MockProvider for unit tests
- Musical validator: scale membership, MIDI range, density, BPM bounds
- Prompt templates for bassline, arpeggio, melody with template variables

### Changed
- Generator architecture split into SingleGenerator and MultiGenerator
- Provider interface uses `Messages[]` instead of flat SystemPrompt/UserPrompt

## [0.2.0] - 2026-04-26

### Added
- Style profiles: `bass_progressive`, `arp_flowing`, `melody_expressive`
- Full renderer pipeline: timing, velocity, gate, portamento, filter sweep
- Theory package: key parser, scales (5 types), chords (6 qualities), progressions (8 templates)
- MIDI Type-0 writer with variable-length encoding
- PatternSpec schema with evolution and automation intents
- Variation seed (UUID v4) for output diversity

## [0.1.0] - 2026-04-25

### Added
- Initial project structure
- CLI entry point with BPM and key flags
- Basic MIDI writer
- Key parser for standard English notation (Am, D, F#m, Bb)
