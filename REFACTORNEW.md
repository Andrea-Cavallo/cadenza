# REFACTORNEW — Roadmap to Production Quality

> **Istruzioni per l'agente:** Questo file e' la checklist ufficiale dei miglioramenti.
> Quando un punto viene completato, spuntare la checkbox `[ ]` → `[x]`.
> Quando un punto viene iniziato ma non completato, annotare lo stato accanto.
> Aggiornare questo file ad ogni modifica rilevante al codebase.
> L'ordine e' logico: prima si pulisce, poi si consolida, poi si aggiunge.

---

## Checklist Riepilogativa

### Phase 1 — Eliminare Dead Code (pulire il terreno)
- [x] #1 Eliminare 5 file 100% dead (~480 righe)
- [x] #2 Rimuovere stub `DumpToJSON`/`LoadFromJSON` rotti — implementati con encoding/json
- [x] #3 Rimuovere/implementare dev stubs inutili — messaggio utile + redirect a flag
- [x] #4 `go mod tidy` — dipendenze phantom (`gorilla/mux`, `rs/cors`) rimosse
- [x] #5 Rendere unexported `ChordsInKey`/`ProgressionPool`

### Phase 2 — Deduplicare e Consolidare
- [x] #6 Eliminare duplicazione utility functions — `theory.ParseChordName` + `theory.QualitySuffix` in chord_util.go
- [x] #7 Rimuovere stdlib reimplementate a mano — sostituite con strings.Split/TrimSpace/ToLower/Join
- [x] #8 Decisione architetturale: service layer — ELIMINATO (zero importers, ~700 righe rimosse)
- [x] #9 Spezzare `main.go` god-file — estratto `provider.go` + `logger.go` (637→506 righe)

### Phase 3 — Fixare Bug Critici
- [x] #10 Cache TTL rotta — `cache.New` ora accetta `ttlDays` (30 = 30 giorni)
- [x] #11 `varLen` sbagliata in service/renderer.go — risolto eliminando service layer (#8)
- [x] #12 Style Profile names — validator allineato al registry + `NewValidatorWithProfiles`
- [x] #13 Ollama URL — letta da config, passata a `buildProvider`

### Phase 4 — Sicurezza
- [x] #14 Gemini API key — spostata da URL query param a header `x-goog-api-key`
- [x] #15 Errori silenziati — ora loggati con slog.Warn (cache, generator, profile)
- [x] #16 Config validation — `AppConfig.Validate()` con check su BPM, velocity, env, format

### Phase 5 — Test Coverage
- [~] #17 Aggiungere test ai 7 package con 0% coverage — fatto per cache (93%), config (96%), llm (15%), metrics. Restano: logger, styleprofile, cmd/cadenza
- [~] #18 Portare i 4 package sotto-soglia a 80%+ — schema 26→61%, midi 49→91%. Restano: generator (60%), llm (15%), renderer (72%), theory (68%)
- [x] #19 Coverage gate in CI — fail se < 80% (awk su `go tool cover -func`)

### Phase 6 — Deployment e CI
- [x] #20 Dockerfile: user non-root aggiunto
- [x] #21 Docker Compose: fix Ollama service — healthcheck + `CADENZA_LLM_OLLAMA_URL` + `service_healthy`
- [x] #22 `make install-tools` — installa golangci-lint v1.64.8 + govulncheck + goimports
- [x] #23 `make ci` — ora include `go build` + cross-compile linux
- [x] #24 Pinnare `golangci-lint` in CI — `go install ...@v1.64.8`
- [x] #25 Release workflow: `windows/arm64` — aggiunto alla matrix
- [x] #26 `make release-snapshot` — recipe aggiunta

### Phase 7 — Osservabilita'
- [x] #27 Metriche — `internal/metrics/metrics.go` con expvar counters (generations, errors, cache, LLM)
- [x] #28 Fix goroutine leak session cache — risolto eliminando `session.go` in Phase 1

### Phase 8 — Qualita' Musicale
- [x] #29 Fix `approachNote` — ora cromatico (semitono sotto via MIDI arithmetic)
- [x] #30 Assegnare DynamicCurve "plateau"/"tension" — `bass_driving`=plateau, `arp_epic`=tension
- [x] #31 ModWheelProfile — gia' attivo su melody_hypnotic (Enabled: true)

