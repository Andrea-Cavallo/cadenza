# Cadenza Desktop — Wails Design Spec

**Date:** 2026-05-01  
**Status:** Approved  
**Approach:** Wails v2 — Go backend bound directly to React frontend, no HTTP layer

---

## 1. Architecture Overview

Cadenza Desktop is a standalone Windows EXE built with Wails v2. The Go MIDI generation engine is exposed directly to a React frontend via Wails bindings. No HTTP server is involved at runtime.

```
cadenza-desktop.exe
├── Go engine (internal/ — unchanged)
│   ├── theory/
│   ├── generator/
│   ├── renderer/
│   ├── llm/
│   ├── midi/
│   └── cache/
├── AppService (cmd/desktop/app.go)
│   └── Binds Go methods → JS via Wails runtime
└── Frontend (frontend/dist/ — embedded via //go:embed)
    └── Vite + React + TypeScript (ported from existing JSX)
```

The three existing entrypoints remain unchanged:
- `cmd/cadenza/` — CLI (unchanged)
- `cmd/api/` — HTTP API server (optional, for DAW/script integrations)
- `cmd/desktop/` — **NEW**: Wails desktop app

---

## 2. Repository Layout Changes

```
backend/
  cmd/
    cadenza/          ← CLI (unchanged)
    api/              ← HTTP server (unchanged)
    desktop/          ← NEW
      main.go         ← wails.Run(app, options)
      app.go          ← AppService struct
      wails.json      ← Wails project config
  internal/           ← MIDI engine (zero changes)
  go.mod              ← adds github.com/wailsapp/wails/v2

frontend/             ← migrated from CDN/Babel to Vite + React + TypeScript
  src/
    components/
      GenerationConsole.tsx
      Pipeline.tsx
      Presets.tsx
      PianoRoll.tsx
      TweaksPanel.tsx
      Topbar.tsx
    App.tsx
    main.tsx
  index.html
  vite.config.ts
  tsconfig.json
  package.json
```

---

## 3. Go Bindings — `AppService`

```go
// backend/cmd/desktop/app.go

type AppService struct {
    ctx context.Context
}

func (a *AppService) Generate(req GenerateRequest) (GenerateResult, error)
func (a *AppService) GetProviders() []string
func (a *AppService) GetModels(provider string) []ModelInfo
func (a *AppService) GetConfig() AppConfig
func (a *AppService) OpenOutputFolder() error
```

### Request / Response types

```go
type GenerateRequest struct {
    BPM      int    `json:"bpm"`
    Key      string `json:"key"`
    Provider string `json:"provider"`
    Model    string `json:"model"`
    NoLLM    bool   `json:"noLlm"`
    Bars     int    `json:"bars"`
}

type GenerateResult struct {
    Files   []string `json:"files"`   // absolute paths of 3 MIDI files
    Elapsed string   `json:"elapsed"`
}

type ModelInfo struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type AppConfig struct {
    DefaultBPM      int    `json:"defaultBpm"`
    DefaultKey      string `json:"defaultKey"`
    DefaultProvider string `json:"defaultProvider"`
    OutputDir       string `json:"outputDir"`
}
```

### Progress events

`Generate` is async. Real-time progress is emitted via Wails Events — no polling:

```go
// Go side (inside Generate)
runtime.EventsEmit(a.ctx, "progress", "Generating chord progression...")
runtime.EventsEmit(a.ctx, "progress", "Generating bassline...")
runtime.EventsEmit(a.ctx, "progress", "Writing MIDI files...")
```

```ts
// Frontend side
import { EventsOn } from '../wailsjs/runtime'
EventsOn("progress", (msg: string) => setLog(prev => [...prev, msg]))
```

---

## 4. Frontend Migration

The existing design system is preserved 1:1. Only the delivery mechanism changes (CDN/Babel → Vite + React + TypeScript).

### Design tokens (unchanged)

```css
--accent: #01FF95;          /* beatport acid lime */
--bg: #000000;
--surface: #0a0a0a;
--border: #1a1a1a;
--font-sans: 'Inter Tight';
--font-mono: 'JetBrains Mono';
--font-serif: 'Fraunces';
```

### Component mapping

| Existing file | Migrated to |
|---|---|
| `frontend/app.jsx` | `src/App.tsx` + `src/components/Topbar.tsx` |
| `frontend/console.jsx` | `src/components/GenerationConsole.tsx` |
| `frontend/pipeline.jsx` | `src/components/Pipeline.tsx` |
| `frontend/presets.jsx` | `src/components/Presets.tsx` |
| `frontend/pianoroll.jsx` | `src/components/PianoRoll.tsx` |
| `frontend/tweaks-panel.jsx` | `src/components/TweaksPanel.tsx` |

### Key UI change

The `curl` / CLI command display in `GenerationConsole` is replaced by:
- A **Generate** button that calls `AppService.Generate()` via Wails binding
- An inline log panel fed by `EventsOn("progress", ...)`
- A file list showing the 3 generated MIDI paths with an **Open Folder** button

---

## 5. Build & Distribution

### Commands

```bash
# Install Wails CLI (once)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Dev mode — HMR frontend + live Go reload
cd backend/cmd/desktop && wails dev

# Production build — single cadenza-desktop.exe (~15-20 MB)
cd backend/cmd/desktop && wails build -platform windows/amd64

# CLI + desktop
cd backend && make build-all
```

### `wails.json` (in `backend/cmd/desktop/`)

```json
{
  "name": "Cadenza",
  "outputfilename": "cadenza-desktop",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dir": "../../../frontend",
  "wailsVersion": "v2.x"
}
```

### Makefile targets (added to `backend/Makefile`)

```makefile
desktop:
	cd cmd/desktop && wails build -platform windows/amd64

desktop-dev:
	cd cmd/desktop && wails dev

build-all: build desktop
```

### Distribution

- Single `cadenza-desktop.exe` — no installer, no runtime dependencies
- Frontend assets embedded via `//go:embed all:frontend/dist`
- Node.js required only at build time, not at runtime

---

## 6. Error Handling

- `Generate` returns `(GenerateResult, error)` — Wails surfaces the error to the frontend as a rejected Promise
- Frontend catches and displays errors inline in the log panel
- Existing graceful fallback (LLM → offline template) is preserved in the Go engine

---

## 7. Out of Scope

- macOS / Linux builds (future)
- Auto-updater
- Code signing / Windows SmartScreen bypass
- REST API integration in the desktop app (stays in `cmd/api/` only)
