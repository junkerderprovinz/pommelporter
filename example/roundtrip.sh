#!/usr/bin/env bash
# Installs the example app for a throwaway home, checks what the installer put
# in place, and uninstalls it again.
set -euo pipefail

installer=$(realpath "$1")
export POMMELPORTER_PAYLOAD=$(realpath "$2")
export HOME=$(mktemp -d)
unset XDG_DATA_HOME XDG_CONFIG_HOME

id=com.junkerderprovinz.pommelexample
dir=$HOME/.local/share/$id
entry=$HOME/.local/share/applications/$id.desktop

expect() {
	if ! eval "$1"; then
		echo "expected $2" >&2
		exit 1
	fi
	echo "ok: $2"
}

"$installer" --silent
expect '[ -x "$dir/pommelexample" ]' "the program in $dir"
expect '[ -x "$dir/uninstall" ]' "the uninstaller in $dir"
expect '[ -f "$entry" ]' "the menu entry"
expect 'grep -q "^Exec=.*pommelexample" "$entry"' "the menu entry starting the program"

"$dir/uninstall" --uninstall --silent
expect '[ ! -e "$dir/pommelexample" ]' "the program removed"
expect '[ ! -e "$entry" ]' "the menu entry removed"
