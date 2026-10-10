# Staging of a NickelHook mod archive as .kobo/KoboRoot.tgz. Sourced by tools/device-session.sh and
# tools/install-to-kobo.sh so the safety check lives in one place. The caller defines
# fail() and STAGED_TGZ (<kobo>/.kobo/KoboRoot.tgz).

# Refuses an archive that would install anything but mod libraries: Nickel unpacks KoboRoot.tgz
# over the root file system as root, so only usr/local/Kobo/imageformats/*.so is allowed.
assert_only_mod_libraries() {
    local tgz=$1 entry
    while IFS= read -r entry; do
        entry=${entry#./}
        entry=${entry%/}
        case "$entry" in
        "" | usr | usr/local | usr/local/Kobo | usr/local/Kobo/imageformats) ;;
        usr/local/Kobo/imageformats/*.so) ;;
        *) fail "$tgz contains '$entry'. Only usr/local/Kobo/imageformats/*.so is allowed. Not staging." ;;
        esac
    done < <(tar tzf "$tgz")
}

# Copies the archive to .kobo/KoboRoot.tgz after the check. Only one archive can wait for a reboot.
stage_mod_archive() {
    local tgz=$1
    [ -f "$tgz" ] || fail "no such file: $tgz"
    [ ! -e "$STAGED_TGZ" ] ||
        fail "a KoboRoot.tgz is already staged. Only one can be installed per reboot; reboot the Kobo first or remove it by hand."
    assert_only_mod_libraries "$tgz"
    cp "$tgz" "$STAGED_TGZ"
}
