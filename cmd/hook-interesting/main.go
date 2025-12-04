package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed wrapper.md
var wrapperTemplate string

const briefFile = "slack-interesting.md"

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		os.Exit(0)
	}

	// Touch marker file for debugging
	markerPath := filepath.Join(homeDir, "tools", "check-slack-for-relevant-stuff", "tmp", "session_start_last_fired")
	os.MkdirAll(filepath.Dir(markerPath), 0755)
	os.WriteFile(markerPath, []byte{}, 0644)

	briefPath := filepath.Join(homeDir, "THE_SINK", "docs", briefFile)

	// No brief file = nothing to report
	data, err := os.ReadFile(briefPath)
	if err != nil || len(data) == 0 {
		os.Exit(0)
	}

	content := string(data)

	// Only show if there's interesting content
	if !strings.Contains(content, "🌟") && !strings.Contains(content, "💡") && !strings.Contains(content, "🔗") {
		os.Exit(0)
	}

	// Output using embedded wrapper template
	output := strings.ReplaceAll(wrapperTemplate, "{{BRIEF}}", content)
	fmt.Print(output)
}
