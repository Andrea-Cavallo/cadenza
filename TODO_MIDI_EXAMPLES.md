# MIDI Examples Module — Remaining Tasks

Plan file: `docs/superpowers/plans/2026-04-29-midi-examples.md`
Spec file:  `docs/superpowers/specs/2026-04-29-midi-examples-design.md`
Branch:     `feature/read-midi`

---

## ✅ Done

- **Task 1** — `internal/examples/types.go` — commit `d9d853b`
  All 5 shared types: RawNote, ExampleProfile, SnippetStep, PatternSnippet, ExampleBank.

---

## ⏳ Remaining (Tasks 2–12)

### Task 2 — MIDI Parser
Files to create:
- `internal/examples/parser.go`
- `internal/examples/parser_test.go`

Pure-Go binary MIDI reader (Type-0 + Type-1), VLQ delta-time decoder, step quantization
(`step = round(absTick / ticksPerStep)`, ticksPerStep = ticksPerBeat/4), BPM from meta 0x51,
key from meta 0x59. Round-trip test using existing `internal/midi` writer.
Commit: `feat(examples): pure-Go MIDI parser with VLQ decoding and step quantization`

---

### Task 3 — Classifier
Files to create:
- `internal/examples/classifier.go`
- `internal/examples/classifier_test.go`

`Classify([]RawNote) string` — returns "bassline" / "arpeggio" / "melody" based on
median MIDI pitch + distinct active step count (density).
Ranges: bass < 48, overlap 48-60 (density<10 → bass, else arp), overlap 60-76
(density<8 → melody, else arp), ≥76 → melody.
Commit: `feat(examples): MIDI track classifier by register and density`

---

### Task 4 — Feature Extractor
Files to create:
- `internal/examples/extractor.go`
- `internal/examples/extractor_test.go`

`ExtractProfile([]RawNote) *ExampleProfile` — StepDensity per bar, AccentRatio
(vel>90), GhostRatio (vel<60), SlideRatio (dur>1), IntervalHist (semitone from root,
normalized), AvgDensity.
`ExtractSnippets(notes, profile, type, maxN) []PatternSnippet` — up to 3 bars
closest to AvgDensity.
Commit: `feat(examples): feature extractor — StepDensity, IntervalHist, PatternSnippets`

---

### Task 5 — Profile Cache
Files to create:
- `internal/examples/cache.go`
- `internal/examples/cache_test.go`

`SaveCache(dir, bank, mtimes)` / `LoadCache(dir) → bank, mtimes, err` /
`IsCacheValid(dir, files) bool` / `CollectMtimes(files)`.
Cache file: `<dir>/.cadenza_cache.json` (JSON with bank + mtime map).
Commit: `feat(examples): disk cache with mtime-based invalidation`

---

### Task 6 — Loader
Files to create:
- `internal/examples/loader.go`
- `internal/examples/loader_test.go`

`Load(dir string, noCache bool) (*ExampleBank, error)` — scans dir for *.mid,
calls ParseFile → Classify → ExtractProfile → ExtractSnippets per track,
detects PresentTypes / MissingTypes, reads/writes cache.
Commit: `feat(examples): loader — scan folder, classify tracks, build ExampleBank with cache`

---

### Task 7 — Similarity Scorer
Files to create:
- `internal/examples/scorer.go`
- `internal/examples/scorer_test.go`

`Score(spec *schema.PatternSpec, profile *ExampleProfile) float64`
Weighted cosine similarity: 0.35×rhythm + 0.25×density + 0.30×interval + 0.10×accent.
Returns 0 for nil profile. Tests: score≥0.7 for similar, score<0.6 for unrelated.
Commit: `feat(examples): weighted cosine similarity scorer against ExampleProfile`

---

### Task 8 — Offline Mode Bias
Files to modify:
- `internal/generator/offline.go`
- `internal/generator/multi_generator.go`

Add `*examples.ExampleProfile` param to `offlineTemplate` + 3 sub-functions.
Add `applyProfileBias(steps, profile)` that adjusts accent/ghost/slide flags.
Add `ExampleBank *examples.ExampleBank` field to `MultiGenerator` struct.
Update `generatePattern` to pass profile to `offlineTemplate`.
Commit: `feat(generator): wire ExampleProfile bias into offline templates`

---

### Task 9 — LLM Prompt Injection
Files to modify:
- `internal/generator/generator.go`
- `prompts/bassline_v1.md`
- `prompts/arpeggio_v1.md`
- `prompts/melody_v1.md`

Add `ExampleSnippets []examples.PatternSnippet` and `ExistingMaterial []examples.PatternSnippet`
to `MusicContext`. Add `{{EXAMPLE_SECTION}}` placeholder to all 3 prompt templates.
`buildExampleSection()` serializes snippets into compact human-readable text.
`serializeSnippets()` formats each snippet as step-by-step interval list.
Commit: `feat(generator): inject example snippets into LLM prompts via {{EXAMPLE_SECTION}}`

---

### Task 10 — Completion Mode
Files to modify:
- `internal/generator/multi_generator.go`

Add `GenerateCompletion(ctx, bpm, keyStr, n int) ([]*GenerationResult, error)`.
For each missing type, sets `ExistingMaterial` from present-type snippets, generates N variants,
computes similarity score, writes files as `output_<type>_variant<n>_<key>_<bpm>_<ts>.mid`.
Commit: `feat(generator): completion mode — generate missing instrument types from partial examples`

---

### Task 11 — CLI Flags Wiring
Files to modify:
- `cmd/cadenza/cli.go`
- `cmd/cadenza/main.go`
- `.gitignore`

Add `ExamplesDir`, `CompleteN`, `NoCacheExamples` to `cliConfig`.
Add `--examples`, `--complete` (default 3), `--no-cache-examples` flags.
In `runGeneration`: call `examples.Load`, use auto-detected BPM/key if flags not set,
branch to `GenerateCompletion` when `MissingTypes` non-empty.
Add `.cadenza_cache.json` to `.gitignore`.
Commit: `feat(cli): --examples, --complete, --no-cache-examples flags with completion mode output`

---

### Task 12 — Final Verification + README
Files to modify:
- `README.md`

Run `go test ./... -race -count=1` — all green.
Cross-compile: `GOOS=linux go build ./...` and `GOOS=darwin go build ./...`.
Add `--examples` usage section to README (after Quickstart).
Commit: `docs: document --examples usage and completion mode in README`

---

## How to resume

Next session: read this file + the full plan at
`docs/superpowers/plans/2026-04-29-midi-examples.md`, then continue from Task 2.

Use subagent-driven development (dispatch one implementer per task, then spec reviewer,
then code quality reviewer before marking complete).
