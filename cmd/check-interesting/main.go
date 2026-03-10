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
	HookSpecificOutput HookSpecificData `json:"hookSpecificOutput"`
}

type HookSpecificData struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

func main() {
	homeDir, _ := os.UserHomeDir()

	docsDir := getDocsDir()
	if docsDir == "" {
		fmt.Fprintf(os.Stderr, "Error: Could not determine docs directory\n")
		os.Exit(1)
	}

	briefPath := filepath.Join(docsDir, briefFile)

	// Read previous brief
	previousBrief := "No previous brief - first run."
	if data, err := os.ReadFile(briefPath); err == nil {
		previousBrief = string(data)
	}

	// Read user reactions to previously surfaced items
	// (showed interest, dismissed, said not relevant, etc.)
	reactionsPath := filepath.Join(docsDir, reactionsFile)
	userReactions := "No reactions recorded yet."
	if data, err := os.ReadFile(reactionsPath); err == nil && len(data) > 0 {
		userReactions = string(data)
	}

	// Build prompt with previous brief and user reactions
	prompt := strings.ReplaceAll(promptTemplate, "{{PREVIOUS_BRIEF}}", previousBrief)
	prompt = strings.ReplaceAll(prompt, "{{USER_REACTIONS}}", userReactions)

	// Run claude CLI with henchman tools
	claudePath := filepath.Join(homeDir, ".local", "bin", "claude")
	mcpConfig := filepath.Join(homeDir, "tools", "check-slack-for-relevant-stuff", "mcp-config.json")
	cmd := exec.Command(claudePath,
		"--mcp-config", mcpConfig,
		"--strict-mcp-config",
		"--allowedTools", "Bash,mcp__henchman__search,mcp__henchman__getThreadDetails,mcp__henchman__channelLookup,mcp__henchman__userLookup",
		"-p", prompt,
	)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

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
		timestamp := time.Now().Format("2006-01-02 15:04:05")

		// Log error to separate error log
		errLogPath := "/tmp/slack-adhd-errors.log"
		if f, err2 := os.OpenFile(errLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err2 == nil {
			fmt.Fprintf(f, "%s | check-interesting | %v | %s\n", timestamp, err, stderr.String())
			f.Close()
		}

		// Write a flat error notice (no nesting of previous brief)
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		failureNotice := fmt.Sprintf("⚠️ Slack check failed at %s: %s", timestamp, errMsg)
		os.WriteFile(briefPath, []byte(failureNotice), 0644)
		os.Exit(1)
	}

	output := stdout.String()
	result := strings.TrimSpace(output)

	// Detect Henchman MCP failure - even if Claude says NO_UPDATE
	resultLower := strings.ToLower(result)
	henchmanDown := strings.Contains(result, "HENCHMAN_UNAVAILABLE") ||
		(strings.Contains(resultLower, "henchman") &&
			(strings.Contains(resultLower, "not connected") ||
				strings.Contains(resultLower, "not available") ||
				strings.Contains(resultLower, "isn't connected") ||
				strings.Contains(resultLower, "no henchman")))
	if henchmanDown {
		now := time.Now()
		warning := fmt.Sprintf("⚠️ **Henchman MCP is unreachable** — Slack checks are not running. You may need to re-authenticate.\n\nLast attempted: %s", now.Format("2006-01-02 15:04 MST"))
		os.WriteFile(briefPath, []byte(warning), 0644)
		os.Exit(0)
	}

	// Check for NO_UPDATE
	if strings.Contains(result, "NO_UPDATE") {
		// Claude says the previous brief is still valid - keep it but strip any error headers
		cleanBrief := previousBrief

		// Strip error messages from previous brief
		if strings.Contains(cleanBrief, "## Slack Brief - FAILED TO UPDATE") {
			parts := strings.Split(cleanBrief, "---")
			if len(parts) >= 2 {
				cleanBrief = strings.TrimSpace(strings.Join(parts[1:], "---"))
			}
		}
		if strings.HasPrefix(cleanBrief, "⚠️") {
			cleanBrief = ""
		}

		// If there's no good content left, write "all clear"
		if cleanBrief == "" || cleanBrief == "No previous brief - first run." || len(cleanBrief) < 50 {
			now := time.Now()
			cleanBrief = fmt.Sprintf(`Nothing particularly interesting in the past 48 hours - all quiet!

Last checked: %s`, now.Format("2006-01-02 15:04 MST"))
		}

		if err := os.WriteFile(briefPath, []byte(cleanBrief), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing clean brief: %v\n", err)
			os.Exit(1)
		}
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
