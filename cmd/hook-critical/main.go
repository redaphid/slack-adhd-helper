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

	// Read the brief file - if it doesn't exist or is empty, exit silently
	data, err := os.ReadFile(briefPath)
	if err != nil {
		log(homeDir, fmt.Sprintf("no brief file: %v", err))
		os.Exit(0)
	}

	content := strings.TrimSpace(string(data))
	if len(content) < 20 {
		log(homeDir, "brief too short, skipping")
		os.Exit(0)
	}

	log(homeDir, fmt.Sprintf("brief length: %d chars", len(content)))

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

	log(homeDir, fmt.Sprintf("output JSON length: %d", len(jsonBytes)))
	fmt.Print(string(jsonBytes))
}
