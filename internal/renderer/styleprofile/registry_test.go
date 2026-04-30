package styleprofile

import "testing"

func TestRegistry_DefaultsAndLookups(t *testing.T) {
	reg := NewRegistry()

	for _, patternType := range []string{"bassline", "arpeggio", "melody"} {
		profile, err := reg.DefaultForType(patternType)
		if err != nil {
			t.Fatalf("DefaultForType(%s) error: %v", patternType, err)
		}
		if profile.PatternType != patternType {
			t.Fatalf("expected %s profile, got %s", patternType, profile.PatternType)
		}
		if profile.Velocity.MaxVelocity > 120 {
			t.Fatalf("max velocity too high for %s: %d", profile.Name, profile.Velocity.MaxVelocity)
		}
		if profile.Velocity.GhostVelocity > 60 {
			t.Fatalf("ghost velocity too high for %s: %d", profile.Name, profile.Velocity.GhostVelocity)
		}
		if profile.Velocity.DynamicCurve == "" {
			t.Fatalf("dynamic curve missing for %s", profile.Name)
		}
	}

	if _, err := reg.LoadForType("bassline", "arp_flowing"); err == nil {
		t.Fatal("expected type mismatch error")
	}
	if _, err := reg.LoadForType("bassline", "missing"); err == nil {
		t.Fatal("expected missing profile error")
	}
	if _, err := reg.DefaultForType("drums"); err == nil {
		t.Fatal("expected missing default error")
	}
}
