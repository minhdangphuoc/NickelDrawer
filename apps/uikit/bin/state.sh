#!/bin/sh
# Prints the screen variables as one JSON object: the stored control values, two read-only facts
# about the device and the gallery tiles. Local files only (well under 1 s).
. "$(dirname "$0")/lib.sh"

MAX_SAMPLES=6
MAX_SAMPLE_BYTES=1048576

firmware_version() {
    version=$(head -n 1 "$KOBO_DIR/version" 2>/dev/null | cut -d, -f3)
    printf '%s' "${version:--}"
}

device_locale() {
    locale=$(sed -n 's/^CurrentLocale=//p' "$KOBO_DIR/Kobo/Kobo eReader.conf" 2>/dev/null | head -n 1 | tr -d '\r')
    printf '%s' "${locale:--}"
}

# Copies up to MAX_SAMPLES small PNG files of Pictures into data/samples as s1.png, s2.png ...
# (with the original name in s1.label): the gallery only shows PNG files below the app's data/.
# The folder is read once; the Rescan header icon forgets the copies.
copy_sample_pictures() {
    mkdir -p "$SAMPLES_DIR" || return 0
    number=0
    for picture in "$PICTURES_DIR"/*.png; do
        [ -f "$picture" ] || continue
        [ "$(wc -c <"$picture")" -le "$MAX_SAMPLE_BYTES" ] || continue
        number=$((number + 1))
        cp "$picture" "$SAMPLES_DIR/s$number.png" || return 0
        basename "$picture" >"$SAMPLES_DIR/s$number.label"
        [ "$number" -lt "$MAX_SAMPLES" ] || break
    done
}

gallery_tiles() {
    tiles=""
    for thumb in "$SAMPLES_DIR"/s*.png; do
        [ -f "$thumb" ] || continue
        id=$(basename "$thumb" .png)
        label=$(json_escape "$(head -n 1 "${thumb%.png}.label" 2>/dev/null)")
        tile="{\"id\":\"$id\",\"thumb\":\"$thumb\",\"image\":\"$thumb\",\"label\":\"$label\",\"info\":\"$label\"}"
        tiles="$tiles${tiles:+,}$tile"
    done
    printf '[%s]' "$tiles"
}

ensure_sample_pictures() {
    if ! ls "$SAMPLES_DIR"/s*.png >/dev/null 2>&1; then
        copy_sample_pictures
    fi
}

note_text() {
    if [ "$(stored_flag shownote)" = true ]; then
        msg "$APP_ID" msg.note_text
    fi
}

ensure_sample_pictures
printf '{"firmware":"%s","locale":"%s","shownote":%s,"hiddennote":"%s",' \
    "$(json_escape "$(firmware_version)")" "$(json_escape "$(device_locale)")" \
    "$(stored_flag shownote)" "$(json_escape "$(note_text)")"
printf '"refreshmode":"%s","searchtext":"%s","pagenumber":"%s","pictures":%s}\n' \
    "$(json_escape "$(stored_value refreshmode balanced)")" "$(json_escape "$(stored_value searchtext '')")" \
    "$(json_escape "$(stored_value pagenumber 1)")" "$(gallery_tiles)"
