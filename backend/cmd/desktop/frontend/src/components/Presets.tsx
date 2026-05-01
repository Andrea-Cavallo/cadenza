export const GENRE_PRESETS = [
  {
    id: 'progressive-warmup',
    num: '01',
    name: 'Progressive',
    italic: 'warmup',
    description: 'Warm melodic build with an emotional Dorian groove for set openers.',
    bpm: 122,
    key: 'Am-dorian',
    groove: 'mpc60',
    style: 'melodic',
  },
  {
    id: 'peak-time-driver',
    num: '02',
    name: 'Peak time',
    italic: 'driver',
    description: 'Dense peak-time energy with full-power Aeolian floor pressure.',
    bpm: 130,
    key: 'Am',
    groove: 'straight',
    style: 'driving',
  },
  {
    id: 'afterhours-hypnotic',
    num: '03',
    name: 'Afterhours',
    italic: 'hypnotic',
    description: 'Deep afterhours textures, sparse motion, and slow-moving Bm focus.',
    bpm: 116,
    key: 'Bm',
    groove: 'humanize',
    style: 'hypnotic',
  },
  {
    id: 'festival-melodic',
    num: '04',
    name: 'Festival',
    italic: 'melodic',
    description: 'Festival euphoria with Mixolydian brightness and anthemic drive.',
    bpm: 126,
    key: 'A-mixolydian',
    groove: 'mpc60',
    style: 'melodic',
  },
]

interface PresetGridProps {
  active: string
  onSelect: (id: string) => void
}

export function PresetGrid({ active, onSelect }: PresetGridProps) {
  return (
    <div className="row" style={{ gridTemplateColumns: 'repeat(4, 1fr)' }}>
      {GENRE_PRESETS.map(p => (
        <button
          type="button"
          key={p.id}
          className={'preset' + (active === p.id ? ' on' : '')}
          onClick={() => onSelect(p.id)}
        >
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
        </button>
      ))}
    </div>
  )
}
