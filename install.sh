#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_NAME="pixelysia"
INSTALL_PATH="/usr/local/bin/$BIN_NAME"

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

download_release_binary() {
	local repo platform url output
	repo="$1"
	platform="$2"
	output="$3"
	url="https://github.com/$repo/releases/latest/download/pixelysia-$platform"

	if ! command -v curl > /dev/null 2>&1; then
		return 1
	fi

	if curl -fsSL "$url" -o "$output"; then
		chmod +x "$output"
		return 0
	fi

	return 1
}

build_local_binary() {
	local output
	output="$1"

	if ! command -v go > /dev/null 2>&1; then
		echo "Go is required for local fallback build" >&2
		exit 1
	fi

	(
		cd "$SCRIPT_DIR"
		CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$output" ./cmd/pixelysia
	)
}

main() {
	local platform repo tmp_bin
	platform="$(detect_platform)"
	repo="$(resolve_repo)"
	tmp_bin="$(mktemp)"
	trap 'rm -f "$tmp_bin"' EXIT

	echo "Installing pixelysia CLI..."
	if ! download_release_binary "$repo" "$platform" "$tmp_bin"; then
		echo "Release download failed; building locally..."
		build_local_binary "$tmp_bin"
	fi

	sudo install -m 0755 "$tmp_bin" "$INSTALL_PATH"
	echo "Running system install..."
	export PIXELYSIA_SOURCE_DIR="$SCRIPT_DIR"
	sudo PIXELYSIA_SOURCE_DIR="$SCRIPT_DIR" pixelysia install
}

main "$@"