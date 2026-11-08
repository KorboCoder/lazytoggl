# lazytoggl

A lazygit-style TUI for Toggl Track, in Go with Bubble Tea.

## Workflow: OpenSpec + Superpowers

OpenSpec owns *what and why*. Superpowers owns *how the work gets executed*.

- **Feature or behavior change → OpenSpec.** Start with `/opsx:explore` to think it
  through, or `/opsx:propose` to write the artifacts. The change folder
  `openspec/changes/<name>/` is the single home for its proposal, design, specs, and
  tasks. Output from superpowers:brainstorming feeds `/opsx:propose`, and `tasks.md` is
  the implementation plan — it takes the place of superpowers:writing-plans and
  `docs/superpowers/`.
- **Checkpoint.** After proposing, stop and wait for the user to approve the change
  before `/opsx:apply`.
- **Executing → Superpowers.** During `/opsx:apply`, work each task with
  superpowers:test-driven-development, use superpowers:systematic-debugging on any
  failure, and pass superpowers:verification-before-completion before checking a task
  off. When all tasks are done, run superpowers:requesting-code-review, then
  `/opsx:archive`.
- **Bug fix or tweak that leaves specified behavior unchanged → Superpowers only**, no
  OpenSpec change.

## Verifying

- `go test -race ./...` is the gate for every task.
- Check UI behavior with `go run . -mock`. Plain `go run .` hits the live Toggl API on
  the user's Free plan (30 requests/hour, 2 spent per launch), so run it only when the
  user asks.

## Tests

- UI flows: add cases to `internal/ui/driver_test.go` (synchronous driver, mock
  fixtures, fixed clock); use `teatest` when the real event loop matters.
- API client: `httptest` fixtures in `internal/toggl/client_test.go` that pin the exact
  request and response shapes.
- A new `toggl.Service` method gets a `mock.go` implementation too, so `-mock` keeps
  working.

## Docs

- `docs/ARCHITECTURE.md` — read before touching `internal/toggl`, adding an API call,
  or changing config/secrets. Its rate-limit budget and API quirks are hard constraints.
- `docs/REQUIREMENTS.md` — feature spec and keymap; check it before adding or changing
  a key.
- `docs/PLAN.md` — phases and decisions log.

When a change alters behavior these docs describe, update them in the same change.
