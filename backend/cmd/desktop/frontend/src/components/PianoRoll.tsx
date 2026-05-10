import { PointerEvent as ReactPointerEvent, useEffect, useMemo, useRef, useState } from 'react'
import { GenerationPreview, StepPreview, TrackPreview } from '../types'
import { isInScale, midiToNoteName, midiToNoteWithOctave, snapToScale } from '../lib/music'
import { useAudio } from '../hooks/useAudio'

const PITCH_MIN = 33
const PITCH_MAX = 96
const PITCH_RANGE = PITCH_MAX - PITCH_MIN   // 63
const BASE_STEP_W = 18
const NOTE_H = 12                           // Fixed row height — visible, consistent
const PIANO_KEY_W = 58                      // Piano keyboard panel width
const PAD_TOP = 28                          // Bar header height
const ROLL_SVG_H = PAD_TOP + PITCH_RANGE * NOTE_H  // 784px total

interface PianoRollProps {
  width?: number
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

function isBlackKey(midi: number): boolean {
  return [1, 3, 6, 8, 10].includes(midi % 12)
}

export function PianoRoll({
  width = 1480,
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
  const [previewPlaying, setPreviewPlaying] = useState(false)
  const [previewTrackType, setPreviewTrackType] = useState<string | null>(null)

  const { playNote } = useAudio({ volume })
  const startRef = useRef<number | null>(null)
  const prevActiveKeysRef = useRef<Set<string>>(new Set())
  const rafRef = useRef<number | null>(null)
  const svgRef = useRef<SVGSVGElement | null>(null)
  const scrollRef = useRef<HTMLDivElement | null>(null)

  const activeTrack = useMemo(() => {
    if (!preview?.patterns.length) return null
    return preview.patterns.find(p => p.patternType === activeType) ?? preview.patterns[0]
  }, [activeType, preview])

  const audioTrack = useMemo(() => {
    if (!preview?.patterns.length) return activeTrack
    if (previewPlaying && previewTrackType) {
      return preview.patterns.find(p => p.patternType === previewTrackType) ?? activeTrack
    }
    return activeTrack
  }, [activeTrack, previewPlaying, previewTrackType, preview])

  useEffect(() => {
    if (!preview?.patterns.some(p => p.patternType === activeType)) {
      setActiveType(preview?.patterns[0]?.patternType ?? 'bassline')
      setSelected(null)
    }
  }, [activeType, preview])

  // Scroll to typical pitch range when switching tracks
  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const targetMidi =
      activeType === 'bassline' ? 43 :
      activeType === 'arpeggio' ? 60 : 69
    const rowIdx = PITCH_MAX - 1 - targetMidi
    const targetY = PAD_TOP + rowIdx * NOTE_H - el.clientHeight / 2
    el.scrollTop = Math.max(0, targetY)
  }, [activeType])

  const stepsPerBar = preview?.stepsPerBar ?? 16
  const totalBars = preview?.bars ?? 16
  const totalSteps = Math.max(totalBars * stepsPerBar, activeTrack?.steps.length ?? stepsPerBar)
  // Full-pattern cycle so playhead sweeps all bars
  const cycleSec = (60 / bpm) * 4 * totalBars

  const effectivePlaying = previewPlaying || playing

  useEffect(() => {
    if (!effectivePlaying) {
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current)
      setT(0)
      startRef.current = null
      prevActiveKeysRef.current.clear()
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
  }, [cycleSec, effectivePlaying])

  const W = Math.max(width, totalSteps * BASE_STEP_W * zoom)
  const stepW = W / totalSteps
  const playheadX = t * W  // spans full pattern width

