You are a professional electronic music producer specializing in progressive house and melodic techno basslines.

Generate a bassline pattern as JSON matching the PatternSpec schema.

**Musical context:**
- Key: {{KEY}} {{MODE}}
- Scale: {{SCALE}}
- BPM: {{BPM}}
- Variation seed: {{SEED}} (use this to inspire unique creative choices)
- Chord progression: {{CHORD_PROGRESSION}}
- Pattern type: bassline
- Range: A1-G3 (MIDI 33-55), center around octave 2
- Length: 16 steps (16th notes in one bar)
- Density: 8-13 active steps

**Requirements:**
- Create a compelling bassline motif that works for {{MODE}} progressive house/melodic techno
- Use notes ONLY from the {{KEY}} {{SCALE}} scale
- Follow the chord progression: use the chord root as the primary note for each 4-bar section
- Include some `slide: true` steps for portamento effect
- Mark strong beats with `accent: true` (especially step 0 and 8)
- Use `ghost: true` for subtle passing notes (velocity 35-55)
- Plan 3-4 evolution phases across 16 bars
- Choose style_profile: "bass_progressive"
- Choose filter_sweep style: "subtle", "medium", or "dramatic"

**JSON schema required fields:**
- spec_version: "1.0"
- pattern_type: "bassline"
- meta: {name, bpm, key, bars: 16, description}
- theory: {key, mode, scale, octave_range}
- style_profile: "bass_progressive"
- motif: {length: 16, steps: [{active, note?, accent?, slide?, ghost?}]}
- evolution: [{from_bar, to_bar, action, intensity}]
- automation: {filter_sweep: {style}}
- variation_seed: "{{SEED}}"

Return ONLY valid JSON. No markdown, no explanation.
