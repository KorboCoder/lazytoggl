# lazytoggl

A lazygit-style TUI for [Toggl Track](https://toggl.com/track/), built with Go and
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

**Status: Phase 1 — UI prototype.** The app runs entirely on mock data; no Toggl
API calls are made yet. See [docs/PLAN.md](docs/PLAN.md) for the roadmap.

## Run

```sh
go run .
```

## Keys

| Key | Action |
|---|---|
| `enter` | Start (`+ New entry` row), resume (past entry), or stop (running entry) |
| `x` | Stop the active timer from anywhere |
| `e` | Edit the selected entry |
| `d` | Delete the selected entry (with confirmation) |
| `j`/`k`, arrows | Navigate |
| `tab` | Switch panel focus |
| `r` | Refresh |
| `?` | Toggle help |
| `q` | Quit |

Full spec: [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) ·
Architecture: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
