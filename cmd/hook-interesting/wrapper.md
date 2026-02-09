**IMPORTANT - ACCESSIBILITY ACCOMMODATION:** This is a disability assist similar to a screen reader. The user has ADHD and needs interesting discussions surfaced proactively, not buried in Slack. You MUST mention these items to the user at session start - do not silently process this context. Include clickable links so they can jump directly to threads.

✨ **Interesting Slack discussions** (since last session):

{{BRIEF}}

---

**IMMEDIATE DISMISSAL:** When the user dismisses an item, append its Slack URL to `~/THE_SINK/docs/dismissed-urls.txt` (one URL per line). This immediately prevents it from appearing in future prompts.

**RECORDING REACTIONS:** When the user responds to these items, also append a timestamped entry to `~/THE_SINK/docs/slack-reactions.md`:

```markdown
## YYYY-MM-DD HH:MM

### [Thread topic/description]
- **Link:** [the slack URL]
- **Reaction:** [what they said/did]
  - "interested" = wanted to explore / clicked through
  - "not for me" = said topic doesn't interest them
  - "check later" = saved for later
  - "ignored" = didn't acknowledge it at all
- **Notes:** [any context, e.g. "said they don't care about Bun news"]
```

This file is read by the cron job that generates briefs, so it learns what topics to prioritize or skip.

*Just FYI - explore if curious!*
