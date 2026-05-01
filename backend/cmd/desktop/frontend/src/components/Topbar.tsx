export function Topbar() {
  return (
    <div className="topbar">
      <div className="brand">
        <div className="brand-mark">C</div>
        <span>cadenza</span>
        <span className="pill" style={{ marginLeft: 4 }}>v0.7.2</span>
        <span className="pill live">desktop</span>
      </div>
      <nav className="topnav">
        <a href="#presets">Presets</a>
        <a href="#generate">Generate</a>
        <a href="#pipeline">Pipeline</a>
      </nav>
      <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
        <a className="btn primary" href="#generate">Generate</a>
      </div>
    </div>
  )
}
