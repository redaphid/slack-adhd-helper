package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func log(homeDir, msg string) {
	logPath := filepath.Join(homeDir, ".claude", "hooks", "logs", "slack-critical.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("%s | %s\n", time.Now().Format(time.RFC3339), msg))
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		os.Exit(0)
	}

	log(homeDir, "hook started")

	briefPath := filepath.Join(homeDir, "THE_SINK", "docs", briefFile)

	// No brief file = nothing to report
	data, err := os.ReadFile(briefPath)
	if err != nil {
		log(homeDir, fmt.Sprintf("failed to read brief: %v", err))
		os.Exit(0)
	}
	if len(data) == 0 {
		log(homeDir, "brief file empty")
		os.Exit(0)
	}

	content := string(data)
	log(homeDir, fmt.Sprintf("brief content length: %d", len(content)))

	// Only show if there's actual content (skip empty or placeholder files)
	if len(strings.TrimSpace(content)) < 50 {
		log(homeDir, "content too short, skipping")
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
		log(homeDir, fmt.Sprintf("json marshal failed: %v", err))
		os.Exit(0)
	}

	log(homeDir, fmt.Sprintf("output length: %d", len(jsonBytes)))
	fmt.Print(string(jsonBytes))
}
