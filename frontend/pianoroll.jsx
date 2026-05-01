// Animated piano roll hero — SVG, looping, deterministic.
// Three voices: bassline (low), arpeggio (mid), melody (high), color-coded.

const ROLL_BARS = 4;          // visible window
const ROLL_STEPS_PER_BAR = 16;
const ROLL_TOTAL = ROLL_BARS * ROLL_STEPS_PER_BAR;  // 64 steps visible
const ROLL_PITCH_MIN = 36;    // C2
const ROLL_PITCH_MAX = 84;    // C6
const ROLL_PITCH_RANGE = ROLL_PITCH_MAX - ROLL_PITCH_MIN;

// Hand-authored notes (pitch, startStep, lengthSteps, voice)
// Voice: 0=bass, 1=arp, 2=melody
const ROLL_NOTES = [
  // Bassline — 1/8 root motion, Am-F-C-G feel
  { p: 45, s: 0,  l: 6,  v: 0 }, { p: 45, s: 8,  l: 6,  v: 0 },
  { p: 41, s: 16, l: 6,  v: 0 }, { p: 41, s: 24, l: 6,  v: 0 },
  { p: 36, s: 32, l: 6,  v: 0 }, { p: 36, s: 40, l: 6,  v: 0 },
  { p: 43, s: 48, l: 6,  v: 0 }, { p: 43, s: 56, l: 6,  v: 0 },
  // Arpeggio — 1/16 chord tones
  { p: 57, s: 0,  l: 1, v: 1 }, { p: 60, s: 2,  l: 1, v: 1 }, { p: 64, s: 4,  l: 1, v: 1 }, { p: 60, s: 6,  l: 1, v: 1 },
  { p: 57, s: 8,  l: 1, v: 1 }, { p: 60, s: 10, l: 1, v: 1 }, { p: 64, s: 12, l: 1, v: 1 }, { p: 67, s: 14, l: 1, v: 1 },
  { p: 53, s: 16, l: 1, v: 1 }, { p: 57, s: 18, l: 1, v: 1 }, { p: 60, s: 20, l: 1, v: 1 }, { p: 65, s: 22, l: 1, v: 1 },
  { p: 53, s: 24, l: 1, v: 1 }, { p: 57, s: 26, l: 1, v: 1 }, { p: 60, s: 28, l: 1, v: 1 }, { p: 57, s: 30, l: 1, v: 1 },
  { p: 48, s: 32, l: 1, v: 1 }, { p: 52, s: 34, l: 1, v: 1 }, { p: 55, s: 36, l: 1, v: 1 }, { p: 60, s: 38, l: 1, v: 1 },
  { p: 48, s: 40, l: 1, v: 1 }, { p: 52, s: 42, l: 1, v: 1 }, { p: 55, s: 44, l: 1, v: 1 }, { p: 64, s: 46, l: 1, v: 1 },
  { p: 55, s: 48, l: 1, v: 1 }, { p: 59, s: 50, l: 1, v: 1 }, { p: 62, s: 52, l: 1, v: 1 }, { p: 67, s: 54, l: 1, v: 1 },
  { p: 55, s: 56, l: 1, v: 1 }, { p: 62, s: 58, l: 1, v: 1 }, { p: 67, s: 60, l: 1, v: 1 }, { p: 71, s: 62, l: 1, v: 1 },
  // Melody — phrase arc
  { p: 72, s: 4,  l: 4,  v: 2 }, { p: 76, s: 12, l: 3, v: 2 },
  { p: 74, s: 18, l: 6,  v: 2 }, { p: 77, s: 26, l: 4, v: 2 },
  { p: 79, s: 34, l: 6,  v: 2 }, { p: 76, s: 42, l: 4, v: 2 },
  { p: 74, s: 50, l: 5,  v: 2 }, { p: 71, s: 58, l: 6, v: 2 },
];

