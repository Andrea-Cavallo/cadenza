import { useEffect, useState } from 'react'
import { InfoModal } from './components/InfoModal'
import { LogDrawer } from './components/LogDrawer'
import { PianoRoll } from './components/PianoRoll'
import { Sidebar } from './components/Sidebar'
import { StatusBar } from './components/StatusBar'
import { TitleBar } from './components/TitleBar'
import { GenerationPreview, Params, ProviderStatus } from './types'

// @ts-ignore
import * as AppService from '../wailsjs/go/main/AppService'

export default function App() {
  const [params, setParams] = useState<Params>({
    bpm: 122,
    key: 'Am-dorian',
    bars: 16,
    groove: 'mpc60',
    style: 'melodic',
    styleFamily: 'groove',
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
  const [infoOpen, setInfoOpen] = useState(false)
  const [outputDir, setOutputDir] = useState('')
  const [lastSeed, setLastSeed] = useState('')
  const [pinnedSeed, setPinnedSeed] = useState('')

  useEffect(() => {
    AppService.GetOutputDir().then((dir: string) => setOutputDir(dir)).catch(() => {})
  }, [])

  useEffect(() => {
    const last = log[log.length - 1] ?? ''
    if (last.startsWith('Error:') || last.includes('[warn]')) setLogOpen(true)
  }, [log])

  const handleChooseOutputDir = async () => {
    try {
      const dir = await AppService.ChooseOutputDir()
      if (dir) setOutputDir(dir)
    } catch {}
  }

  const handleOpenOutputFolder = async () => {
    try {
      await AppService.OpenOutputFolder()
    } catch {}
  }

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

  const updateDisplayPreview = (next: GenerationPreview) => {
    if (abView === 'b' && previewAlt) {
      setPreviewAlt(next)
      return
    }
    setPreview(next)
  }

  const exportEditedPreview = async () => {
    if (!displayPreview) {
      setLog(lines => [...lines, 'Error: Nothing to export.', 'Action: Generate MIDI first, then edit the piano roll.'])
      setLogOpen(true)
      return
    }
    try {
      setLog(lines => [...lines, '[write] exporting edited piano roll'])
      const result = await AppService.ExportEditedPreview({ bpm: params.bpm, preview: displayPreview } as any)
      if (abView === 'b') {
        setFilesAlt(result.files)
      } else {
        setFiles(result.files)
      }
      setLog(lines => [...lines, `[write] exported ${result.files.length} edited MIDI file(s)`])
    } catch (err) {
      setLog(lines => [
        ...lines,
        'Error: Edited export failed.',
        'Action: Check that the output folder is writable, then press Export edited again.',
        `Details: ${err instanceof Error ? err.message : String(err)}`,
      ])
      setLogOpen(true)
    }
  }

  return (
    <div className="app-shell">
      <TitleBar
        provider={params.provider}
        status={status}
        running={running}
        logOpen={logOpen}
        onToggleLog={() => setLogOpen(o => !o)}
        onToggleInfo={() => setInfoOpen(o => !o)}
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
          outputDir={outputDir}
          onChooseOutputDir={handleChooseOutputDir}
          onOpenOutputFolder={handleOpenOutputFolder}
        />

        <section className="workspace">
          <div className="roll-wrap">
            <PianoRoll
              bpm={params.bpm}
              playing={running}
              preview={displayPreview}
              kept={kept}
              onPreviewChange={updateDisplayPreview}
              onKeep={() => setKept(true)}
              onDiscard={discardCurrent}
              onExportEdited={exportEditedPreview}
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

      <InfoModal open={infoOpen} onClose={() => setInfoOpen(false)} />
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
