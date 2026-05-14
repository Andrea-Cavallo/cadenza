# Bassline Specialization + New Tracks — Design Spec

**Data:** 2026-05-10  
**Stato:** Approvato per implementazione  
**Autore:** Brainstorming session

---

## Obiettivo

Aggiungere 3 stili di bassline (Groove / Rolling / Sub) sempre generati in parallelo, più 2 nuovi layer (Chord Pad, Lead/Pluck), con un sistema di coerenza adattiva (`StyleFamily`) che lega tutti i layer in un unico bundle musicalmente coerente di 7 stem MIDI.

---

## Output Bundle

7 stem MIDI su canali separati, generati ad ogni `Generate()`:

| File | Canale | Layer | Note |
|------|--------|-------|------|
| `bass-groove.mid` | CH1 | Bassline Groove | comportamento attuale raffinato |
| `bass-rolling.mid` | CH2 | Bassline Rolling | acid/TB-303, 14-16 attivi/16 |
| `bass-sub.mid` | CH3 | Bassline Sub | deep/dub, 4-6 attivi/16, ottava 1 |
| `arp.mid` | CH4 | Arpeggio | adattato al StyleFamily (coeff. 70%) |
| `melody.mid` | CH5 | Melody | lievemente adattato (coeff. 30%) |
| `chord-pad.mid` | CH6 | Chord Pad | nuovo, coeff. 100% |
| `lead-pluck.mid` | CH7 | Lead/Pluck Stab | nuovo, coeff. 80% |

L'utente importa nel DAW: 1 delle 3 bass (tab scelto) + arp + melody + chord-pad + lead. Tutti condividono la stessa `ChordProgression`, lo stesso `seed` e lo stesso `StyleFamily`.

---

## Architettura: StyleFamily

`StyleFamily` è un enum globale `groove | rolling | sub` che sostituisce l'attuale "Offline flavor" (Melodic / Hypnotic / Driving / Minimal).

**Scope:** StyleFamily si applica **solo in modalità offline** (`provider = "offline"`). Con provider LLM (Claude, Ollama, OpenAI, Gemini) la generazione usa i prompt esistenti e non è influenzata da StyleFamily.

### Flusso di generazione

```
Generate(BPM, Key, StyleFamily, Seed)
  → ChordProgression (condivisa da tutti i layer)
  → parallel:
      basslineGroove(ctx)   → bass-groove.mid  CH1
      basslineRolling(ctx)  → bass-rolling.mid CH2
      basslineSub(ctx)      → bass-sub.mid     CH3
      arpeggio(ctx, SF)     → arp.mid          CH4
      melody(ctx, SF)       → melody.mid       CH5
      chordPad(ctx, SF)     → chord-pad.mid    CH6
      leadPluck(ctx, SF)    → lead-pluck.mid   CH7
```

Tutto gira in parallelo — nessun overhead rispetto all'attuale generazione a 3 tracce.

### Coefficienti di adattamento per layer

| Layer | Coeff. | Cosa adatta |
|-------|--------|-------------|
| Bass ×3 | 100% | pattern sub diversi per stile |
| Arpeggio | 70% | densità e articolazione |
| Chord Pad | 100% | voicing, densità, ottava |
| Lead/Pluck | 80% | gate length, densità, velocity |
| Melody | 30% | density ±1-2 note, articolazione ghost |

---

## Specifiche Layer

### Bassline Groove (CH1)

Comportamento attuale mantenuto. Sub-pattern esistenti 0–6, densità 8–11/16, palette `chooseBassGroovePalette`. Nessuna modifica ai pattern.

### Bassline Rolling (CH2)

**Target densità:** 14–16 attivi/16 — zero rest.  
**Groove:** variazione di velocity, non silenzi.  
**Slides:** ≥4 per 16 step.  
**Ottava:** root+"2" (stesso del groove).

**5 nuovi sub-pattern (R0–R4):**

| # | Nome | Step 0 | Step 1 | Step 2 | Step 3 |
|---|------|--------|--------|--------|--------|
| R0 | acid pulse | root acc+leg | root ghost+stacc | fifth slide | root ghost+stacc |
| R1 | rolling ghost | root acc | root ghost+stacc | approach ghost+slide | root slide | approach = `approachNote()` = nota diatonica sotto la root |
| R2 | chromatic fill | root acc+leg | semitono_sotto ghost+stacc | root | fifth ghost+slide |
| R3 | octave stream | root acc+leg | highRoot ghost+stacc | root ghost | root slide |
| R4 | double root | root acc | root | root ghost | fifth slide |

