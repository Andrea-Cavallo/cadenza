import { useEffect, useMemo, useRef, useState } from 'react'
import { GenerationPreview, TrackPreview } from '../types'

const PITCH_MIN = 33
const PITCH_MAX = 96
const PITCH_RANGE = PITCH_MAX - PITCH_MIN

interface PianoRollProps {
  width?: number
  height?: number
  bpm?: number
  playing?: boolean
  preview?: GenerationPreview | null
  kept?: boolean
  onKeep?: () => void
  onDiscard?: () => void
  onExportAll?: () => void
}

export function PianoRoll({
  width = 1480,
  height = 420,
  bpm = 122,
  playing = false,
  preview = null,
  kept = false,
  onKeep,
  onDiscard,
  onExportAll,
}: PianoRollProps) {
  const [activeType, setActiveType] = useState('bassline')
  const [t, setT] = useState(0)
  const startRef = useRef<number | null>(null)
  const rafRef = useRef<number | null>(null)

  const activeTrack = useMemo(() => {
    if (!preview?.patterns.length) return null
    return preview.patterns.find(pattern => pattern.patternType === activeType) ?? preview.patterns[0]
  }, [activeType, preview])

  useEffect(() => {
    if (!preview?.patterns.some(pattern => pattern.patternType === activeType)) {
      setActiveType(preview?.patterns[0]?.patternType ?? 'bassline')
    }
  }, [activeType, preview])

  const totalSteps = Math.max(activeTrack?.steps.length ?? preview?.stepsPerBar ?? 16, 16)
  const cycleSec = (60 / bpm) * 4

  useEffect(() => {
    if (!playing) {
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current)
      setT(0)
      startRef.current = null
      return
    }
    const tick = (now: number) => {
      if (startRef.current == null) startRef.current = now
      const elapsed = (now - startRef.current) / 1000
      setT((elapsed % cycleSec) / cycleSec)
      rafRef.current = requestAnimationFrame(tick)
    }
    rafRef.current = requestAnimationFrame(tick)
    return () => { if (rafRef.current != null) cancelAnimationFrame(rafRef.current) }
  }, [cycleSec, playing])

  if (!preview || preview.patterns.length === 0) {
    return (
      <div className="real-roll empty-preview">
        <div className="empty-preview-inner">
          <div className="empty-preview-title">No MIDI generated yet</div>
          <div className="empty-preview-copy">Generate a session to inspect Bass, Arp, and Melody notes.</div>
        </div>
      </div>
    )
  }

  const W = width
  const H = height
  const padTop = 58
  const padBot = 34
  const innerH = H - padTop - padBot
  const stepW = W / totalSteps
  const noteH = innerH / PITCH_RANGE
  const playheadX = t * W

  const grids = Array.from({ length: totalSteps + 1 }, (_, step) => ({
    x: step * stepW,
    isBeat: step % 4 === 0,
    isBar: step % (preview.stepsPerBar || 16) === 0,
  }))

  const octaves = []
  for (let p = 36; p <= 96; p += 12) {
    octaves.push(padTop + (PITCH_MAX - p) * noteH)
  }

  const notes = buildRollNotes(activeTrack, stepW, noteH, padTop, playheadX)

  return (
    <div className="real-roll">
      <div className="roll-chords">
        {preview.chords.length === 0 ? (
          <span className="chord-pill muted">No chord progression</span>
        ) : preview.chords.map(chord => (
          <span key={`${chord.name}-${chord.fromBar}-${chord.toBar}`} className="chord-pill">
            <strong>{chord.name}</strong>
            <span>bars {chord.fromBar}-{chord.toBar}</span>
          </span>
        ))}
      </div>

      <div className="roll-tabs">
        {preview.patterns.map(pattern => (
          <button
            key={pattern.patternType}
            type="button"
            className={pattern.patternType === activeTrack?.patternType ? 'on' : ''}
            onClick={() => setActiveType(pattern.patternType)}
          >
            {pattern.label}
          </button>
        ))}
      </div>

      <svg
        className="roll-svg"
        viewBox={`0 0 ${W} ${H}`}
        preserveAspectRatio="none"
      >
        {octaves.map((y, i) => (
          <line key={`o${i}`} x1={0} x2={W} y1={y} y2={y} stroke="rgba(255,255,255,0.07)" strokeWidth="1" />
        ))}
        {grids.map((g, i) => (
          <line
            key={`g${i}`}
            x1={g.x}
            x2={g.x}
            y1={padTop}
            y2={H - padBot}
            stroke={g.isBar ? 'rgba(255,255,255,0.14)' : g.isBeat ? 'rgba(255,255,255,0.08)' : 'rgba(255,255,255,0.035)'}
            strokeWidth="1"
          />
        ))}
        {notes.map(note => (
          <g key={note.key}>
            <rect
              x={note.x + 1}
              y={note.y}
              width={note.w}
              height={note.h}
              fill={noteColor(activeTrack?.patternType, note.active)}
              opacity={note.opacity}
            />
            {note.label && (
              <text
                x={note.x + 7}
                y={note.y + Math.max(note.h - 5, 11)}
                fill="rgba(0,0,0,0.72)"
                fontFamily="var(--mono)"
                fontSize="10"
              >
                {note.label}
              </text>
            )}
          </g>
        ))}
        {playing && (
          <>
            <line x1={playheadX} x2={playheadX} y1={padTop - 4} y2={H - padBot + 4}
              stroke="var(--accent)" strokeWidth="1.5" />
            <circle cx={playheadX} cy={padTop - 4} r="3" fill="var(--accent)" />
          </>
        )}
        {Array.from({ length: Math.ceil(totalSteps / 4) }, (_, beat) => (
          <text
            key={beat}
            x={beat * 4 * stepW + 8}
            y={H - 13}
            fill="rgba(255,255,255,0.25)"
            fontFamily="var(--mono)"
            fontSize="10"
          >
            {String(beat + 1).padStart(2, '0')}
          </text>
        ))}
      </svg>

      <div className="preview-actions">
        <button type="button" className={`btn ghost ${kept ? 'kept' : ''}`} onClick={onKeep}>
          {kept ? 'Kept' : 'Keep'}
        </button>
        <button type="button" className="btn ghost" onClick={onDiscard}>Discard</button>
        <button type="button" className="btn primary" onClick={onExportAll}>Export all</button>
      </div>
    </div>
  )
}

