import { useEffect } from 'react'

interface InfoModalProps {
  open: boolean
  onClose: () => void
}

export function InfoModal({ open, onClose }: InfoModalProps) {
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-panel" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <span className="modal-title">Setup Guide</span>
          <button type="button" className="modal-close" onClick={onClose}>✕</button>
        </div>

        <div className="modal-body">
          <section className="info-section">
            <div className="info-section-title">Offline Mode</div>
            <p className="info-text">
              Generates MIDI patterns entirely on your machine — no API key, no internet
              connection required. Patterns are seeded: the same seed always produces the
              same output. Use the <strong>Offline flavor</strong> selector (Melodic,
              Hypnotic, Driving, Minimal) to control the style. This is the fastest mode
              and always available.
            </p>
          </section>

          <section className="info-section">
            <div className="info-section-title">Ollama (Local LLM)</div>
            <p className="info-text">
              Runs open-source models entirely on your machine — no API key, no internet
              after the one-time model download, 100% private.
            </p>

            <div className="info-platform-label">Step 1 — Install Ollama</div>

            <span className="info-platform-label" style={{ fontSize: 9, borderColor: 'var(--fg-muted)', color: 'var(--fg-muted)', marginTop: 6 }}>macOS</span>
            <ol className="info-steps">
              <li>
                Download the <code>.dmg</code> from{' '}
                <code>ollama.com/download</code> and open it,<br />
                <em>or</em> via Homebrew: <code>brew install ollama</code>
              </li>
              <li>The Ollama icon appears in the menu bar — the server starts automatically on <code>http://localhost:11434</code></li>
            </ol>

            <span className="info-platform-label" style={{ fontSize: 9, borderColor: 'var(--fg-muted)', color: 'var(--fg-muted)', marginTop: 6 }}>Windows</span>
            <ol className="info-steps">
              <li>Download <code>OllamaSetup.exe</code> from <code>ollama.com/download</code> and run it</li>
              <li>Ollama installs as a background service on <code>http://localhost:11434</code> — no extra config needed</li>
              <li>If Windows Firewall prompts, allow access on <strong>private networks only</strong></li>
            </ol>

            <div className="info-platform-label">Step 2 — Download a model</div>
            <p className="info-text">Open a terminal (macOS: Terminal; Windows: PowerShell) and run one of:</p>
            <ol className="info-steps">
              <li><code>ollama pull gemma</code> — Google Gemma, fast + great musical JSON <strong>(recommended)</strong></li>
              <li><code>ollama pull qwen2.5:7b</code> — strong instruction following</li>
              <li><code>ollama pull llama3.2</code> — lightest, best for CPU-only machines</li>
              <li><code>ollama pull mistral</code> — solid all-rounder</li>
            </ol>
            <p className="info-note">Download is one-time (~2–8 GB). Check installed models: <code>ollama list</code></p>

            <div className="info-platform-label">Step 3 — Start the server</div>
            <p className="info-text">
              <strong>macOS:</strong> starts automatically when the menu-bar app launches.<br />
              <strong>Windows:</strong> starts automatically after installation.<br />
              To start manually on either platform: <code>ollama serve</code>
            </p>
            <p className="info-note">
              Verify it is running: open a browser or terminal and check{' '}
              <code>http://localhost:11434/api/tags</code> — you should see a JSON list of your models.
            </p>

            <div className="info-platform-label">Step 4 — Use with Cadenza</div>
            <ol className="info-steps">
              <li>Select <strong>Ollama</strong> in the Provider panel on the left</li>
              <li>Click <strong>Refresh</strong> — Cadenza will detect the running server and list your models</li>
              <li>Pick a model from the dropdown and click <strong>Generate</strong></li>
            </ol>
          </section>

          <section className="info-section">
            <div className="info-section-title">Claude (Anthropic)</div>
            <p className="info-text">
              Uses Anthropic's Claude API for highest-quality generation.
            </p>
            <ol className="info-steps">
              <li>Create an API key at <code>console.anthropic.com/settings/keys</code></li>
              <li>Set the environment variable before launching Cadenza:<br /><code>ANTHROPIC_API_KEY=sk-ant-...</code></li>
              <li>Restart Cadenza, then select <strong>Claude</strong> in the Provider panel</li>
            </ol>
          </section>

          <section className="info-section">
            <div className="info-section-title">OpenAI</div>
            <p className="info-text">
              Uses OpenAI's API (GPT-4o and compatible models).
            </p>
            <ol className="info-steps">
              <li>Create an API key at <code>platform.openai.com/api-keys</code></li>
              <li>Set the environment variable before launching Cadenza:<br /><code>OPENAI_API_KEY=sk-...</code></li>
              <li>Restart Cadenza, then select <strong>OpenAI</strong> in the Provider panel</li>
            </ol>
          </section>

          <section className="info-section">
            <div className="info-section-title">Gemini (Google)</div>
            <p className="info-text">
              Uses Google's Gemini API.
            </p>
            <ol className="info-steps">
              <li>Create an API key at <code>aistudio.google.com/app/apikey</code></li>
              <li>Set the environment variable before launching Cadenza:<br /><code>GEMINI_API_KEY=AI...</code></li>
              <li>Restart Cadenza, then select <strong>Gemini</strong> in the Provider panel</li>
            </ol>
          </section>
        </div>
      </div>
    </div>
  )
}
