# Cadenza Desktop (Wails v2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Windows desktop app (`cadenza-desktop.exe`) using Wails v2 that binds the existing Go MIDI engine directly to a Vite + React + TypeScript frontend — no HTTP layer.

**Architecture:** A new `cmd/desktop/` entrypoint wraps an `AppService` struct whose public methods (`Generate`, `GetProviders`, etc.) are auto-bound to the frontend by the Wails runtime. The frontend lives at `backend/cmd/desktop/frontend/` as a standard Vite + React + TypeScript project and is embedded into the binary at build time via `//go:embed all:frontend/dist`. Existing `cmd/cadenza/` (CLI) and `cmd/api/` (HTTP) entrypoints are untouched.

**Tech Stack:** Go 1.25.9, Wails v2 (latest stable), Vite 5, React 18, TypeScript 5, module `github.com/Andrea-Cavallo/cadenza`.

---

### Task 1: Add Wails v2 to `go.mod` and scaffold `cmd/desktop/`

**Files:**
- Modify: `backend/go.mod` (via `go get`)
- Create: `backend/cmd/desktop/main.go`
- Create: `backend/cmd/desktop/app.go` (stub)
- Create: `backend/cmd/desktop/wails.json`

- [ ] **Step 1: Install Wails CLI (once per machine)**

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails version
```

Expected: `wails v2.x.x`

- [ ] **Step 2: Add Wails v2 dependency**

```powershell
cd backend
go get github.com/wailsapp/wails/v2@latest
go mod tidy
```

Expected: `go.mod` lists `github.com/wailsapp/wails/v2` in `require`.

- [ ] **Step 3: Create `backend/cmd/desktop/app.go` (stub — full implementation in Task 2)**

```go
package main

import "context"

type AppService struct {
	ctx context.Context
}

func NewApp() *AppService { return &AppService{} }

func (a *AppService) startup(ctx context.Context) { a.ctx = ctx }

func (a *AppService) Generate(req GenerateRequest) (GenerateResult, error) {
	return GenerateResult{}, nil
}

func (a *AppService) GetProviders() []string {
	return []string{"claude", "ollama", "openai", "gemini", "offline"}
}

func (a *AppService) GetModels(provider string) []ModelInfo { return nil }

func (a *AppService) GetConfig() AppConfig {
	return AppConfig{DefaultBPM: 122, DefaultKey: "Am", DefaultProvider: "offline"}
}

func (a *AppService) OpenOutputFolder() error { return nil }
```

- [ ] **Step 4: Create `backend/cmd/desktop/main.go`**

```go
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "Cadenza",
		Width:            1280,
		Height:           800,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 1},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
	})
	if err != nil {
		log.Fatal("Error:", err)
	}
}
```

- [ ] **Step 5: Create `backend/cmd/desktop/wails.json`**

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "Cadenza",
  "outputfilename": "cadenza-desktop",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "auto",
  "frontend:dir": "frontend",
  "version": "2"
}
```

- [ ] **Step 6: Verify `go vet` passes (embed error is expected until frontend/dist exists)**

```powershell
cd backend
go vet ./cmd/desktop/
```

Expected: no vet errors.

- [ ] **Step 7: Commit**

```powershell
cd backend
git add go.mod go.sum cmd/desktop/
git commit -m "feat: scaffold Wails v2 desktop entrypoint (cmd/desktop)"
```

---

### Task 2: Implement `AppService` Go methods

**Files:**
- Create: `backend/cmd/desktop/types.go`
- Create: `backend/cmd/desktop/provider.go`
- Modify: `backend/cmd/desktop/app.go` (full implementation)
- Create: `backend/cmd/desktop/app_test.go`

- [ ] **Step 1: Create `backend/cmd/desktop/types.go`**

```go
package main

type GenerateRequest struct {
	BPM      int    `json:"bpm"`
	Key      string `json:"key"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	NoLLM    bool   `json:"noLlm"`
	Bars     int    `json:"bars"`
}