  // Note trigger on playhead crossing
  useEffect(() => {
    if (!effectivePlaying || !audioTrack) {
      prevActiveKeysRef.current.clear()
      return
    }
    const sw = W / totalSteps
    const currentActiveKeys = new Set<string>()
    for (const s of audioTrack.steps) {
      if (!s.active || s.midi <= 0) continue
      const dur = Math.max(s.durationSteps || 1, 1)
      const baseW = Math.max(sw * dur - 2, 5)
      const noteW = s.staccato ? Math.max(baseW * 0.5, 4) : baseW
      const x = s.step * sw
      if (playheadX >= x && playheadX < x + noteW) {
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

  const selectedStep = selected?.patternType === activeTrack?.patternType
    ? activeTrack?.steps[selected!.index]
    : null
  const effectiveScaleNotes = scaleLocked ? (preview.scaleNotes ?? []) : []

  const grids = Array.from({ length: totalSteps + 1 }, (_, step) => ({
    x: step * stepW,
    isBeat: step % 4 === 0,
    isBar: step % stepsPerBar === 0,
  }))

  const notes = buildRollNotes(activeTrack, stepW, NOTE_H, PAD_TOP, playheadX, selected)

  const updateStep = (index: number, updater: (step: StepPreview) => StepPreview) => {
    if (!preview || !activeTrack) return
    const nextPatterns = preview.patterns.map(pattern => {
      if (pattern.patternType !== activeTrack.patternType) return pattern
      return {
        ...pattern,
        steps: pattern.steps.map((step, i) =>
          i === index ? normalizeStep(updater(step), totalSteps) : step
        ),
      }
    })
    onPreviewChange?.({ ...preview, patterns: nextPatterns })
  }

  const startEdit = (event: ReactPointerEvent<SVGElement>, note: ReturnType<typeof buildRollNotes>[number]) => {
    setSelected({ patternType: activeTrack?.patternType ?? '', index: note.index })
    playNote(note.midi, 0.35, note.opacity * 0.7)
    const x = svgX(event)
    const edge = Math.min(14, Math.max(8, note.w / 4))
    const mode: EditMode =
      x <= note.x + edge ? 'resize-left' :
      x >= note.x + note.w - edge ? 'resize-right' :
      'drag'
    setDrag({
      mode,
      index: note.index,
      startStep: note.step,
      startDuration: note.durationSteps,
      startMidi: note.midi,
      pointerStep: snapStep(x, stepW),
      pointerY: svgY(event),
    })
    ;(event.currentTarget as Element).setPointerCapture(event.pointerId)
  }

  const continueEdit = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (!drag) return
    const pointerStep = snapStep(svgX(event), stepW)
    const delta = pointerStep - drag.pointerStep

    if (drag.mode === 'drag') {
      const deltaY = drag.pointerY - svgY(event)
      const deltaMidi = Math.round(deltaY / NOTE_H)
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
    const rawMidi = clamp(PITCH_MAX - 1 - Math.floor((y - PAD_TOP) / NOTE_H), PITCH_MIN, PITCH_MAX - 1)
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

  const toggleTrackPreview = (patternType: string) => {
    if (previewTrackType === patternType && previewPlaying) {
      setPreviewPlaying(false)
      setPreviewTrackType(null)
      setT(0)
      startRef.current = null
    } else {
      setPreviewTrackType(patternType)
      setPreviewPlaying(true)
      setActiveType(patternType)
      setSelected(null)
    }
  }

  return (
    <div
      className="real-roll editable-roll"
      tabIndex={0}
      onKeyDown={event => {
        if ((event.key === 'Delete' || event.key === 'Backspace') && selectedStep) deleteSelected()
      }}
    >
      {/* Row 1 — Chords + Zoom/Volume */}
      <div className="roll-chords">
        {preview.chords.length === 0 ? (
          <span className="chord-pill muted">No chord progression</span>
        ) : preview.chords.map(chord => (
          <span key={`${chord.name}-${chord.fromBar}-${chord.toBar}`} className="chord-pill">
            <strong>{chord.name}</strong>
            <span>bars {chord.fromBar}–{chord.toBar}</span>
          </span>
        ))}
        <div className="roll-zoom">
          <button type="button" onClick={() => setZoom(v => clamp(v - 0.5, 1, 5))}>-</button>
          <span>{zoom.toFixed(1)}x</span>
          <button type="button" onClick={() => setZoom(v => clamp(v + 0.5, 1, 5))}>+</button>
          <span className="vol-label">VOL</span>
          <input
            type="range" className="vol-slider"
            min={0} max={1} step={0.05} value={volume}
            onChange={e => setVolume(Number(e.target.value))}
            title={`Volume: ${Math.round(volume * 100)}%`}
          />
        </div>
      </div>

      {/* Row 2 — Track tabs + per-track play buttons + scale lock */}
      <div className="roll-tabs">
        {preview.patterns.map(pattern => {
          const isActive = pattern.patternType === activeTrack?.patternType
          const isPlaying = previewTrackType === pattern.patternType && previewPlaying
          return (
            <div key={pattern.patternType} className="roll-tab-group">
              <button
                type="button"
                className={`roll-tab-select ${isActive ? 'on' : ''}`}
                onClick={() => { setActiveType(pattern.patternType); setSelected(null) }}
              >
                {pattern.label}
              </button>
              <button
                type="button"
                className={`roll-tab-play ${isPlaying ? 'playing' : ''}`}
                title={isPlaying ? `Stop ${pattern.label}` : `Play ${pattern.label}`}
                onClick={() => toggleTrackPreview(pattern.patternType)}
              >
                {isPlaying ? '■' : '▶'}
              </button>
            </div>
          )
        })}
        <span className="active-track-copy">Editing {activeTrack?.label ?? 'track'} — snap 1/16</span>
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

      {/* Row 3 — Piano keyboard (sticky) + Roll SVG (scrollable) */}
      <div className="roll-scroll" ref={scrollRef}>
        <div className="roll-scroll-inner">

          {/* Piano keyboard panel — sticky to left on horizontal scroll */}
          <svg
            className="piano-keys-svg"
            width={PIANO_KEY_W}
            height={ROLL_SVG_H}
          >
            {/* Header padding area */}
            <rect x={0} y={0} width={PIANO_KEY_W} height={PAD_TOP} fill="var(--bg-panel)" />
            <line x1={0} x2={PIANO_KEY_W} y1={PAD_TOP} y2={PAD_TOP}
              stroke="rgba(255,255,255,0.1)" strokeWidth={1} />

            {Array.from({ length: PITCH_RANGE }, (_, i) => {
              const midi = PITCH_MAX - 1 - i
              const noteName = midiToNoteName(midi)
              const noteWithOct = midiToNoteWithOctave(midi)
              const black = isBlackKey(midi)
              const inScale = effectiveScaleNotes.length > 0
                ? isInScale(noteName, effectiveScaleNotes)
                : false
              const isC = midi % 12 === 0
              const y = PAD_TOP + i * NOTE_H

              return (
                <g key={midi}>
                  {/* Row background */}
                  <rect
                    x={0} y={y}
                    width={PIANO_KEY_W} height={NOTE_H}
                    fill={
                      inScale ? 'rgba(0,224,255,0.13)' :
                      black   ? '#090C13' : '#111620'
                    }
                  />
                  {/* Left key stripe: black vs white indicator */}
                  <rect
                    x={0} y={y}
                    width={black ? 11 : 6} height={NOTE_H}
                    fill={black ? '#050709' : 'rgba(255,255,255,0.06)'}
                  />
                  {/* Separator line — stronger on C */}
                  {!black && (
                    <line
                      x1={6} x2={PIANO_KEY_W}
                      y1={y} y2={y}
                      stroke={isC ? 'rgba(255,255,255,0.28)' : 'rgba(255,255,255,0.06)'}
                      strokeWidth={isC ? 1 : 0.5}
                    />
                  )}
                  {/* Note label: always show C notes, show scale notes when they fit */}
                  {(isC || inScale) && (
                    <text
                      x={PIANO_KEY_W - 5}
                      y={y + NOTE_H - 3}
                      textAnchor="end"
                      fill={inScale ? 'rgba(0,224,255,0.88)' : 'rgba(255,255,255,0.42)'}
                      fontFamily="JetBrains Mono, monospace"
                      fontSize={isC ? 9 : 8}
                      fontWeight={isC ? 700 : 400}
                    >
                      {isC ? noteWithOct : noteName}
                    </text>
                  )}
                </g>
              )
            })}
          </svg>

          {/* Main piano roll SVG */}
          <svg
            ref={svgRef}
            className="roll-svg"
            width={W}
            height={ROLL_SVG_H}
            viewBox={`0 0 ${W} ${ROLL_SVG_H}`}
            onPointerMove={continueEdit}
            onPointerUp={finishEdit}
            onPointerCancel={finishEdit}
          >
            {/* Scale row backgrounds */}
            {Array.from({ length: PITCH_RANGE }, (_, i) => {
              const midi = PITCH_MAX - 1 - i
              const inScale = effectiveScaleNotes.length > 0
                ? isInScale(midiToNoteName(midi), effectiveScaleNotes)
                : true
              return (
                <rect
                  key={`row-${midi}`}
                  x={0} y={PAD_TOP + i * NOTE_H}
                  width={W} height={NOTE_H}
                  fill={inScale ? 'rgba(255,255,255,0.026)' : 'rgba(0,0,0,0.15)'}
                  pointerEvents="none"
                />
              )
            })}

            {/* Octave separator lines */}
            {Array.from({ length: 8 }, (_, i) => {
              const p = 36 + i * 12
              if (p < PITCH_MIN || p > PITCH_MAX) return null
              const y = PAD_TOP + (PITCH_MAX - p) * NOTE_H
              return (
                <line key={`oct${i}`} x1={0} x2={W} y1={y} y2={y}
                  stroke="rgba(255,255,255,0.1)" strokeWidth={1} />
              )
            })}

            {/* Vertical grid lines */}
            {grids.map((g, i) => (
              <line
                key={`g${i}`}
                x1={g.x} x2={g.x}
                y1={PAD_TOP} y2={ROLL_SVG_H}
                stroke={
                  g.isBar  ? 'rgba(255,255,255,0.22)' :
                  g.isBeat ? 'rgba(255,255,255,0.08)' :
                  'rgba(255,255,255,0.03)'
                }
                strokeWidth={g.isBar ? 1.3 : 1}
              />
            ))}

            {/* Bar numbers */}
            {Array.from({ length: totalBars }, (_, bar) => (
              <text
                key={bar}
                x={bar * stepsPerBar * stepW + 6}
                y={18}
                fill="rgba(255,255,255,0.34)"
                fontFamily="JetBrains Mono, monospace"
                fontSize={10}
              >
                B{String(bar + 1).padStart(2, '0')}
              </text>
            ))}

            {/* Transparent canvas — click empty space to toggle a note */}
            <rect
              x={0} y={PAD_TOP}
              width={W} height={ROLL_SVG_H - PAD_TOP}
              fill="transparent"
              style={{ cursor: drag ? 'default' : 'crosshair' }}
              onPointerDown={handleCanvasPointerDown}
            />

            {/* Notes */}
            {notes.map(note => {
              const color = noteColor(activeTrack?.patternType, note.active)
              const nh = Math.max(note.h - 2, 3)
              const edge = Math.min(14, Math.max(8, note.w / 4))
              return (
                <g key={note.key} className={`roll-note ${note.selected ? 'selected' : ''}`}>
                  {/* Main note body */}
                  <rect
                    x={note.x + 1} y={note.y + 1}
                    width={Math.max(note.w - 1, 3)} height={nh}
                    rx={2}
                    fill={color}
                    opacity={note.opacity}
                    style={{ cursor: 'grab' }}
                    onPointerDown={event => startEdit(event, note)}
                  />
                  {/* Left resize handle */}
                  <rect
                    x={note.x + 1} y={note.y + 1}
                    width={edge} height={nh}
                    fill="transparent"
                    style={{ cursor: 'w-resize', pointerEvents: 'all' }}
                    onPointerDown={event => startEdit(event, note)}
                  />
                  {/* Right resize handle */}
                  <rect
                    x={note.x + note.w - edge} y={note.y + 1}
                    width={edge} height={nh}
                    fill="transparent"
                    style={{ cursor: 'e-resize', pointerEvents: 'all' }}
                    onPointerDown={event => startEdit(event, note)}
                  />
                  {/* Legato arrow */}
                  {note.legato && (
                    <polygon
                      points={`${note.x + note.w + 1},${note.y + nh * 0.25 + 1} ${note.x + note.w + 5},${note.y + nh * 0.5 + 1} ${note.x + note.w + 1},${note.y + nh * 0.75 + 1}`}
                      fill={color}
                      opacity={note.opacity * 0.8}
                      pointerEvents="none"
                    />
                  )}
                  {/* Note label — only when note is wide enough */}
                  {note.w > 26 && (
                    <text
                      x={note.x + 8} y={note.y + nh - 2}
                      fill="rgba(0,0,0,0.75)"
                      fontFamily="JetBrains Mono, monospace"
                      fontSize={9}
                      pointerEvents="none"
                    >
                      {note.label}
                    </text>
                  )}
                </g>
              )
            })}

            {/* Playhead */}
            {effectivePlaying && (
              <>
                <line
                  x1={playheadX} x2={playheadX}
                  y1={PAD_TOP - 4} y2={ROLL_SVG_H}
                  stroke="var(--accent)" strokeWidth={1.5}
                />
                <circle cx={playheadX} cy={PAD_TOP - 4} r={3} fill="var(--accent)" />
              </>
            )}
          </svg>
        </div>
      </div>

      {/* Row 4 — Inspector + Actions */}
      <div className="preview-actions editor-actions">
        <div className="note-inspector">
          {selectedStep && selectedStep.active ? (
            <>
              <strong>{selectedStep.note}</strong>
              <span>step {selectedStep.step + 1}</span>
              <label>
                Len
                <input
                  type="number" min={1} max={32}
                  value={selectedStep.durationSteps || 1}
                  onChange={e => updateStep(selected!.index, step => ({ ...step, durationSteps: Number(e.target.value) }))}
                />
              </label>
              <label>
                Vel
                <input
                  type="range" min={1} max={120}
                  value={selectedStep.velocity || 84}
                  onChange={e => updateStep(selected!.index, step => ({ ...step, velocity: Number(e.target.value) }))}
                />
                <span>{selectedStep.velocity || 84}</span>
              </label>
              <label className="mini-check">
                <input type="checkbox" checked={selectedStep.accent}
                  onChange={e => updateStep(selected!.index, step => ({ ...step, accent: e.target.checked }))} />
                Accent
              </label>
              <label className="mini-check">
                <input type="checkbox" checked={selectedStep.ghost}
                  onChange={e => updateStep(selected!.index, step => ({ ...step, ghost: e.target.checked }))} />
                Ghost
              </label>
              <label className="mini-check">
                <input type="checkbox" checked={selectedStep.staccato}
                  onChange={e => updateStep(selected!.index, step => ({ ...step, staccato: e.target.checked, legato: false }))} />
                Staccato
              </label>
              <label className="mini-check">
                <input type="checkbox" checked={selectedStep.legato}
                  onChange={e => updateStep(selected!.index, step => ({ ...step, legato: e.target.checked, staccato: false }))} />
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
    return clamp(((event.clientY - rect.top) / rect.height) * ROLL_SVG_H, 0, ROLL_SVG_H)
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
    .filter(({ step }) => step.active && step.midi > 0)
    .map(({ step, index }) => {
      const durationSteps = Math.max(step.durationSteps || 1, 1)
      const baseW = Math.max(stepW * durationSteps - 2, 5)
      const w = step.staccato ? Math.max(baseW * 0.5, 4) : baseW
      const x = step.step * stepW
      const y = padTop + (PITCH_MAX - clamp(step.midi, PITCH_MIN, PITCH_MAX)) * noteH
      const h = Math.max(noteH - 1, 4)
      const active = playheadX >= x && playheadX < x + w
      const isSelected = selected?.patternType === track.patternType && selected.index === index
      return {
        key: `${track.patternType}-${step.step}-${step.note}-${index}`,
        index,
        step: step.step,
        durationSteps,
        midi: step.midi,
        x, y, w, h,
        active,
        selected: isSelected,
        label: step.note,
        opacity: isSelected ? 1 : active ? 1 : step.ghost ? 0.38 : step.accent ? 0.96 : 0.82,
        staccato: step.staccato ?? false,
        legato: step.legato ?? false,
      }
    })
}

function normalizeStep(step: StepPreview, totalSteps: number): StepPreview {
  const start = clamp(Math.round(step.step), 0, Math.max(totalSteps - 1, 0))
  const duration = clamp(Math.round(step.durationSteps || 1), 1, Math.max(totalSteps - start, 1))
  return { ...step, step: start, durationSteps: duration, velocity: clamp(Math.round(step.velocity || 84), 1, 120) }
}

function snapStep(x: number, stepW: number) {
  return Math.round(x / stepW)
}

function noteColor(patternType = '', active: boolean) {
  if (active) return 'var(--accent)'
  switch (patternType) {
    case 'bassline': return 'color-mix(in srgb, var(--accent) 78%, #F6FBFF)'
    case 'arpeggio': return 'color-mix(in srgb, var(--accent) 55%, #F6FBFF)'
    case 'melody':   return '#F0F6FF'
    default:         return 'var(--fg-dim)'
  }
}

function clamp(value: number, min: number, max: number) {
  return Math.max(min, Math.min(max, value))
}
