package main

import (
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/generator"
	"github.com/Andrea-Cavallo/cadenza/internal/theory"
)

func TestBuildGenerationPreviewScaleNotes(t *testing.T) {
	cases := []struct {
		keyStr      string
		wantInScale []string // notes that must appear in ScaleNotes
		wantMode    string   // substring expected in KeyName
	}{
		{"Am", []string{"A", "C", "E", "G"}, "minor"},
		{"Dm", []string{"D", "F", "A", "C"}, "minor"},
		{"G", []string{"G", "A", "B", "D"}, "major"},
	}

	for _, tc := range cases {
		t.Run(tc.keyStr, func(t *testing.T) {
			key, err := theory.ParseKey(tc.keyStr)
			if err != nil {
				t.Fatalf("ParseKey(%q): %v", tc.keyStr, err)
			}

			ctx := generator.MusicContext{
				Key:  key,
				Bars: 4,
			}

			preview := buildGenerationPreview(ctx, nil)

			if len(preview.ScaleNotes) == 0 {
				t.Fatalf("ScaleNotes is empty for key %q", tc.keyStr)
			}

			if preview.KeyName == "" {
				t.Fatalf("KeyName is empty for key %q", tc.keyStr)
			}

			for _, want := range tc.wantInScale {
				found := false
				for _, got := range preview.ScaleNotes {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("key %q: note %q missing from ScaleNotes %v", tc.keyStr, want, preview.ScaleNotes)
				}
			}

			// KeyName must mention the root note
			if len(preview.KeyName) < 1 || string(preview.KeyName[0]) != string(key.Root[0]) {
				t.Errorf("key %q: KeyName %q does not start with root %q", tc.keyStr, preview.KeyName, key.Root)
			}
		})
	}
}

func TestBuildGenerationPreviewScaleNotesLength(t *testing.T) {
	key, _ := theory.ParseKey("Am")
	ctx := generator.MusicContext{Key: key, Bars: 4}
	preview := buildGenerationPreview(ctx, nil)

	// diatonic scale always has 7 notes
	if len(preview.ScaleNotes) != 7 {
		t.Errorf("expected 7 scale notes for Am, got %d: %v", len(preview.ScaleNotes), preview.ScaleNotes)
	}
}

func TestBuildGenerationPreviewNilSpecs(t *testing.T) {
	key, _ := theory.ParseKey("G")
	ctx := generator.MusicContext{Key: key, Bars: 16}
	preview := buildGenerationPreview(ctx, nil)

	// nil specs → no patterns, but scale info still populated
	if len(preview.Patterns) != 0 {
		t.Errorf("expected 0 patterns with nil specs, got %d", len(preview.Patterns))
	}
	if len(preview.ScaleNotes) == 0 {
		t.Error("ScaleNotes must be populated even when specs is nil")
	}
}
