#!/usr/bin/env sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
out="${INTEGIN_BUILD_OUTPUT:-$root/dist}"
mkdir -p "$out/windows-amd64" "$out/linux-amd64"

build_target() {
  target_os=$1
  target_arch=$2
  target_dir=$3
  suffix=""
  if [ "$target_os" = "windows" ]; then
    suffix=".exe"
  fi
  GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -o "$target_dir/integin-server$suffix" ./cmd/integin-server
  GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -o "$target_dir/integin-pilot-policy-candidate$suffix" ./cmd/integin-pilot-policy-candidate
}

cd "$root"
build_target windows amd64 "$out/windows-amd64"
build_target linux amd64 "$out/linux-amd64"
printf '%s\n' "Target builds written to $out"
