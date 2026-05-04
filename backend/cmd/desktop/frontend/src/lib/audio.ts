// AudioEngine: plays notes via Web Audio API.
// Falls back to a triangle-wave oscillator when WAV samples are not available.
//
// WAV sample convention (place files in /public/samples/):
//   {NoteName}{Octave}.wav  — e.g. C4.wav, A#3.wav, F5.wav
//   Sharps notation only (C#, D#, F#, G#, A#) — consistent with midiToNoteWithOctave.
//   Sparse packs are fine: if C4.wav is missing, the engine falls back to oscillator.

import { midiToNoteWithOctave } from './music'

export class AudioEngine {
  private ctx: AudioContext | null = null
  private masterGain: GainNode | null = null
  private sampleCache: Map<number, AudioBuffer | null> = new Map()
  private loadingPromises: Map<number, Promise<void>> = new Map()

  private getCtx(): AudioContext {
    if (!this.ctx) {
      this.ctx = new AudioContext()
      this.masterGain = this.ctx.createGain()
      this.masterGain.gain.value = 0.6
      this.masterGain.connect(this.ctx.destination)
    }
    if (this.ctx.state === 'suspended') {
      this.ctx.resume()
    }
    return this.ctx
  }

  // Pre-load a WAV sample for the given MIDI note from /samples/{note}.wav.
  // Silently stores null on any load error (oscillator fallback will be used).
  async preload(midi: number): Promise<void> {
    if (this.sampleCache.has(midi)) return
    if (this.loadingPromises.has(midi)) return this.loadingPromises.get(midi)

    const promise = (async () => {
      const ctx = this.getCtx()
      const noteName = midiToNoteWithOctave(midi)
      try {
        const res = await fetch(`/samples/${noteName}.wav`)
        if (!res.ok) throw new Error('not found')
        const arrayBuf = await res.arrayBuffer()
        const audioBuf = await ctx.decodeAudioData(arrayBuf)
        this.sampleCache.set(midi, audioBuf)
      } catch {
        this.sampleCache.set(midi, null)
      }
    })()

    this.loadingPromises.set(midi, promise)
    return promise
  }

  // Pre-load samples for an entire chord / scale range.
  preloadRange(midiValues: number[]): void {
    for (const midi of midiValues) {
      this.preload(midi)
    }
  }

  // Play a note immediately. Uses exact WAV sample if loaded, pitch-shifted
  // nearest sample if available, or falls back to a triangle-wave oscillator.
  playNote(midi: number, durationSec = 0.4, velocity = 0.7): void {
    const ctx = this.getCtx()
    const gain = this.masterGain!
    const exactSample = this.sampleCache.get(midi)

    if (exactSample) {
      this.playSample(ctx, exactSample, gain, durationSec, velocity, 1)
      return
    }

    const nearest = this.findNearestLoadedSample(midi)
    if (nearest !== null) {
      const buffer = this.sampleCache.get(nearest)!
      const playbackRate = Math.pow(2, (midi - nearest) / 12)
      this.playSample(ctx, buffer, gain, durationSec, velocity, playbackRate)
    } else {
      this.playOscillator(ctx, midi, gain, durationSec, velocity)
    }
    // Lazy preload so next play of this exact pitch uses its own sample
    this.preload(midi)
  }

  private findNearestLoadedSample(midi: number): number | null {
    let nearest: number | null = null
    let minDist = Infinity
    for (const [sampleMidi, buffer] of this.sampleCache.entries()) {
      if (buffer === null) continue
      const dist = Math.abs(sampleMidi - midi)
      if (dist < minDist) {
        minDist = dist
        nearest = sampleMidi
      }
    }
    return nearest
  }

  setVolume(value: number): void {
    if (this.masterGain) {
      this.masterGain.gain.value = Math.max(0, Math.min(1, value))
    }
  }

  dispose(): void {
    this.ctx?.close()
    this.ctx = null
    this.masterGain = null
    this.sampleCache.clear()
    this.loadingPromises.clear()
  }

  private playSample(
    ctx: AudioContext,
    buffer: AudioBuffer,
    destination: AudioNode,
    durationSec: number,
    velocity: number,
    playbackRate = 1,
  ): void {
    const source = ctx.createBufferSource()
    const gainNode = ctx.createGain()
    source.buffer = buffer
    source.playbackRate.value = playbackRate
    gainNode.gain.setValueAtTime(velocity, ctx.currentTime)
    gainNode.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + durationSec)
    source.connect(gainNode)
    gainNode.connect(destination)
    source.start()
    source.stop(ctx.currentTime + durationSec + 0.05)
  }

  private playOscillator(
    ctx: AudioContext,
    midi: number,
    destination: AudioNode,
    durationSec: number,
    velocity: number,
  ): void {
    const freq = 440 * Math.pow(2, (midi - 69) / 12)
    const osc = ctx.createOscillator()
    const gainNode = ctx.createGain()
    osc.type = 'triangle'
    osc.frequency.value = freq
    gainNode.gain.setValueAtTime(velocity * 0.4, ctx.currentTime)
    gainNode.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + durationSec)
    osc.connect(gainNode)
    gainNode.connect(destination)
    osc.start()
    osc.stop(ctx.currentTime + durationSec + 0.05)
  }
}

// Singleton — one engine for the entire app.
let _engine: AudioEngine | null = null

export function getAudioEngine(): AudioEngine {
  if (!_engine) _engine = new AudioEngine()
  return _engine
}
