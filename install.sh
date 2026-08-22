#!/usr/bin/env bash

# Strict mode applies when executed directly; when sourced by the offline
# test harness it must not leak errexit into the caller.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
	set -e
fi

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

# Resolve the single release identity that this installation will consume.
# Following the releases/latest redirect exactly once yields one immutable
# tag; every asset is then fetched from that tag so a single installation
# can never combine artifacts from different releases.
resolve_release_tag() {
	local repo url
	repo="$1"

	command -v curl > /dev/null 2>&1 || return 1
	url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")" || return 1
	url="${url##*/}"

	printf '%s' "$url" | grep -Eq '^[A-Za-z0-9._-]+$' || return 1
	printf '%s' "$url"
}

download_asset() {
	local url output
	url="$1"
	output="$2"

	command -v curl > /dev/null 2>&1 || return 1
	curl -fsSL "$url" -o "$output"
}

# Verify one downloaded file against its entry in a SHA256SUMS-style file.
verify_file_hash() {
	local sums file name expected
	sums="$1"
	file="$2"
	name="$3"

	expected="$(awk -v n="$name" '$2 == n { print $1 }' "$sums")"
	if [ -z "$expected" ]; then
		echo "Integrity metadata is missing an entry for $name" >&2
		return 1
	fi
	if ! printf '%s  %s\n' "$expected" "$file" | sha256sum -c --status -; then
		echo "$name failed SHA-256 verification; refusing to continue" >&2
		return 1
	fi
}

# Fail unless the checksum metadata covers exactly the three release assets.
verify_checksum_metadata() {
	local sums="$1"
	if ! command -v sha256sum > /dev/null 2>&1; then
		echo "sha256sum is required for release integrity verification" >&2
		return 1
	fi
	for name in pixelysia-linux-amd64 pixelysia-linux-arm64 pixelysia-payload.tar.gz; do
		grep -Eq '^[0-9a-f]{64}[[:space:]]+'"$name"'$' "$sums" || {
			echo "Integrity metadata is incomplete: missing $name" >&2
			return 1
		}
	done
	return 0
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

# Validate payload archive members before extraction. Only the four known
# payload roots are accepted; anything else (absolute paths, ".." traversal,
# unexpected extras) causes rejection regardless of local tar semantics.
validate_payload_archive() {
	local tarball entry
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
	local repo tag dest tmp_tar sums
	repo="$1"
	tag="$2"
	dest="$3"
	tmp_tar="$(mktemp)"
	sums="$(mktemp)"

	base="${PIXELYSIA_RELEASE_BASE_URL:-https://github.com/$repo/releases/download}/$tag"

	if ! download_asset "$base/pixelysia-checksums.txt" "$sums"; then
		echo "Unable to download release integrity metadata." >&2
		rm -f "$tmp_tar" "$sums"
		return 1
	fi
	verify_checksum_metadata "$sums" || {
		rm -f "$tmp_tar" "$sums"
		return 1
	}

	if ! download_asset "$base/pixelysia-payload.tar.gz" "$tmp_tar"; then
		echo "Unable to download the Pixelysia theme payload." >&2
		rm -f "$tmp_tar" "$sums"
		return 1
	fi
	verify_file_hash "$sums" "$tmp_tar" "pixelysia-payload.tar.gz" || {
		rm -f "$tmp_tar" "$sums"
		return 1
	}

	if ! validate_payload_archive "$tmp_tar"; then
		echo "Release payload archive failed validation; refusing to extract." >&2
		rm -f "$tmp_tar" "$sums"
		return 1
	fi

	mkdir -p "$dest"
	if ! tar -xzf "$tmp_tar" -C "$dest"; then
		rm -f "$tmp_tar" "$sums"
		rm -rf "$dest"
		return 1
	fi
	rm -f "$tmp_tar" "$sums"

	is_valid_source_root "$dest"
}

fetch_verified_binary() {
	local repo tag platform output sums
	repo="$1"
	tag="$2"
	platform="$3"
	output="$4"
	sums="$(mktemp)"

	base="${PIXELYSIA_RELEASE_BASE_URL:-https://github.com/$repo/releases/download}/$tag"

	download_asset "$base/pixelysia-checksums.txt" "$sums" || {
		rm -f "$sums"
		return 1
	}
	verify_checksum_metadata "$sums" || {
		rm -f "$sums"
		return 1
	}
	download_asset "$base/pixelysia-$platform" "$output" || {
		rm -f "$sums"
		return 1
	}
	chmod +x "$output"
	verify_file_hash "$sums" "$output" "pixelysia-$platform" || {
		rm -f "$output" "$sums"
		return 1
	}

	rm -f "$sums"
	return 0
}

main() {
	local platform repo tmp_bin source_dir payload_dir tag bin_src
	platform="$(detect_platform)" || exit 1
	repo="$(resolve_repo)"
	tmp_bin="$(mktemp)"
	payload_dir=""
	bin_src=""
	trap 'rm -f "${tmp_bin:-}"; [ -n "${payload_dir:-}" ] && rm -rf "${payload_dir:-}"' EXIT

	# Resolve an installation source before touching the system. A real
	# repository checkout wins (development installs); otherwise exactly one
	# release identity is resolved and its binary, payload and integrity
	# metadata are fetched together and verified before anything is installed.
	if is_valid_source_root "$SCRIPT_DIR"; then
		source_dir="$SCRIPT_DIR"

		if tag="$(resolve_release_tag "$repo")" &&
		   fetch_verified_binary "$repo" "$tag" "$platform" "$tmp_bin"; then
			bin_src="$tmp_bin"
		else
			echo "Release download or verification failed; building locally..."
			build_local_binary "$tmp_bin"
			bin_src="$tmp_bin"
		fi
	else
		if [ -n "${PIXELYSIA_RELEASE_TAG:-}" ]; then
			tag="${PIXELYSIA_RELEASE_TAG:-}"
		else
			tag="$(resolve_release_tag "$repo")" || {
				echo "Unable to determine the current Pixelysia release." >&2
				exit 1
			}
		fi

		payload_dir="$(mktemp -d)"
		echo "Fetching Pixelysia release $tag..."
		fetch_release_payload "$repo" "$tag" "$payload_dir" || {
			echo "Unable to obtain verified Pixelysia runtime assets; nothing was installed." >&2
			exit 1
		}
		source_dir="$payload_dir"

		if ! fetch_verified_binary "$repo" "$tag" "$platform" "$tmp_bin"; then
			echo "Unable to obtain a verified pixelysia binary; nothing was installed." >&2
			exit 1
		fi
		bin_src="$tmp_bin"
	fi

	echo "Installing pixelysia CLI..."
	sudo install -m 0755 "$bin_src" "$INSTALL_PATH"
	echo "Running system install..."
	sudo PIXELYSIA_SOURCE_DIR="$source_dir" pixelysia install
}

# Allow the offline installer test harness to source this script's
# functions without executing an actual installation.
if [ -z "${PIXELYSIA_SKIP_MAIN:-}" ]; then
	main "$@"
fi
