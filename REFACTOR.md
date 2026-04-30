# REFACTOR — Aree di miglioramento residue

Ultimo aggiornamento: 29 aprile 2026 — dopo completamento dei 10 punti LLM pipeline.

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
