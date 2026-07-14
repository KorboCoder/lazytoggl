# lazytoggl — Requirements

## P0 — MVP features

1. **Onboarding.** On first run (no token in the keychain), prompt for the Toggl API
   token, validate it with `GET /me`, store it in the OS keychain, and cache the default
   `workspace_id` in the config file. The token never touches disk in plaintext.
2. **Start a time entry** with a description.
3. **Stop the active entry.**
4. **Active-timer panel** showing the running entry's description, project, and a
   live-ticking elapsed time (computed locally from the entry's start timestamp — no API
   calls to tick).
5. **Recent entries list** — today plus the past several days, grouped by day with
   per-day totals.
6. **Resume** — select a past entry and start a new entry pre-filled with its
   description/project.
7. **Edit** an entry's description and start/stop times; **delete** with a confirmation
   dialog.
8. **Manual refresh only** (`r`). No background polling. Every mutation updates local
   state from the API response, so normal usage costs 1 request per action plus 1 fetch
   at startup. This is a hard requirement from the Free-plan rate limit
   (see [ARCHITECTURE.md](ARCHITECTURE.md#rate-limits)).

## Non-functional requirements

- Perceived startup under ~100 ms: render immediately from cached/empty state, load
  entries async.
- Graceful offline/error behavior: failures surface in the status bar; the UI never
  crashes or blocks on the network.
- Strict API request budget (Free plan: 30 requests/hour). Rate-limit responses
  (HTTP 402) show a "rate limited — try again later" message with no automatic retries.
- Adaptive styling for light and dark terminals.

## Keymap

Refined after the Phase 1 prototype review (2026-07-15): `c` creates, `space`
toggles, `enter` acts on the selection.

| Key | Context | Action |
|---|---|---|
| `c` | anywhere | Create: open an empty start dialog |
| `space` | anywhere | Pause/resume: stop the running timer, or continue the most recent entry (Toggl has no true pause — resume starts a new entry copying the last one) |
| `enter` | on a past entry | Open start dialog **pre-filled** from that entry (resume); `enter` again to go |
| `enter` | on the running entry | Stop it |
| `enter` | Timer panel | Stop the running timer; open the create dialog when idle |
| `e` / `d` | entries list | Edit / delete (with confirmation) the selected entry |
| `e` / `d` | Timer panel | Edit / delete the **running** entry, regardless of list selection |
| `j` / `k`, arrows | list | Navigate |
| `tab` | anywhere | Cycle panel focus |
| `r` | anywhere | Refresh from API |
| `?` | anywhere | Help overlay |
| `q` / `ctrl+c` | anywhere | Quit (`esc` closes dialogs) |

## UI sketch (draft — refined during the Phase 1 prototype)

```
┌─ Timer ─────────────────────────────────────────┐
│ ▶ Writing project plan          ACME   00:12:34 │
└─────────────────────────────────────────────────┘
┌─ Entries ── Today · 3h 45m ─────────────────────┐
│ > ▶ Writing project plan        ACME   00:12:34 │
│     Standup                     —      00:15:00 │
│     Code review                 ACME   01:02:11 │
│   Yesterday · 6h 12m                            │
│     ...                                         │
└─────────────────────────────────────────────────┘
 space:pause/resume  c:new  enter:resume/stop  e:edit  d:del
```

- When idle, the Timer panel shows a muted "no timer running — press c to start".
- Dialogs (start, edit, delete-confirm, help) render as centered popups over the panels,
  lazygit-style.

## Backlog (post-MVP, toward feature parity)

Roughly in expected priority order; reordered by feedback during Phase 3.

- Project / tag / client pickers with fuzzy search when starting or editing an entry
- Billable flag
- Reports: daily/weekly totals view (Reports API)
- Multi-workspace switching
- Offline queue (start/stop while offline, sync later)
- Pomodoro mode
- Calendar/timeline view
- Deletion undo (grace period before the API call)
