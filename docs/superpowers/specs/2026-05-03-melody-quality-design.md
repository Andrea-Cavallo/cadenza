# Melody Quality — Offline Phrase Builder

**Date:** 2026-05-03  
**Scope:** `backend/internal/generator` only (no renderer, LLM, frontend changes)  
**Status:** Approved

---

## Problem

The offline melody generator produces notes that are structurally valid but musically flat:

1. Step positions within each bar are hardcoded per `chordIdx` case — seed only changes *which* notes, not *where* they fall rhythmically
2. The 4-section narrative arc (statement/call/tension/resolution) exists but has no rhythmic variety between seeds
3. No contour management — `buildHypnoticMotif` can produce consecutive wide leaps in the same direction
4. Melody and arpeggio frequently occupy the same register (octave 5) without separation
5. No measurable quality floor — no way to assert that a generated melody is "musical"

---

## Decision

**Approach: Phrase Builder with two passes + diagnostic scorer**

- Replace `buildMelodyBar` loop with a two-pass phrase builder
- Pass 1 (Contour): decides step positions, roles, and contour shapes seed-driven
- Pass 2 (Fill): resolves concrete notes, octaves, articulation from the contour
- Add `MelodyPhraseScorer` for tests and diagnostics only — no production regen gate
- Register separation via soft octave preference (no cross-pattern dependency)

---

## Architecture

### Flusso

```
melodyTemplate(ctx)
  → buildHypnoticMotif(h, motifLen, scaleNotes)  → motifDegrees[]   [invariato]
  → buildMelodyContour(h, prog, motifDegrees, scaleNotes, key)  → MelodyContour   [NUOVO]
  → buildMelodyFromContour(steps, contour, prog, scaleNotes, key, h)               [NUOVO]
  → applyMelodyOfflineStyle(...)   [invariato]
  → applyPassingNotes(...)         [invariato]
  → ensureMelodyMinDensity(...)    [invariato]
  → ensureGateVariation(...)       [invariato]
```

### File coinvolti

| File | Cambio |
|---|---|
| `generator/offline.go` | aggiunge phrase builder, aggiorna `melodyNote`, depreca `buildMelodyBar` |
| `generator/melody_scorer.go` | **nuovo** — `MelodyPhraseScore`, `ScoreMelodyPhrase` |
| `generator/coverage_test.go` | aggiunge 4 nuovi test |

Tutti gli altri package rimangono invariati.

---

## Sezione 1 — Tipi nuovi

```go
// ContourType descrive la forma melodica di una sezione di 16 step.
type ContourType string

const (
    ContourArch            ContourType = "arch"
    ContourQuestionAnswer  ContourType = "question_answer"
    ContourTensionHold     ContourType = "tension_hold"
    ContourDescRelease     ContourType = "descending_release"
)

// StepIntention descrive il ruolo e la direzione di una singola posizione.
type StepIntention struct {
    Active      bool
    Role        string // "target" | "pickup" | "echo" | "fill"
    Direction   int    // -1 down, 0 stay, +1 up (per contrary motion)
    PreferHigh  bool   // se true, melodyNote preferisce ottava 6 su 5
}

// SectionContour è il contour completo per una sezione di 16 step.
type SectionContour struct {
    Type       ContourType
    Intentions [stepsPerSection]StepIntention
}

// MelodyContour raccoglie i contour delle 4 sezioni del motif.
type MelodyContour struct {
    Sections [4]SectionContour
}
```

---

## Sezione 2 — Contour Pass

### Mapping sezione → ContourType

| Sezione | Ruolo narrativo | Tipo primario | Variante (hash) |
|---|---|---|---|
| 0 — Statement | apertura semplice | `arch` | `question_answer` |
| 1 — Call | sviluppo | `question_answer` | `arch` |
| 2 — Tension | picco denso | `tension_hold` | `arch` (peak) |
| 3 — Resolution | discesa, respiro | `descending_release` | `question_answer` |

Selezione variante:
```go
func chooseSectionContour(chordIdx int, h []byte) ContourType {
    bias := h[(chordIdx*7+3)%32] % 2
    switch chordIdx {
    case 0: return []ContourType{ContourArch, ContourQuestionAnswer}[bias]
    case 1: return []ContourType{ContourQuestionAnswer, ContourArch}[bias]
    case 2: return []ContourType{ContourTensionHold, ContourArch}[bias]
    case 3: return []ContourType{ContourDescRelease, ContourQuestionAnswer}[bias]
    }
    return ContourArch
}
```

### Template ritmici per ContourType (3 varianti per tipo)

Ogni posizione è annotata: `●`=target, `p`=pickup, `g`=ghost/echo, `f`=fill, `.`=rest.