type GenerateResult struct {
	Files   []string `json:"files"`
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

- [ ] **Step 2: Create `backend/cmd/desktop/provider.go`**

```go
package main

import (
	"fmt"

	"github.com/Andrea-Cavallo/cadenza/internal/llm"
)

func buildProvider(noLLM bool, providerName, model string) (llm.Provider, error) {
	if noLLM || providerName == "offline" {
		return &llm.MockProvider{}, nil
	}
	switch providerName {
	case "claude":
		return llm.NewClaudeProvider(model)
	case "ollama":
		return llm.NewOllamaProvider("http://localhost:11434", model), nil
	case "openai":
		return llm.NewOpenAIProvider(model)
	case "gemini":
		return llm.NewGeminiProvider(model)
	default:
		return nil, fmt.Errorf("unknown provider %q (supported: claude, ollama, openai, gemini, offline)", providerName)
	}
}
```

- [ ] **Step 3: Write failing test `backend/cmd/desktop/app_test.go`**

```go
package main

import (
	"context"
	"testing"
)

func TestGetProviders_ContainsOffline(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	providers := app.GetProviders()
	if len(providers) == 0 {
		t.Fatal("expected at least one provider")
	}
	found := false
	for _, p := range providers {
		if p == "offline" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'offline' in providers list")
	}
}

func TestGetConfig_DefaultBPMNonZero(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	cfg := app.GetConfig()
	if cfg.DefaultBPM == 0 {
		t.Error("expected non-zero DefaultBPM")
	}
}

func TestGetModels_Claude_ReturnsEntries(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	models := app.GetModels("claude")
	if len(models) == 0 {
		t.Error("expected at least one model for claude")
	}
}
```

- [ ] **Step 4: Run test — expect FAIL**

```powershell
cd backend
go test ./cmd/desktop/ -v -run TestGetModels_Claude
```

Expected: FAIL (`GetModels` returns nil).

- [ ] **Step 5: Replace `backend/cmd/desktop/app.go` with full implementation**

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/models"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type AppService struct {
	ctx context.Context
}

func NewApp() *AppService { return &AppService{} }

func (a *AppService) startup(ctx context.Context) {
	a.ctx = ctx
	models.Load("")
}

func (a *AppService) Generate(req GenerateRequest) (GenerateResult, error) {
	start := time.Now()

	runtime.EventsEmit(a.ctx, "progress", "Initializing provider...")
	prov, err := buildProvider(req.NoLLM, req.Provider, req.Model)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("provider init: %w", err)
	}

	outputDir := defaultOutputDir()
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return GenerateResult{}, fmt.Errorf("output dir: %w", err)
	}

	bars := req.Bars
	if bars == 0 {
		bars = 16
	}

	v := schema.NewValidator()
	reg := styleprofile.NewRegistry()
	rend := renderer.New()
	w := midipkg.NewWriter(float64(req.BPM))
	mg := generator.NewMultiGenerator(prov, v, reg, rend, w, outputDir)
	mg.NoLLM = req.NoLLM

	parsedKey, err := theory.ParseKey(req.Key)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("invalid key %q: %w", req.Key, err)
	}

	seed := fmt.Sprintf("%d", time.Now().UnixNano())
	prog := theory.SelectProgression(parsedKey.Root, parsedKey.Scale, seed)

	runtime.EventsEmit(a.ctx, "progress", "Generating chord progression...")
	runtime.EventsEmit(a.ctx, "progress", fmt.Sprintf("Rendering %d bars at %d BPM in %s...", bars, req.BPM, req.Key))

	musicCtx := generator.MusicContext{
		BPM:              float64(req.BPM),
		Key:              parsedKey,
		Bars:             bars,
		VariationSeed:    seed,
		ChordProgression: prog,
		Groove:           "straight",
	}

	result, err := mg.GenerateWithContext(a.ctx, musicCtx, bars, 1)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("generation failed: %w", err)
	}

	runtime.EventsEmit(a.ctx, "progress", fmt.Sprintf("Done — %d files written.", len(result.Files)))

	return GenerateResult{
		Files:   result.Files,
		Elapsed: time.Since(start).Round(time.Millisecond).String(),
	}, nil
}

func (a *AppService) GetProviders() []string {
	return []string{"claude", "ollama", "openai", "gemini", "offline"}
}

func (a *AppService) GetModels(provider string) []ModelInfo {
	entries := models.List(provider)
	out := make([]ModelInfo, len(entries))
	for i, e := range entries {
		out[i] = ModelInfo{ID: e.ID, Name: e.Name}
	}
	return out
}

func (a *AppService) GetConfig() AppConfig {
	return AppConfig{
		DefaultBPM:      122,
		DefaultKey:      "Am",
		DefaultProvider: "offline",
		OutputDir:       defaultOutputDir(),
	}
}

func (a *AppService) OpenOutputFolder() error {
	dir := filepath.FromSlash(defaultOutputDir())
	return exec.Command("explorer", dir).Start()
}

func defaultOutputDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "output"
	}
	return filepath.Join(home, "cadenza-output")
}
```

- [ ] **Step 6: Run tests — expect PASS**

```powershell
cd backend
go test ./cmd/desktop/ -v
```

Expected: all 3 tests PASS.

- [ ] **Step 7: Verify build and vet**

```powershell
cd backend
go vet ./cmd/desktop/
go build ./cmd/desktop/
```

Expected: no errors (embed error is expected until frontend/dist exists).

- [ ] **Step 8: Commit**

```powershell
cd backend
git add cmd/desktop/
git commit -m "feat: implement AppService with Generate, GetProviders, GetModels, GetConfig, OpenOutputFolder"
```

---

### Task 3: Scaffold Vite + React + TypeScript frontend

**Files:**
- Create: `backend/cmd/desktop/frontend/package.json`
- Create: `backend/cmd/desktop/frontend/vite.config.ts`
- Create: `backend/cmd/desktop/frontend/tsconfig.json`
- Create: `backend/cmd/desktop/frontend/index.html`
- Create: `backend/cmd/desktop/frontend/src/main.tsx`
- Create: `backend/cmd/desktop/frontend/src/index.css`

- [ ] **Step 1: Create `backend/cmd/desktop/frontend/package.json`**

```json
{
  "name": "cadenza-desktop",
  "private": true,
  "version": "0.7.2",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@types/react": "^18.3.1",
    "@types/react-dom": "^18.3.1",
    "@vitejs/plugin-react": "^4.2.1",
    "typescript": "^5.4.5",
    "vite": "^5.2.11"
  }
}
```

- [ ] **Step 2: Create `backend/cmd/desktop/frontend/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": false,
    "noUnusedParameters": false,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src"]
}
```

- [ ] **Step 3: Create `backend/cmd/desktop/frontend/vite.config.ts`**

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist' },
})
```

