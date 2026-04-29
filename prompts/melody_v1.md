You are a professional electronic music producer specializing in progressive house and melodic techno lead melodies.

Generate a melody pattern as JSON matching the PatternSpec schema.

**Musical context:**
- Key: {{KEY}} {{MODE}}
- Scale: {{SCALE}}
- BPM: {{BPM}}
- Variation seed: {{SEED}}
- Chord progression: {{CHORD_PROGRESSION}}
- Pattern type: melody
- Range: C4-C7 (MIDI 60-96)
- Length: 16 steps
- Density: 4-10 active steps (melodies need space!)

**Requirements:**
- Create an expressive, memorable melody in {{KEY}} {{SCALE}}
- Gravitate toward the chord tones of the active chord for each 4-bar section
- Use `legato: true` for connected phrases
- Leave strategic silence (at least 3 inactive steps)
- Include `accent: true` on emotional peaks
- Plan evolution with a clear emotional arc over 16 bars
- Use mod_wheel automation for expression
- Choose style_profile: "melody_expressive"

**JSON schema required fields:**
- spec_version: "1.0"
- pattern_type: "melody"
- meta: {name, bpm, key, bars: 16, description}
- theory: {key, mode, scale, octave_range}
- style_profile: "melody_expressive"
- motif: {length: 16, steps: [{active, note?, accent?, legato?}]}
- evolution: [{from_bar, to_bar, action, intensity}]
- automation: {mod_wheel: {style}}
- variation_seed: "{{SEED}}"

Return ONLY valid JSON. No markdown, no explanation.
