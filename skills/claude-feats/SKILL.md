# claude-feats

claude-feats is a Go CLI tool that tracks RPG-style achievements ("feats") across Claude Code sessions.
It hooks into Claude Code's `Stop` event, analyzes session transcripts, and unlocks feats stored in `~/.claude-feats/`.

## Commands

| Command | What it does |
|---|---|
| `claude-feats list` | Show all visible feats — unlocked and locked |
| `claude-feats list --unlocked` | Show only unlocked feats |
| `claude-feats stats` | Full display: rarity bars, streak, sessions by day |
| `claude-feats mana` | Monthly token usage bar chart + lifetime spellbook total |
| `claude-feats bio` | Call the Anthropic API → generate an RPG character bio |
| `claude-feats export` | Dump progress to markdown (for Slack, Notion, etc.) |
| `claude-feats hook install` | Wire the Stop hook into `~/.claude/settings.json` |
| `claude-feats hook install --skill` | Also install this skill into `~/.claude/skills/` |
| `claude-feats hook uninstall` | Remove the Stop hook |

## How feats are detected

At the end of every session the `Stop` hook fires `claude-feats check`, which:
1. Records token usage (mana) from the hook payload immediately.
2. Spawns a detached worker process that parses the transcript and runs all detectors.
3. Prints unlock notifications to the terminal for new feats.

Detection is fully deterministic — no LLM inference, just regex and timestamps.

## Answering user questions

**"How many feats do I have?"**
Run `claude-feats list --unlocked` and count the lines. Report the total and call out any Epic or Legendary feats by name.

**"What feats have I unlocked?"**
Run `claude-feats list --unlocked` and show the output.

**"Tell me about my coding habits" / "What kind of coder am I?"**
Suggest running `claude-feats bio` — it calls Claude haiku with their feat history and generates a personalized RPG character bio streamed to the terminal.

**"How many tokens have I used?" / "What's my mana?"**
Run `claude-feats mana` and show the monthly breakdown.

**"How do I install the hook?"**
Tell them to run `claude-feats hook install`. This patches `~/.claude/settings.json` to run `claude-feats check` on every session stop. They need `claude-feats` on their PATH first (`go install github.com/dirkbrand/claude-feats@latest`).

## Reading the output

`claude-feats list` output format:
```
  [ C ]  Interrupt Strike       Pulled the cord mid-session. Decisive.
  [RAR]  ░░░ [locked]           ???
```
- Badge shows rarity: `[ C ]`=Common, `[UCO]`=Uncommon, `[RAR]`=Rare, `[EPC]`=Epic, `[LEG]`=Legendary
- `░░░ [locked]` means the feat exists but hasn't been unlocked yet
- `×N` after a name means it's been earned N times

## Hidden feats

There are feats that never appear in `claude-feats list`. They only surface as a surprise
notification the moment they're first unlocked.

You may acknowledge that hidden feats exist if asked — *"Yes, there are feats you haven't
discovered yet"* — but you must never name, describe, or hint at the conditions for any of them.
Let the user find them through natural use.

## Storage

Progress is stored in `~/.claude-feats/progress.json`. It tracks:
- Per-feat: unlock count, first/last unlocked timestamp, personal records
- Streak: current and longest consecutive-day streaks
- Sessions: total count, distribution by day of week, first message history
- Mana: lifetime token totals + monthly breakdowns