**Schema velocity rolling:**
- accent: 108–115
- hit principale: 72–85
- ghost: 35–50
- ghost2: 38–46

**Regola cromatica (R2):** il semitono sotto la root è l'unico cromatismo ammesso. Massimo 2 per sezione di 16 step, mai su step 0 (downbeat). Eccezione esplicita nel validator.

**Override density bounds:**
- `ensureBassMinDensity`: target ≥14/16 (non 8)
- `ensureBassMaxDensity`: limite 16/16 (nessun ceiling)

### Bassline Sub (CH3)

**Target densità:** 4–6 attivi/16 — il silenzio è parte del suono.  
**Ottava:** root+"1" (un'ottava sotto il groove), cap MIDI ≤ 43 (G2).  
**Articolazione:** legato su tutto, gate ≥80%.  
**Slides:** 0–1 per 16 step.

**4 nuovi sub-pattern (S0–S3):**

| # | Nome | Step 0 | Step 1 | Step 2 | Step 3 |
|---|------|--------|--------|--------|--------|
| S0 | long root | root acc+leg | rest | rest | rest |
| S1 | held fifth | root leg | rest | fifth leg | rest |
| S2 | sub pulse | root acc+leg | rest | rest | root ghost |
| S3 | dub slide | root acc+leg | rest | root ghost+stacc | fifth slide |

**Override density bounds:**
- `ensureBassMaxDensity`: target ≤6/16
- `ensureBassMinDensity`: target ≥4/16

### Arpeggio (CH4) — adattamento 70%

| StyleFamily | Densità | Pattern preferiti | Articolazione |
|-------------|---------|-------------------|---------------|
| Groove | 48–56/64 (attuale) | tutti e 6 | mix legato/staccato |
| Rolling | 60–64/64 | pattern 0, 3, 4 (ascendenti, pendulum) | staccato dominante, slide ogni 4 |
| Sub | 32–40/64 | pattern 1, 5 (discendenti, climbing) | legato su tutto, ghost rimossi |

### Melody (CH5) — adattamento 30%

| StyleFamily | Variazione densità | Variazione articolazione |
|-------------|-------------------|--------------------------|
| Groove | nessuna (attuale) | nessuna |
| Rolling | +1–2 attivi/sezione | accent più forti, staccato in più |
| Sub | −1 attivo/sezione | ghost rimossi, solo note principali |

### Chord Pad (CH6) — nuovo, adattamento 100%

**Ruolo:** voicing sustain dell'accordo. Collante armonico tra arp e bass.  
**Voicing base:** drop-2 su 3 voci — root bassa, quinta centro, terza alta.  
**Range MIDI:** 48–79 (C3–G5).  
**Velocity:** accent 78, hit 55–72, nessun ghost.

| StyleFamily | Densità | Pattern | Voicing |
|-------------|---------|---------|---------|
| Groove | 4 attivi/16 (downbeat) | legato, gate 90% | root3, quinta4, terza5 |
| Rolling | 2 attivi/16 (beat 1 e 9) | legato lungo | root3, quinta4 (drop, senza terza) |
| Sub | 6 attivi/16 (downbeat + off-beat) | legato heavy, gate 100% | root2, quinta3, terza4 (ottava bassa) |

La settima/nona entra solo nel bar di tensione (sezione 3 della progressione).

### Lead / Pluck Stab (CH7) — nuovo, adattamento 80%

**Ruolo:** stesso hook della melody, attacchi brevi e percussivi. Per drop e breakdown.  
**Implementazione:** wrapper su `melodyTemplate` con override totale dell'articolazione.

**Override fissi:**
- tutte le articolazioni → `staccato: true`
- tutti i `legato` → rimossi
- tutti i `ghost` → rimossi (`ghost: false`)
- gate length: 15–30%
- velocity: melody × 0.85 (Groove), melody × 0.95 (Rolling), melody × 0.70 (Sub)

| StyleFamily | Attivi/16 | Gate | Carattere |
|-------------|-----------|------|-----------|
| Groove | 4–6 | 25% | staccato morbido, qualche accent |
| Rolling | 6–8 | 15% | percussivo, accent forti |
| Sub | 1–3 | 30% | quasi assente |

**Range MIDI:** 60–96 (uguale melody).

---

## UI

### Sidebar — StyleFamily selector

Il selector "Offline flavor" viene sostituito da 3 pill quando provider = `offline`:

```
Style
[ Groove ]  [ Rolling ]  [ Sub ]
```

### Pannello bassline — 3 tab

Il piano roll della bassline mostra 3 tab:

```
BASSLINE
┌─────────────┬──────────────┬───────────┐
│   Groove    │   Rolling    │    Sub    │
└─────────────┴──────────────┴───────────┘
[Piano roll del tab selezionato]
```

Il tab attivo evidenzia il file corrispondente con bordo accent nei download.

### Download stem

```
[bass-groove]  [bass-rolling]  [bass-sub]
[arp]  [melody]  [chord-pad]  [lead]
```

### Quick actions (regen singola traccia)

```
[ Same harmony ]
[ Bass ]  [ Arp ]  [ Mel ]  [ Pad ]  [ Lead ]  [ A/B ]
```

`Bass` rigenera tutte e 3 le varianti insieme (stesso seed). Gli altri rigenerano solo quel layer.

---

## Validator — Nuove regole

1. **Eccezione cromatica rolling:** per `pattern_type = "bassline"` con `style_profile = "bass_rolling"`, ammettere note a ±1 semitono dalla scala. Massimo 2 per sezione di 16 step, mai su downbeat.
2. **Density bounds per-stile:** il validator usa bounds diversi per groove (8–13), rolling (14–16), sub (4–6) invece di bounds fissi.
3. **Chord pad:** nuovo `pattern_type = "chord_pad"`, range 48–79, densità 2–6/16, no density check stringente (è sustain).
4. **Lead:** `pattern_type = "lead_stab"`, range 60–96, densità 1–8/16, tutti i passi devono avere `staccato: true`.

---

## Checklist di Implementazione

### Fase 1 — Backend: nuovi pattern bassline

- [x] Aggiungere `StyleFamily` type in `generator/` (`"groove" | "rolling" | "sub"`)
- [x] Aggiungere campo `StyleFamily string` a `MusicContext`
- [x] Implementare sub-pattern R0–R4 in `fillBassSubPattern` (nuovo `case` nello switch)
- [x] Implementare sub-pattern S0–S3 in `fillBassSubPattern`
- [x] Implementare `basslineRolling(ctx)` — usa R0-R4, ottava "2", density ≥14
- [x] Implementare `basslineSub(ctx)` — usa S0-S3, ottava "1", density 4-6
- [x] Parametrizzare `ensureBassMinDensity` e `ensureBassMaxDensity` per stile
- [x] Aggiungere nuovi style profile `bass_rolling` e `bass_sub` nel renderer/styleprofile
- [x] Verificare build: `go build ./...` e `go vet ./...`

### Fase 2 — Backend: validator aggiornato

- [x] Aggiungere eccezione cromatica nel validator per `bass_rolling` (±1 semitono, max 2/sezione, no downbeat)
- [x] Aggiungere density bounds per-stile nel validator (groove 8-13, rolling 14-16, sub 4-6)
- [x] Aggiungere regole validator per `chord_pad` (range 48-79, density 2-6)
- [x] Aggiungere regole validator per `lead_stab` (range 60-96, staccato obbligatorio)
- [x] Test: `go test ./internal/schema/...`

### Fase 3 — Backend: nuovi template (Chord Pad, Lead)

- [x] Implementare `chordPadTemplate(ctx, sf StyleFamily)` in `offline_pad_lead.go`
  - [x] Drop-2 voicing builder (root3, quinta4, terza5)
  - [x] Adattamento Groove: 4 attivi downbeat, legato gate 90%
  - [x] Adattamento Rolling: 2 attivi (beat 1 e 9), legato lungo
  - [x] Adattamento Sub: 6 attivi + ottava bassa (root2, quinta3, terza4)
- [x] Implementare `leadPluckTemplate(ctx, sf StyleFamily)` in `offline_pad_lead.go`
  - [x] Wrapper su `melodyTemplate` con override articolazione
  - [x] Override: staccato=true, legato=false, ghost=false su tutti gli step
  - [x] Adattamento density e velocity per StyleFamily
- [x] Aggiungere `chordPadTemplate` e `leadPluckTemplate` a `offlineTemplate` switch
- [x] Verificare build: `go build ./...`

### Fase 4 — Backend: arpeggio e melody adattivi

- [x] Aggiungere parametro `StyleFamily` a `arpeggioTemplate` (via `styleFamilyOfflineStyle`)
- [x] Implementare adattamento arp Rolling: stile "driving" (alta densità)
- [x] Implementare adattamento arp Sub: stile "hypnotic" (bassa densità)
- [x] Aggiungere parametro `StyleFamily` a `melodyTemplate` (via `styleFamilyOfflineStyle`)
- [x] Implementare adattamento melody Rolling: stile "driving"
- [x] Implementare adattamento melody Sub: stile "hypnotic"
- [x] Test: `go test ./internal/generator/...`

### Fase 5 — Backend: generazione parallela 7 stem

- [x] Aggiornare `multi_generator.go` per generare 7 pattern in parallelo
  - [x] `basslineGroove`, `basslineRolling`, `basslineSub` in goroutine separate
  - [x] `chordPad` e `leadPluck` aggiunti al pool parallelo
- [x] Aggiornare naming file output via `renderAndSaveNamed`: `output_bassline-groove_*.mid`, `output_bassline-rolling_*.mid`, `output_bassline-sub_*.mid`, `output_arp_*.mid`, `output_melody_*.mid`, `output_chord-pad_*.mid`, `output_lead_*.mid`
- [x] Aggiornare Wails AppService in `cmd/desktop/app.go` — passa `StyleFamily` a `MusicContext`
- [x] Aggiornare `types.go` — aggiunto campo `StyleFamily` a `GenerateRequest`
- [x] Aggiornare `normalizeGenerateRequest` — default `StyleFamily = "groove"`
- [x] Verificare build: `GOOS=linux go build ./...`

### Fase 6 — Frontend: Sidebar

- [x] Rimuovere selector "Offline flavor" (Melodic/Hypnotic/Driving/Minimal) — sostituito con pills
- [x] Aggiungere 3 pill StyleFamily `[ Groove ] [ Rolling ] [ Sub ]` per provider=offline
- [x] Aggiornare tipo `Params` in `types.ts` — aggiunto `styleFamily: 'groove' | 'rolling' | 'sub'`
- [x] Aggiornare `GenerateRequest` in `types.ts` — aggiunto `styleFamily: string`
- [x] Aggiungere `Pad` e `Lead` ai pulsanti quick-action regen in Sidebar
- [x] `handleRegenTrack` già supporta stringhe arbitrarie — `'chord_pad'` e `'lead_stab'` funzionano

### Fase 7 — Frontend: PianoRoll e download

- [x] PianoRoll mostra tutti i pattern come tab — bass groove/rolling/sub appaiono automaticamente
- [x] `buildTrackPreviews` in `preview.go` usa ordine canonico a 7 tracce
- [x] `previewLabel` restituisce etichette umane per tutti i 7 tipi (Bass Groove, Bass Rolling, ecc.)
- [x] Download panel usa 2-column grid per ≥5 stem con etichette e click-to-open-folder
- [x] `models.ts` aggiornato con campo `styleFamily` in `GenerateRequest`

### Fase 8 — Qualità e documentazione

- [x] `go test ./...` — tutti i test passano
- [x] `golangci-lint run ./...` — zero Error
- [x] Aggiornare `CHANGELOG.md` con le nuove feature

---

## Dipendenze tra fasi

```
Fase 1 → Fase 2 → Fase 3 → Fase 4 → Fase 5 → Fase 6 → Fase 7 → Fase 8
              ↑                              ↑
         (validator)                   (AppService)
```

Fase 1–4 sono solo backend e possono essere verificate con `go test` prima di toccare il frontend. Fase 5 è il punto di integrazione critico.

---

## Invarianti da non rompere

1. **Step 0 sempre root + accent** — in tutti e 3 gli stili di bass, mai ghost sul downbeat
2. **Slide mai su step 0** — invariante già presente, da rispettare in rolling
3. **MIDI cap sub ≤ 43** — C1 è il limite assoluto (MIDI 24), G2 (43) è il ceiling pratico
4. **Chord coherence validator** — chord pad e lead devono passare il check dei chord tone (≥80%)
5. **Velocity max 120** — mai 127, clippa
6. **Bass rolling cromatismo** — mai su downbeat, max 2 per sezione
7. **StyleFamily non cambia la ChordProgression** — l'armonia è sempre la stessa
8. **Seed deterministico** — stesso seed + stessa key + stesso BPM → stesso output sempre
