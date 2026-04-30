# REFACTOR — Aree di miglioramento residue

Ultimo aggiornamento: 30 aprile 2026 — aggiunti punti architettura, dev mode, service layer API-ready, JSON logging.

Tutti i refactoring precedenti sono stati completati. I punti sotto sono miglioramenti futuri identificati.

---

## 1. Schema inference dalla struct Go (eliminare inferSchemaProperty)

**Problema attuale:**

`buildSchemaFromJSON` e `inferSchemaProperty` in `claude.go` e `ollama.go` inferiscono lo schema JSON da un esempio marshallato. Questo è fragile: se l'esempio omette campi opzionali, lo schema generato è incompleto.

**Soluzione:** Generare lo schema direttamente dalla struct `PatternSpec` usando reflection o una libreria come `invopop/jsonschema` (già in go.mod). Produrrebbe uno schema completo con tipi corretti, enum, required fields.

---

## 2. Chord coherence validation: threshold troppo permissivo

**Problema attuale:**

`validateChordCoherence` richiede solo "almeno 1 chord tone per sezione". Per bassline con 4 note attive per sezione, basta una nota chord-tone su 4 per passare (25%). Un threshold più alto (50%+) produrrebbe risultati più armonici.

**Soluzione:** Differenziare per pattern type:
- `bassline`: almeno 75% chord tones (il basso è l'ancora armonica)
- `arpeggio`: almeno 80% (è letteralmente un arpeggio)
- `melody`: almeno 30% (la melodia usa passing tones legittimamente)

---

## 3. Retry correction prompt: includere esempio positivo

**Problema attuale:**

La correction prompt mostra solo l'output invalido + errori. Non mostra un esempio corretto per il caso specifico.

**Soluzione:** Nella correction, includere un frammento di 4 step corretti per la sezione problematica:

```
<correction_example>
For section 2 (bars 5-8, chord: F), correct step would be:
{"active": true, "note": "F3", "accent": true}
</correction_example>
```

---

## 4. Offline melody: manca il legato across chord boundaries

**Problema attuale:**

In `melodyTemplate`, il legato è assegnato con `noteIdx > 0 && noteIdx%2 == 0`. Questo è puramente meccanico e non tiene conto dei confini tra chord section. Il legato attraverso un cambio di chord suona scorretto.

**Soluzione:** Non assegnare legato alla prima nota di ogni nuova sezione di 4 step.

---

## 5. Filter sweep: phase offset per track type

**Problema attuale:**

I tre track (bass, arp, melody) hanno la stessa curva di sweep perché `bar` e `step` sono uguali. Se importati nello stesso DAW, tutti e tre fanno "cutoff open" nello stesso punto.

**Soluzione:** Aggiungere un offset per pattern type nel calcolo del progress:

```go
offset := map[string]float64{"bassline": 0, "arpeggio": 0.25, "melody": 0.5}
progress := (float64(bar*resolution+step) + offset[patternType]*float64(totalBars*resolution)) / float64(totalBars*resolution)
```

---

## 6. Cache: invalidation su cambio schema/prompt

**Problema attuale:**

La cache key è `provider+type+key+mode+seed`. Se il prompt template o lo schema cambiano (dopo un refactoring), la cache serve risposte obsolete che passavano il vecchio validator ma falliscono il nuovo.

**Soluzione:** Includere un hash del contenuto del prompt template nella cache key.

---

## Miglioramenti ad alto impatto

---

## 7. Seed riproducibile (`--seed`)

**Problema attuale:**

La modalità `--no-llm` usa un UUID casuale internamente ma non lo espone all'utente. È impossibile riprodurre un pattern che è piaciuto.

**Soluzione:** Aggiungere flag `--seed uint64` alla CLI. Se omesso, generare un seed casuale e stamparlo a stdout (`seed: 3847261920`). Se fornito, usarlo deterministicamente in tutti i generatori offline. Fondamentale per workflow creativi.

**Impatto:** Alto — zero costo di rendering, sblocca riproducibilità completa.

---

## 8. Output MIDI Type-1 (`--single-file`)

**Problema attuale:**

Il tool genera 3 file separati. L'import nel DAW richiede 3 drag-and-drop separati e la sincronizzazione manuale delle tracce.

**Soluzione:** Aggiungere flag `--single-file` che produce un unico MIDI Type-1 con 3 tracce nominate (`Bassline`, `Arpeggio`, `Melody`). Il writer MIDI in `internal/midi/` va esteso per supportare Type-1 (header chunk con `format=1`, un MTrk per traccia).

**Impatto:** Alto — semplifica enormemente il workflow DAW.

---

## 9. Più barre (`--bars`)

**Problema attuale:**

16 barre sono un loop. Progressive house e melodic techno hanno bisogno di 32 o 64 barre per strutturare intro/breakdown/climax.

**Soluzione:** Aggiungere flag `--bars int` (default `16`, valori validi: potenze di 2 fino a 128). Il validator e il renderer sono già parametrici; il costo principale è propagare il parametro fino ai generatori e aggiornare i template offline per non ciclare semplicemente ma evolvere.

**Impatto:** Alto — il valore musicale di una run cresce esponenzialmente.

---

## 10. Progressioni personalizzate (`--progression`)

**Problema attuale:**

Step 0 genera la progressione automaticamente. Chi ha già un giro armonico nel proprio progetto DAW non può adattare Cadenza ad esso.

**Soluzione:** Aggiungere flag `--progression "Am-F-C-G"` che bypassa il generatore di progressioni e usa i chord specificati dall'utente. Il parser deve convertire la notazione chord-name in `[]Chord` e iniettarla nel pipeline come chord contract. La coerenza armonica resta garantita dal validator.

**Impatto:** Alto — caso d'uso classico in produzione musicale.

---

## 11. Drum pattern MIDI (`--drums`)

**Problema attuale:**

Bassline + arpeggio + melodia senza drums non è un arrangiamento completo. L'utente deve aggiungere manualmente una drum track nel DAW.

**Soluzione:** Aggiungere un quarto generatore `--drums` con kick/clap/hi-hat/open-hat in stile techno/progressive. Il renderer può riutilizzare la stessa logica di velocity grid e timing offset. Il pattern drum è algoritmico (no LLM necessario); il seed controlla le variazioni. Output su CH10 (standard General MIDI percussion).

**Impatto:** Alto — chiude il cerchio dell'arrangiamento.

---

## 12. Variazioni multiple (`--variations N`)

**Problema attuale:**

Generare N versioni dello stesso input richiede N esecuzioni manuali del comando.

**Soluzione:** Aggiungere flag `--variations int` (default `1`). Con `N>1`, esegue N run con seed incrementali e salva i file con suffisso numerico (`bassline_1.mid`, `bassline_2.mid`, …). Utile per A/B comparison nel DAW senza uscire dal terminale.

**Impatto:** Medio — migliora significativamente il workflow creativo offline.

---

## 13. Groove templates (`--groove`)

**Problema attuale:**

Il timing offset è per-step nel profilo di stile ma non è esposto come preset nominato. Non esiste modo rapido di applicare swing/shuffle noti.

**Soluzione:** Aggiungere flag `--groove <name>` con preset nominati: `straight` (default), `mpc60` (Akai MPC swing 54%), `linndrum` (shuffle 58%), `humanize` (micro-variazioni casuali ±5 tick). I preset traducono in tabelle di timing offset per step e si sovrascrivono al profilo di stile.

**Impatto:** Medio — aggiunge carattere umano immediato senza complessità per l'utente.

---

## 14. Dump e re-render dello spec (`--dump-spec` / `--from-spec`)

**Problema attuale:**

Il `PatternSpec` generato dall'LLM è una black box. Non è possibile ispezionarlo, modificarlo manualmente, o re-renderizzarlo senza una nuova LLM call.

**Soluzione (loop creativo completo):**

1. `--dump-spec <dir>` — scrive il `PatternSpec` grezzo su disco in YAML (o JSON) prima del rendering, ad es. `out/spec_bassline.yaml`.
2. `--from-spec <file>` — legge un `PatternSpec` dal disco, bypassa completamente LLM e generatori, e lo renderizza direttamente.

Questo sblocca il workflow: `genera → ispeziona → modifica a mano → re-renderizza` — Cadenza diventa uno strumento di composizione interattivo, non una black box.

**Impatto:** Altissimo — zero LLM calls aggiuntive, massimo controllo creativo. Combinato con `--seed`, forma il loop creativo più potente del tool.

---

## 15. Watch mode (`--watch`)

**Problema attuale:**

Sessioni creative live richiedono di rieseguire manualmente il comando ad ogni iterazione.

**Soluzione:** Aggiungere flag `--watch` che rimane in ascolto sul stdin. Ogni `Enter` genera una nuova variazione con seed incrementale usando gli stessi parametri. `q` + `Enter` termina. Opzionale: stampare il seed di ogni variazione per poterla riprodurre.

**Impatto:** Medio — ottimo per sessioni creative live senza uscire dal terminale.

---

## 16. Provider OpenAI e Gemini

**Problema attuale:**

L'interfaccia `Provider` è pulita e già astratta, ma i provider supportati sono solo Claude e Ollama.

**Soluzione:**
- `--provider openai` con `gpt-4o` in structured output mode (JSON Schema) — circa 150 righe
- `--provider gemini` con Gemini 2.0 Flash in JSON mode — circa 150 righe

Entrambi usano l'interfaccia `Provider` esistente senza modificare il pipeline.

**Impatto:** Medio — allarga la base utenti a chi non vuole Anthropic o Ollama.

---

## 17. OpenTelemetry traces

**Problema attuale:**

Ogni run è opaca: non si sa quanto tempo impiega l'LLM vs il renderer, quanti retry avvengono, quanti cache hit si ottengono.

**Soluzione:** Strumentare `generator/`, `llm/`, `renderer/` con span OTel usando `go.opentelemetry.io/otel`. Ogni run diventa tracciabile: latenza LLM, latenza rendering, retry count, cache hit/miss. Prerequisito se si vuole esporre Cadenza come servizio HTTP.

**Impatto:** Basso per ora, alto se si scala a servizio.

---

## 18. Benchmark Go (`go test -bench`)

**Problema attuale:**

Il renderer è deterministico ma nessun benchmark misura le sue performance. Regressioni di performance sono invisibili in CI.

**Soluzione:** Aggiungere `BenchmarkRenderer` e `BenchmarkValidator` in `internal/renderer/` e `internal/schema/`. Integrarli in CI con `go test -bench=. -benchmem` e threshold di regressione.

**Impatto:** Basso ma importante per la qualità a lungo termine.

---

## 19. Flag `--dry-run`

**Problema attuale:**

Non è possibile testare il pipeline completo (LLM call, validazione, stima costi API) senza scrivere file su disco.

**Soluzione:** Aggiungere flag `--dry-run` che esegue tutto il pipeline — LLM call, validazione, rendering — ma non scrive file MIDI. Stampa un summary: token usati, costo stimato, validazioni passate/fallite, seed usato.

**Impatto:** Basso — utile principalmente per debugging e stima costi API.

---

## Architettura e osservabilità

---

## 20. Dev mode con CLI interattiva (`--dev` / `APP_ENV=development`)

**Problema attuale:**

Non esiste una modalità sviluppatore. Chi lavora sul codebase deve eseguire una run completa ogni volta per testare una modifica, senza poter isolare step del pipeline o ispezionare lo stato intermedio.

**Soluzione:**

Quando il flag `--dev` è presente (o `APP_ENV=development`), il tool entra in una REPL interattiva invece di uscire dopo la prima run:

```
cadenza [dev] > generate --bpm 122 --key Am --no-llm
cadenza [dev] > render --from-spec ./out/spec_bassline.yaml
cadenza [dev] > validate --spec ./out/spec_bassline.yaml
cadenza [dev] > inspect --step 4 --type bassline
cadenza [dev] > exit
```

Comandi disponibili in dev mode:
- `generate` — esegue il pipeline completo con i parametri dati
- `render` — re-renderizza uno spec già su disco (`--from-spec`)
- `validate` — valida uno spec YAML/JSON e stampa gli errori strutturati
- `inspect` — stampa lo step N di un pattern (note, velocity, timing, gate)
- `chord-progression` — genera e stampa solo la progressione armonica
- `cache-info` — mostra hit/miss rate e chiavi in cache
- `help` — lista comandi disponibili

Il log level in dev mode è sempre `DEBUG`, con output in formato testo colorato (non JSON) per leggibilità. In tutti gli altri ambienti il formato è JSON (vedi punto 22).

**Impatto:** Alto per developer experience — elimina il ciclo manuale genera → ispeziona → itera.

---

## 21. Service layer pubblico API-ready (prerequisito per frontend/HTTP)

**Problema attuale:**

Tutta la logica di business è accoppiata alla CLI in `cmd/cadenza/`. Non esiste un service layer con metodi pubblici chiamabili indipendentemente dalla CLI. Qualsiasi tentativo di esporre Cadenza come API HTTP richiederebbe un refactoring pesante.

**Soluzione:**

Estrarre un service layer esplicito con interfacce pubbliche Go, mantenendo la CLI come thin adapter.

### Struttura target

```
internal/
  service/
    cadenza.go          ← CadenzaService: orchestrazione principale
    generator.go        ← GeneratorService: gestisce il pipeline di generazione
    renderer.go         ← RendererService: rendering PatternSpec → MIDI bytes
    validator.go        ← ValidatorService: validazione PatternSpec
    spec.go             ← SpecService: dump/load PatternSpec da disco
```

### Interfacce pubbliche (esempio)

```go
// GeneratorService — tutto ciò che genera PatternSpec
type GeneratorService interface {
    GenerateAll(ctx context.Context, req GenerateRequest) (GenerateResult, error)
    GeneratePattern(ctx context.Context, req PatternRequest) (*schema.PatternSpec, error)
    GenerateChordProgression(ctx context.Context, key string, bars int) ([]theory.Chord, error)
}

// RendererService — PatternSpec → MIDI bytes
type RendererService interface {
    RenderPattern(ctx context.Context, spec *schema.PatternSpec, profile styleprofile.Profile) ([]byte, error)
    RenderAll(ctx context.Context, specs [3]*schema.PatternSpec) (RenderResult, error)
}

// ValidatorService — validazione con errori strutturati
type ValidatorService interface {
    Validate(spec *schema.PatternSpec) []ValidationError
}

// CadenzaService — punto di ingresso unico per un futuro handler HTTP/gRPC
type CadenzaService interface {
    Run(ctx context.Context, req RunRequest) (RunResult, error)
}
```

### Regole di separazione

- `cmd/cadenza/` — solo parsing CLI, flag, I/O su disco, stampa output. **Zero business logic.**
- `internal/service/` — business logic pura, nessuna dipendenza da `os.Args`, `fmt.Println`, flag packages.
- I service ricevono `context.Context` come primo argomento — prerequisito per tracing OTel (punto 17) e cancellazione HTTP.
- Tutti i metodi pubblici accettano e restituiscono tipi Go (no JSON strings) — la serializzazione è responsabilità del transport layer (CLI o HTTP handler).

### Futuro: `cmd/server/`

Una volta estratto il service layer, aggiungere `cmd/server/` diventa banale:

```go
// cmd/server/main.go
svc := service.NewCadenzaService(cfg)
http.HandleFunc("/generate", handlers.Generate(svc))
http.HandleFunc("/render",   handlers.Render(svc))
```

Zero duplicazione della logica di generazione tra CLI e API.

**Impatto:** Altissimo a lungo termine — è il prerequisito architetturale per ogni integrazione frontend, API HTTP, gRPC, o servizio cloud.

---

## 22. JSON structured logging con livelli corretti (slog)

**Problema attuale:**

Il codebase usa `log/slog` (già importato) ma probabilmente con il handler di default (testo non strutturato). Mancano:
- Formato JSON in produzione
- Livelli applicati in modo consistente (DEBUG/INFO/WARN/ERROR)
- Campi strutturati per correlare log di una stessa run
- Logging di debug granulare nei punti critici del pipeline

**Soluzione:**

### 1. Configurazione del logger per ambiente

```go
// internal/logger/logger.go

func New(env, level string) *slog.Logger {
    var handler slog.Handler
    lvl := parseLevel(level) // default INFO

    if env == "development" {
        // Testo colorato leggibile in dev mode
        handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
    } else {
        // JSON strutturato in produzione / CI / futura API
        handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
    }
    return slog.New(handler)
}
```

Flag CLI: `--log-level debug|info|warn|error` (default `info`). In `--dev` mode il default è `debug`.

### 2. Campi strutturati obbligatori per run

Ogni logger viene arricchito all'inizio di una run con i campi della richiesta:

```go
log := logger.With(
    slog.String("provider",      cfg.Provider),
    slog.String("key",           cfg.Key),
    slog.Int("bpm",              cfg.BPM),
    slog.Int("bars",             cfg.Bars),
    slog.Uint64("seed",          cfg.Seed),
    slog.String("run_id",        uuid.New().String()),
)
```

Ogni log di quella run porta automaticamente tutti i campi — facilita il grep e la correlazione in produzione.

### 3. Livelli e campi per punto del pipeline

| Punto del pipeline | Livello | Campi aggiuntivi |
|---|---|---|
| Inizio run | `INFO` | `provider`, `key`, `bpm`, `seed` |
| Chord progression generata | `INFO` | `chords` |
| LLM call avviata | `DEBUG` | `pattern_type`, `attempt` |
| LLM response ricevuta | `DEBUG` | `pattern_type`, `duration_ms`, `tokens_used` |
| Cache hit | `DEBUG` | `pattern_type`, `cache_key` |
| Cache miss | `DEBUG` | `pattern_type`, `cache_key` |
| Retry LLM | `WARN` | `pattern_type`, `attempt`, `error_type`, `error` |
| Fallback a offline | `WARN` | `pattern_type`, `reason` |
| Validation error | `WARN` | `pattern_type`, `rule`, `step`, `note`, `error` |
| Rendering completato | `INFO` | `pattern_type`, `duration_ms`, `steps_rendered` |
| File scritto su disco | `INFO` | `path`, `size_bytes` |
| Errore fatale | `ERROR` | `error`, `stack` (se disponibile) |

### 4. Esempio di output JSON in produzione

```json
{"time":"2026-04-30T10:22:01Z","level":"INFO","msg":"run started","provider":"claude","key":"Am","bpm":122,"bars":16,"seed":3847261920,"run_id":"a1b2c3"}
{"time":"2026-04-30T10:22:01Z","level":"INFO","msg":"chord progression","chords":["Am","F","C","G"],"run_id":"a1b2c3"}
{"time":"2026-04-30T10:22:01Z","level":"DEBUG","msg":"llm call","pattern_type":"bassline","attempt":1,"run_id":"a1b2c3"}
{"time":"2026-04-30T10:22:02Z","level":"DEBUG","msg":"llm response","pattern_type":"bassline","duration_ms":1243,"tokens_used":412,"run_id":"a1b2c3"}
{"time":"2026-04-30T10:22:02Z","level":"WARN","msg":"llm retry","pattern_type":"bassline","attempt":2,"error_type":"musical","error":"chord coherence < 75% in section 2","run_id":"a1b2c3"}
{"time":"2026-04-30T10:22:03Z","level":"INFO","msg":"pattern rendered","pattern_type":"bassline","duration_ms":4,"steps_rendered":64,"run_id":"a1b2c3"}
{"time":"2026-04-30T10:22:03Z","level":"INFO","msg":"file written","path":"./out/bassline.mid","size_bytes":1842,"run_id":"a1b2c3"}
```

### 5. Regole di disciplina per il logging

- **No `fmt.Println` nel business logic** — solo nel layer CLI per output utente intenzionale
- **No log di segreti** — mai loggare API key, anche a livello DEBUG
- **No log in hot path** — i loop per-step del renderer usano `DEBUG` condizionale (`if logger.Enabled(ctx, slog.LevelDebug)`)
- **Errori sempre con `slog.Any("error", err)`** — non interpolati nella stringa del messaggio
- **`run_id` su tutti i log** — prerequisito per correlare una run in produzione o in OTel

**Impatto:** Alto — prerequisito per debugging efficace, produzione, e futura integrazione OTel (punto 17).
