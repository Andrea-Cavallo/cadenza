# REFACTOR.md — Cadenza Production Readiness

> Generated from deep codebase audit. Items are grouped by priority.
> After each item is completed, check it off with `[x]`.

---

## P0 — Critical / Blocking

- [x] **P0-1**: Fix `ask()` swallowed I/O errors — `cli.go:61`. On stdin close, program hangs. Handle EOF.
- [x] **P0-2**: Fix `lastRun` race condition — `main.go:35-36`. Add `sync.Mutex`.
- [x] **P0-3**: Fix orphaned Ollama process — `desktop/app.go:228-237`. Store `*exec.Cmd`, cleanup on shutdown.
- [ ] **P0-4**: CI does not run `go test` — `.github/workflows/ci.yml`. Add `go test -race -coverprofile=coverage.out ./...`.
- [x] **P0-5**: Error boundary in React app — `App.tsx`. Add `ErrorBoundary` component.
- [x] **P0-6**: OpenAI/Gemini broken in `GenerateAll` — `multi_generator.go:436`. Add `openai` and `gemini` cases.

## P1 — High Priority

- [x] **P1-1**: `main()` config fallback destroys env vars — Use `config.DefaultConfig()`.
- [ ] **P1-2**: Desktop vs CLI Ollama timeout mismatch
- [ ] **P1-3**: `generatePattern` lacks context cancellation
- [ ] **P1-4**: `ensureWritableDir` TOCTOU race
- [x] **P1-5**: API key partial leak in stdout — Mask only in non-JSON mode.
- [ ] **P1-6**: SHA1 usage in session hash
- [ ] **P1-7**: `cadenza.yaml` in CWD risk
- [x] **P1-8**: `--no-color` / `NO_COLOR` flag — ANSI codes made vars, init-based toggle.
- [ ] **P1-9**: Loading state during export
- [ ] **P1-10**: Progress indicator during generation
- [ ] **P1-11**: Missing test files for critical packages
- [ ] **P1-12**: No frontend component tests
- [x] **P1-13**: AGENTS.md directory paths — Added `backend/` prefix to all.
- [x] **P1-14**: SPECS.md wrong Go version — Updated to 1.25.
- [x] **P1-15**: Update SECURITY.md for desktop app
- [x] **P1-16**: `.env.example` stale env vars — Updated to `CADENZA_*`.

## P2 — Medium Priority

- [ ] **P2-1**: Deduplicate `providerLabel`
- [x] **P2-2**: Config temperature never used — Now read from config.
- [x] **P2-3**: Config `max_retries` never used — Now read from config.
- [x] **P2-4**: Config timeout never used by providers — Now read from config.
- [ ] **P2-5**: Dead `prompts/` COPY in Dockerfile
- [ ] **P2-6**: `var version = "dev"` no verification
- [ ] **P2-7**: `useMemo` for `displayFiles`/`displayPreview`
- [ ] **P2-8**: React.memo on `Sidebar`
- [ ] **P2-9**: Batch `setLog` calls
- [ ] **P2-10**: `.gitattributes` for line endings
- [ ] **P2-11**: Remove `go.mod` linter dependencies

## P3 — Low Priority / Polish

- [ ] **P3-1**: `--single-file` advertised but not implemented
- [ ] **P3-2**: `--dump-spec` advertised but not implemented
- [ ] **P3-3**: Unused `modWheelEvents` in renderer
- [ ] **P3-4**: `eventLineWriter` goroutine proliferation
- [ ] **P3-5**: Probe file from `ensureWritableDir` persists on crash
- [x] **P3-6**: AGENTS.md missing directory entries — Added 7 missing dirs.
- [ ] **P3-7**: `testdata/listening/gen_before/main.go` picked up by `go build`
- [ ] **P3-8**: `Makefile.distributions` fragile `cd ..`

## Completed Extras

- [x] Piano roll off-by-one alignment fix — Notes now correctly aligned with piano keyboard.
- [x] Dynamic pitch range — Piano roll auto-crops to active note range.
- [x] Multi-resolution .ico — 8 sizes (16→256px) generated from PNG.
- [x] Splash PNG blend — Radial gradient overlay fades image edges into app background.
- [x] Clean MIDI filenames — `cadenza_bass_Am_122_v1_s847261.mid` format with seed.
- [x] Desktop exe renamed to `Cadenza.exe` with Andrea Cavallo metadata.

---

_Last updated: 2026-05-13 — 15/52 items completed_
