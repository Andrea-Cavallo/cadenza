// @ts-ignore
import * as AppService from '../../wailsjs/go/main/AppService'

const STEM_LABELS: Record<string, string> = {
  'bassline-groove': 'Bass Groove',
  'bassline-rolling': 'Bass Rolling',
  'bassline-sub': 'Bass Sub',
  'arp': 'Arpeggio',
  'melody': 'Melody',
  'chord-pad': 'Chord Pad',
  'lead': 'Lead',
}

function stemLabel(filePath: string): string {
  const name = filePath.replace(/\\/g, '/').split('/').pop() ?? filePath
  // output_<stem>_<key>_<bpm>_<ts>.mid
  const m = name.match(/^output_([^_]+(?:-[^_]+)*)_/)
  if (m) return STEM_LABELS[m[1]] ?? m[1]
  return name.replace(/\.mid$/i, '')
}

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
        <div className="panel-title">Output stems</div>
        {files.length === 0 ? (
          <div className="empty-state">No MIDI files yet</div>
        ) : (
          <div className={`file-list${files.length >= 5 ? ' file-list-grid' : ''}`}>
            {files.map(file => (
              <button
                key={file}
                type="button"
                className="file-row stem-btn"
                title={file}
                onClick={() => AppService.OpenOutputFolder()}
              >
                <span className="stem-label">{stemLabel(file)}</span>
                <span className="stem-ext mono muted">.mid</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
