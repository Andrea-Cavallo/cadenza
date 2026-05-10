import { Dispatch, SetStateAction, useEffect, useState } from 'react'
import { GenerateRequest, GenerateResult, GenerationPreview, Params, ProviderStatus } from './types'

// @ts-ignore
import * as AppService from '../wailsjs/go/main/AppService'
// @ts-ignore
import { EventsOn } from '../wailsjs/runtime/runtime'

const service = AppService as typeof AppService & {
  GetProviderStatus: (provider: string) => Promise<ProviderStatus>
  StartOllama: () => Promise<ProviderStatus>
  OpenProviderSetup: (provider: string) => Promise<void>
  PullOllamaModel: (model: string) => Promise<void>
}

interface UseGeneratorArgs {
  params: Params
  setParams: (p: Params) => void
  setLog: Dispatch<SetStateAction<string[]>>
  setFiles: Dispatch<SetStateAction<string[]>>
  setFilesAlt: Dispatch<SetStateAction<string[]>>
  setPreview: Dispatch<SetStateAction<GenerationPreview | null>>
  setPreviewAlt: Dispatch<SetStateAction<GenerationPreview | null>>
  setRunning: (r: boolean) => void
  setStatus: (s: ProviderStatus | null) => void
  setLastSeed: (s: string) => void
  status: ProviderStatus | null
  pinnedSeed: string
}

