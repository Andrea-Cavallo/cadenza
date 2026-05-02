package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunDoctorCheck(t *testing.T) {
	out := captureStdout(t, func() {
		runDoctorCheck(t.TempDir())
	})
	for _, want := range []string{"CADENZA DOCTOR", "Go runtime", "ANTHROPIC_API_KEY", "Ollama", "Output directory"} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output missing %q; got:\n%s", want, out)
		}
	}
}

func TestPrintDoctorItem(t *testing.T) {
	okOut := captureStdout(t, func() { printDoctorItem("Test item", "all good", true) })
	if !strings.Contains(okOut, "Test item") || !strings.Contains(okOut, "all good") {
		t.Fatalf("unexpected ok doctor item output %q", okOut)
	}

	warnOut := captureStdout(t, func() { printDoctorItem("Test item", "not set", false) })
	if !strings.Contains(warnOut, "Test item") || !strings.Contains(warnOut, "not set") {
		t.Fatalf("unexpected warning doctor item output %q", warnOut)
	}
}

func TestCheckOllamaReachable(t *testing.T) {
	t.Run("reachable server", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		if !checkOllamaReachable(srv.URL) {
			t.Fatal("expected reachable")
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		if checkOllamaReachable("http://127.0.0.1:19999") {
			t.Fatal("expected unreachable for port with no listener")
		}
	})

	t.Run("server error response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		if checkOllamaReachable(srv.URL) {
			t.Fatal("expected false for 500 response")
		}
	})
}
