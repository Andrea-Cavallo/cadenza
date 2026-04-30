# CADENZA

<p align="center">
  <img src="cadenza.png" alt="Cadenza" width="480" />
</p>

<p align="center">

[![Go 1.25](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://golang.org)
[![CI](https://github.com/Andrea-Cavallo/cadenza/actions/workflows/ci.yml/badge.svg)](https://github.com/Andrea-Cavallo/cadenza/actions/workflows/ci.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=coverage)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=Andrea-Cavallo_cadenza&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=Andrea-Cavallo_cadenza)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![CGO_ENABLED=0](https://img.shields.io/badge/CGO-disabled-lightgrey)](https://pkg.go.dev/cmd/cgo)

</p>

> AI-powered MIDI generator for progressive house and melodic techno.  
> Give it a BPM and a musical key, get three harmonically coherent MIDI files back.

---

## Languages

- [English](#english)
- [Italiano](#italiano)
- [Espanol](#espanol)

---

## English

### What is Cadenza?

Cadenza turns a BPM and a key into three MIDI files: bassline, arpeggio, and melody. All three parts share the same chord progression, so the result is coherent and ready to drop into a DAW.

The project can run with an LLM, but one of its biggest strengths is that it also works extremely well without one.

### Offline mode is a core feature

`--no-llm` is not an emergency fallback. It is one of the strongest parts of the project.

Offline mode:

- makes zero API calls
- generates musically useful, hypnotic patterns algorithmically
- keeps harmonic coherence, timing, velocity, and renderer quality
- is fast, reproducible, cheap, and reliable in CI, Docker, and local workflows

If you want to sketch ideas quickly, test arrangements, or avoid API cost and network dependency, offline mode is often the best way to use Cadenza.

### Features

- One command, three MIDI files
- Shared chord progression across bassline, arpeggio, and melody
- Deterministic renderer with timing, velocity, gate, sweep, and portamento rules
- Claude, Ollama, OpenAI, and Gemini support
- Strong offline generation with `--no-llm`
- Graceful fallback to offline templates if an LLM fails
- LLM response cache
- Static binaries with `CGO_ENABLED=0`

### Requirements

To use the project after cloning, you need:

- Go `1.25`
- Git
- Optional: `make`
- Optional: Ollama for local LLM mode
- Optional: API keys for Claude, OpenAI, or Gemini

Check your Go version:

```bash
go version
```

### Use it immediately without building

If you just cloned the repo and want to try it right away:

```bash
go run ./cmd/cadenza/ --bpm 122 --key Am --no-llm
```

This is the fastest first run.

### Build after cloning

#### Windows (PowerShell)

```powershell
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza.exe ./cmd/cadenza/
.\bin\cadenza.exe --bpm 122 --key Am --no-llm
```

#### macOS

```bash
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza ./cmd/cadenza/
./bin/cadenza --bpm 122 --key Am --no-llm
```

#### Linux

```bash
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza ./cmd/cadenza/
./bin/cadenza --bpm 122 --key Am --no-llm
```

Output goes to `./output/` by default.

### Run with hosted LLMs

#### Claude

macOS / Linux:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
./bin/cadenza --bpm 122 --key Am
```

Windows PowerShell:

```powershell
$env:ANTHROPIC_API_KEY="sk-ant-..."
.\bin\cadenza.exe --bpm 122 --key Am
```

#### OpenAI

macOS / Linux:

```bash
export OPENAI_API_KEY=sk-...
./bin/cadenza --bpm 124 --key Em --provider openai
```

Windows PowerShell:

```powershell
$env:OPENAI_API_KEY="sk-..."
.\bin\cadenza.exe --bpm 124 --key Em --provider openai
```

#### Gemini

macOS / Linux:

```bash
export GEMINI_API_KEY=...
./bin/cadenza --bpm 128 --key Dm --provider gemini
```

Windows PowerShell:

```powershell
$env:GEMINI_API_KEY="..."
.\bin\cadenza.exe --bpm 128 --key Dm --provider gemini
```

### Run with Ollama

1. Install Ollama from [ollama.com](https://ollama.com)
2. Pull a model:

```bash
ollama pull qwen2.5:7b
```

3. Start the server:

```bash
ollama serve
```

4. Run Cadenza:

macOS / Linux:

```bash
./bin/cadenza --bpm 126 --key Fm --provider ollama --model qwen2.5:7b
```

Windows PowerShell:

```powershell
.\bin\cadenza.exe --bpm 126 --key Fm --provider ollama --model qwen2.5:7b
```

### Useful commands

```bash
make build
make build-all
make test
make test-race
make test-coverage
make ci
```

### Cross-platform release builds

If you want binaries for all supported platforms:

```bash
make build-all
```

This creates:

- `bin/cadenza-linux-amd64`
- `bin/cadenza-linux-arm64`
- `bin/cadenza-darwin-amd64`
- `bin/cadenza-darwin-arm64`
- `bin/cadenza-windows-amd64.exe`
- `bin/cadenza-windows-arm64.exe`

### Modes at a glance

| Mode | Internet | API Key | Notes |
|---|---|---|---|
| Offline | No | No | Fast, algorithmic, reliable, excellent for sketches and production starters |
| Claude | Yes | Yes | Default hosted LLM mode |
| Ollama | No (after model download) | No | Local LLM mode |
| OpenAI | Yes | Yes | Structured output mode |
| Gemini | Yes | Yes | JSON mode |

### Why `CGO_ENABLED=0`

The project is intentionally built as a pure Go application with static binaries and minimal runtime dependencies. That keeps builds portable and makes CI, Docker, and cross-compilation simpler.

---

## Italiano

### Cos'e Cadenza?

Cadenza trasforma BPM e tonalita in tre file MIDI: bassline, arpeggio e melodia. Tutte e tre le parti condividono la stessa progressione armonica, quindi il risultato e coerente e pronto da importare nella DAW.

Il progetto puo usare un LLM, ma uno dei suoi punti forti principali e che funziona molto bene anche senza LLM.

### La modalita offline e un punto di forza

`--no-llm` non e un ripiego. E una delle forze principali del progetto.

La modalita offline:

- non usa API
- genera pattern utili in modo algoritmico
- mantiene coerenza armonica, timing, velocity e qualita del renderer
- e veloce, riproducibile, economica e affidabile

### Requisiti

Dopo il clone servono:

- Go `1.25`
- Git
- opzionale: `make`
- opzionale: Ollama per la modalita LLM locale
- opzionale: chiavi API per Claude, OpenAI o Gemini

### Provarlo subito senza build

```bash
go run ./cmd/cadenza/ --bpm 122 --key Am --no-llm
```

### Compilazione dopo il clone

#### Windows (PowerShell)

```powershell
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza.exe ./cmd/cadenza/
.\bin\cadenza.exe --bpm 122 --key Am --no-llm
```

#### macOS

```bash
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza ./cmd/cadenza/
./bin/cadenza --bpm 122 --key Am --no-llm
```

#### Linux

```bash
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza ./cmd/cadenza/
./bin/cadenza --bpm 122 --key Am --no-llm
```

L'output va in `./output/`.

### Provider hosted e locali

Claude:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
./bin/cadenza --bpm 122 --key Am
```

Windows PowerShell:

```powershell
$env:ANTHROPIC_API_KEY="sk-ant-..."
.\bin\cadenza.exe --bpm 122 --key Am
```

OpenAI:

```bash
export OPENAI_API_KEY=sk-...
./bin/cadenza --bpm 124 --key Em --provider openai
```

Gemini:

```bash
export GEMINI_API_KEY=...
./bin/cadenza --bpm 128 --key Dm --provider gemini
```

Ollama:

```bash
ollama pull qwen2.5:7b
ollama serve
./bin/cadenza --bpm 126 --key Fm --provider ollama --model qwen2.5:7b
```

### Comandi utili

```bash
make build
make build-all
make test
make test-race
make test-coverage
make ci
```

---

## Espanol

### Que es Cadenza?

Cadenza convierte BPM y tonalidad en tres archivos MIDI: bassline, arpegio y melodia. Las tres partes comparten la misma progresion armonica, asi que el resultado sale coherente y listo para llevar a una DAW.

El proyecto puede usar un LLM, pero una de sus mayores fortalezas es que funciona muy bien incluso sin LLM.

### El modo offline es una fortaleza real

`--no-llm` no es un plan B. Es una parte fuerte del proyecto.

El modo offline:

- no hace llamadas API
- genera patrones utiles de forma algoritmica
- mantiene coherencia armonica, timing, velocity y calidad del renderer
- es rapido, reproducible, barato y fiable

### Requisitos

Despues de clonar el repo necesitas:

- Go `1.25`
- Git
- opcional: `make`
- opcional: Ollama para modo LLM local
- opcional: claves API para Claude, OpenAI o Gemini

### Probarlo sin compilar

```bash
go run ./cmd/cadenza/ --bpm 122 --key Am --no-llm
```

### Compilar despues del clone

#### Windows (PowerShell)

```powershell
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza.exe ./cmd/cadenza/
.\bin\cadenza.exe --bpm 122 --key Am --no-llm
```

#### macOS

```bash
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza ./cmd/cadenza/
./bin/cadenza --bpm 122 --key Am --no-llm
```

#### Linux

```bash
git clone https://github.com/Andrea-Cavallo/cadenza
cd cadenza
go build -o bin/cadenza ./cmd/cadenza/
./bin/cadenza --bpm 122 --key Am --no-llm
```

La salida va a `./output/`.

### Usar proveedores

Claude:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
./bin/cadenza --bpm 122 --key Am
```

OpenAI:

```bash
export OPENAI_API_KEY=sk-...
./bin/cadenza --bpm 124 --key Em --provider openai
```

Gemini:

```bash
export GEMINI_API_KEY=...
./bin/cadenza --bpm 128 --key Dm --provider gemini
```

Ollama:

```bash
ollama pull qwen2.5:7b
ollama serve
./bin/cadenza --bpm 126 --key Fm --provider ollama --model qwen2.5:7b
```

### Comandos utiles

```bash
make build
make build-all
make test
make test-race
make test-coverage
make ci
```

---

## Architecture

```text
User (BPM + Key)
  -> Key parser
  -> Shared chord progression
  -> Parallel generators (bassline, arpeggio, melody)
  -> Validator
  -> Renderer
  -> 3 MIDI files
```

## Repository Notes

- Go version: `1.25`
- Module path: `github.com/Andrea-Cavallo/cadenza`
- Default CLI entry point: `./cmd/cadenza/`
- Default output directory: `./output/`
- Offline mode is a first-class workflow, not a degraded one
