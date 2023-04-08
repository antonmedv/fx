#!/bin/sh

curl -sAx "https://bit.ly/fx-download-counter" >/dev/null

version='24.1.0'
os=$(uname -s | tr '[:upper:]' '[:lower:]')
machine=$(uname -m)

case $os in
linux | darwin)
  ext=''
  ;;
windows)
  os=windows
  ext='.exe'
  ;;
*)
  echo "Unsupported OS: $os" >&2
  exit 1
  ;;
esac

case $machine in
x86_64 | amd64)
  arch=amd64
  ;;
arm64 | aarch64)
  arch=arm64
  ;;
*)
  echo "Unsupported architecture: $machine" >&2
  exit 1
  ;;
esac

temp_dir=$(mktemp -d) || {
  echo "temp dir error" >&2
  exit 1
}
trap 'rm -rf $temp_dir' EXIT INT TERM HUP

asset="fx_${os}_${arch}${ext}"
echo "Installing fx ${version} (${asset})"
curl -Lfs "https://github.com/antonmedv/fx/releases/download/${version}/${asset}" -o "$temp_dir/fx"

chmod +x "$temp_dir/fx"
mv "$temp_dir/fx" /usr/local/bin/fx
