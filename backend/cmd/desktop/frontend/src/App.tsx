import { ReactNode, useEffect, useState } from 'react'
import { GenerationConsole, Params } from './components/GenerationConsole'
import { PianoRoll } from './components/PianoRoll'
import { Pipeline } from './components/Pipeline'
import { GENRE_PRESETS, PresetGrid } from './components/Presets'
import { Topbar } from './components/Topbar'
import { TweakRadio, TweakSection, TweaksPanel, useTweaks } from './components/TweaksPanel'

function Section({ id, eyebrow, title, sub, children }: {
  id: string
  eyebrow: string
  title: string
  sub: string
  children: ReactNode
}) {
  return (
    <section id={id}>
      <div className="shell">
        <div className="section-head">
          <div>
            <div className="eyebrow" style={{ marginBottom: 14 }}>{eyebrow}</div>
            <h2 dangerouslySetInnerHTML={{ __html: title }} />
          </div>
          <p>{sub}</p>
        </div>
        {children}
      </div>
    </section>
  )
}

export default function App() {
  const [tweaks, setTweak] = useTweaks({ accent: 'lime' })
  const [activePreset, setActivePreset] = useState('progressive-warmup')
  const [params, setParams] = useState<Params>({
    bpm: 122,
    key: 'Am-dorian',
    bars: 16,
    groove: 'mpc60',
    style: 'melodic',
    provider: 'offline',
    drums: false,
  })

  useEffect(() => {
    const map: Record<string, string> = { lime: '#01FF95', cyan: '#00E0FF', amber: '#FFB000' }
    document.documentElement.style.setProperty('--accent', map[String(tweaks.accent)] || map.lime)
  }, [tweaks.accent])

  const onSelectPreset = (id: string) => {
    setActivePreset(id)
    const preset = GENRE_PRESETS.find(x => x.id === id)
    if (preset) {
      setParams(prev => ({
        ...prev,
        bpm: preset.bpm,
        key: preset.key,
        groove: preset.groove,
        style: preset.style,
      }))
    }
  }

  return (
    <>
      <Topbar />

      <section style={{ padding: '70px 0 40px' }}>
        <div className="shell" style={{ marginBottom: 36 }}>
          <div className="eyebrow" style={{ marginBottom: 22 }}>
            MIDI generation engine . desktop app for modern electronic music
          </div>
          <h1 className="hero-title">
            Coherent MIDI<br />for the floor - <em>generated.</em>
          </h1>
          <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 60, marginTop: 36, alignItems: 'end' }}>
            <p style={{ fontSize: 17, lineHeight: 1.55, color: 'var(--fg-dim)', maxWidth: '60ch', margin: 0 }}>
              Cadenza renders <span className="mono" style={{ color: 'var(--fg)' }}>bassline</span>,{' '}
              <span className="mono" style={{ color: 'var(--fg)' }}>arpeggio</span> and{' '}
              <span className="mono" style={{ color: 'var(--fg)' }}>melody</span> tracks
              that share one progression and one seed.
            </p>
            <div className="kv">
              <div className="k">tempo</div><div className="v">80 - 150 bpm</div>
              <div className="k">scales</div><div className="v">major . minor . dorian . phrygian . mixolydian . lydian</div>
              <div className="k">providers</div><div className="v">claude . ollama . openai . gemini . offline</div>
            </div>
          </div>
        </div>
        <div className="shell">
          <div className="roll-wrap">
            <PianoRoll bpm={params.bpm} />
            <div className="roll-overlay">
              <div className="meta">
                <span><span style={{ color: 'var(--fg-muted)' }}>BPM</span> <span className="v">{params.bpm}</span></span>
                <span><span style={{ color: 'var(--fg-muted)' }}>KEY</span> <span className="v">{params.key}</span></span>
                <span><span style={{ color: 'var(--fg-muted)' }}>BARS</span> <span className="v">{params.bars}</span></span>
              </div>
              <div />
              <div className="meta" style={{ justifyContent: 'flex-end' }}>
                <span style={{ color: 'var(--fg-muted)' }}>BASSLINE</span>
                <span style={{ color: 'var(--fg-muted)' }}>ARP</span>
                <span style={{ color: 'var(--accent)' }}>MELODY</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <Section
        id="presets"
        eyebrow="04 genre presets . single click"
        title="Pick a <em>vibe.</em><br/>Skip the configuration."
        sub="Each preset bakes BPM, key, mode, groove, and offline style for a specific moment in a set."
      >
        <PresetGrid active={activePreset} onSelect={onSelectPreset} />
      </Section>

      <Section
        id="generate"
        eyebrow="generation . live controls"
        title="The console <em>is</em> the engine."
        sub="Tweak parameters, press Generate, and three MIDI files land in your local cadenza-output folder."
      >
        <GenerationConsole params={params} setParams={setParams} />
      </Section>

      <Section
        id="pipeline"
        eyebrow="under the hood . 4 stages"
        title="From <em>intent</em> to MIDI."
        sub="Models produce validated musical specs; the deterministic Go renderer turns them into MIDI."
      >
        <Pipeline />
      </Section>

      <TweaksPanel title="Tweaks">
        <TweakSection label="Accent">
          <TweakRadio
            value={String(tweaks.accent)}
            onChange={value => setTweak('accent', value)}
            options={[
              { value: 'lime', label: 'Lime' },
              { value: 'cyan', label: 'Cyan' },
              { value: 'amber', label: 'Amber' },
            ]}
          />
        </TweakSection>
      </TweaksPanel>
    </>
  )
}
