import { Params, ProviderStatus } from '../types'

interface StatusBarProps {
  params: Params
  status: ProviderStatus | null
  files: string[]
  running: boolean
  lastSeed: string
  pinnedSeed: string
  onPinSeed: () => void
  onUnpinSeed: () => void
}

export function StatusBar({ params, status, files, running, lastSeed, pinnedSeed, onPinSeed, onUnpinSeed }: StatusBarProps) {
  const providerReady = params.provider === 'offline' || status?.ready
  const pinned = !!pinnedSeed

  return (
    <footer className="statusbar">
      <span className={`status-dot ${providerReady ? 'ready' : 'warn'}`} style={{ width: 6, height: 6 }} />
      <span className="mono">{params.bpm} bpm</span>
      <span className="sb-sep">·</span>
      <span>{params.key}</span>
      <span className="sb-sep">·</span>
      <span>{params.bars} bars</span>

      <span className="statusbar-fill" />

      {lastSeed && (
        <span className="seed-row">
          <span className="mono seed-val">{pinned ? '📌' : 'SEED'} {pinned ? pinnedSeed : lastSeed}</span>
          <button
            type="button"
            className="seed-btn"
            title={pinned ? 'Unpin seed (use random next time)' : 'Pin this seed (reproduce result)'}
            onClick={pinned ? onUnpinSeed : onPinSeed}
          >
            {pinned ? '✕' : '📌'}
          </button>
        </span>
      )}

      <span className="sb-sep">·</span>
      <span>{running ? 'rendering...' : files.length > 0 ? `${files.length} files` : 'idle'}</span>
    </footer>
  )
}
