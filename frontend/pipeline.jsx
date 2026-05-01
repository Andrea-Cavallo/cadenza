// Pipeline diagram: LLM → spec → render → MIDI

const PIPE = [
  {
    n: "01 / INTENT",
    title: "Provider",
    body: "Claude, Ollama, OpenAI, or Gemini receive the musical context — bpm, key, bars, groove — and reason about a coherent pattern.",
    glyph: "claude · ollama · openai · gemini",
  },
  {
    n: "02 / SPEC",
    title: "PatternSpec",
    body: "A schema-validated YAML describing chord motion, rhythmic density, motif arcs. Re-render anytime with --from-spec.",
    glyph: "spec.bassline.yaml  ·  spec.arp.yaml  ·  spec.melody.yaml",
  },
  {
    n: "03 / RENDER",
    title: "Style profile",
    body: "Specs are bound to a style profile (mpc60, linndrum, humanize) and rendered into deterministic MIDI events.",
    glyph: "groove · velocity · swing · timing",
  },
  {
    n: "04 / MIDI",
    title: "Tracks out",
    body: "Three coherent tracks share one progression and seed — drop straight into Ableton, Logic, or Bitwig.",
    glyph: "bassline.mid  ·  arpeggio.mid  ·  melody.mid",
  },
];

function Pipeline() {
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
  );
}

window.Pipeline = Pipeline;