### Phase 9 — Developer Experience
- [x] #32 Godoc comments su tipi pubblici — aggiunto a tutti i tipi esportati nei package core
- [x] #33 API HTTP — decisione: non prevista (CLI-only tool, documentato sotto)

---

## Phase 1: Eliminare Dead Code

> Principio: prima togli il rumore, poi vedi chiaro cosa c'e' da fixare.

### 1. File 100% dead — eliminabili immediatamente (~480 righe)

| File | Righe | Motivo |
|------|-------|--------|
| `internal/examples/types.go` | 46 | Tipi `ExampleBank`, `ExampleProfile`, `PatternSnippet`, `RawNote`, `SnippetStep` mai usati |
| `internal/cache/session.go` | 220 | `SessionCache` mai istanziato. Contiene goroutine leak. Il disco cache (`cache.go`) e' l'unico usato |
| `internal/renderer/groove/groove.go` | 71 | `Profile`, `Get()`, `MPC60()` etc. mai chiamati. Groove reimplementato altrove |
| `internal/midi/writer_type1.go` | 124 | `WriteType1File()` mai chiamato. Type-1 logic duplicata in `service/renderer.go` |
| `internal/service/cadenza.go:389` (`formatChords`) | 18 | Funzione mai chiamata — esiste `formatProgression()` |

### 2. Rimuovere stub `DumpToJSON`/`LoadFromJSON`
- **File:** `internal/schema/spec_io.go:79-87`
- **Problema:** `marshalIndent` restituisce `{}` e un errore. `unmarshal` restituisce sempre errore
- **Fix:** Eliminare completamente (il service layer ha la propria implementazione JSON funzionante)

### 3. Dev mode stubs incompleti

| Funzione | File |
|----------|------|
| `devRender()` | `cmd/cadenza/dev.go:219` |
| `devInspect()` | `cmd/cadenza/dev.go:252` |
| `devFromSpec()` | `cmd/cadenza/dev.go:334` |
| `devDumpSpec()` | `cmd/cadenza/dev.go:338` |

Stampano "not yet fully implemented" e ritornano. Rimuovere o implementare.

### 4. Dipendenze Go inutilizzate
- `github.com/gorilla/mux v1.8.1` — nessun `.go` le importa
- `github.com/rs/cors v1.11.1` — nessun `.go` le importa
- **Fix:** `go mod tidy`

### 5. Funzioni esportate da rendere unexported

| Funzione | File | Motivo |
|----------|------|--------|
| `ChordsInKey()` | `internal/theory/chord.go:50` | Solo uso interno + test |
| `ProgressionPool()` | `internal/theory/progression.go:39` | Solo uso interno |

---

## Phase 2: Deduplicare e Consolidare

> Principio: una sola implementazione per ogni concetto. Il codice duplicato diverge.

### 6. Duplicazione massiva di funzioni utility
- `parseChordName` — duplicata in `main.go` e `service/cadenza.go`
- `qualitySuffix` — in 3 posti (`dev.go`, `generator.go`, `service/cadenza.go`)
- `varLen` e `eventPriority` — in `midi/writer.go` e `service/renderer.go` (con bug!)
- `progressionString` — 3 varianti
- **Fix:** Estrarre in `internal/theory/chord_util.go` per chord helpers, esportare `VarLen` da `internal/midi/`

### 7. Stdlib reimplementate a mano
- **File:** `internal/service/cadenza.go`
- `splitString` → `strings.Split`
- `trimSpace` → `strings.TrimSpace`
- `toLower` → `strings.ToLower`
- `joinStrings` → `strings.Join`
- **Fix:** Sostituire con chiamate stdlib dirette

### 8. Decisione architetturale: Service Layer
- **Stato attuale:** `internal/service/` (~700 righe, 7 file) NON e' importato da `cmd/cadenza/main.go`
- Il CLI usa direttamente `generator.MultiGenerator`, `renderer.Renderer`, `schema.Validator`
- Due pipeline parallele che divergono silenziosamente
- **Opzioni:**
  1. **Wireare** `main.go` al service layer → CLI diventa thin adapter (preferibile se API HTTP pianificata)
  2. **Eliminare** `internal/service/` → meno codice, una sola pipeline
- **Decidere prima di procedere con Phase 3** (i bug nel service layer diventano irrilevanti se lo eliminiamo)

