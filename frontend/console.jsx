// Generation console: BPM/key/bars/groove/provider controls + live curl mirror
// + API playground with endpoint list, example response, and tabbed code.

const KEYS = [
  "Am", "Bm", "Cm", "Dm", "Em", "Fm", "Gm",
  "C", "D", "E", "F", "G", "A", "B",
  "F#m", "C#m", "Bbm",
  "Am-dorian", "Bm-dorian", "Em-dorian",
  "Em-phrygian", "Bm-phrygian",
  "G-mixolydian", "A-mixolydian", "D-mixolydian",
  "C-lydian", "D-lydian",
];
const GROOVES   = ["straight", "mpc60", "linndrum", "humanize"];
const STYLES    = ["—", "hypnotic", "driving", "minimal", "melodic"];
const PROVIDERS = ["claude", "ollama", "openai", "gemini", "offline"];
const BARS_OPTS = [16, 32, 64, 128];

function bpmGenre(bpm) {
  if (bpm < 100) return "Downtempo / Ambient";
  if (bpm < 115) return "Deep house / Nu-disco";
  if (bpm < 122) return "Tech house / Organic";
  if (bpm < 130) return "Progressive / Melodic techno";
  if (bpm < 138) return "Peak-time techno";
  return "Hard techno / Industrial";
}

