#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_NAME="pixelysia"
INSTALL_PATH="/usr/local/bin/$BIN_NAME"

# A valid Pixelysia source root contains the dispatcher entrypoint, the
# theme tree and the bundled fonts. The CLI needs all three to install.
is_valid_source_root() {
	local dir="$1"
	[ -n "$dir" ] || return 1
	[ -f "$dir/Main.qml" ] || return 1
	[ -d "$dir/themes" ] || return 1
	[ -d "$dir/fonts" ] || return 1
	return 0
}

detect_platform() {
	local os arch
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	arch="$(uname -m)"

	if [ "$os" != "linux" ]; then
		echo "Unsupported OS: $os (Linux only)" >&2
		exit 1
	fi

	case "$arch" in
		x86_64)
			arch="amd64"
			;;
		aarch64|arm64)
			arch="arm64"
			;;
		*)
			echo "Unsupported architecture: $arch" >&2
			exit 1
			;;
	esac

	echo "$os-$arch"
}

resolve_repo() {
	local remote
	if command -v git > /dev/null 2>&1; then
		remote="$(git -C "$SCRIPT_DIR" config --get remote.origin.url || true)"
		remote="${remote%.git}"
		remote="${remote#https://github.com/}"
		remote="${remote#http://github.com/}"
		remote="${remote#git@github.com:}"
		remote="${remote#ssh://git@github.com/}"
		if printf '%s' "$remote" | grep -Eq '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$'; then
			echo "$remote"
			return 0
		fi
	fi

	echo "divijg19/Pixelysia"
}

download_release_asset() {
	local repo asset url output
	repo="$1"
	asset="$2"
	output="$3"
	url="https://github.com/$repo/releases/latest/download/$asset"

	command -v curl > /dev/null 2>&1 || return 1
	curl -fsSL "$url" -o "$output"
}

download_release_binary() {
	local repo platform output
	repo="$1"
	platform="$2"
	output="$3"

	if ! download_release_asset "$repo" "pixelysia-$platform" "$output"; then
		return 1
	fi
	chmod +x "$output"
}

# Validate payload archive members before extraction. Only the four known
# payload roots are accepted; anything else (absolute paths, ".." traversal,
# unexpected extras) causes rejection regardless of local tar semantics.
validate_payload_archive() {
	local tarball list entry
	tarball="$1"
	list="$(mktemp)"

	if ! tar -tzf "$tarball" > "$list"; then
		rm -f "$list"
		return 1
	fi
	if ! [ -s "$list" ]; then
		rm -f "$list"
		return 1
	fi

	while IFS= read -r entry; do
		[ -n "$entry" ] || continue
		case "$entry" in
			Main.qml|metadata.desktop|themes/*|fonts/*) ;;
			*)
				rm -f "$list"
				return 1
				;;
		esac
	done < "$list"

	rm -f "$list"
	return 0
}

fetch_release_payload() {
	local repo dest tmp_tar
	repo="$1"
	dest="$2"
	tmp_tar="$(mktemp)"

	if ! download_release_asset "$repo" "pixelysia-payload.tar.gz" "$tmp_tar"; then
		rm -f "$tmp_tar"
		return 1
	fi

	if ! validate_payload_archive "$tmp_tar"; then
		echo "Release payload archive failed validation; refusing to extract." >&2
		rm -f "$tmp_tar"
		return 1
	fi

	mkdir -p "$dest"
	if ! tar -xzf "$tmp_tar" -C "$dest"; then
		rm -f "$tmp_tar"
		rm -rf "$dest"
		return 1
	fi
	rm -f "$tmp_tar"

	is_valid_source_root "$dest"
}

build_local_binary() {
	local output
	output="$1"

	if ! command -v go > /dev/null 2>&1; then
		echo "Go is required for local fallback build" >&2
		exit 1
	fi

	if ! is_valid_source_root "$SCRIPT_DIR"; then
		echo "Local fallback build requires a Pixelysia repository checkout" >&2
		exit 1
	fi

	(
		cd "$SCRIPT_DIR"
		CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$output" ./cmd/pixelysia
	)
}

main() {
	local platform repo tmp_bin source_dir payload_dir
	platform="$(detect_platform)"
	repo="$(resolve_repo)"
	tmp_bin="$(mktemp)"
	payload_dir=""
	trap 'rm -f "$tmp_bin"; [ -n "$payload_dir" ] && rm -rf "$payload_dir"' EXIT

	# Resolve an installation source before touching the system. A real
	# repository checkout wins (development installs); otherwise the release
	# payload archive is downloaded so that a clean machine can install
	# without Go and without keeping the repository around.
	if is_valid_source_root "$SCRIPT_DIR"; then
		source_dir="$SCRIPT_DIR"
	else
		payload_dir="$(mktemp -d)"
		echo "Fetching Pixelysia theme payload..."
		if ! fetch_release_payload "$repo" "$payload_dir"; then
			echo "Unable to obtain Pixelysia runtime assets." >&2
			echo "Run this script from a Pixelysia repository checkout, or check your network connection." >&2
			exit 1
		fi
		source_dir="$payload_dir"
	fi

	echo "Installing pixelysia CLI..."
	if ! download_release_binary "$repo" "$platform" "$tmp_bin"; then
		echo "Release download failed; building locally..."
		build_local_binary "$tmp_bin"
	fi

	sudo install -m 0755 "$tmp_bin" "$INSTALL_PATH"
	echo "Running system install..."
	sudo PIXELYSIA_SOURCE_DIR="$source_dir" pixelysia install
}

main "$@"
