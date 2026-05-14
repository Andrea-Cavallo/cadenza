# Session Persistence — Design Spec

**Data:** 2026-05-12  
**Stato:** Approvato per implementazione  
**Autore:** Andrea Cavallo

---

## Obiettivo

Salvare su disco lo stato completo di una sessione Cadenza — history LLM e stato musicale — in modo che una sessione interrotta possa essere ripresa esattamente dove era, senza rielaborare la history dei messaggi né i pattern MIDI già generati.

---

## Motivazione

Cadenza non ha una KV cache LLM esterna, ma produce due artefatti di sessione che vale la pena persistere:

1. **History messaggi LLM** — l'intera conversazione con il provider (Claude, Ollama, offline). Ricaricarla come contesto iniziale elimina la rigenerazione costosa e preserva il filo creativo.
2. **Stato musicale** — progressione armonica, pattern generati, step evolutivo corrente. Ripartire dall'ultimo step evita ripetizioni e mantiene la coerenza armonica cross-sessione.

Il meccanismo è ispirato a `ds4`: header binario fisso + payload JSON, naming basato su SHA1 del contenuto per deduplicare sessioni identiche.

---

## Struttura dati

```go
type SessionState struct {
    Version    uint32    // versione formato file
    SessionID  string    // UUID della sessione
    CreatedAt  time.Time
    UpdatedAt  time.Time
    SaveReason SaveReason

    // Stato LLM
    Messages []Message

    // Stato musicale
    MusicState MusicState
}

type SaveReason uint8

const (
    SaveReasonManual   SaveReason = iota // cadenza --save
    SaveReasonAuto                       // checkpoint periodico
    SaveReasonEvict                      // sessione soppiantata da un'altra
    SaveReasonShutdown                   // Ctrl+C / exit pulito
)

type Message struct {
    Role    string // "user" | "assistant" | "system"
    Content string
}

type MusicState struct {
    Key            string // es. "C minor"
    Scale          string // es. "dorian"
    BPM            int
    TimeSignature  string // es. "4/4"
    Patterns       []Pattern
    HarmonyHistory []ChordProgression
    EvolutionStep  int
}

type Pattern struct {
    ID        string
    StepIndex int // step evolutivo che ha generato questo pattern
    Notes     []Note
    CreatedAt time.Time
}
```

---

## Formato file su disco

Header binario fisso di 32 byte + payload JSON. Ispirato a `ds4`.

```
[HEADER 32 bytes]
  0   u8[3]  magic = "CZS"   (Cadenza Session)
  3   u8     version = 1
  4   u8     save_reason
  5   u8[3]  reserved (zero)
  8   u32    message_count
 12   u32    pattern_count
 16   u64    created_at (unix timestamp)
 24   u64    updated_at (unix timestamp)

[PAYLOAD]
  JSON di SessionState (tutto il resto, leggibile)
```

**Naming del file:** `<sha1-dei-messaggi>.czs`

SHA1 è calcolato sulla sequenza ordinata di `(role, content)` dei messaggi. Due sessioni con la stessa history LLM condividono il medesimo file — nessuna duplicazione su disco.

---

## Interfaccia Go

```go
type SessionStore interface {
    Save(ctx context.Context, state *SessionState, reason SaveReason) error
    Load(ctx context.Context, sessionID string) (*SessionState, error)
    LoadByMessageHash(ctx context.Context, hash string) (*SessionState, error)
    List(ctx context.Context) ([]SessionMeta, error)
    Delete(ctx context.Context, sessionID string) error
    Evict(ctx context.Context, maxSizeMB int) error
}

type SessionMeta struct {
    SessionID    string
    FilePath     string
    SaveReason   SaveReason
    MessageCount int
    PatternCount int
    CreatedAt    time.Time
    UpdatedAt    time.Time
    SizeBytes    int64
}
```

L'implementazione concreta è `FileSessionStore` in `internal/session/`.

---

## Repository layout

```
backend/internal/session/
├── store.go          // interfaccia SessionStore + tipi
├── file_store.go     // implementazione FileSessionStore
├── header.go         // encode/decode header binario 32 byte
├── hash.go           // SHA1 della message history
├── evict.go          // politica di eviction FIFO
└── session_test.go   // unit test
```

---

## Policy di salvataggio

