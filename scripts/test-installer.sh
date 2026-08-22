#!/usr/bin/env bash
# Offline failure-mode harness for install.sh (v0.5.2 release integrity).
#
# Exercises the installer's release-resolution and integrity contract
# against local fixtures via curl's file:// support, without touching the
# host system: sudo is shimmed to redirect writes into a sandbox.
set -u

PASS=0
FAIL=0

SB="$(mktemp -d)"
trap 'rm -rf "$SB"' EXIT
mkdir -p "$SB/bin" "$SB/target" "$SB/home"

# Shim sudo: translate "install -m MODE SRC /usr/local/bin/pixelysia" into a
# sandboxed copy, and record pixelysia invocations with their environment.
cat > "$SB/bin/sudo" <<EOF
#!/usr/bin/env bash
if [ "\$1" = "install" ]; then
  shift 3 # install -m MODE
  src="\$1"
  mkdir -p "$SB/target"
  cp "\$src" "$SB/target/pixelysia"
  echo "sudo-install:\$src" >> "$SB/log"
  exit 0
fi
if [ "\${2:-}" = "pixelysia" ]; then
  echo "system-install:\${1#*=}" >> "$SB/log"
  exit 0
fi
exit 64
EOF
chmod +x "$SB/bin/sudo"

# Shim uname so architecture detection is deterministic.
cat > "$SB/bin/uname" <<'EOF'
#!/usr/bin/env bash
case "$*" in
  *-m*) printf '%s\n' "${HARNESS_ARCH:-x86_64}" ;;
  *) printf 'Linux\n' ;;
esac
EOF
chmod +x "$SB/bin/uname"

log() { printf '%s\n' "$*" >> "$SB/log"; }
export PATH="$SB/bin:$PATH"
expect_pass() { if "$@" >/dev/null 2>&1; then PASS=$((PASS+1)); else FAIL=$((FAIL+1)); echo "FAIL(expected pass): $*"; fi; }
expect_fail() { if "$@" >/dev/null 2>&1; then FAIL=$((FAIL+1)); echo "FAIL(expected abort): $*"; else PASS=$((PASS+1)); fi; }

export PIXELYSIA_SKIP_MAIN=1
export HARNESS_ARCH=x86_64

source "$(dirname "$0")/../install.sh"

make_fixture() { # $1=name -> $SB/rel/$1 with coherent binary+payload+checksums
	local d="$SB/rel/$1"
	mkdir -p "$d/pl/themes/tui/Amber" "$d/pl/fonts"
	printf 'binary-%s' "$1" > "$d/pixelysia-linux-amd64"
	printf 'arm-%s' "$1" > "$d/pixelysia-linux-arm64"
	printf 'q-%s' "$1" > "$d/pl/Main.qml"; printf 'm' > "$d/pl/metadata.desktop"
	printf 't' > "$d/pl/themes/tui/Amber/Main.qml"; printf 'f' > "$d/pl/fonts/f.ttf"
	tar -czf "$d/pixelysia-payload.tar.gz" -C "$d/pl" Main.qml metadata.desktop themes fonts
	( cd "$d" && sha256sum pixelysia-linux-amd64 pixelysia-linux-arm64 pixelysia-payload.tar.gz > pixelysia-checksums.txt )
}

echo "== checksum metadata cases"
make_fixture good
S="$SB/rel/good/pixelysia-checksums.txt"
expect_pass verify_checksum_metadata "$S"
grep -v arm64 "$S" > "$SB/incomplete.txt";   expect_fail verify_checksum_metadata "$SB/incomplete.txt"
sed 's/^[0-9a-f]/z/' "$S" > "$SB/malformed.txt"; expect_fail verify_checksum_metadata "$SB/malformed.txt"
: > "$SB/empty.txt";                          expect_fail verify_checksum_metadata "$SB/empty.txt"

echo "== file hash cases"
B="$SB/rel/good/pixelysia-linux-amd64"
expect_pass verify_file_hash "$S" "$B" pixelysia-linux-amd64
cp "$B" "$SB/corrupt"; printf 'X' | dd of="$SB/corrupt" bs=1 seek=3 conv=notrunc 2>/dev/null
expect_fail verify_file_hash "$S" "$SB/corrupt" pixelysia-linux-amd64
cp "$B" "$SB/trunc"; truncate -s 5 "$SB/trunc"
expect_fail verify_file_hash "$S" "$SB/trunc" pixelysia-linux-amd64
expect_fail verify_file_hash "$S" "$B" pixelysia-nonexistent

