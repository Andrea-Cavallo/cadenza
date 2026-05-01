// Cadenza dashboard — composition root.

const { useState, useEffect } = React;

function Topbar() {
  return (
    <div className="topbar">
      <div className="brand">
        <div className="brand-mark">C</div>
        <span>cadenza</span>
        <span className="pill" style={{ marginLeft: 4 }}>v0.7.2</span>
        <span className="pill live">api · beta</span>
      </div>
      <nav className="topnav">
        <a href="#presets">Presets</a>
        <a href="#generate">Generate</a>
        <a href="#api">API</a>
        <a href="#pipeline">Pipeline</a>
        <a href="#install">Install</a>
      </nav>
      <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
        <a className="btn ghost" href="#install">Docs</a>
        <a className="btn primary" href="#generate">Generate ↗</a>
      </div>
    </div>
  );
}

function Hero() {
  return (
    <section style={{ padding: "70px 0 40px" }}>
      <div className="shell" style={{ marginBottom: 36 }}>
        <div className="eyebrow" style={{ marginBottom: 22 }}>
          MIDI generation engine · for modern electronic music
        </div>
        <h1 className="hero-title">
          Coherent MIDI<br/>for the floor — <em>generated.</em>
        </h1>
        <div style={{
          display: "grid", gridTemplateColumns: "1.4fr 1fr",
          gap: 60, marginTop: 36, alignItems: "end",
        }}>
          <p style={{
            fontSize: 17, lineHeight: 1.55, color: "var(--fg-dim)",
            maxWidth: "60ch", margin: 0,
          }}>
            Cadenza renders <span className="mono" style={{ color: "var(--fg)" }}>bassline</span>,{" "}
            <span className="mono" style={{ color: "var(--fg)" }}>arpeggio</span> and{" "}
            <span className="mono" style={{ color: "var(--fg)" }}>melody</span> tracks
            that share one progression and one seed — engineered for tech house, melodic
            techno, peak-time, and afterhours sets. CLI today, HTTP service next.
          </p>
          <div className="kv">
            <div className="k">tempo</div><div className="v">80 — 150 bpm</div>
            <div className="k">scales</div><div className="v">major · minor · dorian · phrygian · mixolydian · lydian</div>
            <div className="k">grooves</div><div className="v">straight · mpc60 · linndrum · humanize</div>
            <div className="k">providers</div><div className="v">claude · ollama · openai · gemini</div>
          </div>
        </div>
      </div>

      <div className="shell">
        <div className="roll-wrap">
          <PianoRoll />
          <div className="roll-overlay">
            <div className="meta">
              <span><span style={{ color: "var(--fg-muted)" }}>BPM</span> <span className="v">122</span></span>
              <span><span style={{ color: "var(--fg-muted)" }}>KEY</span> <span className="v">Am-dorian</span></span>
              <span><span style={{ color: "var(--fg-muted)" }}>BARS</span> <span className="v">16</span></span>
              <span><span style={{ color: "var(--fg-muted)" }}>SEED</span> <span className="v">1714560219</span></span>
            </div>
            <div />
            <div className="meta" style={{ justifyContent: "flex-end" }}>
              <span><span style={{ color: "var(--fg-muted)" }}>BASSLINE</span></span>
              <span><span style={{ color: "var(--fg-muted)" }}>ARP</span></span>
              <span><span style={{ color: "var(--accent)" }}>● MELODY</span></span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function Section({ id, eyebrow, title, sub, children }) {
  return (
    <section id={id}>
      <div className="shell">
        <div className="section-head">
          <div>
            <div className="eyebrow" style={{ marginBottom: 14 }}>{eyebrow}</div>
            <h2 dangerouslySetInnerHTML={{ __html: title }} />
          </div>
          <p>{sub}</p>
        </div>
        {children}
      </div>
    </section>
  );
}

function Footer() {
  return (
    <footer id="install">
      <div className="foot-grid">
        <div>
          <div className="brand" style={{ marginBottom: 16 }}>
            <div className="brand-mark">C</div>
            <span className="mono">cadenza</span>
          </div>
          <p style={{ color: "var(--fg-dim)", fontSize: 13.5, lineHeight: 1.6, maxWidth: "44ch", margin: 0 }}>
            A MIDI generation engine for modern electronic music producers — open-source CLI,
            production-ready HTTP service. Built in Go.
          </p>
          <pre className="code" style={{ marginTop: 22, maxWidth: 480 }}>
{`# install
go install github.com/Andrea-Cavallo/cadenza/cmd/cadenza@latest

# first run
cadenza --bpm 122 --key Am --no-llm`}
          </pre>
        </div>
        <div>
          <h5>Product</h5>
          <ul>
            <li><a href="#presets">Genre presets</a></li>
            <li><a href="#generate">Generate</a></li>
            <li><a href="#api">REST API</a></li>
            <li><a href="#pipeline">Pipeline</a></li>
          </ul>
        </div>
        <div>
          <h5>Reference</h5>
          <ul>
            <li><a>CLI flags</a></li>
            <li><a>PatternSpec</a></li>
            <li><a>Style profiles</a></li>
            <li><a>Provider setup</a></li>
          </ul>
        </div>
        <div>
          <h5>Status</h5>
          <ul>
            <li>CLI · stable</li>
            <li>HTTP service · beta</li>
            <li>Web SDK · planned</li>
            <li>DAW plugin · researching</li>
          </ul>
        </div>
      </div>
      <div style={{
        maxWidth: 1480, margin: "60px auto 0",
        paddingTop: 22, borderTop: "1px solid var(--line)",
        display: "flex", justifyContent: "space-between",
        fontFamily: "var(--mono)", fontSize: 11.5,
        letterSpacing: "0.12em", textTransform: "uppercase",
        color: "var(--fg-muted)",
      }}>
        <span>© 2026 cadenza · andrea cavallo</span>
        <span>build 0.7.2 · go 1.23</span>
      </div>
    </footer>
  );
}

function App() {
  const tweakDefaults = /*EDITMODE-BEGIN*/{
    "accent": "lime"
  }/*EDITMODE-END*/;
  const [tweaks, setTweak] = useTweaks(tweakDefaults);

  useEffect(() => {
    const map = { lime: "#01FF95", cyan: "#00E0FF", amber: "#FFB000" };
    document.documentElement.style.setProperty("--accent", map[tweaks.accent] || map.lime);
  }, [tweaks.accent]);

  const [activePreset, setActivePreset] = useState("progressive-warmup");
  const [params, setParams] = useState({
    bpm: 122, key: "Am-dorian", bars: 16,
    groove: "mpc60", style: "melodic",
    provider: "claude", drums: false,
  });

  // when a preset is clicked, sync the generation panel to it
  const onSelectPreset = (id) => {
    setActivePreset(id);
    const p = GENRE_PRESETS.find(x => x.id === id);
    if (p) {
      setParams(prev => ({
        ...prev,
        bpm: p.bpm, key: p.key, groove: p.groove, style: p.style,
      }));
    }
  };

  return (
    <>
      <Topbar />
      <Hero />

      <Section
        id="presets"
        eyebrow="04 genre presets · single command"
        title="Pick a <em>vibe.</em><br/>Skip the configuration."
        sub="Each preset bakes the BPM, key, mode, groove and offline style for a specific moment of a set. Click one — the generator below mirrors it instantly.">
        <PresetGrid active={activePreset} onSelect={onSelectPreset} />
      </Section>

      <Section
        id="generate"
        eyebrow="generation · live mirror"
        title="The console <em>is</em> the API."
        sub="Tweak parameters on the left; the request payload and the equivalent CLI invocation update on the right. Same engine, same output, two surfaces.">
        <GenerationConsole params={params} setParams={setParams} />
      </Section>

      <Section
        id="api"
        eyebrow="rest service · v1 · beta"
        title="HTTP for <em>every</em> DAW workflow."
        sub="Six endpoints. Deterministic seeds. Signed download URLs. Built to slot into render farms, sample-pack pipelines, and live tools.">
        <APIPlayground />
      </Section>

      <Section
        id="pipeline"
        eyebrow="under the hood · 4 stages"
        title="From <em>intent</em> to MIDI."
        sub="Cadenza never asks an LLM to write notes directly. Models produce a validated PatternSpec; a deterministic renderer turns it into MIDI. Musical, fast, reproducible.">
        <Pipeline />
      </Section>

      <Footer />

      {/* Tweaks panel */}
      <TweaksPanel title="Tweaks">
        <TweakSection label="Accent">
          <TweakRadio
            value={tweaks.accent || "lime"}
            onChange={(v) => {
              setTweak("accent", v);
              const map = {
                lime:  "#01FF95",
                cyan:  "#00E0FF",
                amber: "#FFB000",
              };
              document.documentElement.style.setProperty("--accent", map[v] || map.lime);
            }}
            options={[
              { value: "lime",  label: "Lime"  },
              { value: "cyan",  label: "Cyan"  },
              { value: "amber", label: "Amber" },
            ]}
          />
        </TweakSection>
      </TweaksPanel>
    </>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
