package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// runWatchMode enters watch mode: stays in a loop on stdin, each Enter generates
// a new variation with incremental seed, 'q' + Enter exits.
// REFACTOR.md point 15
func runWatchMode(cfg cliConfig) {
	fmt.Printf("\n  %s═══ WATCH MODE ═══%s\n\n", ansiCyan+ansiBold, ansiReset)
	fmt.Printf("  %sPress Enter to generate a new variation%s\n", ansiDim, ansiReset)
	fmt.Printf("  %sType 'q' + Enter to exit%s\n\n", ansiDim, ansiReset)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := bufio.NewReader(os.Stdin)
	varNum := 1

	// Calculate initial seed
	var baseSeed uint64
	if cfg.Seed != nil {
		baseSeed = *cfg.Seed
	} else {
		baseSeed = uint64(time.Now().UnixNano())
	}

	for {
		// Wait for user input
		fmt.Printf("  %s[%d]%s › ", ansiYellow+ansiBold, varNum, ansiReset)
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("\n  %s✗  Input error: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
			break
		}

		line = strings.TrimSpace(strings.ToLower(line))
		if line == "q" || line == "quit" || line == "exit" {
			fmt.Printf("\n  %s✕  Watch mode exited.%s\n\n", ansiDim, ansiReset)
			break
		}

		// Generate variation with incremental seed
		actualSeed := baseSeed + uint64(varNum-1)
		seedStr := fmt.Sprintf("%d", actualSeed)

		fmt.Printf("\n  %s═══ Variation %d (seed: %s) ═══%s\n\n", ansiCyan+ansiBold, varNum, seedStr, ansiReset)

		if err := runSingleGeneration(ctx, cfg, varNum, seedStr); err != nil {
			fmt.Printf("  %s✗  Generation failed: %v%s\n\n", ansiRed+ansiBold, err, ansiReset)
		} else {
			fmt.Printf("  %s✓  Variation %d complete%s\n\n", ansiGreen+ansiBold, varNum, ansiReset)
		}

		varNum++
	}
}
