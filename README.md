# victronctl

`victronctl` is a Go CLI and terminal dashboard for monitoring Victron Energy systems through the VRM API.

## Features

- 🔐 Authenticate with a VRM access token
- 📍 List available VRM sites and installations
- 📊 Show a site overview with summary data and devices
- ⚡ Monitor live battery, solar, inverter, load, and grid status
- 🖥️ Run a Bubble Tea live dashboard in the terminal
- 🧾 Export machine-readable output with `--json`

Implemented commands:
- `auth`
- `sites`
- `overview`
- `live`

The CLI targets `https://vrmapi.victronenergy.com/v2` and reads the API token from `VICTRON_VRM_TOKEN`.

## Example Commands

```bash
VICTRON_VRM_TOKEN=your-vrm-token go run . auth
VICTRON_VRM_TOKEN=your-vrm-token go run . auth --json
VICTRON_VRM_TOKEN=your-vrm-token go run . sites
VICTRON_VRM_TOKEN=your-vrm-token go run . sites --json
VICTRON_VRM_TOKEN=your-vrm-token go run . overview --site 948332
VICTRON_VRM_TOKEN=your-vrm-token go run . overview --site 948332 --json
VICTRON_VRM_TOKEN=your-vrm-token go run . overview
VICTRON_VRM_TOKEN=your-vrm-token go run . live
VICTRON_VRM_TOKEN=your-vrm-token go run . live --site 948332 --once
VICTRON_VRM_TOKEN=your-vrm-token go run . live --site 948332 --once --json
VICTRON_VRM_TOKEN=your-vrm-token go run . live --once
VICTRON_VRM_TOKEN=your-vrm-token go run . live --json --interval 10s
```

TUI controls:
- `q` or `ctrl+c` to quit
- `tab`, `h`, `j`, `k`, `l`, or arrow keys to switch panels
- `c` to toggle compact and expanded layouts

## Releases

Releases are created manually from GitHub Actions using the `Release` workflow.

Release process:
- Merge the changes you want into `main`
- Open the `Release` workflow in GitHub Actions
- Run it manually and choose a version bump strategy

Available bump strategies:
- `auto`: derive the next version from conventional commits since the last tag
- `patch`: increment the patch version
- `minor`: increment the minor version
- `major`: increment the major version

In `auto` mode:
- commits with `BREAKING CHANGE` or `type!:` trigger a major release
- commits with `feat:` trigger a minor release
- all other changes produce a patch release

The workflow will:
- run `go test ./...`
- run `go build ./...`
- create and push the next `v*` tag from `main`
- run GoReleaser
- publish release artifacts to GitHub Releases
