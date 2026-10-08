# Native-mod toolchain

| Item | Value |
|---|---|
| Toolchain | NickelTC (https://github.com/pgaskin/NickelTC), GCC 4.9.4, Qt 5.2.1 sysroot |
| Image | `ghcr.io/pgaskin/nickeltc:1` |
| Digest (pinned in `mod/hello/Makefile`) | `sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840` |
| Mod framework | `third_party/NickelHook/` (see `third_party/README.md` for commit and license) |

## Build

```sh
docker pull --platform linux/amd64 ghcr.io/pgaskin/nickeltc:1
make -C mod/hello docker-build
```

Outputs (gitignored): `build/mod/libndhello.so` and `build/mod/KoboRoot-ndhello.tgz`
(installs to `/usr/local/Kobo/imageformats/libndhello.so`).

The Makefile runs NickelHook's build system inside the container, copies the
results out, then deletes the intermediate files, including the ones NickelHook
generates inside `third_party/NickelHook/`.

## Apple Silicon

The image is amd64-only, so Docker runs it under emulation (`--platform linux/amd64`
is passed everywhere). It works without extra setup; no host cross compiler is needed.
Build time for the hello mod: about 5 s warm (first run also pays the image pull).

## Static checks after a build

```sh
file build/mod/libndhello.so   # ELF 32-bit LSB shared object, ARM, EABI5
docker run --rm --platform linux/amd64 -v "$PWD/build/mod:/o" --entrypoint sh \
  ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840 \
  -c 'arm-nickel-linux-gnueabihf-readelf -d -A /o/libndhello.so; arm-nickel-linux-gnueabihf-nm -D --undefined-only /o/libndhello.so'
```

Expected: `Tag_ABI_VFP_args: VFP registers` (hard float), NEEDED limited to libdl, Qt5Core,
Qt5Gui, libstdc++, libgcc_s, libpthread, libc, and from libstdc++ only `operator new/delete`
plus `__cxa_guard_*`/`__aeabi_atexit` (no STL).

## g++ 4.9 gotchas

- Designated initializers must list every field in declaration order (no skipping); that is why
  `hello.cc` sets `.uninstall_xflag = NULL` and `.hook = NULL` explicitly.
- Qt 5.2 has no `QTimer::singleShot(msec, functor)`; use a `QObject` receiver and slot (`toast.h`, moc'd).
