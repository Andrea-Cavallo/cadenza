package metrics

import "testing"

func TestMetricsInitialized(t *testing.T) {
	if GenerationsTotal == nil {
		t.Fatal("GenerationsTotal should be initialized")
	}
	if GenerationErrors == nil {
		t.Fatal("GenerationErrors should be initialized")
	}
	if CacheHits == nil {
		t.Fatal("CacheHits should be initialized")
	}
	if CacheMisses == nil {
		t.Fatal("CacheMisses should be initialized")
	}
	if LLMCalls == nil {
		t.Fatal("LLMCalls should be initialized")
	}
	if LLMErrors == nil {
		t.Fatal("LLMErrors should be initialized")
	}
	if OfflinePatterns == nil {
		t.Fatal("OfflinePatterns should be initialized")
	}
}

func TestMetricsIncrement(t *testing.T) {
	before := GenerationsTotal.Value()
	GenerationsTotal.Add(1)
	after := GenerationsTotal.Value()
	if after != before+1 {
		t.Fatalf("expected %d, got %d", before+1, after)
	}
}