- [ ] **Step 4: Create `backend/cmd/desktop/frontend/index.html`**

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Cadenza</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter+Tight:wght@300;400;500;600;700&family=JetBrains+Mono:wght@300;400;500;600&family=Fraunces:opsz,wght@9..144,300;9..144,400&display=swap" rel="stylesheet">
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 5: Create `backend/cmd/desktop/frontend/src/main.tsx`**

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
```

- [ ] **Step 6: Create `backend/cmd/desktop/frontend/src/index.css`**

Port the CSS verbatim from `frontend/index.html` (the existing CDN app at the repo root — the `<style>` block inside `<head>`). Copy every CSS rule from the `<style>` tag into this file. The `<style>` block starts at line 10 of `frontend/index.html` and ends at the closing `</style>` tag.

Additionally append these pipeline-specific styles not in index.html (they were inline in pipeline.jsx):

```css
/* ── pipe ──────────────────────────────────────────── */
.pipe-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1px;
  background: var(--line);
  border: 1px solid var(--line);
}
.pipe-step {
  background: var(--bg);
  padding: 32px 28px;
}
.pipe-step .n {
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: var(--accent);
  margin-bottom: 14px;
}
.pipe-step h4 {
  font-size: 18px;
  font-weight: 400;
  margin: 0 0 12px;
  letter-spacing: -0.01em;
}
.pipe-step p {
  font-size: 13.5px;
  line-height: 1.6;
  color: var(--fg-dim);
  margin: 0 0 18px;
}
.pipe-step .glyph {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--fg-muted);
  letter-spacing: 0.06em;
}
```

- [ ] **Step 7: Install npm dependencies**

```powershell
cd backend/cmd/desktop/frontend
npm install
```

Expected: `node_modules/` created, no errors.

- [ ] **Step 8: Create placeholder `backend/cmd/desktop/frontend/src/App.tsx` and verify Vite builds**

```tsx
export default function App() {
  return <div style={{ color: '#01FF95', fontFamily: 'monospace', padding: 40 }}>Cadenza Desktop</div>
}
```

```powershell
cd backend/cmd/desktop/frontend
npm run build
```

Expected: `dist/` created with `index.html` + JS bundle. No errors.

- [ ] **Step 9: Add `frontend/dist` and `frontend/node_modules` to .gitignore**

Append to `backend/.gitignore` (create the file if it doesn't exist):
```
cmd/desktop/frontend/dist/
cmd/desktop/frontend/node_modules/
cmd/desktop/frontend/wailsjs/
```

- [ ] **Step 10: Commit**

```powershell
cd backend
git add cmd/desktop/frontend/ .gitignore
git commit -m "feat: scaffold Vite + React + TypeScript frontend for Wails desktop"
```

---

### Task 4: Port presentational components

**Files:**
- Create: `backend/cmd/desktop/frontend/src/components/Topbar.tsx`
- Create: `backend/cmd/desktop/frontend/src/components/Pipeline.tsx`
- Create: `backend/cmd/desktop/frontend/src/components/PianoRoll.tsx`
- Create: `backend/cmd/desktop/frontend/src/components/Presets.tsx`

Source files to read first: `frontend/app.jsx`, `frontend/pipeline.jsx`, `frontend/pianoroll.jsx`, `frontend/presets.jsx` at the repository root.

- [ ] **Step 1: Create `backend/cmd/desktop/frontend/src/components/Topbar.tsx`**

Adapted from the `Topbar` function in `frontend/app.jsx`. Nav links simplified for desktop (no API docs link):

```tsx
export function Topbar() {
  return (
    <div className="topbar">
      <div className="brand">
        <div className="brand-mark">C</div>
        <span>cadenza</span>
        <span className="pill" style={{ marginLeft: 4 }}>v0.7.2</span>
        <span className="pill live">desktop</span>
      </div>
      <nav className="topnav">
        <a href="#presets">Presets</a>
        <a href="#generate">Generate</a>
        <a href="#pipeline">Pipeline</a>
      </nav>
      <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
        <a className="btn primary" href="#generate">Generate ↗</a>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Create `backend/cmd/desktop/frontend/src/components/Pipeline.tsx`**

Ported from `frontend/pipeline.jsx`. Change `function Pipeline()` → `export function Pipeline()`, remove `window.Pipeline = Pipeline`:

```tsx
const PIPE = [
  {
    n: '01 / INTENT',
    title: 'Provider',
    body: 'Claude, Ollama, OpenAI, or Gemini receive the musical context — bpm, key, bars, groove — and reason about a coherent pattern.',
    glyph: 'claude · ollama · openai · gemini',
  },
  {
    n: '02 / SPEC',
    title: 'PatternSpec',
    body: 'A schema-validated YAML describing chord motion, rhythmic density, motif arcs. Re-render anytime with --from-spec.',
    glyph: 'spec.bassline.yaml  ·  spec.arp.yaml  ·  spec.melody.yaml',
  },
  {
    n: '03 / RENDER',
    title: 'Style profile',
    body: 'Specs are bound to a style profile (mpc60, linndrum, humanize) and rendered into deterministic MIDI events.',
    glyph: 'groove · velocity · swing · timing',
  },
  {
    n: '04 / MIDI',
    title: 'Tracks out',
    body: 'Three coherent tracks share one progression and seed — drop straight into Ableton, Logic, or Bitwig.',
    glyph: 'bassline.mid  ·  arpeggio.mid  ·  melody.mid',
  },
]

export function Pipeline() {
  return (
    <div className="pipe-row">
      {PIPE.map((s, i) => (
        <div key={i} className="pipe-step">
          <div className="n">{s.n}</div>
          <h4>{s.title}</h4>
          <p>{s.body}</p>
          <div className="glyph">{s.glyph}</div>
        </div>
      ))}
    </div>
  )
}
```

- [ ] **Step 3: Create `backend/cmd/desktop/frontend/src/components/PianoRoll.tsx`**

Read `frontend/pianoroll.jsx` in full. Port it to TypeScript:
- Replace `const { useState, useEffect, useRef } = React` with named imports: `import { useState, useEffect, useRef } from 'react'`
- Change `function PianoRoll({ width = 1480, height = 420, bpm = 122 })` → `export function PianoRoll({ width = 1480, height = 420, bpm = 122 }: { width?: number; height?: number; bpm?: number })`
- Change `(now)` parameter in the RAF callback to `(now: number)`
- Remove `window.PianoRoll = PianoRoll` at the bottom
- Keep all SVG rendering logic unchanged — it is pure calculation with no side effects

- [ ] **Step 4: Create `backend/cmd/desktop/frontend/src/components/Presets.tsx`**

Read `frontend/presets.jsx` in full. Port to TypeScript:
- Add prop types: `interface PresetGridProps { active: string; onSelect: (id: string) => void }`
- Change `function PresetGrid({ active, onSelect })` → `export function PresetGrid({ active, onSelect }: PresetGridProps)`
- Remove any `window.*` exports at the bottom
- Keep all preset data and rendering logic unchanged

- [ ] **Step 5: Create `backend/cmd/desktop/frontend/src/components/TweaksPanel.tsx`**

Ported from `frontend/tweaks-panel.jsx`. Read that file in full first.
- Replace `const { useState, useRef, useEffect } = React` with named imports from 'react'
- Add TypeScript prop interfaces for `TweaksPanel`, `TweakSection`, `TweakRadio`
- Export all three components (`export function TweaksPanel`, `export function TweakSection`, `export function TweakRadio`)
- Remove any `window.*` exports
- Export a `useTweaks` hook with the same signature

Example prop types to add:
```ts
interface TweaksPanelProps { title: string; children: React.ReactNode }
interface TweakSectionProps { label: string; children: React.ReactNode }
interface TweakRadioProps {
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
}
```

- [ ] **Step 6: Verify TypeScript check**

```powershell
cd backend/cmd/desktop/frontend
npx tsc --noEmit
```

Expected: no type errors. Fix any that appear.

- [ ] **Step 7: Commit**

```powershell
cd backend
git add cmd/desktop/frontend/src/components/
git commit -m "feat: port Topbar, Pipeline, PianoRoll, Presets, TweaksPanel to TypeScript"
```

---

### Task 5: Port `GenerationConsole` with real Wails binding

**Files:**
- Create: `backend/cmd/desktop/frontend/src/components/GenerationConsole.tsx`

The Wails runtime generates JS bindings at `frontend/wailsjs/` when `wails dev` or `wails build` runs. Until then, imports are suppressed with `@ts-ignore`. This is normal for Wails projects.

- [ ] **Step 1: Create `backend/cmd/desktop/frontend/src/components/GenerationConsole.tsx`**

```tsx
import { useState, useEffect } from 'react'

interface GenerateRequest {
  bpm: number
  key: string
  provider: string
  model: string
  noLlm: boolean
  bars: number
}

interface GenerateResult {
  files: string[]
  elapsed: string
}

// Generated by `wails dev` / `wails build` — suppress TS errors until first build
// @ts-ignore
import { Generate, OpenOutputFolder } from '../../wailsjs/go/main/AppService'
// @ts-ignore
import { EventsOn } from '../../wailsjs/runtime/runtime'

const KEYS = [
  'Am', 'Bm', 'Cm', 'Dm', 'Em', 'Fm', 'Gm',
  'C', 'D', 'E', 'F', 'G', 'A', 'B',
  'F#m', 'C#m', 'Bbm',
  'Am-dorian', 'Bm-dorian', 'Em-dorian',
  'Em-phrygian', 'Bm-phrygian',
  'G-mixolydian', 'A-mixolydian', 'D-mixolydian',
  'C-lydian', 'D-lydian',
]
const GROOVES = ['straight', 'mpc60', 'linndrum', 'humanize']
const STYLES = ['—', 'hypnotic', 'driving', 'minimal', 'melodic']
const PROVIDERS = ['claude', 'ollama', 'openai', 'gemini', 'offline']
const BARS_OPTS = [16, 32, 64, 128]

function bpmGenre(bpm: number): string {
  if (bpm < 100) return 'Downtempo / Ambient'
  if (bpm < 115) return 'Deep house / Nu-disco'
  if (bpm < 122) return 'Tech house / Organic'
  if (bpm < 130) return 'Progressive / Melodic techno'
  if (bpm < 138) return 'Peak-time techno'
  return 'Hard techno / Industrial'
}

interface Params {
  bpm: number
  key: string
  bars: number
  groove: string
  style: string
  provider: string
  drums: boolean
}

interface GenerationConsoleProps {
  params: Params
  setParams: (p: Params) => void
}

export function GenerationConsole({ params, setParams }: GenerationConsoleProps) {
  const { bpm, key, bars, groove, style, provider } = params
  const [log, setLog] = useState<string[]>([])
  const [files, setFiles] = useState<string[]>([])
  const [running, setRunning] = useState(false)

  useEffect(() => {
    EventsOn('progress', (msg: string) => {
      setLog(prev => [...prev, msg])
    })
  }, [])

  const handleGenerate = async () => {
    setRunning(true)
    setLog([])
    setFiles([])
    try {
      const req: GenerateRequest = {
        bpm,
        key,
        provider,
        model: '',
        noLlm: provider === 'offline',
        bars,
      }
      const result: GenerateResult = await Generate(req)
      setFiles(result.files)
      setLog(prev => [...prev, `✓ Done in ${result.elapsed}`])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setLog(prev => [...prev, `✗ Error: ${msg}`])
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="row" style={{ gridTemplateColumns: '1.1fr 1.4fr' }}>
      {/* ── controls ────────────────────────────── */}
      <div className="cell">
        <div className="eyebrow" style={{ marginBottom: 22 }}>Parameters</div>

        {/* BPM */}
        <div style={{ marginBottom: 22 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 8 }}>
            <span className="label">BPM</span>
            <span className="mono" style={{ fontSize: 13, color: 'var(--fg-dim)' }}>{bpmGenre(bpm)}</span>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 70px', gap: 14, alignItems: 'center' }}>
            <input type="range" min="80" max="150" step="1"
              value={bpm}
              onChange={e => setParams({ ...params, bpm: parseInt(e.target.value) })} />
            <div className="mono" style={{ fontSize: 28, fontWeight: 500, textAlign: 'right', letterSpacing: '-0.02em' }}>
              {bpm}
            </div>
          </div>
        </div>

        {/* Key + Bars */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 18, marginBottom: 22 }}>
          <div>
            <div className="label" style={{ marginBottom: 8 }}>Key &amp; mode</div>
            <select value={key} onChange={e => setParams({ ...params, key: e.target.value })}>
              {KEYS.map(k => <option key={k} value={k}>{k}</option>)}
            </select>
          </div>
          <div>
            <div className="label" style={{ marginBottom: 8 }}>Bars</div>
            <select value={bars} onChange={e => setParams({ ...params, bars: parseInt(e.target.value) })}>
              {BARS_OPTS.map(b => <option key={b} value={b}>{b}</option>)}
            </select>
          </div>
        </div>

        {/* Groove + Style */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 18, marginBottom: 22 }}>
          <div>
            <div className="label" style={{ marginBottom: 8 }}>Groove</div>
            <select value={groove} onChange={e => setParams({ ...params, groove: e.target.value })}>
              {GROOVES.map(g => <option key={g} value={g}>{g}</option>)}
            </select>
          </div>
          <div>
            <div className="label" style={{ marginBottom: 8 }}>Offline style</div>
            <select value={style} onChange={e => setParams({ ...params, style: e.target.value })}>
              {STYLES.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
        </div>

        {/* Provider */}
        <div style={{ marginBottom: 22 }}>
          <div className="label" style={{ marginBottom: 8 }}>Provider</div>
          <div style={{ display: 'flex', gap: 1, background: 'var(--line)', border: '1px solid var(--line)' }}>
            {PROVIDERS.map(p => (
              <button key={p}
                onClick={() => setParams({ ...params, provider: p })}
                className="mono"
                style={{
                  flex: 1, padding: '10px 8px',
                  background: provider === p ? 'var(--fg)' : 'var(--bg)',
                  color: provider === p ? 'var(--bg)' : 'var(--fg-dim)',
                  border: 0, cursor: 'pointer',
                  fontSize: 11, letterSpacing: '0.12em', textTransform: 'uppercase',
                }}>
                {p}
              </button>
            ))}
          </div>
        </div>

        {/* Generate */}
        <button
          className="btn primary"
          style={{ width: '100%', fontSize: 13, padding: '14px 0' }}
          onClick={handleGenerate}
          disabled={running}>
          {running ? 'Generating...' : 'Generate ↗'}
        </button>
      </div>

      {/* ── log + results ──────────────────────── */}
      <div className="cell" style={{ background: 'var(--bg-1)', display: 'flex', flexDirection: 'column', gap: 14 }}>
        <div className="eyebrow">Generation log</div>
        <pre className="code" style={{ background: 'var(--bg)', flex: 1, margin: 0, minHeight: 120 }}>
          {log.length === 0 ? '# Press Generate to start...' : log.join('\n')}
        </pre>

        {files.length > 0 && (
          <div style={{ borderTop: '1px solid var(--line)', paddingTop: 14 }}>
            <div className="eyebrow" style={{ marginBottom: 10 }}>Output files</div>
            {files.map(f => (
              <div key={f} className="mono" style={{ fontSize: 12, color: 'var(--fg-dim)', marginBottom: 6 }}>
                ✓ {f}
              </div>
            ))}
            <button
              className="btn ghost"
              style={{ marginTop: 12, width: '100%' }}
              onClick={() => OpenOutputFolder()}>
              Open Folder ↗
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```powershell
cd backend
git add cmd/desktop/frontend/src/components/GenerationConsole.tsx
git commit -m "feat: port GenerationConsole with Wails Generate binding and progress events"
```

---

### Task 6: Wire `App.tsx` and verify first Wails build

**Files:**
- Modify: `backend/cmd/desktop/frontend/src/App.tsx` (replace placeholder)

- [ ] **Step 1: Replace placeholder `App.tsx` with full composition root**

```tsx
import { useState } from 'react'
import { Topbar } from './components/Topbar'
import { PianoRoll } from './components/PianoRoll'
import { PresetGrid } from './components/Presets'
import { GenerationConsole } from './components/GenerationConsole'
import { Pipeline } from './components/Pipeline'

interface Params {
  bpm: number
  key: string
  bars: number
  groove: string
  style: string
  provider: string
  drums: boolean
}

const GENRE_PRESETS = [
  { id: 'progressive-warmup',  bpm: 122, key: 'Am-dorian',  groove: 'mpc60',    style: 'melodic'  },
  { id: 'peak-time-driver',    bpm: 130, key: 'Em',          groove: 'straight', style: 'driving'  },
  { id: 'afterhours-hypnotic', bpm: 120, key: 'Dm',          groove: 'humanize', style: 'hypnotic' },
  { id: 'festival-melodic',    bpm: 126, key: 'F#m',         groove: 'linndrum', style: 'melodic'  },
]

function Section({ id, eyebrow, title, sub, children }: {
  id: string; eyebrow: string; title: string; sub: string; children: React.ReactNode
}) {
  return (
    <section id={id}>
      <div className="shell">
        <div className="section-head">
          <div>
            <div className="eyebrow" style={{ marginBottom: 14 }}>{eyebrow}</div>
            <h2 dangerouslySetInnerHTML={{ __html: title }} />
          </div>
          <p>{sub}</p>
        </div>
        {children}
      </div>
    </section>
  )
}

export default function App() {
  const [activePreset, setActivePreset] = useState('progressive-warmup')
  const [params, setParams] = useState<Params>({
    bpm: 122, key: 'Am-dorian', bars: 16,
    groove: 'mpc60', style: 'melodic',
    provider: 'offline', drums: false,
  })

  const onSelectPreset = (id: string) => {
    setActivePreset(id)
    const p = GENRE_PRESETS.find(x => x.id === id)
    if (p) setParams(prev => ({ ...prev, bpm: p.bpm, key: p.key, groove: p.groove, style: p.style }))
  }

  return (
    <>
      <Topbar />

      <section style={{ padding: '70px 0 40px' }}>
        <div className="shell" style={{ marginBottom: 36 }}>
          <div className="eyebrow" style={{ marginBottom: 22 }}>
            MIDI generation engine · for modern electronic music
          </div>
          <h1 className="hero-title">
            Coherent MIDI<br />for the floor — <em>generated.</em>
          </h1>
          <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 60, marginTop: 36, alignItems: 'end' }}>
            <p style={{ fontSize: 17, lineHeight: 1.55, color: 'var(--fg-dim)', maxWidth: '60ch', margin: 0 }}>
              Cadenza renders <span className="mono" style={{ color: 'var(--fg)' }}>bassline</span>,{' '}
              <span className="mono" style={{ color: 'var(--fg)' }}>arpeggio</span> and{' '}
              <span className="mono" style={{ color: 'var(--fg)' }}>melody</span> tracks
              that share one progression and one seed.
            </p>
            <div className="kv">
              <div className="k">tempo</div><div className="v">80 — 150 bpm</div>
              <div className="k">scales</div><div className="v">major · minor · dorian · phrygian · mixolydian · lydian</div>
              <div className="k">providers</div><div className="v">claude · ollama · openai · gemini · offline</div>
            </div>
          </div>
        </div>
        <div className="shell">
          <div className="roll-wrap">
            <PianoRoll />
            <div className="roll-overlay">
              <div className="meta">
                <span><span style={{ color: 'var(--fg-muted)' }}>BPM</span>{' '}<span className="v">{params.bpm}</span></span>
                <span><span style={{ color: 'var(--fg-muted)' }}>KEY</span>{' '}<span className="v">{params.key}</span></span>
                <span><span style={{ color: 'var(--fg-muted)' }}>BARS</span>{' '}<span className="v">{params.bars}</span></span>
              </div>
              <div />
              <div className="meta" style={{ justifyContent: 'flex-end' }}>
                <span style={{ color: 'var(--fg-muted)' }}>BASSLINE</span>
                <span style={{ color: 'var(--fg-muted)' }}>ARP</span>
                <span style={{ color: 'var(--accent)' }}>● MELODY</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <Section
        id="presets"
        eyebrow="04 genre presets · single click"
        title="Pick a <em>vibe.</em><br/>Skip the configuration."
        sub="Each preset bakes the BPM, key, mode, groove and offline style for a specific moment in a set.">
        <PresetGrid active={activePreset} onSelect={onSelectPreset} />
      </Section>

      <Section
        id="generate"
        eyebrow="generation · live controls"
        title="The console <em>is</em> the engine."
        sub="Tweak parameters, press Generate. Three MIDI files land in ~/cadenza-output ready to import into your DAW.">
        <GenerationConsole params={params} setParams={setParams} />
      </Section>

      <Section
        id="pipeline"
        eyebrow="under the hood · 4 stages"
        title="From <em>intent</em> to MIDI."
        sub="Cadenza never asks an LLM to write notes directly. Models produce a validated PatternSpec; a deterministic renderer turns it into MIDI. Musical, fast, reproducible.">
        <Pipeline />
      </Section>
    </>
  )
}
```

- [ ] **Step 2: Run `wails dev` to generate Wails JS bindings and verify the UI renders**

```powershell
cd backend/cmd/desktop
wails dev
```

Expected:
- Wails CLI builds the frontend (runs `npm install` + `npm run dev`)
- A native Windows window opens showing the Cadenza UI
- `frontend/wailsjs/` directory is generated with auto-generated TypeScript bindings
- No console errors in the Wails webview

If the window opens and the UI renders: **success**. Close with `Ctrl+C`.

- [ ] **Step 3: Run TypeScript check (after wailsjs is generated)**

```powershell
cd backend/cmd/desktop/frontend
npx tsc --noEmit
```

Expected: no errors. If `@ts-ignore` suppressed errors before, remove them now that bindings exist and verify they still compile.

- [ ] **Step 4: Run production build**

```powershell
cd backend/cmd/desktop
wails build -platform windows/amd64
```

Expected: `build/bin/cadenza-desktop.exe` created. Launch it and verify:
- Window opens with Cadenza dark UI
- PianoRoll animation plays
- Preset buttons are visible
- Generate section shows the controls

- [ ] **Step 5: Smoke test the desktop app**
  - Set provider to `offline`, BPM to 122, Key to Am, Bars to 16
  - Click **Generate**
  - Verify log panel shows: "Initializing provider...", "Generating chord progression...", "Rendering 16 bars...", "Done — 3 files written."
  - Verify 3 file paths appear in the output section (e.g. `C:\Users\...\cadenza-output\bassline_...mid`)
  - Click **Open Folder** — verify `cadenza-output` folder opens in Windows Explorer
  - Open the MIDI files in a DAW and verify they play correctly

- [ ] **Step 6: Commit**

```powershell
cd backend
git add cmd/desktop/frontend/src/App.tsx
git commit -m "feat: wire App.tsx and verify Wails build produces working EXE"
```

---

### Task 7: Update Makefile, docs, and CHANGELOG

**Files:**
- Modify: `backend/Makefile`
- Modify: `CLAUDE.md` (repo root)
- Modify: `CHANGELOG.md` (repo root, create if missing)

- [ ] **Step 1: Add desktop targets to `backend/Makefile`**

After the `build-all` target block, add:

```makefile
## Desktop (Wails)

