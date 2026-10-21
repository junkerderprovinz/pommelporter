#!/usr/bin/env bash
# make-dmg.sh packs a macOS app into a disk image whose window shows the
# background from cmd/dmgbackground, with the app on its left slot and a link
# to Applications on its right one. It needs hdiutil, tiffutil and a Finder to
# script, so it runs on a Mac or a macOS runner only.
#
#   make-dmg.sh <App.app> <background-dir> <name> <out.dmg>
#
# <background-dir> holds background.png and background@2x.png, as written by
#
#   go run github.com/junkerderprovinz/pommelporter/cmd/dmgbackground \
#     -logo logo.svg -name <name> -out <background-dir>
#
# An app that depends on pommelporter finds this script in its module cache:
#
#   "$(go list -m -f '{{.Dir}}' github.com/junkerderprovinz/pommelporter)/scripts/make-dmg.sh"
set -euo pipefail

if [ $# -ne 4 ]; then
  echo "usage: $0 <App.app> <background-dir> <name> <out.dmg>" >&2
  exit 2
fi
app=${1%/}
backgrounds=$2
name=$3
out=$4

# The window and the slot centres in points, as cmd/dmgbackground draws them.
width=660
height=400
app_x=180
apps_x=480
slot_y=206

work=$(mktemp -d)
mount=""
cleanup() {
  if [ -n "$mount" ]; then
    hdiutil detach "$mount" -force -quiet || true
  fi
  rm -rf "$work"
}
trap cleanup EXIT

stage="$work/stage"
mkdir -p "$stage/.background"
ditto "$app" "$stage/$(basename "$app")"
ln -s /Applications "$stage/Applications"
# One TIFF with both resolutions, so a Retina screen picks the sharp one.
tiffutil -cathidpicheck "$backgrounds/background.png" "$backgrounds/background@2x.png" \
  -out "$stage/.background/background.tiff"

# hdiutil's own estimate of the scratch image runs short for a large app and
# fails with "No space left on device", so the size is the app's plus slack.
need=$(du -sm "$stage" | cut -f1)
hdiutil create -quiet -ov -srcfolder "$stage" -volname "$name" -fs HFS+ \
  -format UDRW -size "$((need + 50))m" "$work/rw.dmg"

# The last field of hdiutil's last line is the mount point, which carries a
# number when a volume of the same name is already mounted.
mount=$(hdiutil attach -readwrite -noverify -noautoopen "$work/rw.dmg" | tail -1 | cut -f3-)
volume=$(basename "$mount")

# Finder keeps the layout in the volume's .DS_Store. Closing and opening the
# window again makes it write the file before the image is detached. Where
# Finder counts the title bar into the bounds, it hides a strip at the bottom
# of the background, which the generator leaves plain for that reason.
osascript <<EOF
tell application "Finder"
  tell disk "$volume"
    open
    set current view of container window to icon view
    set toolbar visible of container window to false
    set statusbar visible of container window to false
    set the bounds of container window to {200, 120, $((200 + width)), $((120 + height))}
    set opts to the icon view options of container window
    set arrangement of opts to not arranged
    set icon size of opts to 128
    set text size of opts to 13
    set background picture of opts to file ".background:background.tiff"
    set position of item "$(basename "$app")" of container window to {$app_x, $slot_y}
    set position of item "Applications" of container window to {$apps_x, $slot_y}
    close
    open
    update without registering applications
    delay 2
    close
  end tell
end tell
EOF

chmod -Rf go-w "$mount" || true
sync
# Finder can hold the volume for a moment after the window closes.
for _ in 1 2 3 4 5; do
  if hdiutil detach "$mount" -quiet; then
    mount=""
    break
  fi
  sleep 2
done
if [ -n "$mount" ]; then
  echo "could not detach $mount" >&2
  exit 1
fi

hdiutil convert -quiet "$work/rw.dmg" -format UDZO -imagekey zlib-level=9 -ov -o "$out"