function syntaxCurl(text) {
  // Minimal highlighting for curl/json display
  return text
    .replace(/("(?:[^"\\]|\\.)*")/g, '<span class="c-str">$1</span>')
    .replace(/\b(\d+\.?\d*)\b/g, '<span class="c-num">$1</span>')
    .replace(/(--[a-z-]+)/g, '<span class="c-flag">$1</span>')
    .replace(/(#.*$)/gm, '<span class="c-comment">$1</span>');
}

function GenerationConsole({ params, setParams }) {
  const { bpm, key, bars, groove, style, provider, drums } = params;

  const curl = [
    `# generate via HTTP service`,
    `curl -X POST https://api.cadenza.dev/v1/generate \\`,
    `  -H "Authorization: Bearer $CADENZA_KEY" \\`,
    `  -H "Content-Type: application/json" \\`,
    `  -d '${JSON.stringify({
      bpm, key, bars, groove,
      ...(style !== "—" ? { offline_style: style } : {}),
      provider,
      drums,
    }, null, 2).replace(/\n/g, "\n  ")}'`,
  ].join("\n");

  const cli = [
    `# or via CLI`,
    `cadenza --bpm ${bpm} --key ${key} --bars ${bars}` +
      (groove !== "straight" ? ` --groove ${groove}` : "") +
      (style !== "—" ? ` --offline-style ${style} --no-llm` : "") +
      (provider !== "offline" && provider !== "claude" ? ` --provider ${provider}` : "") +
      (drums ? ` --drums` : ""),
  ].join("\n");

  return (
    <div className="row" style={{ gridTemplateColumns: "1.1fr 1.4fr" }}>
      {/* ── controls ──────────────────────────── */}
      <div className="cell">
        <div className="eyebrow" style={{ marginBottom: 22 }}>Request — POST /v1/generate</div>

        {/* BPM */}
        <div style={{ marginBottom: 22 }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 8 }}>
            <span className="label">BPM</span>
            <span className="mono" style={{ fontSize: 13, color: "var(--fg-dim)" }}>{bpmGenre(bpm)}</span>
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 70px", gap: 14, alignItems: "center" }}>
            <input type="range" min="80" max="150" step="1"
                   value={bpm}
                   onChange={e => setParams({ ...params, bpm: parseInt(e.target.value) })} />
            <div className="mono" style={{ fontSize: 28, fontWeight: 500, textAlign: "right", letterSpacing: "-0.02em" }}>
              {bpm}
            </div>
          </div>
        </div>

        {/* Key + Bars */}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 18, marginBottom: 22 }}>
          <div>
            <div className="label" style={{ marginBottom: 8 }}>Key &amp; mode</div>
            <select value={key} onChange={e => setParams({ ...params, key: e.target.value })}>
              {KEYS.map(k => <option key={k} value={k}>{k}</option>)}
            </select>
          </div>
          <div>
            <div className="label" style={{ marginBottom: 8 }}>Bars</div>
            <select value={bars} onChange={e => setParams({ ...params, bars: parseInt(e.target.value) })}>
              {BARS_OPTS.map(b => <option key={b} value={b}>{b}</option>)}
            </select>
          </div>
        </div>

        {/* Groove + Style */}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 18, marginBottom: 22 }}>
          <div>
            <div className="label" style={{ marginBottom: 8 }}>Groove</div>
            <select value={groove} onChange={e => setParams({ ...params, groove: e.target.value })}>
              {GROOVES.map(g => <option key={g} value={g}>{g}</option>)}
            </select>
          </div>
          <div>
            <div className="label" style={{ marginBottom: 8 }}>Offline style</div>
            <select value={style} onChange={e => setParams({ ...params, style: e.target.value })}>
              {STYLES.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
        </div>

        {/* Provider */}
        <div style={{ marginBottom: 22 }}>
          <div className="label" style={{ marginBottom: 8 }}>Provider</div>
          <div style={{ display: "flex", gap: 1, background: "var(--line)", border: "1px solid var(--line)" }}>
            {PROVIDERS.map(p => (
              <button key={p}
                      onClick={() => setParams({ ...params, provider: p })}
                      className="mono"
                      style={{
                        flex: 1, padding: "10px 8px",
                        background: provider === p ? "var(--fg)" : "var(--bg)",
                        color: provider === p ? "var(--bg)" : "var(--fg-dim)",
                        border: 0, cursor: "pointer",
                        fontSize: 11, letterSpacing: "0.12em", textTransform: "uppercase",
                      }}>
                {p}
              </button>
            ))}
          </div>
        </div>

        {/* Drums toggle */}
        <label style={{ display: "flex", alignItems: "center", gap: 12, cursor: "pointer" }}>
          <span style={{
            width: 36, height: 20, borderRadius: 999,
            background: drums ? "var(--accent)" : "var(--line-2)",
            position: "relative", transition: "background .15s",
          }}>
            <span style={{
              position: "absolute", top: 2, left: drums ? 18 : 2,
              width: 16, height: 16, borderRadius: 999,
              background: drums ? "var(--accent-ink)" : "var(--fg-dim)",
              transition: "left .15s",
            }} />
          </span>
          <span className="mono" style={{ fontSize: 12, letterSpacing: "0.08em" }}>
            DRUMS — kick / clap / hi-hat on CH10
          </span>
        </label>
      </div>

      {/* ── live curl + cli ────────────────────── */}
      <div className="cell" style={{ background: "var(--bg-1)", display: "flex", flexDirection: "column", gap: 14 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div className="eyebrow">Live request mirror</div>
          <div style={{ display: "flex", gap: 8 }}>
            <span className="pill"><span className="method post" style={{ border: 0, padding: 0, marginRight: 6 }}>POST</span>v1/generate</span>
          </div>
        </div>
        <pre className="code"
             style={{ background: "var(--bg)", flex: 1, margin: 0 }}
             dangerouslySetInnerHTML={{ __html: syntaxCurl(curl) }} />
        <pre className="code"
             style={{ background: "var(--bg)", margin: 0 }}
             dangerouslySetInnerHTML={{ __html: syntaxCurl(cli) }} />
      </div>
    </div>
  );
}

// ────────────────────────────────────────────────────────────────
// API playground

const ENDPOINTS = [
  { method: "POST", path: "/v1/generate",
    desc: "Generate bassline + arpeggio + melody MIDI tracks." },
  { method: "POST", path: "/v1/generate/preset/:id",
    desc: "Generate from a named genre preset (peak-time-driver, …)." },
  { method: "GET",  path: "/v1/presets",
    desc: "List all genre presets with default parameters." },
  { method: "GET",  path: "/v1/jobs/:id",
    desc: "Poll the status of an async generation job." },
  { method: "GET",  path: "/v1/jobs/:id/midi",
    desc: "Download the rendered MIDI bundle (.zip of 3 tracks)." },
  { method: "POST", path: "/v1/render-from-spec",
    desc: "Re-render a previously dumped PatternSpec YAML." },
];

const SAMPLE_RESPONSE = {
  job_id: "gen_01HZ7M9K3F8X2N",
  status: "completed",
  seed: "1714560219834213000",
  meta: {
    bpm: 122, key: "Am-dorian", bars: 16,
    groove: "mpc60", offline_style: "melodic",
    provider: "claude", model: "claude-opus-4-7",
  },
  progression: ["Am7", "F", "Cmaj7", "G"],
  tracks: [
    { name: "bassline", channel: 1,  events: 64,  bytes: 1284, file: "bassline_122_Am.mid" },
    { name: "arpeggio", channel: 2,  events: 256, bytes: 4912, file: "arpeggio_122_Am.mid" },
    { name: "melody",   channel: 3,  events: 48,  bytes: 1108, file: "melody_122_Am.mid"   },
    { name: "drums",    channel: 10, events: 192, bytes: 3640, file: "drums_122.mid"       },
  ],
  download_url: "https://api.cadenza.dev/v1/jobs/gen_01HZ7M9K3F8X2N/midi",
  expires_at: "2026-05-02T14:03:42Z",
};

function jsonHighlight(obj) {
  const json = JSON.stringify(obj, null, 2);
  return json
    .replace(/("(?:[^"\\]|\\.)*")(\s*:)/g, '<span class="c-key">$1</span>$2')
    .replace(/: ("(?:[^"\\]|\\.)*")/g, ': <span class="c-str">$1</span>')
    .replace(/: (-?\d+\.?\d*)/g, ': <span class="c-num">$1</span>')
    .replace(/: (true|false|null)/g, ': <span class="c-num">$1</span>');
}

function APIPlayground() {
  const [tab, setTab] = React.useState("response");
  const [selected, setSelected] = React.useState(0);

  return (
    <div className="row" style={{ gridTemplateColumns: "1fr 1.4fr" }}>
      {/* endpoint list */}
      <div className="cell" style={{ padding: 0 }}>
        <div style={{ padding: "20px 28px", borderBottom: "1px solid var(--line)" }}>
          <div className="eyebrow">REST API · v1</div>
          <div className="mono" style={{ fontSize: 12, color: "var(--fg-dim)", marginTop: 8 }}>
            api.cadenza.dev
          </div>
        </div>
        {ENDPOINTS.map((e, i) => (
          <div key={e.path}
               onClick={() => setSelected(i)}
               style={{
                 padding: "16px 28px",
                 borderBottom: "1px solid var(--line)",
                 cursor: "pointer",
                 background: selected === i ? "var(--bg-1)" : "transparent",
                 borderLeft: selected === i ? "2px solid var(--accent)" : "2px solid transparent",
               }}>
            <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 4 }}>
              <span className={"method " + e.method.toLowerCase()}>{e.method}</span>
              <span className="mono" style={{ fontSize: 13, color: "var(--fg)" }}>{e.path}</span>
            </div>
            <div style={{ fontSize: 12.5, color: "var(--fg-dim)", lineHeight: 1.45 }}>
              {e.desc}
            </div>
          </div>
        ))}
      </div>

      {/* response viewer */}
      <div className="cell" style={{ padding: 0, background: "var(--bg-1)", display: "flex", flexDirection: "column" }}>
        <div className="tabs" style={{ borderBottom: "1px solid var(--line)" }}>
          {["response", "schema", "headers"].map(t => (
            <div key={t}
                 className={"tab" + (tab === t ? " on" : "")}
                 onClick={() => setTab(t)}>
              {t}
            </div>
          ))}
          <div style={{ flex: 1 }} />
          <div className="mono" style={{
            fontSize: 11, letterSpacing: "0.12em", color: "var(--fg-muted)",
            padding: "14px 22px", textTransform: "uppercase",
          }}>
            200 · 4.92 kb · 1.2 s
          </div>
        </div>
        <div style={{ flex: 1, overflow: "auto", padding: 0 }}>
          {tab === "response" && (
            <pre className="code"
                 style={{ border: 0, margin: 0, background: "transparent", padding: "22px 28px" }}
                 dangerouslySetInnerHTML={{ __html: jsonHighlight(SAMPLE_RESPONSE) }} />
          )}
          {tab === "schema" && (
            <pre className="code"
                 style={{ border: 0, margin: 0, background: "transparent", padding: "22px 28px" }}>
{`type GenerateResponse = {
  job_id:        string         // gen_<ulid>
  status:        "queued" | "running" | "completed" | "failed"
  seed:          string         // deterministic re-render handle
  meta:          GenerationMeta
  progression:   string[]       // chord symbols, e.g. ["Am7","F","Cmaj7","G"]
  tracks:        Track[]        // bassline, arpeggio, melody, [drums]
  download_url:  string         // signed url, expires in 1h
  expires_at:    string         // ISO-8601
}

type Track = {
  name:     "bassline" | "arpeggio" | "melody" | "drums"
  channel:  number
  events:   number
  bytes:    number
  file:     string
}`}
            </pre>
          )}
          {tab === "headers" && (
            <pre className="code"
                 style={{ border: 0, margin: 0, background: "transparent", padding: "22px 28px" }}>
{`HTTP/2 200
content-type:        application/json; charset=utf-8
x-cadenza-seed:      1714560219834213000
x-cadenza-version:   0.7.2
x-cadenza-provider:  claude/claude-opus-4-7
x-ratelimit-limit:   120
x-ratelimit-remain:  118
x-request-id:        req_01HZ7M9K3F8X2N`}
            </pre>
          )}
        </div>
      </div>
    </div>
  );
}

window.GenerationConsole = GenerationConsole;
window.APIPlayground = APIPlayground;
