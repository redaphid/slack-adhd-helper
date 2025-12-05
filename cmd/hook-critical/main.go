package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed wrapper.md
var wrapperTemplate string

const briefFile = "slack-critical.md"

type HookOutput struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

type HookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		os.Exit(0)
	}

	// Touch marker file for debugging
	markerPath := filepath.Join(homeDir, "tools", "check-slack-for-relevant-stuff", "tmp", "user_hook_last_fired")
	os.MkdirAll(filepath.Dir(markerPath), 0755)
	os.WriteFile(markerPath, []byte{}, 0644)

	briefPath := filepath.Join(homeDir, "THE_SINK", "docs", briefFile)

	// No brief file = nothing to report
	data, err := os.ReadFile(briefPath)
	if err != nil || len(data) == 0 {
		os.Exit(0)
	}

	content := string(data)

	// Only show if there's actual content (skip empty or placeholder files)
	if len(strings.TrimSpace(content)) < 50 {
		os.Exit(0)
	}

	// Build the context message using wrapper template
	contextMsg := strings.ReplaceAll(wrapperTemplate, "{{BRIEF}}", content)

	// Output as JSON for Claude Code hooks
	output := HookOutput{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName:     "UserPromptSubmit",
			AdditionalContext: contextMsg,
		},
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		os.Exit(0)
	}
	fmt.Print(string(jsonBytes))
}
