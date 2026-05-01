package models

import "testing"

func TestEmbeddedCatalogListsProviders(t *testing.T) {
	global = nil
	Load("")

	providers := Providers()
	if len(providers) == 0 {
		t.Fatal("expected embedded providers")
	}
	if got := DefaultModel("claude"); got == "" {
		t.Fatal("expected claude default model")
	}
	if got := List("claude"); len(got) == 0 {
		t.Fatal("expected claude model entries")
	}
}

func TestUnknownProviderReturnsEmptyValues(t *testing.T) {
	global = nil
	Load("")

	if got := DefaultModel("unknown"); got != "" {
		t.Fatalf("expected empty default model, got %q", got)
	}
	if got := List("unknown"); got != nil {
		t.Fatalf("expected nil list, got %#v", got)
	}
}

func TestParseRejectsInvalidYAML(t *testing.T) {
	if _, err := parse([]byte("providers: [")); err == nil {
		t.Fatal("expected parse error")
	}
}
