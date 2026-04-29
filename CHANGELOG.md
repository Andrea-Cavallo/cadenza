# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
