# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Bass groove palette per BPM/mode**: `chooseBassGroovePalette` selects genre-matched sub-pattern sets — warehouse techno (bpm ≥ 126 minor), melodic/minimal techno (minor < 126), house (major). Replaces random 4-pattern selection with a coherent 3-pattern palette, giving each style a recognizable groove identity.
- **A-B-A-B bass groove structure**: each 16-step section uses two alternating sub-patterns (beats 1&3 share p0, beats 2&4 share p1) instead of four independent patterns. Echo positions (beats 3-4) swap root↔fifth so the phrase fingerprint differs even when chords repeat across sections; musically equivalent to the standard root-on-1-fifth-on-3 bass technique.
- **Melody `HeightPref` per step**: `StepIntention` gains an `int` field carrying the per-step pitch direction from `heightsForContour` (ContourArch/QuestionAnswer/TensionHold/DescRelease). Echo and fill notes now move in the direction the contour prescribes instead of always ascending; section-level octave preference follows the same height signal.
- **`heightsForContour` helper**: maps `ContourType` → a 16-element `[stepsPerSection]int` array of -1/0/+1 pitch height preferences used by the melody phrase builder.

### Fixed
- **Bass case 3 octave out-of-range**: `root + "1"` produced MIDI values below 33 for roots C–G# (e.g. F1=29); changed to `root + "2"` so all bass notes stay in the validator range [33, 55].
- **TestSingleGenerator_GenerateBranches subtests**: the `% 7` hash mapping added for the break-pattern removal caused case 3 to be selected for F-chord sections, making the offline template fail validation and breaking the MockProvider fallback tests.
- **Arp gate lengths**: `arp_epic.go` NormalGate was 0.88 (near-legato), making arpeggios sound like held chords; reduced to 0.55 with matching AccentGate 0.65.
- **Arp articulation uniformity**: all arp steps were set to `Legato: true`, creating a wall of sustained notes instead of discrete arpeggio triggers; replaced with staccato/normal/accent mix per pattern.
- **Bass density overrun**: removing the forced silent section-3 pattern allowed patterns 0–6 which can add 3–4 active notes per section, pushing total active beyond the validator max of 13; added `ensureBassMaxDensity` to clamp ghost and non-accent steps.
- **Bass last-section silence**: section 3 was always forced to the silent/break pattern (pattern 7); now allows any pattern 0–6 for musical continuity, with optional `% 4` fallback below 120 BPM for a calmer release.
- **Piano roll 1-bar display**: `buildTrackPreview` was only showing 1 bar (16 steps) instead of all 16 bars (256 steps); now expands the motif across all bars so the full arrangement is visible.

### Added
- **Section-aware melody phrase generation**: replaced the uniform legato-stream loop with a 4-role phrase model: section 0 = statement (1–2 held notes, lots of space), section 1 = call (pickup → landing → optional echo), section 2 = tension (character/modal note + syncopated off-beat), section 3 = resolution (descend home, staccato end). Produces recognizable musical phrases instead of a block of equal-length notes.
- **`buildMelodyPhraseSection` helper**: implements per-section note selection using `closestChordTone`, `approachNote`, and `characterDegree` for musically coherent articulation and contour.
- **`melodyNote` helper**: clamps any bare note name + octave to the valid melody range [60, 84]; always safe to call, never produces out-of-range MIDI.
- **Fixed accent color switcher removal**: removed the cyan/lime/amber accent toggle from TitleBar, leaving cyan as the sole fixed visual identity; removed `AccentName` type, `ACCENT_MAP`, accent state and associated `useEffect` from App.tsx, and all `.accent-switch` / `.swatch` CSS.

