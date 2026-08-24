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
	- Otherwise exactly one release is resolved (the `releases/latest` redirect is followed once to its immutable tag), and the binary, theme payload and integrity metadata for that same release are downloaded together.
3. Verifies SHA-256 checksums (`pixelysia-checksums.txt`) against the exact bytes downloaded; any mismatch, truncation or missing metadata aborts before anything is installed
4. Falls back to a local build only when running inside a checkout and the verified binary download fails
5. Installs to `/usr/local/bin/pixelysia`
6. Runs `sudo PIXELYSIA_SOURCE_DIR=<resolved source> pixelysia install`

A single installation therefore always consumes one coherent release, and artifacts that fail integrity verification are never installed. Checksums establish integrity relative to the release they were generated from; they do not provide cryptographic authenticity/signing.

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

# Validate a theme source tree (read-only)
pixelysia validate
pixelysia validate --source /path/to/Pixelysia

# Show the version of the running binary
pixelysia version
```
Release binaries report their Git tag (injected at build time); locally built development binaries report `dev`.

`validate` checks a Pixelysia source tree without touching the system: structural theme validity, `metadata.desktop` presence (advisory), declared `background=` assets resolve to real files, declared `font=` families are represented in each theme's QML usage, and every theme reference in the root dispatcher resolves. It exits non-zero on fatal findings and runs in CI against this repository.

Font validation is textual/semantic within the theme source; it does not parse TTF name tables or prove that a runtime font engine will resolve the family.

Theme identifiers are path-safe: traversal segments (`..`), absolute paths, and separators other than `/` are rejected.

## Uninstall

Remove an individual theme or the full bundle with `remove`; `doctor` and `list` always reflect the current state:

```bash
sudo pixelysia remove forest        # remove one split theme
sudo pixelysia remove tui/Amber     # remove a nested theme
sudo pixelysia remove pixelysia     # remove the full bundle
```

Removing the theme that SDDM currently selects prints a warning; SDDM then falls back to its embedded greeter until another theme is set. To fully uninstall Pixelysia, remove every listed theme, then:

```bash
sudo rm -rf /usr/share/fonts/pixelysia
sudo rm /etc/sddm.conf.d/theme.conf
sudo rm /usr/local/bin/pixelysia
```

## Troubleshooting

Run `pixelysia doctor` first — it verifies fonts, fontconfig discovery, installed themes, configuration presence and whether the configured current theme actually resolves. Common situations:

- **Theme does not appear / SDDM loads its default greeter** — the configured `Current=` names a theme that is not installed (for example after `remove`). Reinstall it (`pixelysia install --theme <id>`) or select another with `pixelysia set <id>`.
- **`doctor` reports "current theme ... is not installed"** — same cause; the configuration is dangling until you `set` an installed theme.
- **Fonts do not render** — check `doctor`: it reports both installed font files and whether fontconfig discovers them. If files exist but are not discovered, run `fc-cache -f` once.
- **Release verification fails during install.sh** — the installer refuses unverified downloads by design. Re-run; if it persists, compare your download against the release's `pixelysia-checksums.txt`.

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
	- `pixelysia validate --source .` against the real theme tree
	- `go vet ./...` + `gofmt` check
- Release workflow: `.github/workflows/release.yml`
	- Trigger: tag push matching `v*`
	- Builds static binaries (`CGO_ENABLED=0`) for `linux-amd64` and `linux-arm64`; the runtime payload is built once in the release job from a single LFS checkout (no large Actions transport artifacts)
	- Publishes `pixelysia-linux-amd64`, `pixelysia-linux-arm64`, `pixelysia-payload.tar.gz`, and `pixelysia-checksums.txt` as release assets

## Development Notes

- Runtime source discovery prioritizes `PIXELYSIA_SOURCE_DIR`; installs require a directory containing `Main.qml`, `themes/`, and `fonts/`
- For local development, run commands from repository root or set `PIXELYSIA_SOURCE_DIR`
- Font cache refresh uses `fc-cache -f`
- Media assets (`*.mp4`, `*.png`) are tracked with Git LFS

## License and Attribution

Pixelysia is licensed under the GNU General Public License v3.0 (see [LICENSE](LICENSE)).

Bundled fonts: [Figtree](https://fonts.google.com/specimen/Figtree), [Pixelify Sans](https://fonts.google.com/specimen/Pixelify+Sans), [Orbitron](https://fonts.google.com/specimen/Orbitron) and [Share Tech Mono](https://fonts.google.com/specimen/Share+Tech+Mono) are distributed under the SIL Open Font License. The bundled Ninja Naruto fan font (`njnaruto.ttf`) originates from the upstream qylock distribution; its exact license could not be established from repository evidence — noted here rather than invented.

The QML themes are derived from [Darkkal44/qylock](https://github.com/Darkkal44/qylock), with modifications for bundled system fonts and Pixelysia packaging. The TUI themes are original Pixelysia work. All upstream credit belongs to darkkal.
