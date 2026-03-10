# slack-pulse

ADHD-friendly Slack monitoring for Claude Code hooks. Surfaces urgent items and interesting discussions without overwhelming you.

## How It Works

Two binaries run every 15 minutes via macOS launchd:
- **check-critical** - Finds blocking/pending work items and @mentions
- **check-interesting** - Discovers cool discussions matching your interests

Each binary:
1. Reads previous brief (for context)
2. Runs Claude CLI with Henchman MCP to search Slack
3. Outputs `NO_UPDATE` if nothing changed, or writes updated brief
4. Detects MCP failures and surfaces warnings instead of silently showing "all clear"

Claude Code hooks then read the brief files and surface them at appropriate times.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ launchd (every 15 min)                                          │
├─────────────────────────────────────────────────────────────────┤
│  bin/check-critical ──► claude CLI ──► ~/THE_SINK/docs/slack-critical.md
│  bin/check-interesting ──► claude CLI ──► ~/THE_SINK/docs/slack-interesting.md
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Claude Code Hooks                                               │
├─────────────────────────────────────────────────────────────────┤
│  UserPromptSubmit ──► slack-critical.sh ──► reads brief, shows if 🔴/🟡
│  SessionStart ──► slack-interesting.sh ──► reads brief, shows if 🌟/💡/🔗
└─────────────────────────────────────────────────────────────────┘
```

## Urgency Levels

**Critical (work items):**
- 🔴 Blocking someone - they explicitly said they're waiting
- 🟡 Question pending - asked but not blocking
- 🟢 FYI only - mentioned, no response needed
- ✅ Resolved - already handled

**Interesting (discovery):**
- 🌟 Highlight - really cool, worth reading
- 💡 Interesting - relevant to your interests
- 🔗 Thread worth following - active discussion

## Setup

### 1. Build binaries

```bash
cd cmd/check-critical && go build -o ../../bin/check-critical
cd cmd/check-interesting && go build -o ../../bin/check-interesting
```

### 2. Schedule with launchd (recommended for macOS)

Use launchd instead of cron. Launchd runs in the user's full GUI session, which means the Claude CLI can access MCP servers that require OAuth (like Henchman). Cron runs in a stripped-down environment and MCP auth will fail silently.

Copy the example plists and load them:

```bash
cp examples/com.slack-pulse.check-critical.plist ~/Library/LaunchAgents/
cp examples/com.slack-pulse.check-interesting.plist ~/Library/LaunchAgents/

# Edit the plists to replace YOUR_USERNAME with your macOS username
sed -i '' "s/YOUR_USERNAME/$(whoami)/g" ~/Library/LaunchAgents/com.slack-pulse.check-*.plist

# Load them
launchctl load ~/Library/LaunchAgents/com.slack-pulse.check-critical.plist
launchctl load ~/Library/LaunchAgents/com.slack-pulse.check-interesting.plist
```

To unload:
```bash
launchctl unload ~/Library/LaunchAgents/com.slack-pulse.check-critical.plist
launchctl unload ~/Library/LaunchAgents/com.slack-pulse.check-interesting.plist
```

To manually trigger a run:
```bash
launchctl kickstart gui/$(id -u)/com.slack-pulse.check-critical
```

### 3. Claude Code hooks

Add to `~/.claude/settings.json`:
```json
{
  "hooks": {
    "UserPromptSubmit": [{
      "matcher": "*",
      "hooks": [{ "type": "command", "command": "~/.claude/hooks/slack-critical.sh" }]
    }],
    "SessionStart": [{
      "hooks": [{ "type": "command", "command": "~/.claude/hooks/slack-interesting.sh" }]
    }]
  }
}
```

### Why launchd over cron?

Claude CLI loads MCP servers (like Henchman) that use OAuth for authentication. OAuth tokens are tied to the user's GUI session (keychain, browser cookies). Cron jobs run in a minimal environment without GUI session access, so MCP servers that require OAuth will fail to connect. Launchd agents run in the user's full session context, inheriting all auth state.

## MCP Failure Detection

If the Henchman MCP server is unreachable (e.g. OAuth expired, server down), the binaries detect this and write a clear warning to the brief file:

> ⚠️ **Henchman MCP is unreachable** — Slack checks are not running. You may need to re-authenticate.

This surfaces in your Claude Code session instead of silently showing "all clear."

The binaries use a probe-and-retry strategy: they attempt a trivial search, and if it fails, retry up to 4 times with 15-second sleeps between attempts (~60 seconds total).

## Dismissal

Items auto-dismiss when:
- You respond to the Slack thread (detected on next run)
- Someone else resolves the issue
- The item ages out of the search window

Manual dismiss: `rm ~/THE_SINK/docs/slack-critical.md`

## Dependencies

- Go 1.21+
- Claude CLI (`~/.local/bin/claude`)
- Henchman MCP server (Slack semantic search) configured in `~/.claude.json`
