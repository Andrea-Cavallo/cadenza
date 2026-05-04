import { PointerEvent as ReactPointerEvent, useEffect, useMemo, useRef, useState } from 'react'
import { GenerationPreview, StepPreview, TrackPreview } from '../types'
import { isInScale, midiToNoteName, midiToNoteWithOctave, snapToScale } from '../lib/music'
import { useAudio } from '../hooks/useAudio'

const PITCH_MIN = 33
const PITCH_MAX = 96
const PITCH_RANGE = PITCH_MAX - PITCH_MIN
const BASE_STEP_W = 18

interface PianoRollProps {
  width?: number
  height?: number
  bpm?: number
  playing?: boolean
  preview?: GenerationPreview | null
  kept?: boolean
  onPreviewChange?: (preview: GenerationPreview) => void
  onKeep?: () => void
  onDiscard?: () => void
  onExportEdited?: () => void
}

type EditMode = 'drag' | 'resize-left' | 'resize-right'

interface DragState {
  mode: EditMode
  index: number
  startStep: number
  startDuration: number
  startMidi: number
  pointerStep: number
  pointerY: number
}

interface SelectedNote {
  patternType: string
  index: number
}

export function PianoRoll({
  width = 1480,
  height = 420,
  bpm = 122,
  playing = false,
  preview = null,
  kept = false,
  onPreviewChange,
  onKeep,
  onDiscard,
  onExportEdited,
}: PianoRollProps) {
  const [activeType, setActiveType] = useState('bassline')
  const [selected, setSelected] = useState<SelectedNote | null>(null)
  const [drag, setDrag] = useState<DragState | null>(null)
  const [zoom, setZoom] = useState(1)
  const [t, setT] = useState(0)
  const [scaleLocked, setScaleLocked] = useState(true)
  const [volume, setVolume] = useState(0.6)
  const { playNote } = useAudio({ volume })
  const startRef = useRef<number | null>(null)
  const prevActiveKeysRef = useRef<Set<string>>(new Set())
  const rafRef = useRef<number | null>(null)
  const svgRef = useRef<SVGSVGElement | null>(null)

  const activeTrack = useMemo(() => {
    if (!preview?.patterns.length) return null
    return preview.patterns.find(pattern => pattern.patternType === activeType) ?? preview.patterns[0]
  }, [activeType, preview])

  useEffect(() => {
    if (!preview?.patterns.some(pattern => pattern.patternType === activeType)) {
      setActiveType(preview?.patterns[0]?.patternType ?? 'bassline')
      setSelected(null)
    }
  }, [activeType, preview])

  const stepsPerBar = preview?.stepsPerBar ?? 16
  const totalBars = preview?.bars ?? 16
  const totalSteps = Math.max(totalBars * stepsPerBar, activeTrack?.steps.length ?? stepsPerBar)
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

  // WAV-3: trigger playNote whenever the playhead crosses a note's leading edge
  useEffect(() => {
    if (!playing || !activeTrack) {
      prevActiveKeysRef.current.clear()
      return
    }
    const sp = preview?.stepsPerBar ?? 16
    const tb = preview?.bars ?? 16
    const ts = Math.max(tb * sp, activeTrack.steps.length > 0 ? activeTrack.steps.length : sp)
    const W = Math.max(width, ts * BASE_STEP_W * zoom)
    const sw = W / ts
    const px = t * sp * sw

    const currentActiveKeys = new Set<string>()
    for (const s of activeTrack.steps) {
      if (!s.active || s.midi <= 0) continue
      const dur = Math.max(s.durationSteps || 1, 1)
      const baseW = Math.max(sw * dur - 2, 5)
      const noteW = s.staccato ? Math.max(baseW * 0.5, 4) : baseW
      const x = s.step * sw
      if (px >= x && px < x + noteW) {
        const key = `${s.step}-${s.midi}`
        currentActiveKeys.add(key)
        if (!prevActiveKeysRef.current.has(key)) {
          const vel = s.ghost ? 0.42 : s.accent ? 0.96 : 0.78
          playNote(s.midi, 0.25, vel * 0.65)
        }
      }
    }
    prevActiveKeysRef.current = currentActiveKeys
  })

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

  const W = Math.max(width, totalSteps * BASE_STEP_W * zoom)
  const H = height
  const padTop = 60
  const padBot = 28
  const innerH = H - padTop - padBot
  const stepW = W / totalSteps
  const noteH = innerH / PITCH_RANGE
  const playheadX = t * stepsPerBar * stepW
  const selectedStep = selected?.patternType === activeTrack?.patternType ? activeTrack?.steps[selected!.index] : null
  const effectiveScaleNotes = scaleLocked ? (preview.scaleNotes ?? []) : []

  const grids = Array.from({ length: totalSteps + 1 }, (_, step) => ({
    x: step * stepW,
    isBeat: step % 4 === 0,
    isBar: step % stepsPerBar === 0,
  }))

  const octaves = []
  for (let p = 36; p <= 96; p += 12) {
    octaves.push(padTop + (PITCH_MAX - p) * noteH)
  }

  const notes = buildRollNotes(activeTrack, stepW, noteH, padTop, playheadX, selected)

  const updateStep = (index: number, updater: (step: StepPreview) => StepPreview) => {
    if (!preview || !activeTrack) return
    const nextPatterns = preview.patterns.map(pattern => {
      if (pattern.patternType !== activeTrack.patternType) return pattern
      return {
        ...pattern,
        steps: pattern.steps.map((step, i) => (i === index ? normalizeStep(updater(step), totalSteps) : step)),
      }
    })
    onPreviewChange?.({ ...preview, patterns: nextPatterns })
  }

  const startEdit = (event: ReactPointerEvent<SVGRectElement>, note: ReturnType<typeof buildRollNotes>[number]) => {
    setSelected({ patternType: activeTrack?.patternType ?? '', index: note.index })
    playNote(note.midi, 0.35, note.opacity * 0.7)
    const x = svgX(event)
    const edge = Math.min(10, Math.max(5, note.w / 3))
    const mode: EditMode = x <= note.x + edge ? 'resize-left' : x >= note.x + note.w - edge ? 'resize-right' : 'drag'
    const pointerStep = snapStep(x, stepW)
    setDrag({
      mode,
      index: note.index,
      startStep: note.step,
      startDuration: note.durationSteps,
      startMidi: note.midi,
      pointerStep,
      pointerY: svgY(event),
    })
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  const continueEdit = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (!drag) return
    const pointerStep = snapStep(svgX(event), stepW)
    const delta = pointerStep - drag.pointerStep

    if (drag.mode === 'drag') {
      const deltaY = drag.pointerY - svgY(event)
      const deltaMidi = Math.round(deltaY / noteH)
      const rawMidi = clamp(drag.startMidi + deltaMidi, PITCH_MIN, PITCH_MAX - 1)
      const newMidi = snapToScale(rawMidi, effectiveScaleNotes)
      updateStep(drag.index, step => ({
        ...step,
        step: clamp(drag.startStep + delta, 0, totalSteps - 1),
        midi: newMidi,
        note: midiToNoteWithOctave(newMidi),
      }))
      return
    }

    updateStep(drag.index, step => {
      if (drag.mode === 'resize-right') {
        return { ...step, durationSteps: clamp(drag.startDuration + delta, 1, totalSteps - drag.startStep) }
      }
      const nextStep = clamp(drag.startStep + delta, 0, drag.startStep + drag.startDuration - 1)
      const endStep = drag.startStep + drag.startDuration
      return { ...step, step: nextStep, durationSteps: clamp(endStep - nextStep, 1, totalSteps - nextStep) }
    })
  }

  const finishEdit = () => setDrag(null)

  const deleteSelected = () => {
    if (!selectedStep || selected?.patternType !== activeTrack?.patternType) return
    updateStep(selected!.index, step => ({ ...step, active: false }))
    setSelected(null)
  }

  const handleCanvasPointerDown = (event: ReactPointerEvent<SVGRectElement>) => {
    if (!activeTrack || !preview) return
    const x = svgX(event)
    const y = svgY(event)
    const clickStep = clamp(Math.floor(x / stepW), 0, totalSteps - 1)
    const rawMidi = clamp(PITCH_MAX - 1 - Math.floor((y - padTop) / noteH), PITCH_MIN, PITCH_MAX - 1)
    const newMidi = snapToScale(rawMidi, effectiveScaleNotes)
    const stepIdx = activeTrack.steps.findIndex(s => s.step === clickStep)
    if (stepIdx < 0) return
    updateStep(stepIdx, s => ({
      ...s,
      active: true,
      midi: newMidi,
      note: midiToNoteWithOctave(newMidi),
      velocity: 84,
      accent: false,
      ghost: false,
      staccato: false,
      legato: false,
    }))
    playNote(newMidi, 0.35, 0.6)
    setSelected({ patternType: activeTrack.patternType, index: stepIdx })
  }

  return (
    <div
      className="real-roll editable-roll"
      tabIndex={0}
      onKeyDown={event => {
        if ((event.key === 'Delete' || event.key === 'Backspace') && selectedStep) deleteSelected()
      }}
    >
      <div className="roll-chords">
        {preview.chords.length === 0 ? (
          <span className="chord-pill muted">No chord progression</span>
        ) : preview.chords.map(chord => (
          <span key={`${chord.name}-${chord.fromBar}-${chord.toBar}`} className="chord-pill">
            <strong>{chord.name}</strong>
            <span>bars {chord.fromBar}-{chord.toBar}</span>
          </span>
        ))}
        <div className="roll-zoom">
          <button type="button" onClick={() => setZoom(value => clamp(value - 0.5, 1, 5))}>-</button>
          <span>{zoom.toFixed(1)}x</span>
          <button type="button" onClick={() => setZoom(value => clamp(value + 0.5, 1, 5))}>+</button>
          <span className="vol-label">VOL</span>
          <input
            type="range"
            className="vol-slider"
            min={0}
            max={1}
            step={0.05}
            value={volume}
            onChange={event => setVolume(Number(event.target.value))}
            title={`Volume: ${Math.round(volume * 100)}%`}
          />
        </div>
      </div>

      <div className="roll-tabs">
        {preview.patterns.map(pattern => (
          <button
            key={pattern.patternType}
            type="button"
            className={pattern.patternType === activeTrack?.patternType ? 'on' : ''}
            onClick={() => {
              setActiveType(pattern.patternType)
              setSelected(null)
            }}
          >
            {pattern.label}
          </button>
        ))}
        <span className="active-track-copy">Editing {activeTrack?.label ?? 'track'} - snap 1/16</span>
        {(preview.scaleNotes?.length ?? 0) > 0 && (
          <button
            type="button"
            className={`scale-lock-btn ${scaleLocked ? 'on' : ''}`}
            onClick={() => setScaleLocked(v => !v)}
            title={scaleLocked
              ? `Scale lock ON (${preview.keyName}) — click to disable`
              : `Scale lock OFF — click to enable (${preview.keyName})`}
          >
            {scaleLocked ? '🔒' : '🔓'} {preview.keyName}
          </button>
        )}
      </div>

      <div className="roll-scroll">
        <svg
          ref={svgRef}
          className="roll-svg"
          width={W}
          height={H}
          viewBox={`0 0 ${W} ${H}`}
          preserveAspectRatio="none"
          onPointerMove={continueEdit}
          onPointerUp={finishEdit}
          onPointerCancel={finishEdit}
        >
          {/* Scale row backgrounds — in-scale rows slightly lighter, out-of-scale darker */}
          {Array.from({ length: PITCH_RANGE }, (_, i) => {
            const midi = PITCH_MAX - 1 - i
            const inScale = effectiveScaleNotes.length > 0
              ? isInScale(midiToNoteName(midi), effectiveScaleNotes)
              : true
            return (
              <rect
                key={`row-${midi}`}
                x={0}
                y={padTop + i * noteH}
                width={W}
                height={noteH}
                fill={inScale ? 'rgba(255,255,255,0.025)' : 'rgba(0,0,0,0.14)'}
                pointerEvents="none"
              />
            )
          })}

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
              stroke={g.isBar ? 'rgba(255,255,255,0.22)' : g.isBeat ? 'rgba(255,255,255,0.09)' : 'rgba(255,255,255,0.035)'}
              strokeWidth={g.isBar ? 1.3 : 1}
            />
          ))}
          {Array.from({ length: totalBars }, (_, bar) => (
            <text
              key={bar}
              x={bar * stepsPerBar * stepW + 8}
              y={26}
              fill="rgba(255,255,255,0.34)"
              fontFamily="var(--mono)"
              fontSize="10"
            >
              B{String(bar + 1).padStart(2, '0')}
            </text>
          ))}

          {/* Transparent canvas — click on empty space to add a note */}
          <rect
            x={0}
            y={padTop}
            width={W}
            height={H - padTop - padBot}
            fill="transparent"
            style={{ cursor: drag ? 'default' : 'crosshair' }}
            onPointerDown={handleCanvasPointerDown}
          />

          {notes.map(note => (
            <g key={note.key} className={`roll-note ${note.selected ? 'selected' : ''}`}>
              <rect
                x={note.x + 1}
                y={note.y}
                width={note.w}
                height={note.h}
                rx={1}
                fill={noteColor(activeTrack?.patternType, note.active)}
                opacity={note.opacity}
                onPointerDown={event => startEdit(event, note)}
              />
              <rect className="note-edge" x={note.x + 1} y={note.y} width={5} height={note.h} />
              <rect className="note-edge right" x={note.x + note.w - 4} y={note.y} width={5} height={note.h} />
              {/* Legato indicator — small arrow at right edge */}
              {note.legato && (
                <polygon
                  points={`${note.x + note.w + 1},${note.y + note.h * 0.25} ${note.x + note.w + 5},${note.y + note.h * 0.5} ${note.x + note.w + 1},${note.y + note.h * 0.75}`}
                  fill={noteColor(activeTrack?.patternType, note.active)}
                  opacity={note.opacity * 0.8}
                  pointerEvents="none"
                />
              )}
              {note.label && (
                <text
                  x={note.x + 7}
                  y={note.y + Math.max(note.h - 5, 11)}
                  fill="rgba(0,0,0,0.72)"
                  fontFamily="var(--mono)"
                  fontSize="10"
                  pointerEvents="none"
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
        </svg>
      </div>

      <div className="preview-actions editor-actions">
        <div className="note-inspector">
          {selectedStep && selectedStep.active ? (
            <>
              <strong>{selectedStep.note}</strong>
              <span>step {selectedStep.step + 1}</span>
              <label>
                Len
                <input
                  type="number"
                  min={1}
                  max={32}
                  value={selectedStep.durationSteps || 1}
                  onChange={event => updateStep(selected!.index, step => ({ ...step, durationSteps: Number(event.target.value) }))}
                />
              </label>
              <label>
                Vel
                <input
                  type="range"
                  min={1}
                  max={120}
                  value={selectedStep.velocity || 84}
                  onChange={event => updateStep(selected!.index, step => ({ ...step, velocity: Number(event.target.value) }))}
                />
                <span>{selectedStep.velocity || 84}</span>
              </label>
              <label className="mini-check">
                <input
                  type="checkbox"
                  checked={selectedStep.accent}
                  onChange={event => updateStep(selected!.index, step => ({ ...step, accent: event.target.checked }))}
                />
                Accent
              </label>
              <label className="mini-check">
                <input
                  type="checkbox"
                  checked={selectedStep.ghost}
                  onChange={event => updateStep(selected!.index, step => ({ ...step, ghost: event.target.checked }))}
                />
                Ghost
              </label>
              <label className="mini-check">
                <input
                  type="checkbox"
                  checked={selectedStep.staccato}
                  onChange={event => updateStep(selected!.index, step => ({ ...step, staccato: event.target.checked, legato: false }))}
                />
                Staccato
              </label>
              <label className="mini-check">
                <input
                  type="checkbox"
                  checked={selectedStep.legato}
                  onChange={event => updateStep(selected!.index, step => ({ ...step, legato: event.target.checked, staccato: false }))}
                />
                Legato
              </label>
              <button type="button" className="btn ghost danger" onClick={deleteSelected}>Delete note</button>
            </>
          ) : (
            <span className="inspector-empty">Select a note to edit velocity, accent, ghost, length, or delete it.</span>
          )}
        </div>
        <button type="button" className={`btn ghost ${kept ? 'kept' : ''}`} onClick={onKeep}>
          {kept ? 'Kept' : 'Keep'}
        </button>
        <button type="button" className="btn ghost" onClick={onDiscard}>Discard</button>
        <button type="button" className="btn primary" onClick={onExportEdited}>Export edited</button>
      </div>
    </div>
  )

  function svgX(event: { clientX: number }): number {
    const svg = svgRef.current
    if (!svg) return 0
    const rect = svg.getBoundingClientRect()
    return clamp(((event.clientX - rect.left) / rect.width) * W, 0, W)
  }

  function svgY(event: { clientY: number }): number {
    const svg = svgRef.current
    if (!svg) return 0
    const rect = svg.getBoundingClientRect()
    return clamp(((event.clientY - rect.top) / rect.height) * H, 0, H)
  }
}

function buildRollNotes(
  track: TrackPreview | null,
  stepW: number,
  noteH: number,
  padTop: number,
  playheadX: number,
  selected: SelectedNote | null,
) {
  if (!track) return []
  return track.steps
    .map((step, index) => ({ step, index }))
    .filter(item => item.step.active && item.step.midi > 0)
    .map(({ step, index }) => {
      const durationSteps = Math.max(step.durationSteps || 1, 1)
      const baseW = Math.max(stepW * durationSteps - 2, 5)
      const w = step.staccato ? Math.max(baseW * 0.5, 4) : baseW
      const x = step.step * stepW
      const y = padTop + (PITCH_MAX - clamp(step.midi, PITCH_MIN, PITCH_MAX)) * noteH
      const h = Math.max(noteH * (step.accent ? 1.45 : 1.05), 4)
      const active = playheadX >= x && playheadX < x + w
      const isSelected = selected?.patternType === track.patternType && selected.index === index
      return {
        key: `${track.patternType}-${step.step}-${step.note}-${index}`,
        index,
        step: step.step,
        durationSteps,
        midi: step.midi,
        x,
        y,
        w,
        h,
        active,
        selected: isSelected,
        label: step.note,
        opacity: isSelected ? 1 : active ? 1 : step.ghost ? 0.42 : step.accent ? 0.96 : 0.78,
        staccato: step.staccato ?? false,
        legato: step.legato ?? false,
      }
    })
}

function normalizeStep(step: StepPreview, totalSteps: number): StepPreview {
  const start = clamp(Math.round(step.step), 0, Math.max(totalSteps - 1, 0))
  const duration = clamp(Math.round(step.durationSteps || 1), 1, Math.max(totalSteps - start, 1))
  return {
    ...step,
    step: start,
    durationSteps: duration,
    velocity: clamp(Math.round(step.velocity || 84), 1, 120),
  }
}

function snapStep(x: number, stepW: number) {
  return Math.round(x / stepW)
}

function noteColor(patternType = '', active: boolean) {
  if (active) return 'var(--accent)'
  switch (patternType) {
    case 'bassline':
      return 'color-mix(in srgb, var(--accent) 82%, #F6FBFF)'
    case 'arpeggio':
      return 'color-mix(in srgb, var(--accent) 68%, #F6FBFF)'
    case 'melody':
      return '#F6FBFF'
    default:
      return 'var(--fg-dim)'
  }
}

function clamp(value: number, min: number, max: number) {
  return Math.max(min, Math.min(max, value))
}
