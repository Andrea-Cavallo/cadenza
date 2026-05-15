import { useEffect } from 'react'

interface FallbackWarningModalProps {
  open: boolean
  onClose: () => void
  warnings: string[]
}

export function FallbackWarningModal({ open, onClose, warnings }: FallbackWarningModalProps) {
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [open, onClose])

  if (!open || warnings.length === 0) return null

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-panel" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <span className="modal-title" style={{ color: 'var(--warning)' }}>Fallback Warning</span>
          <button type="button" className="modal-close" onClick={onClose}>✕</button>
        </div>

        <div className="modal-body">
          <section className="info-section">
            <div className="info-section-title" style={{ color: 'var(--warning)' }}>
              LLM provider failed — offline fallback used
            </div>
            <p className="info-text">
              Cadenza fell back to the <strong>offline engine</strong> because the selected provider
              returned an error. The generated MIDI was produced by the built-in algorithmic engine
              instead of the LLM.
            </p>
          </section>

          <section className="info-section">
            <div className="info-section-title">Warnings</div>
            {warnings.map((w, i) => (
              <p key={i} className="info-text" style={{
                background: 'var(--bg)',
                border: '1px solid var(--border)',
                padding: '8px 12px',
                fontFamily: 'var(--mono)',
                fontSize: 10,
                color: 'var(--warning)',
              }}>
                {w}
              </p>
            ))}
          </section>

          <section className="info-section">
            <div className="info-section-title">What to do</div>
            <ol className="info-steps">
              <li>Check that your API key is valid and has credits</li>
              <li>Verify the provider service is operational (status pages)</li>
              <li>Try a different provider or switch to <strong>Offline</strong> mode explicitly</li>
              <li>Press <strong>Generate</strong> again to retry with the LLM</li>
            </ol>
          </section>
        </div>
      </div>
    </div>
  )
}