### Added
- Wails v2 desktop app (`backend/cmd/desktop/`) with a standalone `cadenza-desktop.exe` binding the Go MIDI engine directly to a React frontend, no HTTP layer.
- `AppService` methods exposed to Wails (`Generate`, `GetProviders`, `GetModels`, `GetConfig`, `OpenOutputFolder`) plus progress events for the desktop generation log.
- Vite + React + TypeScript desktop frontend ported from the JSX prototype, including presets, piano roll, pipeline view, and local generation console.
- `make desktop`, `make desktop-dev`, and `make desktop-manual`; the manual target follows Wails' documented npm-build-plus-Go-production-build flow.
- Desktop output defaults to `~/cadenza-output/`, with a cross-platform Open Folder command.
- Desktop frontend toolchain uses Vite 8 with a clean `npm audit` result.
- Desktop provider setup panel now shows Claude/OpenAI/Gemini API-key readiness, Ollama install/runtime/model status, local model selection, Refresh, Start Ollama, and provider setup links.
- PowerShell distribution builder (`scripts/build-distributions.ps1`) packages Windows, macOS, and Linux ZIPs with cross-compiled CLI binaries and Wails desktop binaries when the target build is available locally.
- Desktop frontend redesign first pass: fixed black/cyan app shell with title bar, preset sidebar, generation controls, collapsible log drawer, output panel, and status bar.
- Desktop app logger now writes to `~/cadenza-output/logs/cadenza-desktop.log` by default and mirrors backend `slog` lines into the UI log drawer in realtime.
- Desktop MIDI preview now replaces the animated demo roll with a real post-generation PatternSpec viewer, including Bass/Arp/Melody tabs, chord progression pills, active notes, and Keep/Discard/Export all actions.
- Desktop log drawer now shows step-by-step generation progress (`provider`, `plan`, `generate`, `render`, `write`) and operational error messages with the next button or setup action to use.
- Desktop packaging now includes cross-platform distribution scripts, a multi-OS GitHub Actions matrix, Windows ZIP content validation, and the Cadenza icon for Wails desktop binaries.
- Desktop sidebar now uses a minimal producer workflow: key, BPM, bars, provider, and Generate only; frontend genre presets and the Advanced drawer were removed.
- Desktop piano roll is now editable after generation: notes can be dragged on the 1/16 grid, resized, deleted, adjusted for velocity/accent/ghost, zoomed across long timelines, and exported as fresh edited MIDI files.
- Desktop theme is now a single cyan identity: leftover track colors were neutralized, lime is reserved for success/ready states, amber for warning/setup states, and muted text contrast was raised for sidebar, log, and piano roll readability.
- Desktop offline style is now a secondary `Offline flavor` control shown only for the Offline provider, backed by listening fixtures that assert `melodic`, `hypnotic`, `driving`, and `minimal` produce distinct musical fingerprints.
- Desktop preset cleanup completed: no frontend preset structs, preset mappings, or desktop preset workflow remain; CLI presets are kept separate from the desktop UI.

### Fixed
- Desktop generation now preflights selected providers before rendering, so missing API keys or unavailable Ollama produce clear UI errors instead of silently falling back to offline output.
- Desktop provider status no longer truncates important Ollama/setup messages and now separates readiness summary, model count, and action buttons in one visible provider section.
- `install-tools` now installs golangci-lint v2 and `.golangci.yml` declares Go `1.25.9`, keeping lint aligned with the module toolchain.
- Prompt files are now embedded into the binary via `//go:embed` (`internal/prompts/`); the app no longer falls back to offline mode when the binary is run from a directory other than the project root.
- Ollama/small-model validation failures: system prompt, correction examples, and all three prompt templates now explicitly list the valid evolution `action` catalog and enforce `intensity` as a float in `[0.0, 1.0]`, preventing the "unknown action" and "intensity out of range" retry loops.
- Dynamic model catalog (`internal/models/models.yaml`) embedded in binary; add new models by editing YAML without recompiling. Local override at `~/.cadenza/models.yaml` or `./models.yaml` takes precedence. Interactive CLI now shows a numbered model menu after engine selection. 12 Ollama models pre-configured (Qwen 2.5 7B/14B/32B, Llama 3.x, Mistral, Mixtral, Gemma2, Phi-4, DeepSeek-R1, Qwen Coder).
- Range correction examples are now pattern-type-specific (bassline/arpeggio/melody octave guidance), reducing wrong-octave retries.
- Critic revision prompt now includes explicit density constraints per pattern type (bassline 8-13, arpeggio 12-16, melody 4-10), preventing revision from stripping notes below minimum density.

