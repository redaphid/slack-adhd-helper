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

const briefFile = "slack-interesting.md"

func getDocsDir() string {
	if dir := os.Getenv("SLACK_PULSE_DOCS_DIR"); dir != "" {
		return dir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, "THE_SINK", "docs")
}

type HookOutput struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

type HookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

func log(homeDir, msg string) {
	logPath := filepath.Join(homeDir, ".claude", "hooks", "logs", "slack-interesting.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("%s | %s\n", time.Now().Format(time.RFC3339), msg))
}

func main() {
	docsDir := getDocsDir()
	if docsDir == "" {
		os.Exit(0)
	}

	homeDir, _ := os.UserHomeDir()
	log(homeDir, "hook started")

	briefPath := filepath.Join(docsDir, briefFile)

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
			HookEventName:     "SessionStart",
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
