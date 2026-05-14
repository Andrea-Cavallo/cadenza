import { Component, ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100vh',
          background: 'var(--bg)',
          color: 'var(--fg-dim)',
          fontFamily: 'var(--mono)',
          gap: 16,
          padding: 40,
          textAlign: 'center',
        }}>
          <div style={{
            fontSize: 48,
            color: 'var(--danger)',
            marginBottom: 8,
          }}>!</div>
          <div style={{
            fontSize: 13,
            fontWeight: 700,
            letterSpacing: '0.12em',
            textTransform: 'uppercase',
            color: 'var(--fg)',
          }}>
            Something went wrong
          </div>
          <div style={{
            fontSize: 11,
            color: 'var(--fg-muted)',
            maxWidth: 480,
            lineHeight: 1.6,
          }}>
            {this.state.error?.message ?? 'An unexpected error occurred.'}
          </div>
          <button
            type="button"
            className="btn primary"
            style={{ marginTop: 8 }}
            onClick={() => {
              this.setState({ hasError: false, error: null })
              window.location.reload()
            }}
          >
            Reload
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
