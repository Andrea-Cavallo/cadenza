import { ProviderStatus } from '../types'

interface TitleBarProps {
  provider: string
  status: ProviderStatus | null
  running: boolean
  logOpen: boolean
  onToggleLog: () => void
  onToggleInfo: () => void
}

export function TitleBar({ provider, status, running, logOpen, onToggleLog, onToggleInfo }: TitleBarProps) {
  const ready = provider === 'offline' || status?.ready
  const statusText = running ? 'Generating' : ready ? 'Ready' : 'Setup needed'

  return (
    <header className="titlebar">
      <div className="titlebar-center">
        <span className={`status-dot ${ready ? 'ready' : 'warn'}`} />
        <span className="mono">{provider}</span>
        <span className="muted">/</span>
        <span>{statusText}</span>
      </div>

      <div className="titlebar-actions">
        <button type="button" className="icon-btn text" onClick={onToggleInfo}>
          Info
        </button>
        <button type="button" className="icon-btn text" onClick={onToggleLog}>
          {logOpen ? 'Hide log' : 'Show log'}
        </button>
      </div>
    </header>
  )
}
