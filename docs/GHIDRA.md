# Reverse-engineering setup (Ghidra)

Used in Phase 2+ to map libnickel for the native drawer. Kobo binaries stay in `~/.cache/nickeldrawer/` and Ghidra projects outside the repo; only symbol names and our notes are committed (see `re/README.md`).

## Versions (pinned)

| Component | Version | Why |
|---|---|---|
| Ghidra | 11.3.2 (`ghidra_11.3.2_PUBLIC_20250415.zip`, SHA-256 `99d45035bdcc3d6627e7b1232b7b379905a9fad76c772c920602e2b5d8b2dac2`) | Pinned so analysis results are reproducible |
| JDK | OpenJDK 21 (`brew install openjdk@21`) | Ghidra 11.3 requires JDK 21 |

## Install (macOS, no admin rights needed)

```sh
brew install openjdk@21
# Ghidra 11.3.2 from github.com/NationalSecurityAgency/ghidra/releases → ~/Applications/ghidra_11.3.2_PUBLIC (verify SHA-256 above)
# point Ghidra at JDK 21:
#   support/launch.properties: JAVA_HOME_OVERRIDE=/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home
```

## Analyse a firmware build

```sh
re/scripts/fetch-fw.sh <version>          # downloads + extracts libnickel into ~/.cache/nickeldrawer
PATH=/opt/homebrew/opt/openjdk@21/bin:$PATH ~/Applications/ghidra_11.3.2_PUBLIC/support/analyzeHeadless ~/.local/share/nickeldrawer/ghidra-projects \
  libnickel-<version> -import ~/.cache/nickeldrawer/fw/<version>/root/usr/local/Kobo/libnickel.so.1.0.0
```

## Browse a build (GUI)

1. Run Ghidra with JDK 21 on the PATH (Ghidra's launcher needs a working `java` before it reads `JAVA_HOME_OVERRIDE`; macOS only has a stub):
   ```sh
   JAVA_HOME=/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home PATH=/opt/homebrew/opt/openjdk@21/bin:$PATH ~/Applications/ghidra_11.3.2_PUBLIC/ghidraRun
   ```
2. File → Open Project → `~/.local/share/nickeldrawer/ghidra-projects/libnickel-<version>.gpr`.
3. Double-click `libnickel.so.1.0.0` to open the CodeBrowser.
