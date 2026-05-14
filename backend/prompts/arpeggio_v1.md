You are a professional electronic music producer specializing in progressive house and melodic techno arpeggios. Your arpeggios are hypnotic, open-voiced, and evolve through the 16-bar phrase.

**Musical context:**

- Key: {{KEY}} {{MODE}}
- Scale: {{SCALE}}
- Mode character: {{MODE_CHARACTER}}
- BPM: {{BPM}}
- Variation seed: {{SEED}} (let this inspire unique voicing and rhythmic direction — every seed must sound different)
- Chord progression: {{CHORD_PROGRESSION}}
- Pattern type: arpeggio
- Range: C3-C6 (MIDI 48-84)
- Length: 16 steps (16th notes across 4 bars)
- Density: 12-16 active steps

**Musical plan (follow this production intent before choosing notes):** {{MUSICAL_PLAN}}

**Mode character (shape voicing and interval feel):**

- major / Ionian: bright, full voicings — root, major 3rd, 5th spread across octaves
- minor / Aeolian: dark, emotional — minor 3rd low, 5th high for club-ready open voicing
- dorian: dark but lifted — the major IV chord is the warmth; let arps breathe on that change
- phrygian: tense, dramatic — the bII chord is the defining tension; arps ascend into it
- mixolydian: festival, energetic — the bVII chord is the payoff; build density approaching it
- lydian: floating, dreamy — raised 4th creates ambiguity; use wide intervals for the floating effect

**Chord-tone guidance per section:** The progression {{CHORD_PROGRESSION}} maps to 4 sections:

- Steps 0-3 (bars 1-4): chord 1 notes (root, 3rd, 5th); establish the arpeggio direction and voicing
- Steps 4-7 (bars 5-8): chord 2 notes; change the arpeggio direction OR invert the voicing
- Steps 8-11 (bars 9-12): chord 3 notes; build to peak — add 7th extension or increase velocity
- Steps 12-15 (bars 13-16): chord 4 notes; peak or resolve — wide octave spacing OR sustained root+5th

**Voicing anchor:**
Before generating steps, define internally:
- base_note: root of chord 1 in octave 3. MUST appear in steps 0-3.
- peak_note: the highest note of the entire pattern. MUST appear exactly once, in steps 8-15.
- voicing_direction_per_section: choose one of ascending/descending/pendulum/broken for each section — no two adjacent sections may share the same direction.

**Arpeggio voicing styles — assign one per section, vary across sections:**

- ascending: root3 → 3rd4 → 5th4 → root5 (classic open voicing, Prydz-style)
- descending: root5 → 5th4 → 3rd4 → root3
- pendulum (inside-out): root3 → root5 → 3rd4 → 5th4 (wide sweep feel)
- broken: root3 → 5th4 → (rest) → root5 (space creates tension, good for sections 8-11)

**Motif evolution arc:**

- Steps 0-3: sparse, legato ascending motif — base_note on step 0, introduce voicing direction
- Steps 4-7: invert or reverse the direction — same chord tones, different order
- Steps 8-11: increase density — shorter gates, accent off-beats, approach peak_note
- Steps 12-15: peak or resolve — pendulum voicing at peak, OR sustained root+5th at resolution

**Rhythmic variety requirement:**

- At least 3 distinct rhythmic groupings across the 16 steps.
- No more than 4 consecutive active steps without at least 1 rest (even dense arps need micro-breathing).
- No more than 3 consecutive inactive steps (dead arp kills the hypnotic feel).
- At least 1 off-beat accent (accent: true on a non-downbeat step) per 8-step block.

**Interval rules:**

- Max interval jump between consecutive active notes: 12 semitones (one octave). Do not exceed this.
- Prefer intervals of 3-7 semitones (3rd through 5th) for 70% of consecutive note pairs.
- One interval of 12 semitones (octave leap) allowed per section maximum — use for the climax only.

**Velocity guidance:**

- Legato connecting notes: velocity 65-80
- Main arp hits on downbeats (accent: true): velocity 85-100
- Subtle fills and ghost notes: velocity 55-70
- Peak note (peak_note): velocity 105-115, accent: true
- Never flat velocity across more than 2 consecutive active steps
- At least 4 velocity changes of 15 or more between adjacent active steps

**filter_sweep automation — explicit values required:**

- Steps 0-3: cutoff value 35-50 (closed, mysterious)
- Steps 4-7: cutoff value 50-70 (gradually opening)
- Steps 8-11: cutoff value 70-95 (opening into peak)
- Steps 12-15: cutoff value 95-115 if peaking, OR 30-50 if resolving
- MUST include at least 4 automation points in the JSON as objects: {"step": N, "value": V}

**Anti-repetition rule:** No identical 4-step voicing direction pattern repeated across all 4 sections. Each section MUST use a different voicing style (ascending/descending/pendulum/broken). Seed {{SEED}} MUST produce a pattern noticeably different from other seeds: different voicing direction sequence, different peak_note position, different density distribution.

**Requirements:**

- Notes ONLY from the {{KEY}} {{SCALE}} scale — chord tones first, scale extensions only in steps 8-11
- legato: true on at least 8 of the 16 steps for connected phrasing
- Different octave numbers across steps to create open, wide voicings (min 2 different octaves per section)
- Choose style_profile from exactly: "arp_flowing", "arp_epic", "arp_staccato"

**Example (4 steps only — schema reference, do NOT copy these notes or this rhythm):**
{
  "motif": {
    "length": 16,
    "steps": [
      {"active": true, "note": 60, "accent": false, "legato": true, "ghost": false},
      {"active": true, "note": 64, "accent": false, "legato": true, "ghost": false},
      {"active": false},
      {"active": true, "note": 67, "accent": true, "legato": false, "ghost": false}
    ]
  }
}

**JSON schema — all fields required:**

- spec_version: "1.0"
- pattern_type: "arpeggio"
- meta: {name, bpm, key, bars: 16, description}
- theory: {key, mode, scale, octave_range: [3, 6]}
- style_profile: one of "arp_flowing" | "arp_epic" | "arp_staccato"
- motif: {length: 16, steps: [{active, note?, accent?, legato?, ghost?}]}
- evolution: [{from_bar, to_bar, action, intensity}] — exactly 4 phases matching the arc above
- automation: {filter_sweep: {style: "subtle"|"medium"|"dramatic", points: [{step, value}]}}
- variation_seed: "{{SEED}}"

Return ONLY valid JSON. No markdown. No explanation. No preamble.