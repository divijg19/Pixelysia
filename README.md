# Pixelysia

Pixelysia is a Linux-native SDDM theming runtime and CLI.

## What It Installs

Pixelysia installs fonts and themes using the `pixelysia` CLI.

Fonts:

- Source: `fonts/*.ttf`
- Destination: `/usr/share/fonts/pixelysia/`

Themes:

- Full mode: `/usr/share/sddm/themes/pixelysia/`
- Split mode: `/usr/share/sddm/themes/<theme-name>/`

SDDM config:

- `/etc/sddm.conf.d/theme.conf`

## Requirements

- Linux
- SDDM
- `sudo`
- `curl` for release download (optional if building locally)
- Go toolchain for local build fallback in `install.sh`

## Installation

Clone and install:

```bash
git clone https://github.com/divijg19/Pixelysia.git
cd Pixelysia
./install.sh
```

`install.sh` performs bootstrap only:

1. Detects Linux architecture (`amd64` or `arm64`)
2. Downloads latest `pixelysia` release binary from GitHub
3. Falls back to local build: `CGO_ENABLED=0 go build -o pixelysia ./cmd/pixelysia`
4. Installs binary to `/usr/local/bin/pixelysia`
5. Executes: `sudo PIXELYSIA_SOURCE_DIR="$PWD" pixelysia install`

## CLI Usage

Install full runtime bundle:

```bash
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install
```

Install themes in split mode:

```bash
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install --split
```

Install one theme:

```bash
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install --theme pixel-dusk-city
```

Set active theme:

```bash
sudo pixelysia set pixelysia
```

List installed themes:

```bash
pixelysia list
```

Show current theme:

```bash
pixelysia current
```

Remove an installed theme:

```bash
sudo pixelysia remove pixel-dusk-city
```

Run system diagnostics:

```bash
pixelysia doctor
```

## Testing

Run all tests:

```bash
go test ./...
```

Run with coverage:

```bash
go test ./... -cover
```

All tests use temporary directories and do not require root.

### Runtime-style CLI tests

Runtime execution tests run `pixelysia` through subprocess calls (`go run ./cmd/pixelysia ...`) in isolated temporary environments.
These tests do not write to real system paths.

## CI

GitHub Actions workflow: `.github/workflows/ci.yml`

The CI pipeline runs on `ubuntu-latest` for:

1. Build: `go build ./...`
2. Tests: `go test ./... -v`
3. Lint: `go vet ./...` and `gofmt` check

## Development Notes

- Runtime source discovery prioritizes `PIXELYSIA_SOURCE_DIR`
- For local development, run commands from repository root or set `PIXELYSIA_SOURCE_DIR`
- Font cache refresh uses `fc-cache -f`
