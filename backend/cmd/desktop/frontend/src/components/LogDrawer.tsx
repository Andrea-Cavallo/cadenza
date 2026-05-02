// @ts-ignore
import * as AppService from '../../wailsjs/go/main/AppService'

interface LogDrawerProps {
  open: boolean
  log: string[]
  files: string[]
}

export function LogDrawer({ open, log, files }: LogDrawerProps) {
  return (
    <section className={`log-drawer ${open ? 'open' : ''}`}>
      <div className="log-panel">
        <div className="log-head">
          <div>
            <div className="panel-title">Generation log</div>
            <div className="muted small">Live progress and backend Go logs</div>
          </div>
          {files.length > 0 && (
            <button type="button" className="btn ghost" onClick={() => AppService.OpenOutputFolder()}>
              Open folder
            </button>
          )}
        </div>
        <pre className="code log-code">
          {log.length === 0 ? '# Ready. Press Generate to start.' : log.join('\n')}
        </pre>
      </div>

      <div className="files-panel">
        <div className="panel-title">Output</div>
        {files.length === 0 ? (
          <div className="empty-state">No MIDI files yet</div>
        ) : (
          <div className="file-list">
            {files.map(file => (
              <div key={file} className="file-row mono">{file}</div>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
