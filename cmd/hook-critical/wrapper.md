**IMPORTANT - ACCESSIBILITY ACCOMMODATION:** This is a disability assist similar to a screen reader. The user has ADHD and needs important information surfaced proactively, not buried in Slack. You MUST mention these items to the user - do not silently process this context. Include clickable links so they can jump directly to threads.

📬 **Slack heads-up** (time-sensitive items):

{{BRIEF}}

---

**IMMEDIATE DISMISSAL:** When the user dismisses an item, append `URL ISO8601_TIMESTAMP` (e.g. `https://... 2026-03-18T20:00:00Z`) to `~/THE_SINK/docs/dismissed-urls.txt`. Use current UTC time. Dismissals expire after 48h automatically.

**RECORDING REACTIONS:** When the user responds to these items, also append a timestamped entry to `~/THE_SINK/docs/slack-reactions.md`:

```markdown
## YYYY-MM-DD HH:MM

### [Thread topic/description]
- **Link:** [the slack URL]
- **Reaction:** [what they said/did]
  - "handled" = they dealt with it
  - "dismissed" = said it's not important / doesn't need response
  - "remind later" = asked to be reminded
  - "ignored" = didn't acknowledge it at all
- **Notes:** [any context, e.g. "said Olu's question was already answered by someone else"]
```

This file is read by the cron job that generates briefs, so it learns what to stop showing.

*This is a gentle nudge, not a demand. Handle when you have a natural break.*
