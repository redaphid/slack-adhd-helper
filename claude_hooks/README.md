# Slack ADHD Assist - Claude Code Hooks

Proactive Slack awareness system for Claude Code sessions, designed as an ADHD accommodation to surface important information without requiring manual Slack checking.

## What This Does

This system automatically:
1. **Searches Slack** every 15 minutes via cron jobs
2. **Uses Claude AI** to analyze and summarize findings
3. **Surfaces briefs** at the start of Claude Code sessions via hooks
4. **Learns your preferences** by tracking reactions to avoid repetition

Think of it as a screen reader for Slack - it brings important discussions to you instead of making you hunt through channels.

## Architecture

```
┌─────────────────┐
│   Cron Jobs     │ Every 15 minutes
│  (check-*)      │
└────────┬────────┘
         │ Invoke Claude CLI with prompts
         ▼
┌─────────────────┐
│ Henchman MCP    │ Searches Slack
│ (semantic)      │
└────────┬────────┘
         │ Returns threads
         ▼
┌─────────────────┐
│  Claude AI      │ Analyzes & summarizes
│  (analyzes)     │
└────────┬────────┘
         │ Writes markdown briefs
         ▼
┌─────────────────┐
│ ~/THE_SINK/docs │
│  slack-*.md     │ Brief files
└────────┬────────┘
         │ Read by hooks
         ▼
┌─────────────────┐
│  Hook Scripts   │ Inject into Claude Code
│  (shell)        │
└────────┬────────┘
         │ Surface in session
         ▼
┌─────────────────┐
│   You see it!   │ At session start
└─────────────────┘
```

## Two Brief Types

### 1. Critical (time-sensitive)
- Blockers and urgent items
- @mentions of you
- Questions waiting for response
- Fires on `UserPromptSubmit` hook

### 2. Interesting (discovery)
- Cool technical discussions
- Shipped features and wins
- Architecture decisions
- AI/ML insights
- Fires on `SessionStart` hook

## Installation

### Prerequisites

1. **Henchman MCP** must be configured in your Claude Code MCP settings
2. **Docs directory** for brief storage (configurable, defaults to `~/THE_SINK/docs/`)
3. **Shell environment** - Works with zsh, bash, or fish

### Step 0: Configure Docs Directory (Optional)

By default, briefs are stored in `~/THE_SINK/docs/`. To use a different location, set the `SLACK_PULSE_DOCS_DIR` environment variable in your shell config:

**For zsh** (`~/.zshrc`):
```bash
# Slack Pulse - ADHD Assist
export SLACK_PULSE_DOCS_DIR="$HOME/THE_SINK/docs"
```

**For bash** (`~/.bashrc` or `~/.bash_profile`):
```bash
# Slack Pulse - ADHD Assist
export SLACK_PULSE_DOCS_DIR="$HOME/THE_SINK/docs"
```

**For fish** (`~/.config/fish/config.fish`):
```fish
# Slack Pulse - ADHD Assist
set -x SLACK_PULSE_DOCS_DIR "$HOME/THE_SINK/docs"
```

After adding, reload your shell:
```bash
source ~/.zshrc  # or ~/.bashrc, etc.
```

### Step 1: Build the Binaries

```bash
cd ~/tools/check-slack-for-relevant-stuff

# Build cron job binaries
cd cmd/check-critical && go build -o ../../bin/check-critical
cd ../check-interesting && go build -o ../../bin/check-interesting

# Build hook binaries
cd ../hook-critical && go build -o ../../bin/hook-critical
cd ../hook-interesting && go build -o ../../bin/hook-interesting
```

### Step 2: Install Hooks

```bash
# Create hooks directory if it doesn't exist
mkdir -p ~/.claude/hooks

# Symlink the hook scripts
ln -s ~/tools/check-slack-for-relevant-stuff/claude_hooks/slack-critical.sh \
      ~/.claude/hooks/slack-critical.sh

ln -s ~/tools/check-slack-for-relevant-stuff/claude_hooks/slack-interesting.sh \
      ~/.claude/hooks/slack-interesting.sh

# Make them executable
chmod +x ~/.claude/hooks/slack-*.sh
```

### Step 3: Set Up Cron Jobs

```bash
crontab -e
```

Add these lines (using zsh with login shell to load your environment):

```cron
# Slack ADHD Assist - check every 15 minutes
*/15 * * * * /bin/zsh -l -c "/Users/YOUR_USERNAME/tools/check-slack-for-relevant-stuff/bin/check-critical" >> /tmp/slack-critical.log 2>&1
*/15 * * * * /bin/zsh -l -c "/Users/YOUR_USERNAME/tools/check-slack-for-relevant-stuff/bin/check-interesting" >> /tmp/slack-interesting.log 2>&1
```

**Replace `YOUR_USERNAME` with your actual username!**

**Alternative shells:**

For fish:
```cron
*/15 * * * * /opt/homebrew/bin/fish -l -c "/Users/YOUR_USERNAME/tools/check-slack-for-relevant-stuff/bin/check-critical" >> /tmp/slack-critical.log 2>&1
*/15 * * * * /opt/homebrew/bin/fish -l -c "/Users/YOUR_USERNAME/tools/check-slack-for-relevant-stuff/bin/check-interesting" >> /tmp/slack-interesting.log 2>&1
```

