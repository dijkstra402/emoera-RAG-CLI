#!/bin/sh
set -eu

REPOSITORY="${EMOERA_INSTALL_REPOSITORY:-dijkstra402/emoera-RAG-CLI}"
INSTALL_DIR="${EMOERA_INSTALL_DIR:-$HOME/.local/bin}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "缺少依赖: $1" >&2
    exit 1
  }
}

need curl
need tar

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) echo "暂不支持当前系统: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "暂不支持当前架构: $(uname -m)" >&2; exit 1 ;;
esac

if [ -n "${EMOERA_VERSION:-}" ]; then
  tag="${EMOERA_VERSION}"
else
  latest_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPOSITORY/releases/latest")"
  tag="${latest_url##*/}"
fi

version="${tag#v}"
archive="emoera-cli_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/$REPOSITORY/releases/download/$tag"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/emoera-install.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

echo "正在安装 Emoera Agent CLI $tag ($os/$arch)…"
curl -fsSL "$base_url/$archive" -o "$work_dir/$archive"
curl -fsSL "$base_url/SHA256SUMS" -o "$work_dir/SHA256SUMS"

expected="$(awk -v file="$archive" '$2 == file { print $1; exit }' "$work_dir/SHA256SUMS")"
if [ -z "$expected" ]; then
  echo "SHA256SUMS 中没有找到 $archive" >&2
  exit 1
fi

if command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$work_dir/$archive" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$work_dir/$archive" | awk '{print $1}')"
else
  echo "缺少 shasum 或 sha256sum，无法校验安装包" >&2
  exit 1
fi

if [ "$actual" != "$expected" ]; then
  echo "安装包 SHA-256 校验失败" >&2
  exit 1
fi

tar -xzf "$work_dir/$archive" -C "$work_dir"
mkdir -p "$INSTALL_DIR"
install -m 0755 "$work_dir/emoera-cli_${version}_${os}_${arch}/emoera" "$INSTALL_DIR/emoera"

echo "安装完成: $INSTALL_DIR/emoera"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo "请将下面一行加入你的 shell 配置，然后重新打开终端："
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac
"$INSTALL_DIR/emoera" --version