function PianoRoll({ width = 1480, height = 420, bpm = 122 }) {
  const [t, setT] = React.useState(0);
  const startRef = React.useRef(null);
  const rafRef = React.useRef(null);

  // 4 bars at given BPM = (60/bpm) * 4 * 4 seconds (4/4 time)
  const cycleSec = (60 / bpm) * 4 * ROLL_BARS;

  React.useEffect(() => {
    const tick = (now) => {
      if (startRef.current == null) startRef.current = now;
      const elapsed = (now - startRef.current) / 1000;
      const phase = (elapsed % cycleSec) / cycleSec;
      setT(phase);
      rafRef.current = requestAnimationFrame(tick);
    };
    rafRef.current = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(rafRef.current);
  }, [cycleSec]);

  const W = width;
  const H = height;
  const padTop = 24, padBot = 36;
  const innerH = H - padTop - padBot;
  const stepW = W / ROLL_TOTAL;
  const noteH = innerH / ROLL_PITCH_RANGE;

  const playheadX = t * W;

  // Voice colors
  const voiceColor = (v, active) => {
    if (active) return "var(--accent)";
    if (v === 0) return "#3a3a3a";   // bassline — dim grey
    if (v === 1) return "#888888";   // arp — mid grey
    return "#ffffff";                // melody — white
  };
  const voiceOpacity = (v) => v === 0 ? 0.85 : v === 1 ? 0.9 : 1;

  // Build notes (loop wrap not needed because all notes fit in window)
  const notes = ROLL_NOTES.map((n, i) => {
    const x = n.s * stepW;
    const w = Math.max(stepW * n.l - 1, 2);
    const y = padTop + (ROLL_PITCH_MAX - n.p) * noteH;
    const h = Math.max(noteH - 1, 2);
    const active = playheadX >= x && playheadX < x + w;
    return { i, x, y, w, h, v: n.v, active };
  });

  // Vertical grid lines: every beat (4 steps), thicker every bar (16 steps)
  const grids = [];
  for (let s = 0; s <= ROLL_TOTAL; s++) {
    if (s % 4 !== 0) continue;
    const isBar = s % ROLL_STEPS_PER_BAR === 0;
    grids.push({ x: s * stepW, isBar });
  }
  // Horizontal lines: octave divisions
  const octs = [];
  for (let p = ROLL_PITCH_MIN; p <= ROLL_PITCH_MAX; p += 12) {
    octs.push(padTop + (ROLL_PITCH_MAX - p) * noteH);
  }

  return (
    <svg className="roll-svg" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none">
      {/* horizontal octave lines */}
      {octs.map((y, i) => (
        <line key={`o${i}`} x1={0} x2={W} y1={y} y2={y}
              stroke="var(--line)" strokeWidth="1" />
      ))}
      {/* vertical beat grid */}
      {grids.map((g, i) => (
        <line key={`g${i}`} x1={g.x} x2={g.x} y1={padTop} y2={H - padBot}
              stroke={g.isBar ? "var(--line-2)" : "var(--line)"}
              strokeWidth="1" />
      ))}
      {/* notes */}
      {notes.map(n => (
        <rect key={n.i}
              x={n.x + 1} y={n.y} width={n.w} height={n.h}
              fill={voiceColor(n.v, n.active)}
              opacity={n.active ? 1 : voiceOpacity(n.v)}
              style={{ transition: "opacity 80ms linear" }} />
      ))}
      {/* playhead */}
      <line x1={playheadX} x2={playheadX} y1={padTop - 4} y2={H - padBot + 4}
            stroke="var(--accent)" strokeWidth="1.5" />
      <circle cx={playheadX} cy={padTop - 4} r="3" fill="var(--accent)" />

      {/* bar labels */}
      {[0,1,2,3].map(b => (
        <text key={b}
              x={b * ROLL_STEPS_PER_BAR * stepW + 8}
              y={H - 14}
              fill="var(--fg-muted)"
              fontFamily="var(--mono)"
              fontSize="10"
              letterSpacing="2">
          {String(b + 1).padStart(2, "0")}
        </text>
      ))}
    </svg>
  );
}

window.PianoRoll = PianoRoll;
