import { Dispatch, SetStateAction } from 'react'
import { useGenerator } from '../useGenerator'
import { GenerationPreview, Params, ProviderStatus } from '../types'

const KEYS = [
  'Am', 'Bm', 'Cm', 'Dm', 'Em', 'Fm', 'Gm',
  'C', 'D', 'E', 'F', 'G', 'A', 'B',
  'F#m', 'C#m', 'Bbm',
  'Am-dorian', 'Bm-dorian', 'Em-dorian',
  'Em-phrygian', 'Bm-phrygian',
  'G-mixolydian', 'A-mixolydian', 'D-mixolydian',
  'C-lydian', 'D-lydian',
]
const BARS_OPTS = [16, 32, 64, 128]
const PROVIDERS = ['offline', 'claude', 'ollama', 'openai', 'gemini']

const PROVIDER_KEY_NAMES: Record<string, string> = {
  claude: 'ANTHROPIC_API_KEY',
  openai: 'OPENAI_API_KEY',
  gemini: 'GEMINI_API_KEY',
}

interface SidebarProps {
  params: Params
  setParams: (p: Params) => void
  setLog: Dispatch<SetStateAction<string[]>>
  setFiles: Dispatch<SetStateAction<string[]>>
  setFilesAlt: Dispatch<SetStateAction<string[]>>
  setPreview: Dispatch<SetStateAction<GenerationPreview | null>>
  setPreviewAlt: Dispatch<SetStateAction<GenerationPreview | null>>
  files: string[]
  running: boolean
  setRunning: (r: boolean) => void
  status: ProviderStatus | null
  setStatus: (s: ProviderStatus | null) => void
  pinnedSeed: string
  setLastSeed: (s: string) => void
}

