# victronctl

`victronctl` is a lightweight Go CLI for working with the [Victron VRM API](https://vrmapi.victronenergy.com/v2).

It gives you a fast way to validate access, list sites, inspect an installation, and monitor live system data from the terminal. It also supports JSON output for scripts and automation.

## Why victronctl?

- ⚡ Quick access to VRM data without opening the browser
- 🖥️ Terminal-friendly live monitoring with an interactive dashboard
- 🤖 JSON output for automation, tooling, and shell pipelines

## Features

- 🔐 `auth` validates your `VICTRON_VRM_TOKEN` against the VRM API
- 📍 `sites` lists the VRM installations available to the authenticated user
- 📊 `overview` shows a site summary plus discovered devices
- ⚡ `live` polls key battery, solar, inverter, load, and grid metrics
- 🖥️ Interactive Bubble Tea dashboard for continuous live monitoring
- 🧾 `--json` output for machine-readable integrations
- 🏷️ `version` prints build and release metadata

## Victron VRM API

This project talks directly to the Victron VRM API:

- 🔗 API base URL: [`https://vrmapi.victronenergy.com/v2`](https://vrmapi.victronenergy.com/v2)

## Getting Started

`victronctl` reads your API token from `VICTRON_VRM_TOKEN`.

```bash
export VICTRON_VRM_TOKEN="your-vrm-token"
```

Then validate access:

```bash
go run . auth
```

## Develop Locally

Get a local development loop running:

```bash
git clone <your-fork-or-repo-url>
cd victronctl
export VICTRON_VRM_TOKEN="your-vrm-token"
go run . auth
```

Useful local workflows:

- 🛠️ Run the CLI directly with `go run . <command>` while iterating
- 🔄 Build a local binary with `go build -o victronctl .`
- 🧪 Verify changes with `go test ./...`
- ✅ Check release config with `go run github.com/goreleaser/goreleaser/v2@v2.15.4 check`

Examples during development:

```bash
go run . sites
go run . overview --site <site-id>
go run . live --once
go run . live
```

## Usage

Common commands:

```bash
go run . auth
go run . sites
go run . overview --site <site-id>
go run . live
go run . live --once
go run . live --once --json
go run . version
```

Notes:

- `overview` and `live` can infer the site automatically when your account has exactly one accessible site.
- `live` runs an interactive terminal UI by default.
- Use `--once` for a single snapshot and `--json` for machine-readable output.

## Build Locally

Build the project with Go:

```bash
go build ./...
```

To produce a local binary in the project root:

```bash
go build -o victronctl .
```

## Test Locally

Run the full test suite:

```bash
go test ./...
```

Run a focused test in the `cmd` package:

```bash
go test ./cmd -run TestName
```

Validate the GoReleaser configuration:

```bash
go run github.com/goreleaser/goreleaser/v2@v2.15.4 check
```

## Release Workflow

- 🚀 Releases are created through the GitHub Actions `Release` workflow in `.github/workflows/release.yml`

## License

Released under the [MIT License](./LICENSE).
