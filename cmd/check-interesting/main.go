package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed prompt.md
var promptTemplate string

const briefFile = "slack-interesting.md"
const reactionsFile = "slack-reactions.md"

type HookOutput struct {
	HookSpecificOutput HookSpecificData `json:"hookSpecificOutput"`
}

type HookSpecificData struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home dir: %v\n", err)
		os.Exit(1)
	}

	briefPath := filepath.Join(homeDir, "THE_SINK", "docs", briefFile)

	// Read previous brief
	previousBrief := "No previous brief - first run."
	if data, err := os.ReadFile(briefPath); err == nil {
		previousBrief = string(data)
	}

	// Read user reactions to previously surfaced items
	// (showed interest, dismissed, said not relevant, etc.)
	reactionsPath := filepath.Join(homeDir, "THE_SINK", "docs", reactionsFile)
	userReactions := "No reactions recorded yet."
	if data, err := os.ReadFile(reactionsPath); err == nil && len(data) > 0 {
		userReactions = string(data)
	}

	// Build prompt with previous brief and user reactions
	prompt := strings.ReplaceAll(promptTemplate, "{{PREVIOUS_BRIEF}}", previousBrief)
	prompt = strings.ReplaceAll(prompt, "{{USER_REACTIONS}}", userReactions)

	// Run claude CLI with henchman tools
	cmd := exec.Command("/opt/homebrew/bin/claude",
		"--allowedTools", "mcp__henchman__search,mcp__henchman__getThreadDetails,mcp__henchman__channelLookup,mcp__henchman__userLookup",
		"-p", prompt,
	)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Log Claude's output to /tmp for debugging
	logPath := "/tmp/slack-interesting.log"
	if f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(f, "\n=== %s ===\n", timestamp)
		if stdout.Len() > 0 {
			fmt.Fprintf(f, "Claude output:\n%s\n", stdout.String())
		} else {
			fmt.Fprintf(f, "Claude output: (empty)\n")
		}
		if stderr.Len() > 0 {
			fmt.Fprintf(f, "Stderr:\n%s\n", stderr.String())
		}
		f.Close()
	}

	if err != nil {
		// Log to stderr for cron logs
		fmt.Fprintf(os.Stderr, "Error running claude: %v\nStderr: %s\n", err, stderr.String())

		// Write failure notice to brief file so the hook surfaces it
		failureNotice := fmt.Sprintf(`## Slack Brief - FAILED TO UPDATE

⚠️ **Slack check failed** - couldn't fetch latest messages
   Error: %v
   When: just now

The previous brief (below) may be stale. Cron will retry in 15 minutes.

---
%s`, err, previousBrief)

		os.WriteFile(briefPath, []byte(failureNotice), 0644)
		os.Exit(1)
	}

	output := stdout.String()
	result := strings.TrimSpace(output)

	// Check for NO_UPDATE
	if strings.Contains(result, "NO_UPDATE") {
		// Touch the file so user can see when we last checked
		now := time.Now()
		os.Chtimes(briefPath, now, now)
		os.Exit(0)
	}

	// Write brief to file (for history/debugging)
	if err := os.MkdirAll(filepath.Dir(briefPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(briefPath, []byte(result), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing brief: %v\n", err)
		os.Exit(1)
	}

	// Build friendly wrapper - this is discovery, not obligations
	context := fmt.Sprintf(`Some cool stuff happened in Slack while you were away. No action needed - just interesting things you might enjoy checking out when you have a moment.

%s

---
**These are just for fun** - rabbit holes to explore if you're curious, not tasks to complete.`, result)

	// Output JSON for hook consumption
	hookOutput := HookOutput{
		HookSpecificOutput: HookSpecificData{
			HookEventName:     "SessionStart",
			AdditionalContext: context,
		},
	}

	jsonOut, _ := json.MarshalIndent(hookOutput, "", "  ")
	fmt.Println(string(jsonOut))
}
