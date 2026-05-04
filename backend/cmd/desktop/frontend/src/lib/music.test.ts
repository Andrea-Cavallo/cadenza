import { describe, it, expect } from 'vitest'
import { midiToNoteName, midiToNoteWithOctave, isInScale, snapToScale } from './music'

describe('midiToNoteName', () => {
  it('maps MIDI 60 → C', () => expect(midiToNoteName(60)).toBe('C'))
  it('maps MIDI 61 → C#', () => expect(midiToNoteName(61)).toBe('C#'))
  it('maps MIDI 69 → A', () => expect(midiToNoteName(69)).toBe('A'))
  it('maps MIDI 70 → A#', () => expect(midiToNoteName(70)).toBe('A#'))
  it('maps MIDI 0 → C', () => expect(midiToNoteName(0)).toBe('C'))
  it('maps MIDI 127 → G', () => expect(midiToNoteName(127)).toBe('G'))
})

describe('midiToNoteWithOctave', () => {
  it('maps MIDI 60 → C4', () => expect(midiToNoteWithOctave(60)).toBe('C4'))
  it('maps MIDI 69 → A4', () => expect(midiToNoteWithOctave(69)).toBe('A4'))
  it('maps MIDI 45 → A2', () => expect(midiToNoteWithOctave(45)).toBe('A2'))
  it('maps MIDI 48 → C3', () => expect(midiToNoteWithOctave(48)).toBe('C3'))
  it('maps MIDI 36 → C2', () => expect(midiToNoteWithOctave(36)).toBe('C2'))
  it('maps MIDI 72 → C5', () => expect(midiToNoteWithOctave(72)).toBe('C5'))
})

describe('isInScale', () => {
  const amScale = ['A', 'B', 'C', 'D', 'E', 'F', 'G']
  const majorScale = ['C', 'D', 'E', 'F', 'G', 'A', 'B']

  it('A is in Am', () => expect(isInScale('A', amScale)).toBe(true))
  it('G is in Am', () => expect(isInScale('G', amScale)).toBe(true))
  it('C# is NOT in Am', () => expect(isInScale('C#', amScale)).toBe(false))
  it('A# is NOT in Am', () => expect(isInScale('A#', amScale)).toBe(false))
  it('enharmonic Bb matches A# — NOT in Am', () => expect(isInScale('Bb', amScale)).toBe(false))
  it('enharmonic Db matches C# — in C# scale', () => expect(isInScale('Db', ['C#', 'E', 'G'])).toBe(true))
  it('empty scale → always false', () => expect(isInScale('A', [])).toBe(false))
  it('C is in major', () => expect(isInScale('C', majorScale)).toBe(true))
  it('F# is NOT in C major', () => expect(isInScale('F#', majorScale)).toBe(false))
})

describe('snapToScale', () => {
  const amScale = ['A', 'B', 'C', 'D', 'E', 'F', 'G']

  it('in-scale note is returned unchanged', () => {
    expect(snapToScale(69, amScale)).toBe(69) // A4
    expect(snapToScale(71, amScale)).toBe(71) // B4
  })

  it('A#4 (MIDI 70) snaps to nearest: A4=69 or B4=71', () => {
    const result = snapToScale(70, amScale)
    expect([69, 71]).toContain(result)
  })

  it('direction up: A#4 (70) → B4 (71)', () => {
    expect(snapToScale(70, amScale, 'up')).toBe(71)
  })

  it('direction down: A#4 (70) → A4 (69)', () => {
    expect(snapToScale(70, amScale, 'down')).toBe(69)
  })

  it('C#4 (61) direction up → D4 (62)', () => {
    expect(snapToScale(61, amScale, 'up')).toBe(62) // D4
  })

  it('empty scale returns midi unchanged', () => {
    expect(snapToScale(70, [])).toBe(70)
    expect(snapToScale(61, [], 'up')).toBe(61)
  })

  it('result is always in scale for full piano roll range (Am)', () => {
    for (let midi = 33; midi < 96; midi++) {
      const snapped = snapToScale(midi, amScale)
      expect(
        isInScale(midiToNoteName(snapped), amScale),
        `MIDI ${midi} snapped to ${snapped} (${midiToNoteName(snapped)}) is out of scale`
      ).toBe(true)
    }
  })

  it('result is always in scale for full piano roll range (D dorian)', () => {
    const dorianScale = ['D', 'E', 'F', 'G', 'A', 'B', 'C']
    for (let midi = 33; midi < 96; midi++) {
      const snapped = snapToScale(midi, dorianScale)
      expect(isInScale(midiToNoteName(snapped), dorianScale)).toBe(true)
    }
  })
})
