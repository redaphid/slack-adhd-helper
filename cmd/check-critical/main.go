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

const briefFile = "slack-critical.md"

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
