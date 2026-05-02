// Genre presets — mirrors cmd/cadenza/cli.go genrePresets exactly.

const GENRE_PRESETS = [
  {
    id: "progressive-warmup",
    num: "01",
    name: "Progressive",
    italic: "warmup",
    description: "Warm melodic build — emotional Dorian groove for set openers.",
    bpm: 122, key: "Am-dorian", groove: "mpc60", style: "melodic",
  },
  {
    id: "peak-time-driver",
    num: "02",
    name: "Peak time",
    italic: "driver",
    description: "Dense peak-time energy — full-power Aeolian floor filler.",
    bpm: 130, key: "Am", groove: "straight", style: "driving",
  },
  {
    id: "afterhours-hypnotic",
    num: "03",
    name: "Afterhours",
    italic: "hypnotic",
    description: "Deep afterhours — sparse, meditative, slow-moving Bm textures.",
    bpm: 116, key: "Bm", groove: "humanize", style: "hypnotic",
  },
  {
    id: "festival-melodic",
    num: "04",
    name: "Festival",
    italic: "melodic",
    description: "Festival euphoria — Mixolydian brightness with anthemic drive.",
    bpm: 126, key: "A-mixolydian", groove: "mpc60", style: "melodic",
  },
];

function PresetGrid({ active, onSelect }) {
  return (
    <div className="row" style={{ gridTemplateColumns: "repeat(4, 1fr)" }}>
      {GENRE_PRESETS.map(p => (
        <div key={p.id}
             className={"preset" + (active === p.id ? " on" : "")}
             onClick={() => onSelect(p.id)}>
          <div>
            <div className="num">[{p.num}]</div>
            <h3>{p.name} <em>{p.italic}</em></h3>
            <p>{p.description}</p>
          </div>
          <div className="params">
            <span><strong>{p.bpm}</strong> bpm</span>
            <span><strong>{p.key}</strong></span>
            <span><strong>{p.groove}</strong></span>
            <span><strong>{p.style}</strong></span>
          </div>
        </div>
      ))}
    </div>
  );
}

window.GENRE_PRESETS = GENRE_PRESETS;
window.PresetGrid = PresetGrid;
