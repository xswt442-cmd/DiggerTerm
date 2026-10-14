#!/usr/bin/env bash
# Native Linux release packaging (Ubuntu 24.04 x86_64):
# one real AppImage + one Ubuntu/Debian amd64 package.
set -Eeuo pipefail

usage() {
  echo "Usage: $0 vX.Y.Z [output-dir]" >&2
  exit 2
}

[[ $# -ge 1 && $# -le 2 ]] || usage
version="$1"
[[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || usage
[[ "$(uname -s)" == Linux && "$(uname -m)" == x86_64 ]] || {
  echo 'Linux x86_64 native build host required' >&2; exit 1;
}

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$repo"
current="$(node -p 'require("./frontend/package.json").version')"
[[ "$version" == "v$current" ]] || { echo "Version mismatch: $version vs $current" >&2; exit 1; }
grep -Eq "^  version: \"$current\"" build/config.yml || {
  echo 'build/config.yml version mismatch' >&2; exit 1;
}
[[ -s frontend/dist/index.html ]] || { echo 'Run the frontend production build first' >&2; exit 1; }

for tool in go gcc pkg-config dpkg-deb gdk-pixbuf-thumbnailer curl sha256sum file; do
  command -v "$tool" >/dev/null || { echo "Missing executable $tool" >&2; exit 1; }
done
pkg-config --exists gtk4 webkitgtk-6.0 libsoup-3.0 || {
  echo 'Missing GTK4 / WebKitGTK 6 / libsoup 3 development libraries' >&2; exit 1;
}

out="${2:-dist}"
[[ "$out" = /* ]] || out="$repo/$out"
mkdir -p "$out"
out="$(cd "$out" && pwd -P)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# Official upstream tools locked to the reviewed binary hashes.
# If a mutable upstream download changes, fail closed and review its new SHA.
download_tool() {
  local name="$1" url="$2" expected="$3" supplied="$4"
  local executable
  if [[ -n "$supplied" ]]; then
    executable="$supplied"
    [[ -f "$executable" ]] || { echo "Missing $name: $executable" >&2; exit 1; }
  else
    executable="$tmp/$name.AppImage"
    curl --fail --location --retry 3 --connect-timeout 20 \
      --max-time 240 -sS "$url" -o "$executable"
  fi
  echo "$expected  $executable" | sha256sum --check --status || {
    echo "$name checksum changed; do not run an unreviewed packaging tool" >&2; exit 1;
  }
  chmod +x "$executable"
  printf '%s\n' "$executable"
}

linuxdeploy="$(download_tool linuxdeploy \
  'https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage' \
  '8aea8da0f7f7039d2a2cecb14657d752a222a5e1d3825caeef186c82f751cdd1' \
  "${LINUXDEPLOY_BIN:-}")"
appimagetool="$(download_tool appimagetool \
  'https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage' \
  '95cbe7cce9717fce90c484e34052ee7c7f1d7635b33c12525b4776826a7d29b6' \
  "${APPIMAGETOOL_BIN:-}")"

echo '=== Compile actual Wails GTK/WebKit application ==='
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -tags production -trimpath -buildvcs=false -ldflags='-w -s' \
    -o "$tmp/diggerterm" .
file "$tmp/diggerterm"
if ldd "$tmp/diggerterm" | grep -F 'not found'; then
  echo 'Linux binary has unresolved dynamic library dependencies' >&2
  exit 1
fi

desktop="$tmp/diggerterm.desktop"
cat > "$desktop" <<'DESKTOP'
[Desktop Entry]
Type=Application
Version=1.0
Name=DiggerTerm
Comment=Lightweight terminal workspace
Exec=diggerterm
Icon=diggerterm
Terminal=false
Categories=Development;Utility;
StartupWMClass=DiggerTerm
DESKTOP

# linuxdeploy requires a conventional fixed-size icon, not the source 1024px.
gdk-pixbuf-thumbnailer -s 256 build/appicon.png "$tmp/diggerterm.png"
[[ -s "$tmp/diggerterm.png" ]]

echo '=== Package Ubuntu/Debian amd64 ==='
pkg="$tmp/deb"
install -d "$pkg/DEBIAN" "$pkg/usr/bin" "$pkg/usr/share/applications" \
  "$pkg/usr/share/icons/hicolor/256x256/apps" "$pkg/usr/share/doc/diggerterm"
install -m755 "$tmp/diggerterm" "$pkg/usr/bin/diggerterm"
install -m644 "$desktop" "$pkg/usr/share/applications/diggerterm.desktop"
install -m644 "$tmp/diggerterm.png" "$pkg/usr/share/icons/hicolor/256x256/apps/diggerterm.png"
install -m644 LICENSE "$pkg/usr/share/doc/diggerterm/copyright"
cat > "$pkg/DEBIAN/control" <<CONTROL
Package: diggerterm
Version: ${version#v}
Section: devel
Priority: optional
Architecture: amd64
Maintainer: DiggerTerm Maintainers
Depends: libgtk-4-1, libwebkitgtk-6.0-4, libayatana-appindicator3-1
Description: Lightweight terminal workspace
 DiggerTerm provides local, WSL, and SSH terminal sessions with
 daemon-managed PTY ownership and read-only process observation.
CONTROL
chmod 0755 "$pkg/DEBIAN"
chmod 0644 "$pkg/DEBIAN/control"
deb="$out/DiggerTerm-$version-linux-amd64.deb"
dpkg-deb --root-owner-group --build "$pkg" "$deb"
[[ "$(dpkg-deb --field "$deb" Architecture)" == amd64 ]]
[[ "$(dpkg-deb --field "$deb" Version)" == "${version#v}" ]]

echo '=== Package AppImage with verified linuxdeploy + AppImageKit ==='
appdir="$tmp/DiggerTerm.AppDir"
install -d "$appdir/usr/bin" "$appdir/usr/share/doc/diggerterm"
install -m755 "$tmp/diggerterm" "$appdir/usr/bin/diggerterm"
install -m644 "$tmp/diggerterm.png" "$appdir/diggerterm.png"
install -m644 LICENSE "$appdir/usr/share/doc/diggerterm/LICENSE"
mkdir -p "$tmp/tools"
ln -s "$appimagetool" "$tmp/tools/appimagetool"
export ARCH=x86_64 APPIMAGE_EXTRACT_AND_RUN=1
export PATH="$tmp/tools:$PATH"
(
  cd "$tmp"
  if ! "$linuxdeploy" --verbosity=2 --appdir="$appdir" \
      --executable="$tmp/diggerterm" --desktop-file="$desktop" \
      --icon-file="$tmp/diggerterm.png" --output appimage \
      > "$tmp/linuxdeploy.log" 2>&1; then
    tail -n 100 "$tmp/linuxdeploy.log" >&2
    exit 1
  fi
)
tail -n 12 "$tmp/linuxdeploy.log"
# The download helper may place its two tool AppImages in the same temp dir.
# Only collect the generated application image, never the packaging tools.
mapfile -t images < <(find "$tmp" -maxdepth 1 -type f -name '*.AppImage' \
  ! -name 'linuxdeploy.AppImage' ! -name 'appimagetool.AppImage')
[[ "${#images[@]}" == 1 ]] || { echo "Expected one AppImage, got ${#images[@]}" >&2; exit 1; }
appimage="$out/DiggerTerm-$version-linux-x86_64.AppImage"
install -m755 "${images[0]}" "$appimage"
file "$appimage"
"$appimage" --appimage-version
[[ "$(stat -c %s "$appimage")" -gt 1048576 ]]
[[ "$(stat -c %s "$deb")" -gt 1048576 ]]

echo "Linux release packages: $deb  $appimage"
sha256sum "$deb" "$appimage"
