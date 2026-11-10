# lazytoggl — Requirements

## P0 — MVP features

1. **Onboarding.** On first run (no token in the keychain), prompt for the Toggl API
   token, validate it with `GET /me`, store it in the OS keychain, and cache the default
   `workspace_id` in the config file. The token never touches disk in plaintext.
2. **Start a time entry** with a description and, optionally, a project.
3. **Stop the active entry.**
4. **Log a past entry** (`m`) — like Toggl's manual mode: a form for a finished
   entry that opens on the 30 minutes ending now, with the edit form's fields, rules,
   and project picker. `enter` on the description logs it (1 request); it never stops
   a running timer.
5. **Active-timer panel** showing the running entry's description, project, and a
   live-ticking elapsed time (computed locally from the entry's start timestamp — no API
   calls to tick).
6. **Recent entries list** — today plus the past several days, grouped by day with
   per-day totals.
7. **Resume** — select a past entry and start a new entry pre-filled with its
   description/project.
8. **Projects** — pick, clear, or create an entry's project in the start and edit
   dialogs, from a project row's picker or by typing `@` in the description (as in
   the Toggl app). The list is "No project" (when there is one to remove), then
   active projects A–Z; typing filters by substring and offers `+ Create "<name>"`
   when no name matches exactly. A new project is created only when the entry is
   saved (1 extra request).
9. **Edit** an entry's description, project, start/stop date and time, and duration (times are
   edited as `HH:MM`, with the date beside them in the same field), reviewing a
   before → after summary before saving; **delete** with a confirmation dialog.
10. **Manual refresh only** (`r`). No background polling. Every mutation updates local
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
| `m` | anywhere | Log a finished entry (Toggl's manual mode): open the log-entry dialog on the last 30 minutes |
| `space` | anywhere | Pause/resume: toggles whatever the Timer panel already shows. Running → stops it via the API but keeps it **pinned** to the panel (⏸, frozen clock) instead of clearing to idle. Paused → continues it (Toggl has no true pause — resume starts a new entry copying the paused one, like the desktop app's "continue" button). Idle (nothing running or paused) → **no-op**; it never reaches into history to resume something on its own |
| `enter` | on a past entry | Open start dialog **pre-filled** from that entry (resume); `enter` again to go |
| `enter` | on the running entry | Stop it (clears the Timer panel to idle — unlike `space`) |
| `enter` | Timer panel | Stop the running timer; open the create dialog when idle |
| `e` / `d` | entries list | Edit / delete (with confirmation) the selected entry |
| `e` / `d` | Timer panel | Edit / delete whatever the panel is showing — the **running** entry, or the **paused** one if nothing is running — regardless of list selection |
| `j` / `k`, arrows | list | Navigate |
| `tab` | anywhere | Cycle panel focus |
| `r` | anywhere | Refresh from API |
| `?` | anywhere | Help overlay |
| `q` / `ctrl+c` | anywhere | Quit (`esc` closes dialogs) |

### Start dialog

| Key | Context | Action |
|---|---|---|
| `tab` / `shift+tab`, `down` / `up` | form | Move between description and project |
| `enter` | description | Start the timer (while no `@` dropdown is open) |
| `enter` | project row | Open the project picker |
| `esc` | form | Cancel |

### Log-entry dialog

The edit form for a new, finished entry: the same rows, fields, keys, and project
picker as the edit dialog, with these differences.

| Key | Context | Action |
|---|---|---|
| `enter` | description | Log the entry (while no `@` dropdown is open); 1 request, 2 with a `(new)` project |
| `esc` | form | Cancel: closes at once if nothing changed from the defaults, otherwise asks `Discard this entry?` |
| `y` | discard prompt | Discard and close |
| `n` / `esc` | discard prompt | Back to the form |

### Projects (start and edit dialogs)

| Key | Context | Action |
|---|---|---|
| `@` | description, at the start or after a space | Open the project dropdown below the description; the text after `@` is its search (spaces included). `@` inside a word types normally |
| letters, `backspace` | dropdown | Edit the description and refilter; deleting the `@` or moving the cursor before it closes the dropdown |
| letters, `backspace` | picker | Edit the picker's own search |
| `up` / `down`, `ctrl+p` / `ctrl+n` | dropdown, picker | Move the selection |
| `enter`, `tab` | dropdown, picker | Pick: set the project row (the dropdown also removes `@search` from the description). Never starts or saves |
| `esc` | dropdown | Close it, keeping the typed text |
| `esc` | picker | Close it, project unchanged |

### Edit dialog

| Key | Context | Action |
|---|---|---|
| `tab` / `shift+tab`, `down` / `up` | form | Move between rows (wraps) |
| `enter` | description row | Move to the next row |
| `enter` | start / stop / duration row | Open that row's field: date + `HH:MM` for start / stop, `HH:MM` for duration |
| `enter` | project row (last row) | Open the project picker |
| `esc` | form | Close: no changes → closes with no API call; otherwise opens the review (while the `@` dropdown is open, `esc` only closes the dropdown) |
| `0`–`9` | time field | Overwrite the digit under the cursor and advance (skips the `:`) |
| `h` / `l`, `^` / `$`, `w` / `b` | time field | Move one digit; first / last digit; minutes / hours |
| `tab` / `shift+tab` | start / stop field | Switch between the date and the time (opens on the time; no effect in the duration field) |
| `ctrl+a` / `ctrl+x` | time field | +1 / −1 on the hours or minutes under the cursor (minutes carry into hours; on start / stop, hours carry across midnight into the date; duration stays within 00:01–23:59) |
| `ctrl+a` / `ctrl+x` | date (start / stop field) | +1 / −1 day (digits and motions do nothing here) |
| `enter` / `esc` | time field | Accept (hours 00–23, minutes 00–59) / revert the field |
| `y` / `enter` | review | Save (one API request; two when it creates a `(new)` project first) |
| `esc` | review | Back to the form, edits intact |
| `n` | review | Discard all edits and close |

Editing start keeps stop fixed; editing stop or duration keeps start fixed. Stepping
**start's date** moves the whole entry (stop moves by the same days); stepping stop's
date moves only stop. A typed time crosses midnight when that makes sense: a stop lands
on its first occurrence after start, a start on its last occurrence before stop (or
now, for a running entry), if the entry is then 12 hours or shorter; otherwise it stays
on the field's date. The field's date updates live as you type, and a date you changed
by hand is kept. An entry is never shorter than 1 minute (Toggl rejects zero-length
entries): a time that would cross the other end lands 1 minute from it. No date goes past
today. A running entry's stop and duration can't be edited, its start can be at most 1
minute before now, and its start date no earlier than 7 days ago (so a refresh still
finds it). A finished entry can move outside the list's window (before the last 8 days, or to a
start later than now): the review marks it `(outside list)` and the status line warns
that it drops off on the next refresh.

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

- Tag / client pickers when starting or editing an entry (projects are done; fuzzy
  matching for the project list is a possible follow-up)
- Billable flag
- Reports: daily/weekly totals view (Reports API)
- Multi-workspace switching
- Offline queue (start/stop while offline, sync later)
- Pomodoro mode
- Calendar/timeline view
- Deletion undo (grace period before the API call)
