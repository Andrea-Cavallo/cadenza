# Cadenza REST API — Backend / Frontend Integration Spec
> Version: 0.1-draft · Date: 2026-05-01

## 1. Goal

Replace the CLI as the user-facing interface with a minimal HTTP JSON API.
The frontend sends one request, the backend generates three MIDI files,
and the user downloads them — all without touching a terminal.
The CLI (`cmd/cadenza`) is preserved for local / dev use only.

## 2. Architecture

```
Browser (frontend/)
  └─ POST /api/v1/generate  ──►  backend/cmd/api  (net/http)
                                      │
                                 internal/generator  (existing)
                                      │
                                 internal/renderer   (existing)
                                      │
                                 .mid files → temp dir
                                      │
  ◄── 200 JSON { download_url }  ◄───┘
  └─ GET /api/v1/download/{token}/{file}  →  binary MIDI
```

**No new dependencies.** The API server uses only stdlib `net/http`.
The existing generator pipeline is called directly as Go functions — no subprocess.

## 3. Endpoints

### 3.1 Health check

```
GET /api/v1/health
```

Response `200 OK`:
```json
{ "status": "ok", "version": "0.7.2" }
```

### 3.2 Configuration (for frontend dropdowns)

```
GET /api/v1/config
```

Response `200 OK`:
```json
{
  "scales":    ["major", "minor", "dorian", "phrygian", "mixolydian", "lydian"],
  "grooves":   ["straight", "mpc60", "linndrum", "humanize"],
  "providers": ["claude", "ollama", "offline"],
  "styles":    ["melodic", "peak", "progressive", "afterhours"],
  "bpm_range": { "min": 80, "max": 150 }
}
```

### 3.3 Generate MIDI

```
POST /api/v1/generate
Content-Type: application/json
```

Request body:
```json
{
  "bpm":      122,
  "key":      "Am",
  "scale":    "dorian",
  "bars":     16,
  "provider": "offline",
  "style":    "melodic",
  "groove":   "mpc60",
  "seed":     0
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `bpm` | int | yes | 80–150 |
| `key` | string | yes | e.g. `"Am"`, `"C"`, `"F#m"` |
| `scale` | string | no | default `"minor"` |
| `bars` | int | no | default `16`, valid: 8/16/32 |
| `provider` | string | no | default `"offline"` |
| `style` | string | no | default `"melodic"` |
| `groove` | string | no | default `"straight"` |
| `seed` | int64 | no | `0` = random |

Response `200 OK` (synchronous — generation completes before response):
```json
{
  "job_id":      "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "bpm":         122,
  "key":         "Am-dorian",
  "seed":        1714560219,
  "duration_ms": 847,
  "files": {
    "bassline": "/api/v1/download/f47ac10b-58cc-4372-a567-0e02b2c3d479/bassline.mid",
    "arpeggio": "/api/v1/download/f47ac10b-58cc-4372-a567-0e02b2c3d479/arpeggio.mid",
    "melody":   "/api/v1/download/f47ac10b-58cc-4372-a567-0e02b2c3d479/melody.mid",
    "zip":      "/api/v1/download/f47ac10b-58cc-4372-a567-0e02b2c3d479/cadenza.zip"
  }
}
```

Response `422 Unprocessable Entity` (validation error):
```json
{
  "error": "bpm must be between 80 and 150",
  "field": "bpm"
}
```

Response `500 Internal Server Error`:
```json
{
  "error": "generation failed: <reason>"
}
```

### 3.4 Download file

```
GET /api/v1/download/{job_id}/{filename}
```

- `filename` is one of: `bassline.mid`, `arpeggio.mid`, `melody.mid`, `cadenza.zip`
- Returns binary MIDI or ZIP with appropriate `Content-Type`
- Files are stored in a temp directory for **15 minutes** then deleted
- Returns `404` if job_id is unknown or file expired

Response headers for MIDI:
```
Content-Type: audio/midi
Content-Disposition: attachment; filename="bassline.mid"
```

Response headers for ZIP:
```
Content-Type: application/zip
Content-Disposition: attachment; filename="cadenza-122bpm-Am.zip"
```

## 4. Server Configuration

Environment variables (same `.env.example` pattern):

| Variable | Default | Notes |
|----------|---------|-------|
| `PORT` | `8080` | HTTP listen port |
| `ANTHROPIC_API_KEY` | — | Required for `provider=claude` |
| `OLLAMA_URL` | `http://localhost:11434` | For `provider=ollama` |
| `TEMP_DIR_TTL` | `15m` | How long generated files are kept |
| `CORS_ORIGINS` | `*` | Comma-separated allowed origins |

## 5. CORS

For local dev, allow all origins. In production, restrict to the frontend origin.
Use a simple middleware on all `/api/*` routes.

## 6. Frontend Integration Contract

The frontend (`frontend/console.jsx`) already has a generation console.
It must:
1. Call `GET /api/v1/config` on page load to populate dropdowns
2. Call `POST /api/v1/generate` on form submit
3. Show the response `job_id` and provide download buttons for the 4 files
4. The download buttons are simple `<a href={url} download>` tags — no JS needed

No frontend build step is added. All changes remain plain JSX loaded via Babel CDN.

## 7. Dev Mode — CLI

The CLI (`backend/cmd/cadenza`) is unchanged and continues to work for local dev:

```bash
cd backend
go run ./cmd/cadenza/ --bpm 122 --key Am --no-llm
```

The API server is the new default entrypoint for end users:

```bash
cd backend
go run ./cmd/api/ --port 8080
```

## 8. Out of Scope (v1)

- Authentication / API keys
- Rate limiting
- Persistent storage (SQLite/Postgres)
- Async job queue (jobs are synchronous, typical generation: < 2s offline, < 10s with LLM)
- WebSocket progress updates
- DAW plugin
