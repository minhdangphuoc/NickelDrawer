#!/bin/sh
# Prints the screen variables as one JSON object: the stored control values, the readout of the
# last event, the fake job's progress, two read-only facts about the device, the sample books and
# the gallery tiles. Local files only (well under 1 s).
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
    if [ "$(stored_flag shownote false)" = true ]; then
        msg "$APP_ID" msg.note_text
    fi
}

# The sample list of the List and Menu pages; the titles are the books of the examples.
sample_books() {
    printf '[["Truyện Kiều","Nguyễn Du"],["Số đỏ","Vũ Trọng Phụng"],["Dế Mèn phiêu lưu ký","Tô Hoài"],["Pride and Prejudice","Jane Austen"]]'
}

# The sample checklist of the List page: an item [title, "✓"] is done, a plain title is open.
sample_reading() {
    printf '[["Truyện Kiều","✓"],"Số đỏ",["Dế Mèn phiêu lưu ký","✓"],"Pride and Prejudice"]'
}

# The readout rows exist once an event was recorded: their labels are empty until then, and a kv
# row whose key and value are both empty is hidden.
readout_fields() {
    event=$(stored_value lastEvent '')
    value=$(stored_value lastValue '')
    label_event=""
    label_value=""
    if [ -n "$event" ]; then
        label_event=$(msg "$APP_ID" msg.label_event)
        label_value=$(msg "$APP_ID" msg.label_value)
        value=${value:-—}
    fi
    printf '"lastEvent":"%s","lastValue":"%s","labelEvent":"%s","labelValue":"%s"' \
        "$(json_escape "$event")" "$(json_escape "$value")" "$(json_escape "$label_event")" "$(json_escape "$label_value")"
}

# The fake job (steps.sh): "<label key>|<done>|<total>|<detail>" while it runs or after it ended.
job_fields() {
    running=false
    label=""
    done_count=""
    total=""
    detail=""
    if [ -s "$JOB_FILE" ]; then
        IFS='|' read -r key done_count total detail <"$JOB_FILE"
        label=$(msg "$APP_ID" "$key")
        if [ -f "$DATA_DIR/job.pid" ] && kill -0 "$(cat "$DATA_DIR/job.pid")" 2>/dev/null; then
            running=true
        fi
    fi
    printf '"jobRunning":%s,"progressLabel":"%s","progressDone":"%s","progressTotal":"%s","progressDetail":"%s"' \
        "$running" "$(json_escape "$label")" "$(json_escape "$done_count")" "$(json_escape "$total")" "$(json_escape "$detail")"
}

ensure_sample_pictures
printf '{"firmware":"%s","locale":"%s","sampleenabled":%s,"autorescan":%s,"shownote":%s,"noteText":"%s",' \
    "$(json_escape "$(firmware_version)")" "$(json_escape "$(device_locale)")" \
    "$(stored_flag sampleenabled true)" "$(stored_flag autorescan false)" "$(stored_flag shownote false)" \
    "$(json_escape "$(note_text)")"
printf '"refreshmode":"%s","searchtext":"%s","pagenumber":"%s","books":%s,"reading":%s,"pictures":%s,%s,%s}\n' \
    "$(json_escape "$(stored_value refreshmode balanced)")" "$(json_escape "$(stored_value searchtext '')")" \
    "$(json_escape "$(stored_value pagenumber 1)")" "$(sample_books)" "$(sample_reading)" "$(gallery_tiles)" "$(readout_fields)" "$(job_fields)"
