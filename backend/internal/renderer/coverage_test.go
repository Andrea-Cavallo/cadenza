package renderer

import (
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
	"github.com/Andrea-Cavallo/cadenza/internal/schema"
)

func TestDynamicCurveScale_CoversVariants(t *testing.T) {
	profile := styleprofile.BassProgressive
	profile.Velocity.DynamicCurve = "crescendo"
	if got := dynamicCurveScale(15, 16, &profile); got <= 0.9 {
		t.Fatalf("crescendo should rise, got %f", got)
	}

	profile.Velocity.DynamicCurve = "arch"
	if got := dynamicCurveScale(4, 16, &profile); got <= 0.75 {
		t.Fatalf("arch should rise early, got %f", got)
	}

	profile.Velocity.DynamicCurve = "plateau"
	if got := dynamicCurveScale(8, 16, &profile); got != 1.0 {
		t.Fatalf("plateau middle should be 1.0, got %f", got)
	}

	profile.Velocity.DynamicCurve = "tension"
	if got := dynamicCurveScale(9, 16, &profile); got <= 0.7 {
		t.Fatalf("tension should spike, got %f", got)
	}

	profile.Velocity.DynamicCurve = "unknown"
	if got := dynamicCurveScale(1, 16, &profile); got != 1.0 {
		t.Fatalf("default curve should be 1.0, got %f", got)
	}
}

func TestFindNextActiveTick_AndResolveGate(t *testing.T) {
	profile := &styleprofile.BassProgressive
	steps := []schema.StepSpec{
		{Active: true, Note: "A2"},
		{Active: false},
		{Active: true, Note: "E2"},
	}

	next := findNextActiveTick(steps, 0, 0, profile)
	if next <= resolveTick(0, 0, profile) {
		t.Fatalf("expected next active tick after current tick")
	}

	lastNext := findNextActiveTick(steps, 0, 2, profile)
	if lastNext != resolveTick(1, 0, profile) {
		t.Fatalf("expected last step to resolve to next bar downbeat, got %d", lastNext)
	}

	if gate := resolveGateTicksExt(stepFlags{Slide: true}, profile, 10, 40); gate != 35 {
		t.Fatalf("unexpected slide gate %d", gate)
	}
	if gate := resolveGateTicksExt(stepFlags{Legato: true}, profile, 0, 0); gate <= 0 {
		t.Fatalf("expected positive legato gate, got %d", gate)
	}
	if gate := resolveGateTicksExt(stepFlags{Staccato: true}, profile, 0, 0); gate >= ticksPerStep {
		t.Fatalf("expected shortened staccato gate, got %d", gate)
	}
}

func TestSweepHelpers_CoverFallbacks(t *testing.T) {
	profile := styleprofile.BassProgressive
	profile.FilterSweep.KickbackEnd = true
	profile.FilterSweep.Resolution = 4
	profile.FilterSweep.Jitter = 1
	profile.FilterSweep.Range = map[string][2]int{
		"medium": {50, 90},
	}

	if got := computeSweepProgress(0, 0, 4, 4, -0.25); got < 0 || got >= 1 {
		t.Fatalf("progress should stay normalized, got %f", got)
	}
	if got := applyCurve(0.5, "exponential"); got <= 0 {
		t.Fatalf("applyCurve exponential should be positive, got %f", got)
	}
	if got := applyCurve(0.5, "other"); got != 0.5 {
		t.Fatalf("default curve mismatch: %f", got)
	}

	events := filterSweepEvents(15, 16, "unknown", "seed", "melody", &profile, []schema.EvolutionStep{
		{FromBar: 1, ToBar: 16, Action: "build", Intensity: 0.75},
	})
	if len(events) != profile.FilterSweep.Resolution {
		t.Fatalf("expected %d sweep events, got %d", profile.FilterSweep.Resolution, len(events))
	}
	if events[len(events)-1].Value == 0 {
		t.Fatal("expected non-zero final sweep value")
	}
}
