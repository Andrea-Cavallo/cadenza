import { useCallback, useEffect, useRef } from 'react'
import { AudioEngine, getAudioEngine } from '../lib/audio'

interface UseAudioOptions {
  // MIDI values to pre-load on mount (e.g. the current pattern's active notes)
  preloadMidi?: number[]
  volume?: number
}

interface UseAudioReturn {
  playNote: (midi: number, durationSec?: number, velocity?: number) => void
  preloadRange: (midiValues: number[]) => void
  setVolume: (v: number) => void
}

export function useAudio(options: UseAudioOptions = {}): UseAudioReturn {
  const engineRef = useRef<AudioEngine>(getAudioEngine())
  const { preloadMidi, volume = 0.6 } = options

  useEffect(() => {
    engineRef.current.setVolume(volume)
  }, [volume])

  useEffect(() => {
    if (preloadMidi && preloadMidi.length > 0) {
      engineRef.current.preloadRange(preloadMidi)
    }
  }, [preloadMidi])

  const playNote = useCallback(
    (midi: number, durationSec = 0.4, velocity = 0.7) => {
      engineRef.current.playNote(midi, durationSec, velocity)
    },
    [],
  )

  const preloadRange = useCallback((midiValues: number[]) => {
    engineRef.current.preloadRange(midiValues)
  }, [])

  const setVolume = useCallback((v: number) => {
    engineRef.current.setVolume(v)
  }, [])

  return { playNote, preloadRange, setVolume }
}
