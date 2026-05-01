const PIPE = [
  {
    n: '01 / INTENT',
    title: 'Provider',
    body: 'Claude, Ollama, OpenAI, or Gemini receive bpm, key, bars, and groove, then reason about a coherent pattern.',
    glyph: 'claude . ollama . openai . gemini',
  },
  {
    n: '02 / SPEC',
    title: 'PatternSpec',
    body: 'A schema-validated musical plan describes chord motion, rhythmic density, motif arcs, and track roles.',
    glyph: 'bassline . arpeggio . melody',
  },
  {
    n: '03 / RENDER',
    title: 'Style profile',
    body: 'Specs are bound to deterministic timing, velocity, gate, sweep, and evolution rules.',
    glyph: 'groove . velocity . sweep . timing',
  },
  {
    n: '04 / MIDI',
    title: 'Tracks out',
    body: 'Three coherent tracks share one progression and seed, ready for Ableton, Logic, Bitwig, or your DAW of choice.',
    glyph: 'bassline.mid . arpeggio.mid . melody.mid',
  },
]

export function Pipeline() {
  return (
    <div className="pipe-row">
      {PIPE.map((s, i) => (
        <div key={i} className="pipe-step">
          <div className="n">{s.n}</div>
          <h4>{s.title}</h4>
          <p>{s.body}</p>
          <div className="glyph">{s.glyph}</div>
        </div>
      ))}
    </div>
  )
}
