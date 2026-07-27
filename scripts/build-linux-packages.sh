#!/usr/bin/env bash

set -euo pipefail

version="${1:?usage: build-linux-packages.sh <version> <amd64|arm64> [output-dir]}"
arch="${2:?usage: build-linux-packages.sh <version> <amd64|arm64> [output-dir]}"
output_dir="${3:-dist}"

case "$arch" in
  amd64|arm64)
    ;;
  *)
    echo "unsupported Linux architecture: $arch" >&2
    exit 2
    ;;
esac

for command in go nfpm; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "$command is required" >&2
    exit 2
  fi
done

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/emoera-linux-packages.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT

normalized_version="${version#v}"
mkdir -p "$output_dir"
output_dir="$(cd "$output_dir" && pwd)"

binary="$work_dir/emoera"
(
  cd "$repo_root"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/dijkstra402/emoera-RAG-CLI/cmd.version=$normalized_version" \
    -o "$binary" .
)
chmod 0755 "$binary"

package_config="$work_dir/nfpm.yaml"
cat >"$package_config" <<EOF
name: emoera-cli
arch: $arch
platform: linux
version: $normalized_version
release: "1"
section: utils
priority: optional
maintainer: Emoera <dijkstra402@163.com>
description: Secure document search, streaming RAG chat and automation-friendly output.
vendor: Emoera
homepage: https://github.com/dijkstra402/emoera-RAG-CLI
license: Apache-2.0
contents:
  - src: ./emoera
    dst: /usr/local/bin/emoera
    file_info:
      mode: 0755
EOF

(
  cd "$work_dir"
  nfpm package --config nfpm.yaml --packager deb \
    --target "$output_dir/emoera-cli_${normalized_version}_linux_${arch}.deb"
  nfpm package --config nfpm.yaml --packager rpm \
    --target "$output_dir/emoera-cli_${normalized_version}_linux_${arch}.rpm"
)

printf '%s\n' \
  "$output_dir/emoera-cli_${normalized_version}_linux_${arch}.deb" \
  "$output_dir/emoera-cli_${normalized_version}_linux_${arch}.rpm"
