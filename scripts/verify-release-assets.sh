#!/usr/bin/env bash
# Validate exactly the three release installers and write a unified SHA256SUMS.
set -Eeuo pipefail
[[ $# -ge 1 && $# -le 2 && "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  echo "Usage: $0 vX.Y.Z [release-directory]" >&2; exit 2;
}
version="$1"
dir="${2:-dist}"
cd "$dir"
win="DiggerTerm-$version-windows-amd64.exe"
image="DiggerTerm-$version-linux-x86_64.AppImage"
deb="DiggerTerm-$version-linux-amd64.deb"
test -s "$win" && test -s "$image" && test -s "$deb" || {
  echo 'One or more required release installers are missing' >&2; exit 1;
}
shopt -s nullglob
files=(*)
[[ "${#files[@]}" == 3 ]] || {
  printf 'Expected exactly 3 installers, found: %s\n' "${files[*]}" >&2; exit 1;
}
file "$win" | grep -E 'PE32\+.*x86-64'
file "$image" | grep -E 'ELF 64-bit.*x86-64'
[[ "$(dpkg-deb --field "$deb" Architecture)" == amd64 ]]
[[ "$(dpkg-deb --field "$deb" Version)" == "${version#v}" ]]
chmod +x "$image"
"./$image" --appimage-version
sha256sum "$win" "$image" "$deb" > SHA256SUMS
sha256sum -c SHA256SUMS
echo "Verified three v${version#v} installers; checksums:"
cat SHA256SUMS
