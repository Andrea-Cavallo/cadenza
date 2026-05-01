package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// runDoctorCheck prints a diagnostic report for the current environment.
func runDoctorCheck(outputDir string) {
	fmt.Printf("\n  %sCADENZA DOCTOR%s\n\n", ansiDim+ansiWhite, ansiReset)

	printDoctorItem("Go runtime", runtime.Version(), true)

	apiKeys := [][2]string{
		{"Claude  (ANTHROPIC_API_KEY)", "ANTHROPIC_API_KEY"},
		{"OpenAI  (OPENAI_API_KEY)", "OPENAI_API_KEY"},
		{"Gemini  (GEMINI_API_KEY)", "GEMINI_API_KEY"},
	}
	for _, k := range apiKeys {
		if os.Getenv(k[1]) != "" {
			printDoctorItem(k[0], "configured", true)
		} else {
			printDoctorItem(k[0], "not set (optional)", false)
		}
	}

	ollamaURL := "http://localhost:11434"
	if checkOllamaReachable(ollamaURL) {
		printDoctorItem("Ollama", "reachable at "+ollamaURL, true)
	} else {
		printDoctorItem("Ollama", "not reachable (optional, needed for --provider ollama)", false)
	}

	if outputDir == "" {
		outputDir = "output"
	}
	abs, _ := filepath.Abs(outputDir)
	if err := ensureWritableDir(outputDir); err == nil {
		printDoctorItem("Output directory", abs+" (writable)", true)
	} else {
		printDoctorItem("Output directory", abs+" ("+err.Error()+")", false)
	}

	fmt.Println()
	fmt.Printf("  %sTo generate: cadenza --bpm 122 --key Am --no-llm%s\n\n", ansiDim, ansiReset)
}

func printDoctorItem(label, status string, ok bool) {
	icon := ansiGreen + ansiBold + "✓" + ansiReset
	if !ok {
		icon = ansiYellow + ansiBold + "!" + ansiReset
	}
	fmt.Printf("  %s  %-38s %s%s%s\n", icon, label, ansiDim, status, ansiReset)
}

func checkOllamaReachable(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}