export function Sidebar({
  params, setParams, setLog, setFiles, setFilesAlt, setPreview, setPreviewAlt, files,
  running, setRunning, status, setStatus,
  pinnedSeed, setLastSeed,
}: SidebarProps) {
  const {
    statusLoading, pulling, modelChoices, canGenerate,
    refreshProvider, selectProvider, handleStartOllama,
    handlePullModel, handleGenerate, handleSameHarmony,
    handleRegenTrack, handleAB, openProviderSetup,
  } = useGenerator({
    params, setParams, setLog, setFiles, setFilesAlt,
    setPreview, setPreviewAlt,
    setRunning, setStatus, setLastSeed, status, pinnedSeed,
  })

  const providerReady = params.provider === 'offline' || !!status?.ready
  const isCloudProvider = PROVIDER_KEY_NAMES[params.provider] !== undefined
  const keyName = PROVIDER_KEY_NAMES[params.provider]
  const ollamaRunningNoModels = params.provider === 'ollama' && !!status?.running && !status?.ready
  const hasFiles = files.length > 0
  const providerSummary = providerStatusSummary(params.provider, status)
  const providerDetail = providerStatusDetail(params.provider, status)

  const setBPM = (next: number) => {
    const bpm = Math.max(80, Math.min(150, Math.round(next || 80)))
    setParams({ ...params, bpm })
  }

  return (
    <aside className="sidebar">
      <div className="sidebar-scroll">
        <div className="sidebar-section">
          <div className="panel-title">Session</div>
          <div className="session-grid">
            <label className="sb-control">
              <span className="sb-label">Key &amp; mode</span>
              <select value={params.key} onChange={e => setParams({ ...params, key: e.target.value })}>
                {KEYS.map(k => <option key={k} value={k}>{k}</option>)}
              </select>
            </label>

            <label className="sb-control">
              <span className="sb-label">BPM</span>
              <div className="bpm-box">
                <button type="button" onClick={() => setBPM(params.bpm - 1)} disabled={running}>-</button>
                <input
                  type="number"
                  min="80"
                  max="150"
                  step="1"
                  value={params.bpm}
                  onChange={e => setBPM(parseInt(e.target.value, 10))}
                />
                <button type="button" onClick={() => setBPM(params.bpm + 1)} disabled={running}>+</button>
              </div>
            </label>

            <label className="sb-control">
              <span className="sb-label">Bars</span>
              <select value={params.bars} onChange={e => setParams({ ...params, bars: parseInt(e.target.value, 10) })}>
                {BARS_OPTS.map(b => <option key={b} value={b}>{b}</option>)}
              </select>
            </label>
          </div>
        </div>

        <div className="sidebar-section provider-section">
          <div className="provider-head">
            <div className="panel-title">Provider</div>
            <button
              type="button"
              className="refresh-btn labeled"
              title="Refresh provider status"
              onClick={() => refreshProvider()}
              disabled={statusLoading}
            >
              Refresh
            </button>
          </div>

          <div className="provider-pills all-providers">
            {PROVIDERS.map(p => (
              <button
                key={p}
                type="button"
                className={`pill ${params.provider === p ? 'on' : ''}`}
                onClick={() => selectProvider(p)}
              >
                {p}
              </button>
            ))}
          </div>

          <div className="provider-card">
            <div className="provider-status-line">
              <span className={`dot ${providerReady ? 'ready' : 'warn'}`} />
              <strong className="mono">{providerSummary}</strong>
            </div>
            {providerDetail && <div className="provider-detail mono small">{providerDetail}</div>}

            {isCloudProvider && (
              <div className="provider-detail mono small">
                {status?.authConfigured ? `${keyName} configured` : `${keyName} missing`}
              </div>
            )}

            {params.provider !== 'offline' && (
              <label className="sb-control provider-model">
                <span className="sb-label">{params.provider === 'ollama' ? 'Local model' : 'Model'}</span>
                <select
                  value={params.model}
                  onChange={e => setParams({ ...params, model: e.target.value })}
                  disabled={modelChoices.length === 0}
                >
                  {modelChoices.length === 0 && <option value="">No models available</option>}
                  {modelChoices.map(m => <option key={m.id} value={m.id}>{m.name || m.id}</option>)}
                </select>
              </label>
            )}

            <div className="provider-actions-grid">
              {params.provider === 'ollama' && !status?.running && (
                <button type="button" className="btn ghost" onClick={handleStartOllama} disabled={statusLoading}>
                  Start Ollama
                </button>
              )}
              {ollamaRunningNoModels && (
                <button
                  type="button"
                  className="btn ghost"
                  onClick={() => handlePullModel('qwen2.5:7b')}
                  disabled={pulling || running}
                >
                  {pulling ? 'Pulling...' : 'Pull qwen2.5:7b'}
                </button>
              )}
              {isCloudProvider && !status?.authConfigured && (
                <button type="button" className="btn ghost" onClick={() => openProviderSetup(params.provider)}>
                  Setup
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      <div className="sidebar-generate">
        <button
          type="button"
          className={`btn primary generate-btn${running ? ' running' : ''}`}
          onClick={() => handleGenerate()}
          disabled={running || !canGenerate}
        >
          {running ? 'Generating...' : 'Generate'}
        </button>

        {hasFiles && !running && (
          <div className="quick-actions">
            <button
              type="button"
              className="qa-btn qa-wide"
              title="Keep the same chord progression, generate new motifs"
              onClick={handleSameHarmony}
              disabled={running}
            >
              Same harmony
            </button>
            <div className="qa-tracks">
              {(['bassline', 'arpeggio', 'melody'] as const).map(track => (
                <button
                  key={track}
                  type="button"
                  className="qa-btn"
                  title={`Regenerate ${track} only`}
                  onClick={() => handleRegenTrack(track)}
                  disabled={running}
                >
                  {track === 'bassline' ? 'Bass' : track === 'arpeggio' ? 'Arp' : 'Mel'}
                </button>
              ))}
              <button
                type="button"
                className="qa-btn qa-ab"
                title="Generate two variations for comparison"
                onClick={handleAB}
                disabled={running}
              >
                A/B
              </button>
            </div>
          </div>
        )}
      </div>
    </aside>
  )
}

function providerStatusSummary(provider: string, status: ProviderStatus | null): string {
  if (provider === 'offline') return 'Offline ready'
  if (!status) return 'Checking provider'
  if (provider === 'ollama') {
    if (!status.installed) return 'Ollama not installed'
    if (!status.running) return 'Ollama stopped'
    if (status.ready) return 'Ollama ready'
    return 'Ollama needs a model'
  }
  if (status.ready) return `${providerLabel(provider)} ready`
  return `${providerLabel(provider)} not configured`
}

function providerStatusDetail(provider: string, status: ProviderStatus | null): string {
  if (provider === 'offline') return 'No API key needed. Deterministic local generation.'
  if (!status) return 'Reading provider status...'
  if (provider === 'ollama' && status.localModels?.length > 0) {
    return `${status.localModels.length} local model${status.localModels.length === 1 ? '' : 's'} available`
  }
  return status.setupHint || status.message
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
