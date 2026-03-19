package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

const dismissalTTL = 48 * time.Hour

// slackMsgTime extracts the send-time from a Slack message URL.
// Slack encodes Unix seconds in the path: /archives/CXXX/pSSSSSSSSSSmmmmmm
func slackMsgTime(url string) (time.Time, bool) {
	idx := strings.LastIndex(url, "/p")
	if idx == -1 || len(url)-idx < 12 {
		return time.Time{}, false
	}
	secs, err := strconv.ParseInt(url[idx+2:idx+12], 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(secs, 0), true
}

func loadDismissedURLs(docsDir string) map[string]bool {
	dismissedPath := filepath.Join(docsDir, dismissedFile)
	content, err := os.ReadFile(dismissedPath)
	if err != nil {
		return nil
	}

	now := time.Now()
	dismissed := make(map[string]bool)
	var keepLines []string
	changed := false

	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			changed = true
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		url := parts[0]

		// Determine age: prefer explicit timestamp, fall back to Slack message time
		var age time.Duration
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339, parts[1]); err == nil {
				age = now.Sub(t)
			}
		}
		if age == 0 {
			if t, ok := slackMsgTime(url); ok {
				age = now.Sub(t)
			}
		}

		if age > dismissalTTL {
			changed = true
			continue
		}

		keepLines = append(keepLines, line)
		dismissed[url] = true
	}

	if changed {
		newContent := strings.Join(keepLines, "\n")
		if len(keepLines) > 0 {
			newContent += "\n"
		}
		os.WriteFile(dismissedPath, []byte(newContent), 0644)
	}

	return dismissed
}

func isItemStart(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "🔴") ||
		strings.HasPrefix(t, "🟡") ||
		strings.HasPrefix(t, "🟢") ||
		strings.HasPrefix(t, "✅")
}

func filterBrief(brief string, dismissed map[string]bool) string {
	if len(dismissed) == 0 {
		return brief
	}

	lines := strings.Split(brief, "\n")

	// Two-pass: group into preamble + item blocks, then drop entire blocks
	// that contain a dismissed URL anywhere (not just after the link line).
	type block struct {
		lines      []string
		hasDismiss bool
	}

	var preamble []string
	var blocks []block
	var cur *block

	for _, line := range lines {
		if isItemStart(line) {
			if cur != nil {
				blocks = append(blocks, *cur)
			}
			cur = &block{}
		}
		if cur == nil {
			preamble = append(preamble, line)
			continue
		}
		cur.lines = append(cur.lines, line)
		for url := range dismissed {
			if strings.Contains(line, url) {
				cur.hasDismiss = true
				break
			}
		}
	}
	if cur != nil {
		blocks = append(blocks, *cur)
	}

	var filtered []string
	filtered = append(filtered, preamble...)
	for _, b := range blocks {
		if !b.hasDismiss {
			filtered = append(filtered, b.lines...)
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
