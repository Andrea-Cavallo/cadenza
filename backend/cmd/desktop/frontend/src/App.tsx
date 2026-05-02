import { useEffect, useState } from 'react'
import { LogDrawer } from './components/LogDrawer'
import { PianoRoll } from './components/PianoRoll'
import { Sidebar } from './components/Sidebar'
import { StatusBar } from './components/StatusBar'
import { TitleBar } from './components/TitleBar'
import { AccentName, GenerationPreview, Params, ProviderStatus } from './types'

// @ts-ignore
import * as AppService from '../wailsjs/go/main/AppService'

const ACCENT_MAP: Record<AccentName, string> = {
  cyan: '#00E0FF',
  lime: '#01FF95',
  amber: '#FFB000',
}

export default function App() {
  const [accent, setAccent] = useState<AccentName>('cyan')
  const [params, setParams] = useState<Params>({
    bpm: 122,
    key: 'Am-dorian',
    bars: 16,
    groove: 'mpc60',
    style: 'melodic',
    provider: 'offline',
    model: '',
    drums: false,
  })
  const [log, setLog] = useState<string[]>([])
  const [files, setFiles] = useState<string[]>([])
  const [filesAlt, setFilesAlt] = useState<string[]>([])
  const [preview, setPreview] = useState<GenerationPreview | null>(null)
  const [previewAlt, setPreviewAlt] = useState<GenerationPreview | null>(null)
  const [abView, setAbView] = useState<'a' | 'b'>('a')
  const [kept, setKept] = useState(false)
  const [running, setRunning] = useState(false)
  const [status, setStatus] = useState<ProviderStatus | null>(null)
  const [logOpen, setLogOpen] = useState(false)
  const [lastSeed, setLastSeed] = useState('')
  const [pinnedSeed, setPinnedSeed] = useState('')

  useEffect(() => {
    document.documentElement.style.setProperty('--accent', ACCENT_MAP[accent])
  }, [accent])

  useEffect(() => {
    const last = log[log.length - 1] ?? ''
    if (last.startsWith('Error:')) setLogOpen(true)
  }, [log])

  useEffect(() => {
    if (filesAlt.length === 0) setAbView('a')
  }, [filesAlt])

  const displayFiles = abView === 'b' && filesAlt.length > 0 ? filesAlt : files
  const displayPreview = abView === 'b' && previewAlt ? previewAlt : preview

  const discardCurrent = () => {
    if (abView === 'b') {
      setFilesAlt([])
      setPreviewAlt(null)
      setAbView('a')
      return
    }
    setFiles([])
    setPreview(null)
    setKept(false)
  }

  return (
    <div className="app-shell">
      <TitleBar
        accent={accent}
        onAccentChange={setAccent}
        provider={params.provider}
        status={status}
        running={running}
        logOpen={logOpen}
        onToggleLog={() => setLogOpen(o => !o)}
      />

      <main className="app-main">
        <Sidebar
          params={params}
          setParams={setParams}
          setLog={setLog}
          setFiles={setFiles}
          setFilesAlt={setFilesAlt}
          setPreview={setPreview}
          setPreviewAlt={setPreviewAlt}
          files={files}
          running={running}
          setRunning={setRunning}
          status={status}
          setStatus={setStatus}
          pinnedSeed={pinnedSeed}
          setLastSeed={setLastSeed}
        />

        <section className="workspace">
          <div className="roll-wrap">
            <PianoRoll
              bpm={params.bpm}
              playing={running}
              preview={displayPreview}
              kept={kept}
              onKeep={() => setKept(true)}
              onDiscard={discardCurrent}
              onExportAll={() => AppService.OpenOutputFolder()}
            />

            {!running && filesAlt.length > 0 && (
              <div className="ab-toggle">
                <button
                  type="button"
                  className={abView === 'a' ? 'on' : ''}
                  onClick={() => setAbView('a')}
                >A</button>
                <button
                  type="button"
                  className={abView === 'b' ? 'on' : ''}
                  onClick={() => setAbView('b')}
                >B</button>
              </div>
            )}

            <div className="roll-overlay">
              <span><span>BPM</span> <strong>{params.bpm}</strong></span>
              <span><span>KEY</span> <strong>{params.key}</strong></span>
              <span><span>BARS</span> <strong>{params.bars}</strong></span>
            </div>
          </div>
        </section>
      </main>

      <LogDrawer open={logOpen} log={log} files={displayFiles} />
      <StatusBar
        params={params}
        status={status}
        files={displayFiles}
        running={running}
        lastSeed={lastSeed}
        pinnedSeed={pinnedSeed}
        onPinSeed={() => setPinnedSeed(lastSeed)}
        onUnpinSeed={() => setPinnedSeed('')}
      />
    </div>
  )
}
