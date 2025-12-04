# slack-pulse

ADHD-friendly Slack monitoring for Claude Code hooks. Surfaces urgent items and interesting discussions without overwhelming you.

## How It Works

Two binaries run via cron every 15 minutes:
- **check-critical** - Finds blocking/pending work items
- **check-interesting** - Discovers cool discussions matching your interests

Each binary:
1. Reads previous brief (for context)
2. Runs Claude CLI with Henchman MCP to search Slack
3. Outputs `NO_UPDATE` if nothing changed, or writes updated brief

Claude Code hooks then read the brief files and surface them at appropriate times.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ Cron (every 15 min)                                             │
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

### Build binaries
```bash
cd cmd/check-critical && go build -o ../../bin/check-critical
cd cmd/check-interesting && go build -o ../../bin/check-interesting
```

### Cron jobs
```cron
*/15 * * * * ~/tools/check-slack-for-relevant-stuff/bin/check-critical >> /tmp/slack-critical.log 2>&1
*/15 * * * * ~/tools/check-slack-for-relevant-stuff/bin/check-interesting >> /tmp/slack-interesting.log 2>&1
```

### Claude Code hooks
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

## Dismissal

Items auto-dismiss when:
- You respond to the Slack thread (detected on next cron run)
- Someone else resolves the issue
- The item ages out of the search window

Manual dismiss: `rm ~/THE_SINK/docs/slack-critical.md`

## Dependencies

- Go 1.21+
- Claude CLI (`/opt/homebrew/bin/claude`)
- Henchman MCP server (Slack semantic search)
