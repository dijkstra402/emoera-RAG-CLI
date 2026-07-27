#!/usr/bin/env bash

set -euo pipefail

version="${1:?usage: build-macos-pkg.sh <version> <amd64|arm64> [output-dir]}"
arch="${2:?usage: build-macos-pkg.sh <version> <amd64|arm64> [output-dir]}"
output_dir="${3:-dist}"

case "$arch" in
  amd64|arm64) ;;
  *)
    echo "unsupported macOS architecture: $arch" >&2
    exit 2
    ;;
esac

if ! command -v pkgbuild >/dev/null 2>&1; then
  echo "pkgbuild is required; run this script on macOS" >&2
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/emoera-pkg.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT

payload="$work_dir/payload"
mkdir -p "$payload/usr/local/bin" "$output_dir"

normalized_version="${version#v}"
package_version="${normalized_version%%-*}"
binary="$payload/usr/local/bin/emoera"

(
  cd "$repo_root"
  CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/dijkstra402/emoera-RAG-CLI/cmd.version=$normalized_version" \
    -o "$binary" .
)
chmod 0755 "$binary"

package="$output_dir/emoera-cli_${normalized_version}_darwin_${arch}.pkg"
pkgbuild \
  --root "$payload" \
  --identifier "cn.emoera.cli" \
  --version "$package_version" \
  --install-location / \
  --ownership recommended \
  "$package"

echo "$package"
