# Pixelysia

Pixelysia is a Linux-native SDDM theming runtime and CLI.

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/divijg19/Pixelysia/main/install.sh | bash
```

Or clone and run locally:

```bash
git clone https://github.com/divijg19/Pixelysia.git
cd Pixelysia
./install.sh
```

`install.sh` is release-first:

1. Detects Linux architecture (`amd64` or `arm64`)
2. Downloads prebuilt binary from GitHub Releases (`pixelysia-linux-amd64` / `pixelysia-linux-arm64`)
3. Falls back to local build only if release download fails
4. Installs to `/usr/local/bin/pixelysia`
5. Runs `sudo PIXELYSIA_SOURCE_DIR="$PWD" pixelysia install`

## Installed Paths

Pixelysia installs fonts and themes using the `pixelysia` CLI.

- Fonts:
	- Source: `fonts/*.ttf`
	- Destination: `/usr/share/fonts/pixelysia/`

- Themes:
	- Full mode: `/usr/share/sddm/themes/pixelysia/`
	- Split mode: `/usr/share/sddm/themes/<theme-name>/`

- SDDM config:
	- `/etc/sddm.conf.d/theme.conf`

## Requirements

- Linux
- SDDM
- `sudo`
- `curl` for release download (optional if building locally)
- Go toolchain for local build fallback in `install.sh`

## CLI Usage

```bash
# Install full runtime bundle
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install

# Install themes in split mode
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install --split

# Install one theme
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install --theme pixel-dusk-city

# Set active theme
sudo pixelysia set pixelysia

# List installed themes
pixelysia list

# Show current theme
pixelysia current

# Remove an installed theme
sudo pixelysia remove pixel-dusk-city

# Run system diagnostics
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

## CI and Releases

- CI workflow: `.github/workflows/ci.yml`
	- `go build ./...`
	- `go test ./... -v`
	- `go vet ./...` + `gofmt` check
- Release workflow: `.github/workflows/release.yml`
	- Trigger: tag push matching `v*`
	- Builds static binaries (`CGO_ENABLED=0`) for `linux-amd64` and `linux-arm64`
	- Publishes `pixelysia-linux-amd64` and `pixelysia-linux-arm64` as release assets

## Development Notes

- Runtime source discovery prioritizes `PIXELYSIA_SOURCE_DIR`
- For local development, run commands from repository root or set `PIXELYSIA_SOURCE_DIR`
- Font cache refresh uses `fc-cache -f`