desktop: ## Build Wails desktop EXE (Windows amd64)
	cd cmd/desktop && wails build -platform windows/amd64

desktop-dev: ## Start Wails dev mode with HMR frontend
	cd cmd/desktop && wails dev
```

In the `help` target, add:
```makefile
@echo "  make desktop        Build Wails desktop EXE (Windows)"
@echo "  make desktop-dev    Start Wails dev mode (HMR)"
```

- [ ] **Step 2: Update CLAUDE.md repository layout table**

Add these two rows to the layout table in CLAUDE.md:

```
| `backend/cmd/desktop/` | Wails desktop app entrypoint + AppService |
| `backend/cmd/desktop/frontend/` | Vite + React + TypeScript frontend (embedded into EXE) |
```

Add to the "Running" section:
```bash
# Wails desktop app
cd backend/cmd/desktop && wails dev        # dev mode with HMR
cd backend && make desktop                 # production EXE (cadenza-desktop.exe)
```

- [ ] **Step 3: Add CHANGELOG entry**

Create or update `CHANGELOG.md` at the repo root under `[Unreleased]`:

```markdown
## [Unreleased]

### Added
- Wails v2 desktop app (`cmd/desktop/`) — standalone `cadenza-desktop.exe` binding the Go engine directly to a React frontend, no HTTP layer
- `AppService` Go struct with `Generate`, `GetProviders`, `GetModels`, `GetConfig`, `OpenOutputFolder` methods exposed to the frontend via Wails runtime
- Progress events (`runtime.EventsEmit`) for real-time generation log in the desktop UI
- Vite + React + TypeScript frontend at `cmd/desktop/frontend/` — design system ported 1:1 from the CDN-based JSX prototype
- `make desktop` and `make desktop-dev` Makefile targets
- Output files land in `~/cadenza-output/` and the Open Folder button opens them in Windows Explorer
```

- [ ] **Step 4: Run `go test ./...` to ensure nothing is broken**

```powershell
cd backend
go test ./... -count=1
```

Expected: all tests PASS.

- [ ] **Step 5: Run `golangci-lint run ./...`**

```powershell
cd backend
golangci-lint run ./...
```

Expected: zero `Error:` lines. Fix any that appear before committing.

- [ ] **Step 6: Final commit**

```powershell
cd backend
git add Makefile ../CLAUDE.md ../CHANGELOG.md
git commit -m "docs: update Makefile, CLAUDE.md, CHANGELOG for Wails desktop"
```
