package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed prompt.md
var promptTemplate string

const briefFile = "slack-interesting.md"

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

	// Build prompt with previous brief
	prompt := strings.ReplaceAll(promptTemplate, "{{PREVIOUS_BRIEF}}", previousBrief)

	// Run claude CLI with henchman tools
	cmd := exec.Command("/opt/homebrew/bin/claude",
		"--allowedTools", "mcp__henchman__search,mcp__henchman__getThreadDetails,mcp__henchman__channelLookup,mcp__henchman__userLookup",
		"-p", prompt,
	)

	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running claude: %v\n", err)
		os.Exit(1)
	}

	result := strings.TrimSpace(string(output))

	// Check for NO_UPDATE
	if strings.Contains(result, "NO_UPDATE") {
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
