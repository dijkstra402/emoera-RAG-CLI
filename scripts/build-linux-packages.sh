#!/usr/bin/env bash

set -euo pipefail

version="${1:?usage: build-linux-packages.sh <version> <amd64|arm64> [output-dir]}"
arch="${2:?usage: build-linux-packages.sh <version> <amd64|arm64> [output-dir]}"
output_dir="${3:-dist}"

case "$arch" in
  amd64)
    deb_arch="amd64"
    rpm_arch="x86_64"
    ;;
  arm64)
    deb_arch="arm64"
    rpm_arch="aarch64"
    ;;
  *)
    echo "unsupported Linux architecture: $arch" >&2
    exit 2
    ;;
esac

for command in go dpkg-deb rpmbuild; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "$command is required" >&2
    exit 2
  fi
done

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/emoera-linux-packages.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT

normalized_version="${version#v}"
rpm_version="${normalized_version//-/_}"
mkdir -p "$output_dir"

binary="$work_dir/emoera"
(
  cd "$repo_root"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/dijkstra402/emoera-RAG-CLI/cmd.version=$normalized_version" \
    -o "$binary" .
)
chmod 0755 "$binary"

deb_root="$work_dir/deb"
install -D -m 0755 "$binary" "$deb_root/usr/local/bin/emoera"
mkdir -p "$deb_root/DEBIAN"
cat >"$deb_root/DEBIAN/control" <<EOF
Package: emoera-cli
Version: $normalized_version
Section: utils
Priority: optional
Architecture: $deb_arch
Maintainer: Emoera <dijkstra402@163.com>
Homepage: https://github.com/dijkstra402/emoera-RAG-CLI
Description: Agent CLI for the Emoera RAG knowledge platform
 Secure document search, streaming RAG chat and automation-friendly output.
EOF
dpkg-deb --root-owner-group --build "$deb_root" \
  "$output_dir/emoera-cli_${normalized_version}_linux_${arch}.deb"

rpm_root="$work_dir/rpmbuild"
mkdir -p "$rpm_root"/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS}
install -m 0755 "$binary" "$rpm_root/SOURCES/emoera"
cat >"$rpm_root/SPECS/emoera-cli.spec" <<EOF
Name:           emoera-cli
Version:        $rpm_version
Release:        1%{?dist}
Summary:        Agent CLI for the Emoera RAG knowledge platform
License:        Apache-2.0
URL:            https://github.com/dijkstra402/emoera-RAG-CLI
BuildArch:      $rpm_arch

%description
Secure document search, streaming RAG chat and automation-friendly output.

%prep

%build

%install
install -D -m 0755 %{_sourcedir}/emoera %{buildroot}%{_bindir}/emoera

%files
%{_bindir}/emoera

%changelog
* Sun Jul 27 2026 Emoera <dijkstra402@163.com> - $rpm_version-1
- Automated release package
EOF
rpmbuild --define "_topdir $rpm_root" --target "$rpm_arch" \
  -bb "$rpm_root/SPECS/emoera-cli.spec"
rpm_file="$(find "$rpm_root/RPMS" -type f -name '*.rpm' -print -quit)"
test -n "$rpm_file"
cp "$rpm_file" "$output_dir/emoera-cli_${normalized_version}_linux_${arch}.rpm"

printf '%s\n' \
  "$output_dir/emoera-cli_${normalized_version}_linux_${arch}.deb" \
  "$output_dir/emoera-cli_${normalized_version}_linux_${arch}.rpm"