**`arch`** — sale al picco ~step 7-9, scende
```
V0: [●  .  ●  .  ●  .  .  ●  ●  .  ●  .  .  g  .  p]   7 attivi
V1: [●  .  .  ●  .  ●  ●  .  ●  .  .  ●  .  g  .  .]   7 attivi
V2: [●  p  .  .  ●  .  ●  .  ●  .  .  ●  .  .  g  .]   7 attivi
```
Densità: 43% active (57% vuoto)

**`question_answer`** — frase corta, pausa, risposta
```
V0: [●  p  .  ●  .  .  g  .  .  ●  .  ●  .  g  .  p]   7 attivi
V1: [●  .  p  ●  .  g  .  .  .  ●  p  .  ●  .  g  .]   7 attivi
V2: [p  ●  .  .  ●  .  g  .  .  .  ●  .  p  ●  .  .]   6 attivi
```
Densità: 38-44% active

**`tension_hold`** — figura ripetuta, push, risoluzione
```
V0: [●  .  ●  .  ●  g  .  ●  ●  .  ●  .  .  ●  .  .]   8 attivi
V1: [●  .  .  ●  .  ●  .  ●  .  ●  g  .  ●  .  ●  .]   9 attivi
V2: [●  g  .  ●  .  .  ●  ●  .  ●  .  ●  .  .  ●  g]   9 attivi
```
Densità: 50-56% active (sezione di picco, densità maggiore ammessa)

**`descending_release`** — discesa stepwise, molto spazio
```
V0: [●  .  .  ●  .  ●  .  g  .  .  ●  .  g  .  .  .]   5 attivi
V1: [●  .  ●  .  .  ●  .  .  g  .  .  ●  .  g  .  .]   5 attivi
V2: [●  .  .  .  ●  .  ●  .  .  g  .  .  ●  .  .  g]   5 attivi
```
Densità: 31% active (massimo respiro in chiusura)

### Selezione variante ritmica

```go
func chooseRhythmicVariant(h []byte, sectionIdx int) int {
    return int(h[(sectionIdx*11+17)%32]) % 3
}
```

Garantisce variazione ritmica reale tra seed diversi per la stessa sezione.

### Contour guard (in `buildHypnoticMotif`)

Dopo la generazione degli intervalli, prevent di salti consecutivi nella stessa direzione:
```go
// Se due salti consecutivi > 3 semitoni nella stessa direzione,
// il secondo viene invertito per contrary motion.
if direction(prev→curr) == direction(curr→next) && leapSize(curr→next) > 3 {
    next = mirrorInterval(curr, next)
}
```

---

## Sezione 3 — Fill Pass

### Risoluzione nota per Role

```
"target"  → resolveTargetNote(chordTones, scaleNotes, degree, key)
             preferisce character degree → 7th → 3rd → closestChordTone
             ottava: PreferHigh ? 6 : 5, clamped [60,84]

"pickup"  → approachNote(target, key)
             stessa ottava del target
             Ghost=true, Staccato=true

"echo"    → target o ±1 scale degree
             Ghost=true

"fill"    → scala passing tone tra note adiacenti
             Staccato=true
```

### Preferenza target note (3rd/7th/charTone > root)

```go
func resolveTargetNote(chordTones, scaleNotes []string, degree int, key theory.Key) string {
    charDeg := characterDegree(key)
    candidates := []string{
        scaleNotes[normalizeDegree(charDeg, len(scaleNotes))],
        extendedChordTone(chordTones, scaleNotes, 3), // 7th
        chordTones[1%len(chordTones)],                // 3rd
    }
    for _, c := range candidates {
        if withinLeap(c, scaleNotes[degree%len(scaleNotes)], 4) {
            return c
        }
    }
    return closestChordTone(chordTones, scaleNotes, degree)
}
```

`withinLeap` accetta il candidato solo se distanza cromatica ≤ 4 semitoni dal degree motif — evita che la preferenza 7th introduca salti non voluti.

### Articolazione da ContourType

| ContourType | Step | Articolazione |
|---|---|---|
| `arch` | picco (note più alta) | `Accent=true, Legato=true` |
| `arch` | discesa dopo picco | `Staccato=true` |
| `question_answer` | fine domanda | `Staccato=true` |
| `question_answer` | inizio risposta | `Accent=true, Legato=true` |
| `tension_hold` | figura ripetuta | `Legato=true` |
| `tension_hold` | push finale | `Accent=true` |
| `descending_release` | nota d'arrivo (step 0) | `Accent=true, Legato=true` |
| `descending_release` | discesa | `Staccato=true` |

### Register separation (soft)

`melodyNote` aggiornata con parametro `preferHigher bool`:

