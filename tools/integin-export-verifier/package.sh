#!/usr/bin/env sh
# package.sh - POSIX script to package integin-export-verifier regulatory release
set -eu

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"

VERSION="${VERSION:-0.1.0}"
OUTPUT_DIR="${OUTPUT_DIR:-dist}"
TARGET="${TARGET:-x86_64-unknown-linux-musl}"

echo "==> [1/4] Building Rust verifier release binary for target: $TARGET"
if [ "${SKIP_BUILD:-0}" = "1" ]; then
    echo "    Skipping build step (pre-built binary requested)"
    BINARY="target/$TARGET/release/integin-export-verifier"
elif [ -n "$TARGET" ]; then
    cargo build --release --target "$TARGET"
    BINARY="target/$TARGET/release/integin-export-verifier"
else
    cargo build --release
    BINARY="target/release/integin-export-verifier"
fi

PKG_NAME="integin-export-verifier-v${VERSION}-${TARGET}"
DIST_PATH="$OUTPUT_DIR/$PKG_NAME"

echo "==> [2/4] Assembling distribution bundle at $DIST_PATH"
rm -rf "$DIST_PATH"
mkdir -p "$DIST_PATH/testdata"

cp "$BINARY" "$DIST_PATH/"
cp "README.md" "$DIST_PATH/"
cp "AUDITOR_INSTRUCTIONS.md" "$DIST_PATH/"

if [ -f "testdata/manifest.json" ]; then cp "testdata/manifest.json" "$DIST_PATH/testdata/"; fi
if [ -f "testdata/public_key.hex" ]; then cp "testdata/public_key.hex" "$DIST_PATH/testdata/"; fi
if [ -f "testdata/export.zip" ]; then cp "testdata/export.zip" "$DIST_PATH/testdata/"; fi

echo "==> [3/4] Creating compressed tarball archive"
mkdir -p "$OUTPUT_DIR"
TAR_FILE="$OUTPUT_DIR/${PKG_NAME}.tar.gz"
tar -czf "$TAR_FILE" -C "$OUTPUT_DIR" "$PKG_NAME"

echo "==> [4/4] Emitting SHA-256 release checksum"
cd "$OUTPUT_DIR"
sha256sum "${PKG_NAME}.tar.gz" >> SHA256SUMS.txt

echo "=========================================================="
echo "✅ Packaged successfully: $TAR_FILE"
echo "=========================================================="
