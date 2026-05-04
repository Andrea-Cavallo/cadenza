const CHROMATIC_SHARPS = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B']

const ENHARMONIC: Record<string, string> = {
  Db: 'C#', Eb: 'D#', Gb: 'F#', Ab: 'G#', Bb: 'A#',
}

function normalize(name: string): string {
  return ENHARMONIC[name] ?? name
}

export function midiToNoteName(midi: number): string {
  return CHROMATIC_SHARPS[((midi % 12) + 12) % 12]
}

export function midiToNoteWithOctave(midi: number): string {
  const octave = Math.floor(midi / 12) - 1
  return midiToNoteName(midi) + octave
}

export function isInScale(noteName: string, scaleNotes: string[]): boolean {
  const norm = normalize(noteName)
  return scaleNotes.some(sn => normalize(sn) === norm)
}

export function snapToScale(
  midi: number,
  scaleNotes: string[],
  direction: 'nearest' | 'up' | 'down' = 'nearest',
): number {
  if (scaleNotes.length === 0) return midi
  if (isInScale(midiToNoteName(midi), scaleNotes)) return midi

  let upDelta = Infinity
  let downDelta = Infinity

  for (let d = 1; d <= 12; d++) {
    if (upDelta === Infinity && isInScale(midiToNoteName(midi + d), scaleNotes)) upDelta = d
    if (downDelta === Infinity && isInScale(midiToNoteName(midi - d), scaleNotes)) downDelta = d
    if (upDelta !== Infinity && downDelta !== Infinity) break
  }

  if (direction === 'up') return upDelta < Infinity ? midi + upDelta : midi
  if (direction === 'down') return downDelta < Infinity ? midi - downDelta : midi
  return upDelta <= downDelta ? midi + upDelta : midi - downDelta
}