```go
func melodyNote(noteName, preferredOct string, preferHigher bool) string {
    oct := preferredOct
    if preferHigher && oct == "5" {
        oct = "6"
    }
    note := noteName + oct
    midi, err := theory.NoteToMIDI(note)
    if err != nil { return noteName + preferredOct }
    if midi < 60  { return noteName + "5" }
    if midi > 84  { return noteName + "4" }
    return note
}
```

- Sezioni 0, 1, 2: `preferHigher=true` → melody in ottava 6 (sopra arp ottava 5)
- Sezione 3 (resolution): `preferHigher=false` → discesa in ottava 5/4

### Leap validation (contrary motion)

Prima di scrivere ogni StepSpec, il fill pass verifica:
```go
// Salto > 5 semitoni → la nota successiva attiva deve muoversi per contrary motion.
if prevActive && leapSize(prev, curr) > 5 {
    intentions[nextActiveIdx].Direction = -sign(curr - prev)
}
```
Non blocca il salto — segna un "debito di risoluzione" che il fill pass onora scegliendo la nota successiva per movimento contrario.

---

## Sezione 4 — MelodyPhraseScorer

### Struttura

```go
// In generator/melody_scorer.go

type MelodyPhraseScore struct {
    RestRatio         float64 // % steps inattivi (target ottimale: 0.50-0.70)
    PickupPresence    float64 // % sezioni con ≥1 pickup note
    ContourScore      float64 // % sezioni senza 3 salti consecutivi stesso senso
    ChordToneStrength float64 // % active steps su chord tone o char degree
    RegisterSep       float64 // % steps melody NOT in collisione con arp ottava 5
    PhraseScore       float64 // media ponderata, range 0.0-1.0
}

func ScoreMelodyPhrase(melodySpec *schema.PatternSpec,
                       arpSteps []schema.StepSpec,
                       prog theory.ChordProgression,
                       key theory.Key) MelodyPhraseScore
```

### Pesi PhraseScore

| Metrica | Peso |
|---|---|
| RestRatio in [0.50, 0.70] | 0.20 |
| PickupPresence | 0.20 |
| ContourScore | 0.25 |
| ChordToneStrength | 0.25 |
| RegisterSep | 0.10 |

Soglia minima per i test: **PhraseScore ≥ 0.65**

Uso: solo in test e diagnostica. Non è un gate in produzione — l'utente può sempre rigenerare manualmente.

---

## Sezione 5 — Test Strategy

### Test 1 — Qualità frase (nuovi)

```go
// TestMelodyPhraseQuality: 10 seed × 4 chiavi
// Keys: Am (minor_natural), Dm (dorian), G (mixolydian), E (phrygian)
// Assert: PhraseScore >= 0.65
// Assert: RestRatio in [0.50, 0.75]  -- misurato globalmente su tutti i 64 step,
//         NON per sezione. tension_hold può avere 44-50% rest locale (è il picco),
//         ma la media globale delle 4 sezioni resta >= 0.50.
// Assert: PickupPresence >= 0.50
```

### Test 2 — Contour diversity (nuovo)

```go
// TestMelodyContourDiversity: 10 seed → fingerprint contour
// fingerprint = concatenazione dei 4 ContourType
// Assert: ≥ 3 combinazioni distinte su 10 seed
```

### Test 3 — Leap resolution (nuovo)

```go
// TestMelodyNoUnresolvedLeaps: 20 seed
// Per ogni sezione: nessun salto > 5 semitoni non seguito da contrary motion
```

### Test 4 — Listening fixtures (nuovo)

```go
// TestListeningFixturesMelody
// Genera MIDI per seeds ["fix-1","fix-2","fix-3"] con nuova logica
// Scrive in testdata/listening/melody_after/
// testdata/listening/melody_before/ committato come baseline
// Non fa assertion — produce file per A/B in DAW
```

### Test esistenti che devono continuare a passare

- `TestOfflineSubModesValidateAndShapeDensity` — densità minimal=16, driving arp=64, hypnotic bass=32
- `TestOfflineSeedDiversity` — ≥6 fingerprint su 20 seed
- `TestOfflineKeyDifferentiation` — Am ≠ Dm con stesso seed
- Tutti i test del validator

---

## Invarianti preservate

1. `validator.ValidateWithChords` passa su tutti i 4 stili offline
2. `minimal` melody = 16 active steps (post-processing `applyMelodyOfflineStyle` invariato)
3. `ensureMelodyMinDensity` e `ensureGateVariation` girano dopo il phrase builder
4. `buildMelodyPhraseSection` e `buildMelodyBar` restano compilabili (non rimossi)
5. Range melody: [60, 84] — invariato
6. `melodyTemplate` firma pubblica invariata
7. `OfflineTemplate` firma pubblica invariata

---

## Fuori scope (sprint successivi)

- Aggiornamento piano roll frontend
- Aggiornamento prompt LLM melody
- Aggiornamento critic LLM
- Documentazione SPECS.md musical rules
