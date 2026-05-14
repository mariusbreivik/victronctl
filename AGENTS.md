## Current State

- The repository now has a minimal Go CLI scaffold: `main.go`, `cmd/root.go`, `cmd/auth.go`, `cmd/sites.go`, `cmd/overview.go`, `cmd/live.go`, `cmd/vrm.go`, `go.mod`, and `go.sum`.
- Implemented commands are currently `auth`, `sites`, `overview`, and `live`. Most features described in `README.md` are still planned, not present in the codebase.

## Working Rules

- The module path is `github.com/mariusbreivik/victronctl`, derived from the git remote.
- The CLI entrypoint is `main.go`, which delegates to `cmd.Execute()`. Add commands under `cmd/` and register them from `init()`.
- The current auth scaffold targets `https://vrmapi.victronenergy.com/v2`, reads tokens only from `VICTRON_VRM_TOKEN`, and validates tokens with `GET /users/me` using the `x-authorization: Token <token>` header format.
- `overview` and `live` can infer the site ID when the authenticated user has exactly one site; otherwise they require `--site`.
- `overview` now reuses the normalized live snapshot model to render a site summary before listing devices.
- The `live` command derives battery, solar charger, and VE.Bus instances from `GET /installations/{idSite}/system-overview`, then queries instance-scoped widget endpoints. The default, non-instance widget calls did not return usable current values on this repo's live site.
- On this repo's live site, `Status` input-voltage fields are a better signal for "grid present" than input power alone. When input voltage is absent, `live` reports grid import/export as `n/a` instead of `0 W`.
- In continuous human-readable mode, `live` now runs a Bubble Tea TUI. `--once` and `--json` still use the non-TUI snapshot path.
- The current `live` TUI is panel-based. In the wide layout, the top row is `System`, `Battery`, `Solar`, and `Inverter / Load`, with `Grid / Warnings` below. Prefer evolving that layout rather than reverting to a flat list.
- The `live` TUI now has a centered header with health badges, trend sparklines plus direction labels, compact/expanded mode, and a selected-panel detail view. Prefer preserving those interactions when refining the dashboard.
- The `live` TUI auto-compacts on smaller viewports. Avoid assuming there is room for the expanded detail pane on a 13-inch screen.
- Human-readable timestamps should use the local machine timezone, not UTC.
- There are no repo-local instruction files besides this one and no `opencode.json` in the root.

## Validation

- There is no CI, lint, or test configuration yet. Use focused Go verification commands directly.
- Verified commands for the current scaffold: `go run . auth --help`, `source .envrc && go run . auth`, `source .envrc && go run . auth --json`, `source .envrc && go run . sites`, `source .envrc && go run . sites --json`, `source .envrc && go run . overview`, `source .envrc && go run . overview --site 948332`, `source .envrc && go run . overview --site 948332 --json`, `source .envrc && go run . live --once`, `source .envrc && go run . live --site 948332 --once`, `source .envrc && go run . live --site 948332 --once --json`, and `go build ./...`.
