# Cadenza — FAQ

1. **Che linguaggio di programmazione hai scelto e perché?**
   Go 1.25. Scelto per: compilazione nativa senza runtime, cross-compilazione banale (`CGO_ENABLED=0`), performance da linguaggio compilato, `embed` nativo per i prompt, e `log/slog` strutturato nella stdlib. Nessuna dipendenza pesante.

2. **Qual è l'architettura del progetto?**
   CLI + Desktop app (Wails v2) che condividono lo stesso backend Go. Input: BPM + Key → KeyParser → Chord Progression → MusicalPlan (LLM) → 3 generatori paralleli (bass/arp/melody) → Validator → StyleProfile → Renderer → 3 file MIDI Type-0. Offline mode senza LLM disponibile.

3. **Perché generi 3 file MIDI separati invece di uno solo?**
   Ogni traccia (bassline, arpeggio, melody) ha il suo channel MIDI, range di note, densità e style profile dedicati. File separati permettono al producer di importarli su tracce diverse nel DAW con processing indipendente (compressione sul bass, riverbero sull'arp, etc.).

4. **Come garantisci coerenza armonica tra le 3 tracce?**
   Una chord progression condivisa (Step 0) fa da "contratto armonico". Tutti e 3 i generatori ricevono la stessa progressione. Il validator controlla che ogni sezione di 4 bar contenga almeno un chord tone. Le note fuori scala vengono bloccate.

5. **Quali LLM supporti e come li integri?**
   Claude (Anthropic) via `tool_use`, Ollama (locale) via JSON schema mode, OpenAI e Gemini. Interfaccia `llm.Provider` unificata. Retry fino a 3 tentativi con classificazione errori (strutturali vs musicali) e prompt di correzione diversi. Cache SHA-256 su disco, 30 giorni TTL.

6. **Come funziona la modalità offline (`--no-llm`)?**
   Generazione algoritmica deterministica basata su seed. Template per pattern type con variazione ritmica, contour diversity, chord awareness. Nessuna chiamata API. Stesso seed = stesso output. 4 style family: Groove, Rolling, Sub.

7. **Perché il frontend desktop è in React + TypeScript e non in altro?**
   Wails v2 Embedding nativo di Vite + React. TypeScript per type safety sul contratto Go↔Frontend (bindings auto-generati). CSS custom properties per theming. JetBrains Mono come font principale per l'estetica da music tool.

8. **Come gestisci timing e velocity nei MIDI?**
   Il Renderer è deterministico: gli StyleProfile applicano timing offset, velocity grid, gate length, filter sweep (S-curve esponenziale), portamento (CC65) e DynamicCurve (crescendo 0.7→1.0 per bass/melody, arch 0.75→1.0→0.85 per arp). Velocity max: 120, mai 127 (clipping). Downbeat sempre on-grid.

9. **Quali chiavi e modalità supporti?**
   Maggiore, minore naturale, dorico, frigio, misolidio, lidio. Notazione: `Am`, `C`, `F#m`, `Am-dorian`, `G-mixolydian`, etc. KeyParser con scale degrees e note↔MIDI mapping.

10. **Come validi l'output dell'LLM?**
    Validator a 5 livelli: range MIDI per pattern type (bass: 33-55, arp: 48-84, melody: 60-96), appartenenza alla scala, densità, coerenza accordale (chord tone per sezione), soft musical scoring (contour, ripetizione, chiarezza motivica). Errori diventano prompt di correzione per il retry.

11. **Il progetto ha dei test?**
    15 package con `_test.go`, table-driven, AAA pattern. `go test -race` pulito. Coverage target ≥80%. Test specifici: validazione MIDI, parsing chiavi, generazione progressioni, retry LLM, scrittura file. Frontend: `music.test.ts` per note/scale.

12. **Come gestisci la configurazione?**
    Viper: `cadenza.yaml` + env vars (`CADENZA_*` prefix) + flag CLI. Priorità: flag CLI > env vars > config file > defaults. Supporto per `--no-color` e `NO_COLOR`. Temperature, max_retries e timeout LLM configurabili.

13. **Quali sono le performance?**
    Offline mode: <100ms per 3 tracce. LLM mode: 2-8 secondi dipende dal provider. MIDI Type-0, 480 ticks/beat, 120 ticks/step. Frontend Vite build <100ms, bundle JS ~185KB gzippato ~59KB.

14. **Come funziona la cache LLM?**
    SHA-256 di: provider + pattern type + key + mode + seed + prompt hash + planner version + style-card version + critic version. 30 giorni TTL su disco. Se il prompt o lo style profile cambiano, la cache si invalida automaticamente.

15. **Quali sono i piani futuri?**
    Singolo file MIDI Type-1, drum pattern su CH10, A/B comparison nel desktop, progress bar durante la generazione, test automatici in CI, landing page pubblica con API playground, plugin DAW (ricerca in corso).

16. **Chi è l'autore e qual è la licenza?**
    Andrea Cavallo. Open source. Repository: `github.com/Andrea-Cavallo/cadenza`. Il progetto è in sviluppo attivo.

17. **Come si installa e si usa?**
    ```bash
    go install github.com/Andrea-Cavallo/cadenza/cmd/cadenza@latest
    cadenza --bpm 122 --key Am --no-llm
    ```
    Desktop: scaricare lo zip dalla release, eseguire `Cadenza.exe`. Per Claude: `ANTHROPIC_API_KEY=sk-ant-...`.

18. **Perché il nome Cadenza?**
    La cadenza musicale è la progressione armonica che chiude una frase. Il progetto genera progressioni coerenti che "risolvono" musicalmente. Nome breve, memorabile, dominio `.dev` disponibile.