| Evento | SaveReason | Descrizione |
|--------|------------|-------------|
| Ogni N step evolutivi | `Auto` | Checkpoint periodico (default: ogni 3 step) |
| Nuova sessione avviata con sessione attiva | `Evict` | Salva corrente prima di sostituirla |
| `cadenza --resume` | Cold load | Caricamento esplicito da disco, nessun salvataggio |
| Ctrl+C / exit | `Shutdown` | Salvataggio finale prima di uscire |
| `cadenza --save` | `Manual` | Salvataggio esplicito richiesto dall'utente |

---

## Comportamento al resume

1. Carica `SessionState` dal file `.czs` (by session-id o per ultimo `UpdatedAt`)
2. Ricostruisce la history messaggi e la invia all'LLM come contesto iniziale (campo `Messages` nel primo turno)
3. Ricostruisce `MusicState`: key, BPM, scala, progressione armonica
4. Riprende dall'`EvolutionStep` successivo all'ultimo salvato
5. Stampa riepilogo a stdout:

```
Resuming session a3f9b2c1...
  Key: C minor  BPM: 128  Scale: dorian
  Patterns: 6  Evolution step: 4 → continuing from step 5
  Last saved: 2026-05-12 14:22 (Auto checkpoint)
```

---

## CLI integration

```bash
# Sessione normale
cadenza --key "C minor" --bpm 128

# Riprende l'ultima sessione salvata (per UpdatedAt più recente)
cadenza --resume

# Riprende una sessione specifica
cadenza --resume <session-id>

# Lista sessioni disponibili
cadenza --list-sessions

# Salvataggio manuale durante una sessione
cadenza --save

# Configura directory e checkpoint interval
cadenza --session-dir ~/.cadenza/sessions --checkpoint-interval 3
```

---

## Configurazione

```go
type SessionConfig struct {
    Dir                string // default: ~/.cadenza/sessions
    MaxSizeMB          int    // default: 512 MB totali
    CheckpointInterval int    // ogni N step evolutivi, default: 3
    MaxSessions        int    // default: 20; oltre questo evict FIFO
}
```

Le configurazioni seguono la precedenza standard Cadenza: CLI flag > ENV > YAML config file > default.

---

## Algoritmo di eviction

Quando `MaxSessions` è raggiunto o `MaxSizeMB` è superato, `Evict` applica **FIFO su `UpdatedAt`**:

1. Lista tutti i `.czs` in `Dir`
2. Ordina per `UpdatedAt` ascending (più vecchi prima)
3. Elimina i file più vecchi finché `count ≤ MaxSessions` e `totalSize ≤ MaxSizeMB`
4. Log ogni eliminazione con session-id, save-reason originale e size

---

## Vincoli KISS — cosa NON implementare

- **Niente compressione** — payload JSON leggibile è più utile per debug
- **Niente encryption** — le sessioni non contengono segreti utente
- **Niente sync remoto** — filesystem locale è sufficiente
- **Niente merge di sessioni** — ogni `.czs` è uno snapshot indipendente
- **Niente locking distribuito** — Cadenza è single-process; `sync.Mutex` locale basta

---

## Integrazione con la pipeline esistente

```
Generate(BPM, Key, ...)
  → ChordProgression
  → parallel generators
  → Validator
  → Renderer
  → [SessionStore.Save(Auto)]   ← hook dopo ogni N step evolutivi
  → MIDI output files
```

Al resume, i `Messages` ricostruiti vengono passati al provider LLM prima del primo turno della nuova generazione. Il provider li tratta come contesto preesistente — nessun cambio all'interfaccia `LLMProvider`.

---

## Definition of Done

- [ ] `internal/session/` compila con zero errori (`go build ./...`)
- [ ] Header binario encode/decode round-trip testato
- [ ] SHA1 message hash deterministic (stesso input → stesso nome file)
- [ ] `FileSessionStore` implementa tutti i metodi di `SessionStore`
- [ ] `cadenza --resume` riprende correttamente dall'`EvolutionStep` successivo
- [ ] `cadenza --list-sessions` stampa tabella ordinata per `UpdatedAt`
- [ ] Eviction FIFO rispetta `MaxSessions` e `MaxSizeMB`
- [ ] Shutdown hook salva la sessione prima di uscire (Ctrl+C)
- [ ] Test coverage ≥ 80% per `internal/session/`
- [ ] `CHANGELOG.md` aggiornato
- [ ] `CLAUDE.md` aggiornato con il nuovo package `internal/session/`
