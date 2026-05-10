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
              Runs open-source models on your machine. No API key required, fully private.
            </p>
            <ol className="info-steps">
              <li>Download and install Ollama from <code>ollama.com/download</code></li>
              <li>In a terminal, start the server: <code>ollama serve</code></li>
              <li>Pull a model: <code>ollama pull qwen2.5:7b</code></li>
              <li>Select <strong>Ollama</strong> in the Provider panel, then click <strong>Refresh</strong></li>
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