export function useGenerator({
  params,
  setParams,
  setLog,
  setFiles,
  setFilesAlt,
  setPreview,
  setPreviewAlt,
  setRunning,
  setStatus,
  setLastSeed,
  status,
  pinnedSeed,
}: UseGeneratorArgs) {
  const [statusLoading, setStatusLoading] = useState(false)
  const [pulling, setPulling] = useState(false)

  useEffect(() => {
    const off = EventsOn('progress', (msg: string) => setLog(prev => [...prev, msg]))
    const offLog = EventsOn('log', (msg: string) => setLog(prev => [...prev, msg]))
    return () => {
      if (typeof off === 'function') off()
      if (typeof offLog === 'function') offLog()
    }
  }, [setLog])

  const refreshProvider = async (nextProvider = params.provider) => {
    setStatusLoading(true)
    try {
      const result = await service.GetProviderStatus(nextProvider)
      setStatus(result)
      const choices = result.localModels?.length > 0 ? result.localModels : result.models
      if (!choices.some((e: { id: string }) => e.id === params.model)) {
        setParams({ ...params, provider: nextProvider, model: result.defaultModel || choices[0]?.id || '' })
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setStatus({
        provider: nextProvider,
        ready: false,
        installed: false,
        running: false,
        authConfigured: false,
        message: msg,
        setupHint: 'Check failed.',
        defaultModel: '',
        models: [],
        localModels: [],
      })
    } finally {
      setStatusLoading(false)
    }
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { refreshProvider(params.provider) }, [params.provider])

  const selectProvider = (nextProvider: string) => {
    setFiles([])
    setFilesAlt([])
    setPreview(null)
    setPreviewAlt(null)
    setLog([])
    setParams({ ...params, provider: nextProvider, model: '' })
  }

  const handleStartOllama = async () => {
    setStatusLoading(true)
    setLog(prev => [...prev, 'Starting Ollama...'])
    try {
      const result = await service.StartOllama()
      setStatus(result)
      setLog(prev => [...prev, result.message])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setLog(prev => [...prev, ...formatOperationalError(`Ollama start failed: ${msg}`, params)])
      await refreshProvider('ollama')
    } finally {
      setStatusLoading(false)
    }
  }

  const handlePullModel = async (model: string) => {
    setPulling(true)
    setLog(prev => [...prev, `Pulling ${model}...`])
    try {
      await service.PullOllamaModel(model)
      await refreshProvider('ollama')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setLog(prev => [...prev, ...formatOperationalError(`Pull failed: ${msg}`, params)])
    } finally {
      setPulling(false)
    }
  }

  const guardProvider = (p: Params): string[] | null => {
    if (p.provider === 'offline') return null
    if (p.provider === 'ollama' && !status?.running) {
      return [
        'Error: Ollama is not running.',
        'Action: click Start Ollama, then press Refresh and Generate again.',
      ]
    }
    if (p.provider !== 'ollama' && !status?.authConfigured) {
      return [
        `Error: API key missing for ${providerLabel(p.provider)}.`,
        'Action: click Setup, add the API key, restart Cadenza, then press Refresh.',
      ]
    }
    return null
  }

  const buildReq = (p: Params, extra?: Partial<GenerateRequest>): GenerateRequest => ({
    bpm: p.bpm,
    key: p.key,
    provider: p.provider,
    model: p.model,
    noLlm: p.provider === 'offline',
    bars: p.bars,
    groove: p.groove,
    offlineStyle: p.style,
    styleFamily: p.styleFamily || 'groove',
    seed: pinnedSeed || '',
    reuseChords: false,
    track: '',
    ...extra,
  })

  // overrideParams lets callers pass temporary values before React state settles.
  const handleGenerate = async (overrideParams?: Partial<Params>) => {
    const p = overrideParams ? { ...params, ...overrideParams } : params
    const err = guardProvider(p)
    if (err) { setLog(err); return }

    setRunning(true)
    setLog([])
    setFiles([])
    setFilesAlt([])
    setPreview(null)
    setPreviewAlt(null)
    try {
      const result: GenerateResult = await AppService.Generate(buildReq(p))
      setFiles(result.files)
      setPreview(result.preview)
      setLastSeed(result.seed)
      setLog(prev => [...prev, `Done in ${result.elapsed}`, `Seed ${result.seed}`])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setLog(prev => [...prev, ...formatOperationalError(msg, p)])
    } finally {
      setRunning(false)
    }
  }

  const handleSameHarmony = async () => {
    const err = guardProvider(params)
    if (err) { setLog(err); return }

    setRunning(true)
    setLog([])
    setFiles([])
    setFilesAlt([])
    setPreview(null)
    setPreviewAlt(null)
    try {
      const result: GenerateResult = await AppService.Generate(buildReq(params, { reuseChords: true, seed: '' }))
      setFiles(result.files)
      setPreview(result.preview)
      setLastSeed(result.seed)
      setLog(prev => [...prev, `New motifs — same harmony`, `Done in ${result.elapsed}`, `Seed ${result.seed}`])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setLog(prev => [...prev, ...formatOperationalError(msg, params)])
    } finally {
      setRunning(false)
    }
  }

  const handleRegenTrack = async (track: string) => {
    const err = guardProvider(params)
    if (err) { setLog(err); return }

    setRunning(true)
    setLog([`Regenerating ${track}...`])
    try {
      const result: GenerateResult = await AppService.Generate(buildReq(params, { track, seed: '' }))
      // Replace only the matching track in the current file list
      const prefix = `output_${track}`
      setFiles(prev => {
        const idx = prev.findIndex(f => (f.split(/[/\\]/).pop() ?? '').startsWith(prefix))
        if (idx === -1) return [...prev, ...result.files]
        const next = [...prev]
        next[idx] = result.files[0]
        return next
      })
      setPreview(prev => mergeTrackPreview(prev, result.preview))
      setLastSeed(result.seed)
      setLog(prev => [...prev, `Done — seed ${result.seed}`])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setLog(prev => [...prev, ...formatOperationalError(msg, params)])
    } finally {
      setRunning(false)
    }
  }

  const handleAB = async () => {
    const err = guardProvider(params)
    if (err) { setLog(err); return }

    setRunning(true)
    setLog(['Generating A...'])
    setFiles([])
    setFilesAlt([])
    setPreview(null)
    setPreviewAlt(null)
    try {
      const resultA: GenerateResult = await AppService.Generate(buildReq(params))
      setFiles(resultA.files)
      setPreview(resultA.preview)
      setLastSeed(resultA.seed)
      setLog(prev => [...prev, `A ready — seed ${resultA.seed}`, 'Generating B...'])

      const resultB: GenerateResult = await AppService.Generate(buildReq(params, { seed: '' }))
      setFilesAlt(resultB.files)
      setPreviewAlt(resultB.preview)
      setLog(prev => [...prev, `B ready — seed ${resultB.seed}`])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setLog(prev => [...prev, ...formatOperationalError(msg, params)])
    } finally {
      setRunning(false)
    }
  }

  const modelChoices = status?.localModels?.length ? status.localModels : (status?.models ?? [])
  const canGenerate = params.provider === 'offline' || !!status?.ready

  return {
    statusLoading,
    pulling,
    modelChoices,
    canGenerate,
    refreshProvider,
    selectProvider,
    handleStartOllama,
    handlePullModel,
    handleGenerate,
    handleSameHarmony,
    handleRegenTrack,
    handleAB,
    openProviderSetup: (p: string) => service.OpenProviderSetup(p),
  }
}

function mergeTrackPreview(current: GenerationPreview | null, update: GenerationPreview): GenerationPreview {
  if (!current) return update
  const byType = new Map(current.patterns.map(pattern => [pattern.patternType, pattern]))
  for (const pattern of update.patterns) {
    byType.set(pattern.patternType, pattern)
  }
  const order = ['bassline', 'arpeggio', 'melody']
  return {
    ...current,
    patterns: order
      .map(patternType => byType.get(patternType))
      .filter((pattern): pattern is NonNullable<typeof pattern> => Boolean(pattern)),
  }
}

function formatOperationalError(message: string, params: Params): string[] {
  const details = message.trim()
  const lower = details.toLowerCase()
  if (lower.includes('ollama') && (lower.includes('not running') || lower.includes('localhost:11434'))) {
    return [
      'Error: Ollama is not running.',
      'Action: click Start Ollama, then press Refresh and Generate again.',
      `Details: ${details}`,
    ]
  }
  if (lower.includes('api key') || lower.includes('anthropic') || lower.includes('openai') || lower.includes('gemini')) {
    return [
      `Error: ${providerLabel(params.provider)} is missing a working API key.`,
      'Action: click Setup, add the API key, restart Cadenza, then press Refresh.',
      `Details: ${details}`,
    ]
  }
  if (lower.includes('no previous session')) {
    return [
      'Error: there is no previous session to reuse.',
      'Action: press Generate once, then use Regenerate bass/arp/melody.',
      `Details: ${details}`,
    ]
  }
  if (lower.includes('invalid key')) {
    return [
      `Error: key "${params.key}" is not valid.`,
      'Action: choose a supported key or mode, for example Am, Dm-dorian, or G-mixolydian.',
      `Details: ${details}`,
    ]
  }
  if (lower.includes('output dir') || lower.includes('permission denied')) {
    return [
      'Error: Cadenza cannot write to the output folder.',
      'Action: check folder permissions, then retry Generate or open the output folder from the log drawer.',
      `Details: ${details}`,
    ]
  }
  return [
    `Error: ${details}`,
    'Action: open the log drawer for details. If the selected provider keeps failing, switch to Offline and retry.',
  ]
}

function providerLabel(provider: string): string {
  switch (provider) {
    case 'claude':
      return 'Claude'
    case 'openai':
      return 'OpenAI'
    case 'gemini':
      return 'Gemini'
    case 'ollama':
      return 'Ollama'
    default:
      return provider || 'Provider'
  }
}
