#!/bin/sh
# Background worker (not a menu action). Builds <Images>/Albums/<name>.cbz for every folder that
# holds png/jpg files and skips albums that are newer than all their images.

. "$(dirname "$0")/lib.sh"

made=0
skipped=0
failed=0

is_album_current() {
    album=$1
    folder=$2
    [ -f "$album" ] || return 1
    [ -z "$(find "$folder" -maxdepth 1 -type f -newer "$album" | head -n 1)" ]
}

build_album_for() {
    folder=$1
    album=$ALBUMS_DIR/$(album_name_for "$folder").cbz
    if is_album_current "$album" "$folder"; then
        skipped=$((skipped + 1))
    elif "$MKCBZ" -out "$album" "$folder"; then
        made=$((made + 1))
    else
        failed=$((failed + 1))
    fi
}

folder_list=$STATE_DIR/imageview-folders.txt

mkdir -p "$ALBUMS_DIR"
list_album_folders >"$folder_list"
# Reading from a file keeps the loop in this shell, so the counters survive.
while read -r folder; do
    folder_has_images "$folder" && build_album_for "$folder"
done <"$folder_list"
rm -f "$folder_list" "$PID_FILE"
say msg.status_result made="$made" skipped="$skipped" failed="$failed" >"$RESULT_FILE"