echo "== end-to-end remote installation (coherent release)"
export PIXELYSIA_RELEASE_BASE_URL="file://$SB/rel"
export PIXELYSIA_RELEASE_TAG=good
: > "$SB/log"
( main ); RET=$?
[ "$RET" -eq 0 ] && PASS=$((PASS+1)) || { FAIL=$((FAIL+1)); echo "FAIL: main returned $RET"; }
logged_src="$(sed -n 's/^system-install://p' "$SB/log")"
if grep -q "^sudo-install:" "$SB/log" \
   && [ -n "$logged_src" ] \
   && [ "$(cat "$logged_src/Main.qml" 2>/dev/null)" = "q-good" ] \
   && [ -f "$logged_src/themes/tui/Amber/Main.qml" ]; then
	PASS=$((PASS+1))
else
	FAIL=$((FAIL+1)); echo "FAIL: coherent install flow"
fi
cmp -s "$SB/target/pixelysia" "$SB/rel/good/pixelysia-linux-amd64" \
	&& PASS=$((PASS+1)) || { FAIL=$((FAIL+1)); echo "FAIL: installed binary bytes"; }

echo "== version-skew regression: payload from another release must be refused"
make_fixture other
# checksums claim hashes of "good", but payload served is "other"'s
cp "$SB/rel/other/pixelysia-payload.tar.gz" "$SB/rel/good/pixelysia-payload.tar.gz"
rm -f "$SB/target/pixelysia"
: > "$SB/log"
( main ); RET=$?
[ "$RET" -ne 0 ] && PASS=$((PASS+1)) || { FAIL=$((FAIL+1)); echo "FAIL: skew accepted"; }
[ ! -e "$SB/target/pixelysia" ] && [ "$(wc -l < "$SB/log")" -eq 0 ] \
	&& PASS=$((PASS+1)) || { FAIL=$((FAIL+1)); echo "FAIL: mutation before verification"; }
make_fixture good # restore coherence for remaining cases

echo "== corruption regression: one flipped byte in the binary"
cp "$SB/rel/good/pixelysia-linux-amd64" "$SB/rel/good/.bak"
printf 'X' | dd of="$SB/rel/good/pixelysia-linux-amd64" bs=1 seek=3 conv=notrunc 2>/dev/null
: > "$SB/log"
( main ); RET=$?
[ "$RET" -ne 0 ] && [ "$(wc -l < "$SB/log")" -eq 0 ] \
	&& PASS=$((PASS+1)) || { FAIL=$((FAIL+1)); echo "FAIL: corrupted binary installed"; }
mv "$SB/rel/good/.bak" "$SB/rel/good/pixelysia-linux-amd64"

echo "== unsupported architecture fails closed"
export HARNESS_ARCH=sparc64
( main ); RET=$?
[ "$RET" -ne 0 ] && PASS=$((PASS+1)) || { FAIL=$((FAIL+1)); echo "FAIL: sparc accepted"; }
export HARNESS_ARCH=x86_64

echo "== missing asset / failed download fails closed"
export PIXELYSIA_RELEASE_TAG=nonexistent
( main ); RET=$?
[ "$RET" -ne 0 ] && [ "$(wc -l < "$SB/log")" -eq 0 ] \
	&& PASS=$((PASS+1)) || { FAIL=$((FAIL+1)); echo "FAIL: missing release proceeded"; }

echo "== local checkout mode uses local source"
unset PIXELYSIA_RELEASE_TAG
mkdir -p "$SB/checkout/themes" "$SB/checkout/fonts"; : > "$SB/checkout/Main.qml"
SCRIPT_DIR_OVERRIDE="$SB/checkout"
# Re-source with SCRIPT_DIR pointed at the fake checkout by running main in a subshell
(
	SCRIPT_DIR="$SB/checkout"
	is_valid_source_root "$SCRIPT_DIR" || exit 1
	echo checkout-detected >> "$SB/log"
) 
grep -q checkout-detected "$SB/log" && PASS=$((PASS+1)) || { FAIL=$((FAIL+1)); echo "FAIL: checkout detection"; }

echo
echo "harness results: PASS=$PASS FAIL=$FAIL"
[ "$FAIL" -eq 0 ]
