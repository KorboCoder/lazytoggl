# lazytoggl

A lazygit-style TUI for [Toggl Track](https://toggl.com/track/), built with Go and
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

**Status: Phase 2 — MVP.** Talks to the real Toggl API. See [docs/PLAN.md](docs/PLAN.md)
for the roadmap.

## Run

```sh
go run .
```

On first run, it prompts for your Toggl API token (find it at
track.toggl.com → Profile settings → API Token), validates it, and stores it in your
OS keychain — it's never written to disk in plaintext. Subsequent runs skip this.

To run against seeded fake data instead of the real API (no account needed):

```sh
go run . -mock
```

`$TOGGL_API_TOKEN` overrides the keychain if set, for headless/dev use.

## Test

```sh
go test -race ./...
```

UI flows are covered two ways: `teatest` drives the real Bubble Tea event loop
end-to-end, and a synchronous driver (`internal/ui/driver_test.go`) exercises
every dialog flow deterministically against fixture data pinned to a fixed
clock.

## Keys

| Key | Action |
|---|---|
| `c` | Create a new time entry (from anywhere) |
| `m` | Log a finished entry, like Toggl's manual mode: opens on the last 30 minutes; `enter` on the description logs it |
| `space` | Pause/resume: toggles whatever the Timer panel already shows (running → pause, paused → resume); a no-op when the panel is idle |
| `enter` | Resume the selected entry (pre-filled dialog), or stop it if running |
| `e` | Edit — the selection in the list, or whatever the Timer panel shows (running, else paused) when it's focused |
| `d` | Delete with confirmation — same targeting as `e` |
| `j`/`k`, arrows | Navigate |
| `tab` | Switch panel focus |
| `r` | Refresh |
| `?` | Toggle help |
| `q` | Quit |

Full spec: [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) ·
Architecture: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
