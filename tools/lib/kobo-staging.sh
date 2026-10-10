# shellcheck shell=bash
# Finding the mounted Kobo and staging a NickelHook mod archive as .kobo/KoboRoot.tgz. Sourced by
# tools/device-session.sh, tools/install-to-kobo.sh and helpers/statusbar-screenshot/install.sh so the
# safety check lives in one place. The caller defines fail() and STAGED_TGZ (<kobo>/.kobo/KoboRoot.tgz).

# Where a desktop mounts the Kobo (volume label KOBOeReader), most likely first.
KOBO_VOLUME_NAME=KOBOeReader
KOBO_SEARCH_DIRS=${KOBO_SEARCH_DIRS:-/Volumes /media/${USER:-$(id -un)} /run/media/${USER:-$(id -un)} /mnt}

# Prints the first mounted Kobo volume, or the volume path in the first search dir when none
# is mounted (on macOS that is /Volumes, so the "no Kobo found" message names a path the user
# knows; an overridden search list never falls back to a real mount).
detect_kobo_root() {
    local searchDir firstSearchDir=""
    for searchDir in $KOBO_SEARCH_DIRS; do
        firstSearchDir=${firstSearchDir:-$searchDir}
        if [ -d "$searchDir/$KOBO_VOLUME_NAME/.kobo" ]; then
            echo "$searchDir/$KOBO_VOLUME_NAME"
            return
        fi
    done
    echo "${firstSearchDir:-/Volumes}/$KOBO_VOLUME_NAME"
}

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
