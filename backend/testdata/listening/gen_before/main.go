//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

func main() {
	type Entry struct {
		Seed        string `json:"seed"`
		Key         string `json:"key"`
		Fingerprint string `json:"fingerprint"`
	}

	seeds := []string{"fix-1", "fix-2", "fix-3"}
	keys := []struct{ root, scale, mode string }{
		{"A", "minor_natural", "minor"},
		{"D", "dorian", "dorian"},
	}

	var entries []Entry
	for _, kc := range keys {
		key := theory.Key{Root: kc.root, Scale: kc.scale, Mode: kc.mode}
		for _, seed := range seeds {
			prog := theory.SelectProgression(kc.root, kc.scale, seed)
			ctx := generator.MusicContext{
				BPM:              122,
				Key:              key,
				ChordProgression: prog,
				VariationSeed:    seed,
				Bars:             16,
			}
			spec := generator.OfflineTemplate("melody", ctx)
			if spec == nil {
				fmt.Fprintf(os.Stderr, "nil spec for %s %s\n", kc.root, seed)
				continue
			}
			entries = append(entries, Entry{
				Seed:        seed,
				Key:         kc.root + "-" + kc.scale,
				Fingerprint: musicalFingerprint(spec.Motif.Steps),
			})
		}
	}

	raw, _ := json.MarshalIndent(entries, "", "  ")
	outPath := "testdata/listening/melody_before_fingerprints.json"
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Written", outPath)
}

func musicalFingerprint(steps []schema.StepSpec) string {
	var b strings.Builder
	for _, s := range steps {
		if !s.Active {
			b.WriteByte('-')
			continue
		}
		// Strip octave digit from note name
		note := s.Note
		if len(note) > 0 {
			last := note[len(note)-1]
			if last >= '0' && last <= '9' {
				note = note[:len(note)-1]
			}
		}
		b.WriteString(note)
		switch {
		case s.Accent:
			b.WriteByte('!')
		case s.Ghost:
			b.WriteByte('g')
		default:
			b.WriteByte('.')
		}
		b.WriteByte('|')
	}
	return b.String()
}