### Added
- **LLM MusicalPlan foundation**: `internal/generator/planner.go` adds a production-intent plan before PatternSpec generation, including mood, energy, tension curve, rhythmic identity, motif concept, density target, call/response, section intentions, track separation, and revision priorities.
- **Producer style cards**: LLM prompts now receive compact style-card direction (`afterlife-dark`, `prydz-progressive`, `minimal-hypnotic`, `festival-melodic`, `warehouse-driving`) derived from mode, BPM, and `MusicContext.OfflineStyle`.
- **`{{MUSICAL_PLAN}}` prompt injection**: bassline, arpeggio, and melody prompts now include the generated musical plan before note selection, making AI mode more intentional and less one-shot/mechanical.
- **Musical scoring report**: `Validator.ScoreMusicality()` adds soft quality metrics for repeated 4-step phrases, downbeat chord-tone ratio, pitch contour diversity, articulation flatness, section densities, and warnings.
- **Planner-aware cache keys**: LLM PatternSpec cache keys now include planner version, style-card version, and style-card name to avoid reusing stale responses after musical-intent changes.
- **Critic + targeted revision loop**: valid LLM PatternSpecs now receive a critic pass and, when needed, one targeted revision round that fixes only weak musical dimensions while preserving schema, seed, key, scale, and chord alignment.
- **Anti-loop validation**: validator now rejects 4-step phrases repeated more than twice without rhythmic, pitch, or articulation variation.
- **Cross-track arrangement scoring**: `schema.ScoreArrangement()` reports shared density peaks and melody/arpeggio pitch/register collisions across the generated trio.
- **Listening fixtures**: added `testdata/listening/llm_quality_cases.json` for A/B evaluation of one-step LLM generation vs planner+critic generation.
- **Post-run key and part iteration**: added "same motifs, new key", single-part regeneration for bassline/arpeggio/melody, and lock-progression mode to the interactive post-run menu.
- **`--json` output mode**: prints machine-readable generation summaries for scripts and DAW integrations.
- **`cadenza config init`**: scaffolds an annotated starter `cadenza.yaml`; supports `--force` to overwrite.
- **Representative example MIDI sets**: generated local examples for Am 122 BPM, Dm-dorian 128 BPM, and G-mixolydian 124 BPM under `output/examples/`.
- **Provider failure choice screen** (`handleProviderFailure`): when provider init fails in interactive mode, shows a menu — retry / switch to offline / switch to a different provider / cancel — instead of hard-exiting. Controlled by `cliConfig.Interactive` so flag mode and CI are unaffected.
- **`--doctor` flag**: runs `runDoctorCheck`, which prints a diagnostic report: Go runtime version, API key presence (Claude, OpenAI, Gemini), Ollama reachability, and output directory writability.
- **`--non-interactive` flag**: explicitly forces flag mode (skips TUI); useful in CI and headless scripts. Requires `--bpm` and `--key` (or `--from-spec`) to be present.
- **Absolute output paths**: the success screen now prints `filepath.Abs(f)` for each generated MIDI file so the path is directly usable in any working directory.
- **`cliConfig.Interactive` field**: set to `true` by `runInteractiveCLI`; gates interactive-only behaviour like the provider failure screen.
- **Doctor tests**: `TestRunDoctorCheck`, `TestPrintDoctorItem`, `TestCheckOllamaReachable` (with httptest server).
- **Provider failure tests**: `TestHandleProviderFailure` covers retry, switch-to-offline, switch-provider, cancel, and invalid-then-cancel paths.
- **Interactive flag test**: `TestRunInteractiveCLI_SetsInteractive` asserts `cfg.Interactive = true` after a successful guided session.


