# Proposal

## Why

The commonest thing to log after the fact is a gap between two entries ("I forgot to
track the bit between standup and code review"). Today that means `m`, then typing
both times. The list is already in time order, so the selected entry's neighbours
give the gap's start and stop exactly. That makes it one key, the way vim's `o` / `O`
open a line below or above.

## What Changes

- **`o` on a selected entry** opens the log-entry dialog on the gap **below** it in
  the list (earlier). It starts where the previous entry that day ends and stops
  where the selected entry starts.
- **`O`** opens it on the gap **above** it (later). It starts where the selected entry
  ends and stops where the next entry that day starts. A running entry counts as that
  next entry, at its start. With no later entry that day it stops at now (to the
  minute) for today, or 30 minutes after the selected entry for a past day.
- **Neighbours come from the selected entry's own day.** With no earlier entry that
  day, `o` opens on the 30 minutes before the selected entry (the same 30-minute
  default as `m`, anchored to the entry).
- **Overlaps are respected.** The gap below starts at the latest stop among that
  day's entries that started earlier; the gap above ends at the earliest start among
  those that start later. A gap is never laid over an existing entry.
- **No gap, no dialog.** When the entries touch or overlap, or the gap is under a
  minute, the status line says `no gap below "<description>"` (or `above`). `m` still
  logs anything.
- **The running entry:** `o` fills the gap before it; `O` says
  `nothing after the running entry`.
- **Everything else is the log dialog as it is**, with the same rows, rules, `@`,
  `enter` to log, and `esc` with the discard prompt. The gap's exact times, seconds
  included, are the dialog's defaults, so a logged gap butts exactly against its
  neighbours.
- `o` / `O` work in the entries list only. On the timer panel they do nothing.

Decided during exploration: neighbours stay within the day. Proposed defaults the user
moved past without changing: the edge-case rules above (running entry, no gap,
overlaps, now as the upper end).

Out of scope: crossing day boundaries, and filling several gaps at once.

### Keybindings

| Context | Key | Action |
|---|---|---|
| Entries list | `o` | Log into the gap below (earlier than) the selected entry **(new)** |
| Entries list | `O` | Log into the gap above (later than) the selected entry **(new)** |

`o` / `O` appear in the full help (`?`), not the short help bar, which is already full
at common widths.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `entry-logging`: adds opening the log dialog on a gap with `o` / `O` (neighbours,
  fallbacks, overlaps, no-gap and running-entry cases). The "Default times"
  requirement is narrowed to `m`, since `o` / `O` open on the gap's times.

## Impact

- `internal/ui/gaps.go` (new): pure functions computing the gap below or above a
  selected entry from the loaded entries and now.
- `internal/ui/dialogs/edit.go`: `NewLogAt(start, stop, now, projects)`, so `NewLog`
  delegates to it with the 30-minute default.
- `internal/ui/app.go`, `keys.go`: `o` / `O` bindings (full help only), and the status
  lines for the no-gap and running cases.
- Tests: `internal/ui/gaps_test.go` (table-driven), driver cases in `log_test.go`.
- Docs: `docs/REQUIREMENTS.md` (keymap, log section), `docs/PLAN.md` (the follow-up
  is now done), `README.md` keys table.
- API: none. Logging a gap costs what `m` costs (1 request, or 2 with a new project).
  Pressing `o` / `O` itself costs nothing.
