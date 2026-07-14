# lazytoggl

A lazygit-style TUI for [Toggl Track](https://toggl.com/track/), built with Go and
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

**Status: Phase 1 — UI prototype.** The app runs entirely on mock data; no Toggl
API calls are made yet. See [docs/PLAN.md](docs/PLAN.md) for the roadmap.

## Run

```sh
go run .
```

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