For bash (or without shell wrapper):
```cron
*/15 * * * * /Users/YOUR_USERNAME/tools/check-slack-for-relevant-stuff/bin/check-critical >> /tmp/slack-critical.log 2>&1
*/15 * * * * /Users/YOUR_USERNAME/tools/check-slack-for-relevant-stuff/bin/check-interesting >> /tmp/slack-interesting.log 2>&1
```

The `-l` flag ensures your login shell environment is loaded, so Claude CLI and MCP servers can be found.

### Step 4: Create Brief Storage Directory

Create the directory you configured in Step 0 (or use the default):

```bash
# If you set SLACK_PULSE_DOCS_DIR, use that value:
mkdir -p "$SLACK_PULSE_DOCS_DIR"
touch "$SLACK_PULSE_DOCS_DIR/slack-critical.md"
touch "$SLACK_PULSE_DOCS_DIR/slack-interesting.md"
touch "$SLACK_PULSE_DOCS_DIR/slack-reactions.md"

# Or if using the default:
mkdir -p ~/THE_SINK/docs
touch ~/THE_SINK/docs/slack-critical.md
touch ~/THE_SINK/docs/slack-interesting.md
touch ~/THE_SINK/docs/slack-reactions.md
```

## Configuration

### Customizing Search Queries

Edit the embedded prompt files and rebuild:

- `cmd/check-critical/prompt.md` - Critical item search logic
- `cmd/check-interesting/prompt.md` - Interesting discussion logic

After editing, rebuild the binaries (see Step 1).

### Customizing Hook Messages

Edit the wrapper templates:

- `cmd/hook-critical/wrapper.md` - Critical brief presentation
- `cmd/hook-interesting/wrapper.md` - Interesting brief presentation

After editing, rebuild the hook binaries.

## How the Learning System Works

When you react to items surfaced in briefs, Claude is instructed to log your reaction to `~/THE_SINK/docs/slack-reactions.md`:

```markdown
## 2025-12-12 15:30

### Nick's Build Performance Thread
- **Link:** https://sibi-workspace.slack.com/...
- **Reaction:** interested
- **Notes:** Clicked through, asked follow-up questions
```

The cron job binaries read this file and include it in the prompt to Claude, so it learns to:
- Stop surfacing threads you've dismissed
- Prioritize topics you engage with
- Avoid repetition

## Troubleshooting

### Hooks Not Firing

Check Claude Code hook logs:
```bash
tail -f ~/.claude/hooks/logs/slack-critical.log
tail -f ~/.claude/hooks/logs/slack-interesting.log
```

### Cron Jobs Not Running

Check cron logs:
```bash
tail -f /tmp/slack-critical.log
tail -f /tmp/slack-interesting.log
```

Verify cron is running:
```bash
crontab -l
```

### "Henchman MCP not available"

If you see this message in the briefs:
1. Check that Henchman MCP is in your Claude Code MCP settings
2. Restart your Claude Code session
3. Verify MCP connection: The hooks will auto-recover once it's available

### Empty Briefs

If briefs are always empty:
1. Check that cron jobs are running (see above)
2. Verify `~/THE_SINK/docs/slack-*.md` files exist
3. Try running a binary manually to test:
   ```bash
   ~/tools/check-slack-for-relevant-stuff/bin/check-critical
   ```

## Customization for Your Team

### Change Slack Channels

Edit the search queries in:
- `cmd/check-critical/prompt.md`
- `cmd/check-interesting/prompt.md`

Look for `excludeChannels` arrays and channel-specific filters.

### Change Update Frequency

Modify the cron schedule. Examples:

```cron
# Every 30 minutes
*/30 * * * * ...

# Every hour
0 * * * * ...

# Every 5 minutes (aggressive!)
*/5 * * * * ...

# Only during work hours (9 AM - 6 PM)
*/15 9-18 * * * ...
```

### Disable One Type

Just remove the corresponding cron job or hook symlink:

```bash
# Disable interesting briefs
rm ~/.claude/hooks/slack-interesting.sh

# Or disable the cron
crontab -e  # and comment out the interesting line
```

## Files Overview

```
claude_hooks/
├── README.md                    # This file
├── slack-critical.sh            # Hook script (UserPromptSubmit)
└── slack-interesting.sh         # Hook script (SessionStart)

cmd/
├── check-critical/              # Cron job for critical items
│   ├── main.go
│   └── prompt.md               # Search & analysis logic
├── check-interesting/           # Cron job for interesting items
│   ├── main.go
│   └── prompt.md               # Search & analysis logic
├── hook-critical/               # Hook binary for critical
│   ├── main.go
│   └── wrapper.md              # Presentation template
└── hook-interesting/            # Hook binary for interesting
    ├── main.go
    └── wrapper.md              # Presentation template

bin/
├── check-critical               # Cron binary
├── check-interesting            # Cron binary
├── hook-critical                # Hook binary
└── hook-interesting             # Hook binary
```

## Privacy & Security

- All processing happens locally via Claude Code
- No data sent to third parties (except normal Claude API usage)
- Slack searches use Henchman MCP (local or self-hosted)
- Brief files stored in `~/THE_SINK/docs` (local)
- Reaction logs help train the system for you personally

## Credits

Built by [@hypnodroid](https://github.com/hypnodroid) as an ADHD accommodation tool.

Uses:
- [Claude Code](https://claude.ai/download) by Anthropic
- [Henchman MCP](https://github.com/cryingpotat0/henchman) for Slack search
- Go for cron/hook binaries

## License

MIT - Use freely, modify as needed, share with your team!
