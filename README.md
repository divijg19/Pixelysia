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
2. Resolves an installation source:
	- If run from a Pixelysia repository checkout, that checkout is used directly.
	- Otherwise the versioned `pixelysia-payload.tar.gz` release asset (QML dispatcher, themes, fonts) is downloaded and extracted to a temporary directory.
3. Downloads the prebuilt binary from GitHub Releases (`pixelysia-linux-amd64` / `pixelysia-linux-arm64`) and falls back to a local build only when running inside a checkout and the download fails
4. Installs to `/usr/local/bin/pixelysia`
5. Runs `sudo PIXELYSIA_SOURCE_DIR=<resolved source> pixelysia install`

If neither a checkout nor the payload archive can be obtained, the script fails before installing anything.

## Installed Paths

Pixelysia installs fonts and themes using the `pixelysia` CLI.

- Fonts:
	- Source: `fonts/*.ttf`
	- Destination: `/usr/share/fonts/pixelysia/`

- Themes:
	- Full mode: `/usr/share/sddm/themes/pixelysia/`
	- Split mode: `/usr/share/sddm/themes/<theme-id>/`

- SDDM config:
	- `/etc/sddm.conf.d/theme.conf`

## Themes

Themes are discovered structurally: any directory under `themes/` that contains a `Main.qml` entrypoint is an installable theme, at any depth. The canonical theme identifier is its path relative to `themes/`, so the terminal-style themes are addressed as `tui/Amber`, `tui/Emerald`, and so on.

Category directories without their own `Main.qml` (such as `themes/tui`) are containers only and are never installed as themes. A directory containing `metadata.desktop` but no `Main.qml` is rejected as malformed.

## Requirements

- Linux
- SDDM
- `sudo`
- `curl` and `tar` for release downloads (not required when installing from a local build)
- Go toolchain only for the local build fallback in `install.sh`

## CLI Usage

```bash
# Install full runtime bundle (dispatcher + all themes)
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install

# Install every theme individually
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install --split

# Install one theme by identifier
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install --theme pixel-dusk-city
sudo PIXELYSIA_SOURCE_DIR="$(pwd)" pixelysia install --theme tui/Amber

# Set active theme
sudo pixelysia set pixelysia
sudo pixelysia set tui/Amber

# List installed themes
pixelysia list

# Show current theme
pixelysia current

# Remove an installed theme
sudo pixelysia remove pixel-dusk-city

# Run system diagnostics
pixelysia doctor
```

Theme identifiers are path-safe: traversal segments (`..`), absolute paths, and separators other than `/` are rejected.

## Testing

Run all tests:

```bash
go test ./...
```

Run with coverage:

```bash
go test ./... -cover
```

All tests use temporary directories and do not require root. When the suite runs from a repository checkout, discovery and validation are additionally exercised against the real theme tree.

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
	- Publishes `pixelysia-linux-amd64`, `pixelysia-linux-arm64`, and `pixelysia-payload.tar.gz` as release assets

## Development Notes

- Runtime source discovery prioritizes `PIXELYSIA_SOURCE_DIR`; installs require a directory containing `Main.qml`, `themes/`, and `fonts/`
- For local development, run commands from repository root or set `PIXELYSIA_SOURCE_DIR`
- Font cache refresh uses `fc-cache -f`
- Media assets (`*.mp4`, `*.png`) are tracked with Git LFS

## License and Attribution

Pixelysia is licensed under the GNU General Public License v3.0 (see [LICENSE](LICENSE)).

The QML themes are derived from [Darkkal44/qylock](https://github.com/Darkkal44/qylock), with modifications for bundled system fonts and Pixelysia packaging. The TUI themes are original Pixelysia work. All upstream credit belongs to darkkal.