- **Genre presets**: four built-in presets (`progressive-warmup`, `peak-time-driver`, `afterhours-hypnotic`, `festival-melodic`) — each pre-fills key, BPM, groove, and offline style. Accessible via `--preset <name>` flag and via interactive preset picker before the full guided session.
- **Expanded post-run actions**: interactive menu now offers 8 options — new session, same setup, same harmony + new motifs (locked progression), A/B compare (outputs to `A/` and `B/` subdirs), faster (+6 BPM), slower (-6 BPM), busier (driving style), sparser (hypnotic style).
- **BPM hint after key confirmation**: `keyBPMHint()` shows typical BPM range for the chosen mode in the interactive key selector.
- **`progressionToCLIString()`**: converts a `ChordProgression` to `Am-F-C-G` format for the `--progression` flag; used by same-harmony action to lock the progression.
- **`lastRunInfo` state tracking**: successful generations update a package-level `lastRun` struct, enabling post-run actions to reference the prior seed and chord progression.
- **`clampBPM()`**: bounds BPM to [80, 150] for faster/slower directional actions.
- **README rewrite (P1)**: opening paragraph leads with what you hear; added "Install (one-liner)", "Why Cadenza?", "Producer Workflow" sections; updated Features list with all new capabilities.
- **`--offline-style` CLI flag**: exposes `MusicContext.OfflineStyle` to non-interactive flag mode (`hypnotic | driving | minimal | melodic`); validated by `validateFlags`.
- **Energy selector (1–5) in interactive CLI**: new `selectEnergy()` step between mode selection and tempo; maps energy level to groove preset and offline style suggestion; Enter skips.
- **Musical mood description after key confirmation**: `keyMoodDescription()` prints a producer-friendly emotional character summary (e.g. "A natural minor — dark, introspective, club-friendly") immediately after scale notes.
- **Reproduce command after every successful run**: `reproduceCmd()` builds and prints the exact CLI command to recreate any generation, including seed, provider, groove, offline style, and custom progression.
- **`OfflineStyle` shown in interactive summary**: `printSummary()` now shows GROOVE and STYLE when non-default values are active.
- **Coverage milestone**: `cmd/cadenza` now at 80.3% (up from 67.9%) — P3 SonarCloud quality gate passed for this package.
- **New dev mode tests**: `TestRunDevMode_QuitAndCommands`, `TestDevGenerate`, `TestDevValidate` cover the interactive REPL, generation, and validation helpers.
- **New CLI helper tests**: `TestKeyMoodDescription`, `TestReproduce`, `TestRunInteractiveCLI_WithEnergy`, `TestConfirmInteractiveRender_Branches`, `TestProviderHelpers`, `TestModeLabel_AllModes`, `TestSelectEnergy_InvalidInput`.