function buildRollNotes(track: TrackPreview | null, stepW: number, noteH: number, padTop: number, playheadX: number) {
  if (!track) return []
  return track.steps
    .filter(step => step.active && step.midi > 0)
    .map((step, index) => {
      const x = step.step * stepW
      const length = step.legato || step.slide ? 1.85 : step.staccato ? 0.56 : 0.92
      const w = Math.max(stepW * length - 2, 4)
      const y = padTop + (PITCH_MAX - clamp(step.midi, PITCH_MIN, PITCH_MAX)) * noteH
      const h = Math.max(noteH * (step.accent ? 1.45 : 1.05), 4)
      const active = playheadX >= x && playheadX < x + w
      return {
        key: `${track.patternType}-${step.step}-${step.note}-${index}`,
        x,
        y,
        w,
        h,
        active,
        label: step.note,
        opacity: active ? 1 : step.ghost ? 0.42 : step.accent ? 0.96 : 0.78,
      }
    })
}

function noteColor(patternType = '', active: boolean) {
  if (active) return 'var(--accent)'
  switch (patternType) {
    case 'bassline':
      return '#01FF95'
    case 'arpeggio':
      return '#FFB000'
    case 'melody':
      return '#F6FBFF'
    default:
      return 'var(--fg-dim)'
  }
}

function clamp(value: number, min: number, max: number) {
  return Math.max(min, Math.min(max, value))
}
