# Reverse-engineering setup (Ghidra + Ghidra)

Used in Phase 2+ to map libnickel for the native drawer. Kobo binaries stay in `~/.cache/nickeldrawer/` and Ghidra projects outside the repo; only symbol names and our notes are committed (see `re/README.md`).

## Versions (pinned)

| Component | Version | Why |
|---|---|---|
| Ghidra | 11.3.2 (`ghidra_11.3.2_PUBLIC_20250415.zip`, SHA-256 `99d45035bdcc3d6627e7b1232b7b379905a9fad76c772c920602e2b5d8b2dac2`) | Ghidra 1.4 is built for exactly 11.3.2 |
| JDK | OpenJDK 21 (`brew install openjdk@21`) | Ghidra 11.3 requires JDK 21 |
| Ghidra | 1.4 (LaurieWired/Ghidra) | plugin + `bridge_mcp_ghidra.py` |
| Python MCP SDK | `mcp<2` (1.x) | the bridge uses `mcp.server.fastmcp`, removed in 2.x |

## Install (macOS, no admin rights needed)

```sh
brew install openjdk@21
# Ghidra 11.3.2 from github.com/NationalSecurityAgency/ghidra/releases → ~/Applications/ghidra_11.3.2_PUBLIC (verify SHA-256 above)
# point Ghidra at JDK 21:
#   support/launch.properties: JAVA_HOME_OVERRIDE=/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home
# Ghidra plugin: unzip Ghidra-1-4.zip into ~/Applications/ghidra_11.3.2_PUBLIC/Ghidra/Extensions/
# bridge:
python3.12 -m venv ~/.local/share/nickeldrawer/Ghidra/venv
~/.local/share/nickeldrawer/Ghidra/venv/bin/pip install 'mcp<2' requests

  ~/.local/share/nickeldrawer/Ghidra/bridge_mcp_ghidra.py --ghidra-server http://127.0.0.1:8080/
```

## Analyse a firmware build

```sh
re/scripts/fetch-fw.sh <version>          # downloads + extracts libnickel into ~/.cache/nickeldrawer
PATH=/opt/homebrew/opt/openjdk@21/bin:$PATH ~/Applications/ghidra_11.3.2_PUBLIC/support/analyzeHeadless ~/.local/share/nickeldrawer/ghidra-projects \
  libnickel-<version> -import ~/.cache/nickeldrawer/fw/<version>/root/usr/local/Kobo/libnickel.so.1.0.0
```

## Use with the agent (GUI steps, once per session)

1. Run Ghidra with JDK 21 on the PATH (Ghidra's launcher needs a working `java` before it reads `JAVA_HOME_OVERRIDE`; macOS only has a stub):
   ```sh
   JAVA_HOME=/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home PATH=/opt/homebrew/opt/openjdk@21/bin:$PATH ~/Applications/ghidra_11.3.2_PUBLIC/ghidraRun
   ```
2. File → Open Project → `~/.local/share/nickeldrawer/ghidra-projects/libnickel-<version>.gpr`.
3. Double-click `libnickel.so.1.0.0` to open the CodeBrowser.
4. First time only: File → Configure → Developer → tick **GhidraPlugin** (its HTTP server listens on 127.0.0.1:8080).
5. Start a new the maintainer session in the repo; the `ghidra` MCP tools are then available.
