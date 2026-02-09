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
const dismissedFile = "dismissed-urls.txt"
const defaultBaseFolder = "THE_SINK/docs"

func getDocsDir() string {
	// Full path override takes precedence
	if dir := os.Getenv("SLACK_ADHD_DOCS_DIR"); dir != "" {
		return dir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	// Allow overriding just the relative folder (defaults to THE_SINK/docs)
	baseFolder := os.Getenv("SLACK_ADHD_BASE_FOLDER")
	if baseFolder == "" {
		baseFolder = defaultBaseFolder
	}
	return filepath.Join(homeDir, baseFolder)
}

func loadDismissedURLs(docsDir string) map[string]bool {
	dismissedPath := filepath.Join(docsDir, dismissedFile)
	content, err := os.ReadFile(dismissedPath)
	if err != nil {
		return nil
	}

	dismissed := make(map[string]bool)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			dismissed[line] = true
		}
	}
	return dismissed
}

func filterBrief(brief string, dismissed map[string]bool) string {
	if len(dismissed) == 0 {
		return brief
	}

	lines := strings.Split(brief, "\n")
	var filtered []string
	skip := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// New item starts with emoji - reset skip state
		if strings.HasPrefix(trimmed, "🔴") ||
			strings.HasPrefix(trimmed, "🟡") ||
			strings.HasPrefix(trimmed, "🟢") ||
			strings.HasPrefix(trimmed, "✅") {
			skip = false
		}

		// Check if this line contains a dismissed URL
		for url := range dismissed {
			if strings.Contains(line, url) {
				skip = true
				break
			}
		}

		if !skip {
			filtered = append(filtered, line)
		}
	}

	result := strings.TrimSpace(strings.Join(filtered, "\n"))
	if result == "" || len(result) < 20 {
		return ""
	}
	return result
}

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

	// Filter out dismissed URLs
	dismissed := loadDismissedURLs(docsDir)
	if len(dismissed) > 0 {
		log(homeDir, fmt.Sprintf("loaded %d dismissed URLs", len(dismissed)))
		content = filterBrief(content, dismissed)
		if content == "" {
			log(homeDir, "all items dismissed, skipping")
			os.Exit(0)
		}
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
