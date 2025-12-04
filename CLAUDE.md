# slack-pulse Development

## What This Is

Go binaries that invoke Claude CLI to search Slack via Henchman MCP and generate briefs. The briefs are consumed by lightweight shell hooks in Claude Code.

## Architecture

**Binaries (run via cron):**
- `cmd/check-critical/main.go` - urgent work items
- `cmd/check-interesting/main.go` - discovery/interests

**Prompts (embedded at compile time):**
- `cmd/check-critical/prompt.md`
- `cmd/check-interesting/prompt.md`

**Output files:**
- `~/THE_SINK/docs/slack-critical.md`
- `~/THE_SINK/docs/slack-interesting.md`

## Building

```bash
cd cmd/check-critical && go build -o ../../bin/check-critical
cd cmd/check-interesting && go build -o ../../bin/check-interesting
```

Prompts are embedded via `//go:embed` - rebuild after prompt changes.

## Key Behavior

- Previous brief is fed to Claude for context (avoid repeating resolved items)
- `NO_UPDATE` output = no file write = hook shows nothing
- Items auto-dismiss when you respond (detected via Henchman search)

## MCP Dependency

Requires Henchman MCP for Slack semantic search:
- `mcp__henchman__search`
- `mcp__henchman__getThreadDetails`
- `mcp__henchman__channelLookup`
- `mcp__henchman__userLookup`

## Shell Scripts

The `.sh` files in repo root are deprecated - hooks now use `~/.claude/hooks/slack-*.sh` which just read the brief files.