### 9. Spezzare `cmd/cadenza/main.go` (637 righe)
- Mescola: flag parsing, provider construction, output formatting, spec re-rendering, MIDI routing, chord parsing
- **Fix dopo decisione #8:**
  - Se service layer wired: `main.go` chiama solo `CadenzaService.Run()`, diventa <100 righe
  - Se eliminato: estrarre in `flags.go`, `provider.go`, `output.go`

---

## Phase 3: Fixare Bug Critici

> Principio: ora che il codice e' pulito e non duplicato, i fix si applicano in un solo posto.

### 10. Cache TTL rotta — 30 secondi invece di 30 giorni
- **File:** `internal/generator/multi_generator.go:73`, `internal/service/generator.go:22`
- **Problema:** `cache.New(30, ".cache")` crea TTL di 30 *secondi*
- **Impatto:** Cache funzionalmente inutile — chiamate API ridondanti e costi
- **Fix:** Refactorare `cache.New` per accettare `ttlDays int` e moltiplicare `ttlDays * 24 * 3600` internamente

### 11. `varLen` sbagliata in service/renderer.go
- **File:** `internal/service/renderer.go:261`
- **Problema:** Variable-length encoding MIDI incorretta per valori >= 128 ticks
- **Impatto:** MIDI Type-1 con delta-time corrotti — file inapribili
- **Fix:** Se service layer eliminato (#8), il bug sparisce. Se wired, importare da `internal/midi`

### 12. Style Profile names divergenti
- **File:** `internal/schema/validator.go:23` vs `internal/renderer/styleprofile/registry.go`
- Nel registry ma rifiutati dal validator: `bass_sub`, `arp_epic`, `arp_staccato`, `melody_hypnotic`
- Nel validator ma senza implementazione: `bass_melodic_dark`, `bass_deep`, `arp_rhythmic`, `arp_ambient`, `arp_plucky`, `melody_minimal`, `melody_soaring`, `melody_rhythmic`
- **Fix:** Il validator deve derivare la lista valida dal registry (unica source of truth)

### 13. Ollama URL hardcodata
- **File:** `cmd/cadenza/main.go` (funzione `buildProvider`)
- L'URL e' hardcoded `"http://localhost:11434"` ignorando `config.OllamaURL` / env `CADENZA_LLM_OLLAMA_URL`
- **Impatto:** Docker Compose `cadenza-ollama` non funziona
- **Fix:** Leggere `cfg.OllamaURL` e passarla al provider

---

## Phase 4: Sicurezza

### 14. Gemini API key nell'URL (query parameter)
- **File:** `internal/llm/gemini.go:150`
- `?key=<apikey>` nell'URL — se appare in un log, la chiave e' esposta
- **Fix:** Header `x-goog-api-key` (pattern Google raccomandato)

### 15. Errori silenziati in path critici
- `_ = os.Remove(path)` in cache cleanup (`cache.go:67`)
- `_ = g.cache.Set(...)` in `generator.go:119`
- `profile, _ = reg.DefaultForType(...)` in `main.go:558` — possibile nil pointer dereference
- **Fix:** `slog.Warn` per cache ops; return error per profile lookup

### 16. Config validation assente
- Nessuna validazione: `max_velocity > 127`, `ttl_days < 0`, `app.env` non riconosciuto sono accettati
- **Fix:** `config.Validate()` chiamata all'avvio con errori chiari

---

## Phase 5: Test Coverage

> Principio: ora che il codice e' pulito e corretto, testalo per non regredire.

### 17. Package con 0% coverage
| Package | Azione |
|---------|--------|
| `internal/cache` | Test Get/Set/TTL/cleanup/expiry |
| `internal/config` | Test parsing, defaults, env override, invalidi |
| `internal/llm` | Test con MockProvider, retry, error classification |
| `internal/logger` | Test JSON/text format, level filtering |
| `internal/service` | Integration test (se non eliminato) |
| `cmd/cadenza` | Test flag parsing, output routing |

### 18. Package sotto soglia 80%
| Package | Coverage | Target |
|---------|----------|--------|
| `internal/schema` | 26.5% | Validator: chord coherence, density, range |
| `internal/midi` | 49.6% | varLen edge cases, event ordering |
| `internal/generator` | 58.6% | Offline templates, cache integration |
| `internal/renderer` | 72.8% | Evolution, dynamic curve, portamento |

### 19. Coverage gate in CI
- **File:** `.github/workflows/ci.yml`
- `go tool cover -func` stampa ma non fallisce sotto 80%
- **Fix:** `go tool cover -func=coverage.out | awk '/^total/ { if ($3+0 < 80) { print "FAIL: Coverage " $3 " < 80%"; exit 1 } }'`

---

## Phase 6: Deployment e CI

### 20. Dockerfile: user non-root
```dockerfile
RUN adduser -D -g '' cadenza
USER cadenza
```

### 21. Docker Compose: fix Ollama service
- `depends_on: ollama` senza `condition: service_healthy`
- `OLLAMA_HOST` settato ma ignorato (vedi #13)
- `cadenza-claude` senza fallback per `ANTHROPIC_API_KEY` mancante

### 22. `make install-tools`
- `goimports`, `golangci-lint`, `govulncheck` richiesti ma nessun target per bootstrap
- Nuovo contributor non sa cosa installare

### 23. `make ci` include `go build`
- Un build break non catturato dalla pipeline locale
- Aggiungere `go build ./...` e `GOOS=linux go build ./...`

### 24. Pinnare `golangci-lint` in CI
- Attualmente `@latest` — puo' rompere senza preavviso
- Pinnare a `v1.64.8` (o versione corrente)

### 25. Release workflow: `windows/arm64`
- Il Makefile lo builda, il release workflow no

### 26. Fix `make release-snapshot`
- Listato in `.PHONY` senza recipe — silenziosamente non fa nulla

---

## Phase 7: Osservabilita'

### 27. Metriche
- Nessun endpoint Prometheus/expvar
- Servono: durata generazione, cache hit rate, errori LLM, token usage
- **Fix:** `expvar` (zero dipendenze) o `prometheus/client_golang`

### 28. Goroutine leak session cache
- Se `session.go` eliminato in Phase 1 (#1), questo punto e' automaticamente risolto
- Altrimenti: accettare `context.Context`, fermare ticker su `ctx.Done()`

---

## Phase 8: Qualita' Musicale

### 29. `approachNote` e' diatonico, non cromatico
- **File:** `internal/generator/offline.go`
- Commento dice "chromatic approach" ma usa grado della scala sotto
- **Fix:** `targetMIDI - 1` per vero cromatico, oppure aggiornare commento

### 30. DynamicCurve "plateau"/"tension" dead code
- Implementate in `renderer.go:dynamicCurveScale` ma nessun profilo le usa
- **Fix:** Assegnare a profili (`bass_sub` → plateau, `arp_epic` → tension) o rimuovere

### 31. ModWheelProfile mai attivato
- `ModWheel.Enabled = false` in tutti i profili
- `melody_hypnotic` dovrebbe avere mod wheel lento (specificato nel vecchio REFACTOR.md)
- **Fix:** Attivare e testare rendering CC1

---

## Phase 9: Developer Experience

### 32. Godoc comments mancanti
- Tipi pubblici in `internal/service/interfaces.go`, `internal/schema/`, `internal/theory/` senza doc
- **Fix:** Una riga di godoc per tipo/interfaccia esportata

### 33. API HTTP — decisione: NON PREVISTA
- `internal/service/` eliminato in Phase 2 (#8) — non serviva come fondazione HTTP
- Cadenza e' uno strumento CLI per musicisti/produttori, non un servizio web
- Se in futuro servira' un'API, si partira' da zero con architettura adeguata
- **Stato: CHIUSO** — nessuna azione richiesta

---

## Riepilogo

| Phase | Focus | Items | Effetto |
|-------|-------|-------|---------|
| 1 | Eliminare | #1-5 | -480 righe, zero rumore |
| 2 | Deduplicare | #6-9 | Una sola pipeline, zero divergenza |
| 3 | Bug fix | #10-13 | Cache funziona, MIDI validi, Docker ok |
| 4 | Sicurezza | #14-16 | No key leak, no nil panic, config safe |
| 5 | Test | #17-19 | 80%+ coverage, CI che fallisce se regredisce |
| 6 | Deploy/CI | #20-26 | Container sicuro, CI robusto, DX bootstrap |
| 7 | Osservabilita' | #27-28 | Visibilita' runtime |
| 8 | Musicale | #29-31 | Output piu' espressivo e corretto |
| 9 | DX | #32-33 | Onboarding chiaro, futuro definito |
