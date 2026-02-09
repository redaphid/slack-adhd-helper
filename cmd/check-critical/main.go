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

const briefFile = "slack-critical.md"
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
	Decision           string           `json:"decision,omitempty"`
	Reason             string           `json:"reason,omitempty"`
	HookSpecificOutput HookSpecificData `json:"hookSpecificOutput"`
}

type HookSpecificData struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

func main() {
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
	// (handled, dismissed, asked to be reminded, said not important, etc.)
	reactionsPath := filepath.Join(docsDir, reactionsFile)
	userReactions := "No reactions recorded yet."
	if data, err := os.ReadFile(reactionsPath); err == nil && len(data) > 0 {
		userReactions = string(data)
	}

	// Build prompt with previous brief and user reactions
	prompt := strings.ReplaceAll(promptTemplate, "{{PREVIOUS_BRIEF}}", previousBrief)
	prompt = strings.ReplaceAll(prompt, "{{USER_REACTIONS}}", userReactions)

	// Run claude CLI with henchman tools
	cmd := exec.Command("/opt/homebrew/bin/claude",
		"--allowedTools", "Bash,mcp__henchman__search,mcp__henchman__getThreadDetails,mcp__henchman__channelLookup,mcp__henchman__userLookup",
		"-p", prompt,
	)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Log Claude's output to /tmp for debugging
	logPath := "/tmp/slack-critical.log"
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
			fmt.Fprintf(f, "%s | check-critical | %v | %s\n", timestamp, err, stderr.String())
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

	result := strings.TrimSpace(string(output))

	// Check for NO_UPDATE
	if strings.Contains(result, "NO_UPDATE") {
		// Claude says the previous brief is still valid - keep it but strip any error headers
		cleanBrief := previousBrief

		// Strip "FAILED TO UPDATE" header if present
		if strings.Contains(cleanBrief, "## Slack Brief - FAILED TO UPDATE") {
			// Find the separator line
			parts := strings.Split(cleanBrief, "---")
			if len(parts) >= 2 {
				// Keep everything after the first separator (the actual content)
				cleanBrief = strings.TrimSpace(strings.Join(parts[1:], "---"))
			}
		}

		// If there's no good content left, write "all clear"
		if cleanBrief == "" || cleanBrief == "No previous brief - first run." || len(cleanBrief) < 50 {
			now := time.Now()
			cleanBrief = fmt.Sprintf(`All clear! No urgent items, blockers, or @mentions in the past 48 hours.

Last checked: %s`, now.Format("2006-01-02 15:04 MST"))
		}

		if err := os.WriteFile(briefPath, []byte(cleanBrief), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing clean brief: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Write clinical brief to file (for history/debugging)
	if err := os.MkdirAll(filepath.Dir(briefPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(briefPath, []byte(result), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing brief: %v\n", err)
		os.Exit(1)
	}

	// Build compassionate wrapper for Claude to present
	hasBlocking := strings.Contains(result, "🔴")

	var opener string
	if hasBlocking {
		opener = "Hey - just a heads up, someone mentioned they're waiting on something. No judgment, just information so you can decide what to do."
	} else {
		opener = "Some Slack threads bubbled up. Nothing urgent - just keeping you in the loop so things don't pile up in your head."
	}

	context := fmt.Sprintf(`%s

%s

---
**Remember:** You're allowed to finish what you're doing first. These links are here when you're ready - clicking one might actually feel good to knock out.

If any of this feels overwhelming, that's okay. We can look at it together, or you can tell me to remind you later.`, opener, result)

	// Output JSON for hook consumption
	hookOutput := HookOutput{
		Decision: "allow",
		Reason:   "Slack items to surface",
		HookSpecificOutput: HookSpecificData{
			HookEventName:     "UserPromptSubmit",
			AdditionalContext: context,
		},
	}

	jsonOut, _ := json.MarshalIndent(hookOutput, "", "  ")
	fmt.Println(string(jsonOut))
}