- **Offline key differentiation**: seedHash for all three offline pattern generators (bass, arp, melody) now includes `key.Root + key.Scale` so identical seeds produce distinct rhythmic patterns for different keys (not just different root notes).
- **Chord third in bass patterns**: `chordThird()` helper added; cases 0 and 4 of `basslineTemplate` now incorporate the chord's third note (minor 3rd for dark minor feel, major 3rd for brighter major/Dorian quality).
- **Expanded melody rhythm pool**: two additional rhythm patterns added (`triplet-ish push`, `tension-hold`) for 7 total — ensures ≥6 distinct patterns across 20 seeds.
- **Bass density guarantee**: `ensureBassMinDensity()` enforces ≥8 active steps by filling off-beat positions with ghost notes when the hash selects too many sparse patterns.
- **Key-character offline shaping**: basslines now prefer mode character tones when chord-safe, arpeggios bias direction by key+seed, and melodies choose ascending, descending, or tension-hold contours from the same key-aware seed hash.
- **Cache seed regression test**: `TestKeyIncludesVariationSeed` verifies different variation seeds produce different SHA256 disk-cache keys.
- **Interactive modal key regression test**: CLI key selection is covered for modal examples so Dorian, Phrygian, Mixolydian, and Lydian prompts stay visible.
- **Mode-character prompt injection**: LLM prompts now receive a key-specific mode description with scale notes, interval character, and emotional color through `{{MODE_CHARACTER}}`.
- **Offline sub-modes**: `MusicContext.OfflineStyle` now supports `melodic`, `hypnotic`, `driving`, and `minimal` deterministic templates for bassline, arpeggio, and melody.
- **Offline passing notes and gate variation**: offline templates add seed-controlled 15-25% passing-note opportunities where scale-safe, and every generated pattern mixes legato and staccato articulation.
- **Offline quality tests**: added regression coverage for rhythmic figure counts, sub-mode density/validation, passing notes, and gate variation.
- **Tests `TestOfflineSeedDiversity` and `TestOfflineKeyDifferentiation`**: assert ≥6 distinct step fingerprints from 20 seeds (all three pattern types) and that Am/Dm with the same seed produce different rhythmic structures.
- **Modal scale support**: `ParseKey` now accepts `-dorian`, `-phrygian`, `-mixolydian`, `-lydian` suffixes (e.g. `Am-dorian`, `G-mixolydian`), case-insensitive, with or without leading `m`.
- **Expanded chord quality map**: `scaleChordQualities` now covers Dorian, Phrygian, Mixolydian, and Lydian modes with correct diatonic chord qualities computed from each mode's interval structure.
- **Expanded progression pools**: Minor (Aeolian) and Major (Ionian) pools grown from 4 to 12 progressions each; Dorian, Phrygian, Mixolydian, and Lydian each have 8 dedicated progressions. Seed entropy now sufficient: 10 consecutive seeds produce ≥ 6 distinct progressions.
- **Mixolydian and Lydian scales** added to `scaleIntervals` (previously only Dorian and Phrygian were present).
- **Musical regression tests**: `TestProgressionSeedDiversity`, `TestKeyDifferentiation`, `TestProgressionPool_Modal`, `TestParseKey_Modal`, `TestParseKey_Modal_Invalid` added to `internal/theory`.
- Rewrote `TODO.md` with prioritised P0/P1/P2/P3 structure, checkboxes, and root-cause analysis for the zero-star problem.

### Fixed
- **`approachNote` now stays diatonic**: was generating a chromatic semitone below the chord root, violating the "notes must be in the declared scale" invariant and causing the validator to reject offline templates for certain chord roots (e.g. Dm → C#2 in A minor context). Now returns the nearest scale degree below the root instead.
- **`chooseBassProfile` profile name corrected**: was returning `"bass_techno_driving"` (not in the validator's allowed profile list) for high-BPM minor keys; corrected to `"bass_driving"`.

### Removed
- **`cmd/llmidi-gen/` deleted**: the original prototype binary was superseded by `cmd/cadenza` (which uses `GenerateWithContext`, supports 4 providers, variations, grooves, drums, dry-run, watch mode, and dev mode). Having two entry points confused contributors and users.
- Fixed `docker-compose-up` Makefile target which still referenced the deleted `llmidi-gen` service.

### Added
- Added `TODO.md` coverage plan quantifying the gap to 80% and prioritizing missing tests by package.

### Changed
- Updated `README.md` to clearly mark the project as beta and document that CLI logs are written under `<output-dir>/logs/cadenza.log` by default.
- Refreshed `CONTRIBUTING.md` with current `backend/`-based setup instructions and a stronger "Contributors wanted" section for new collaborators.
- Reduced validator and dev-mode flag parsing complexity by extracting focused helper functions.
- Run the Docker smoke test container with the runner UID/GID so mounted `output/` remains writable without restoring world-writable permissions.
- Pinned `JetBrains/qodana-action` in the Qodana workflow to a full commit SHA to satisfy dependency pinning security checks.
- Refactored CLI, generator, and LLM coverage tests into smaller helpers to clear remaining Sonar cognitive-complexity and Go idiom warnings without reducing coverage.
- Rewrote the README in English, Italian, and Spanish and highlighted offline algorithmic generation as a core strength rather than a fallback.
- Expanded the README with practical post-clone setup, build, and run instructions for Windows, macOS, and Linux users.
- Improved the interactive CLI with a quick-start offline sketch path, clearer provider availability guidance, and output-directory writability checks before rendering.
- Expanded the interactive CLI post-run flow with explicit next actions, including rerunning the same setup with a fresh seed.
- Moved backend application logs into an explicit `logs/` subdirectory under the output folder so beta builds produce easier-to-find runtime logs.

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
