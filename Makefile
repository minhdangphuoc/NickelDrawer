# koboapps build and packaging. `make check build dist` before tools/install-to-kobo.sh.
# `make release VERSION=x.y.z` additionally signs dist/ (maintainer only, see docs/UPDATES.md).
# `make dist` writes the drawer (dist/onboard, dist/koboapps-<v>.zip) and one zip per app (dist/apps).
VERSION ?= 1.0.0-beta.5
ARM_ENV = CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7
GO_BUILD = go build -trimpath -ldflags='-s -w'
APPSCTL_LDFLAGS = -s -w -X main.version=$(VERSION)
PRIVATE_KEY = $(HOME)/.config/nickeldrawer/release-ed25519.key
DIST_ONBOARD = dist/onboard
HOST_DIR = $(DIST_ONBOARD)/.adds/koboapps

.PHONY: check build dist release sync-embedded tokens schema golden-test shellcheck shellcheck-docker clean

# go:embed cannot reach outside a package folder, so two source files are mirrored into the
# packages that embed them; tests fail if a mirror is stale. cmp first keeps mtimes stable.
sync-embedded:
	@cmp -s host/locales/en.json cmd/appsctl/embedded/en.json || cp host/locales/en.json cmd/appsctl/embedded/en.json
	@cmp -s spec/release-ed25519.pub internal/release/release-ed25519.pub || cp spec/release-ed25519.pub internal/release/release-ed25519.pub

# CopperUI: lib/copperui/tokens/copperui.tokens.json (and the shared text vectors) generate the C++
# header, the Go constants, the token table in docs/ and the mod's test vectors. `make check`
# fails when any of them is stale.
tokens:
	go run ./cmd/copperui-tokens

# Rewrites spec/ui.schema.json from the component registry in internal/uikit.
schema:
	go test ./internal/uikit -run TestSchemaFileIsCurrent -update

# The drawer model and screen JSON that the mod parses, compared with cmd/appsctl/testdata/drawer.
# Rewrite them after a deliberate change: go test ./cmd/appsctl -run Golden -update
golden-test:
	go test ./cmd/appsctl -run Golden

check: sync-embedded
	go run ./cmd/copperui-tokens -check
	@unformatted=$$(gofmt -l cmd internal); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	go test ./...
	@$(MAKE) --no-print-directory shellcheck

# shellcheck is optional: without it the check says so and moves on.
SHELL_SOURCES = tools/*.sh helpers/statusbar-screenshot/*.sh tools/lib/*.sh host/bin/*.sh host/lib/common.sh apps/*/bin/*.sh examples/*/bin/*.sh
shellcheck:
	@if command -v shellcheck >/dev/null 2>&1; then \
		shellcheck -x -S warning $(SHELL_SOURCES) && echo "shellcheck: ok"; \
	else \
		echo "shellcheck not installed: skipped (brew install shellcheck, or make shellcheck-docker)"; \
	fi

# The same check as CI, through the pinned image, for machines without shellcheck.
shellcheck-docker:
	docker run --rm -v "$$PWD":/mnt -w /mnt koalaman/shellcheck:v0.10.0 -x -S warning $(SHELL_SOURCES) && echo "shellcheck: ok"

build: sync-embedded
	mkdir -p build
	$(ARM_ENV) go build -trimpath -ldflags='$(APPSCTL_LDFLAGS)' -o build/appsctl-linux-armv7 ./cmd/appsctl
	$(ARM_ENV) $(GO_BUILD) -o build/ghsync-linux-armv7 ./cmd/ghsync
	$(ARM_ENV) $(GO_BUILD) -o build/wallpaper-linux-armv7 ./cmd/wallpaper
	$(ARM_ENV) $(GO_BUILD) -o build/library-linux-armv7 ./cmd/library
	$(ARM_ENV) $(GO_BUILD) -o build/todo-linux-armv7 ./cmd/todo
	cd third_party/localgo && $(ARM_ENV) $(GO_BUILD) -o ../../build/localgo-linux-armv7 ./cmd/localgo
	go build -ldflags='-X main.version=$(VERSION)' -o build/appsctl-host ./cmd/appsctl
	go build -o build/ndrelease-host ./cmd/ndrelease
	go build -o build/ndpreview-host ./cmd/ndpreview

# dist/onboard holds the drawer only (host tool + mod); every app is packed on its own into
# dist/apps/app-<id>-<version>.zip (see apps/README.md). dist/.stamp is what
# tools/install-to-kobo.sh compares against the sources.
APPS_TREE = build/apps-tree
APPS_ZIP_DIR = dist/apps
# <app id>:<built binary>:<name inside bin/>; apps without a binary are not listed.
APP_BINARIES = localsend:localgo:localgo ghsync:ghsync:ghsync wallpaper:wallpaper:wallpaper library:library:library todo:todo:todo
APP_IDS = $(sort $(notdir $(patsubst %/,%,$(dir $(wildcard apps/*/app.json)))))

dist: build
	rm -rf dist $(APPS_TREE)
	mkdir -p $(HOST_DIR)/bin $(HOST_DIR)/lib $(HOST_DIR)/locales $(HOST_DIR)/config $(HOST_DIR)/screens $(HOST_DIR)/icons/ui
	cp build/appsctl-linux-armv7 $(HOST_DIR)/bin/appsctl
	cp host/lib/common.sh $(HOST_DIR)/lib/common.sh
	cp host/bin/*.sh $(HOST_DIR)/bin/
	cp host/locales/*.json $(HOST_DIR)/locales/
	cp host/config/routing.json $(HOST_DIR)/config/routing.json
	cp host/config/project.json $(HOST_DIR)/config/project.json
	build/appsctl-host project update-config --file host/config/project.json > $(HOST_DIR)/config/update.json
	cp host/icons/*.png $(HOST_DIR)/icons/
	cp host/icons/ui/*.png $(HOST_DIR)/icons/ui/
	cp -R host/screens/. $(HOST_DIR)/screens/
	if [ -f build/mod/KoboRoot-nickeldrawer.tgz ]; then mkdir -p $(HOST_DIR)/mod && cp build/mod/KoboRoot-nickeldrawer.tgz $(HOST_DIR)/mod/; fi
	chmod 755 $(HOST_DIR)/bin/*
	chmod 644 $(HOST_DIR)/lib/common.sh $(HOST_DIR)/locales/*.json $(HOST_DIR)/config/routing.json $(HOST_DIR)/config/update.json $(HOST_DIR)/config/project.json $(HOST_DIR)/icons/*.png $(HOST_DIR)/icons/ui/*.png $(HOST_DIR)/screens/settings/ui.json $(HOST_DIR)/screens/settings/locales/*.json
	if [ -d $(HOST_DIR)/mod ]; then chmod 644 $(HOST_DIR)/mod/*; fi
	cd dist && COPYFILE_DISABLE=1 zip -qrX koboapps-$(VERSION).zip onboard
	@for app in $(APP_IDS); do \
		binary=$$(printf '%s\n' $(APP_BINARIES) | awk -F: -v id="$$app" '$$1 == id { print "build/" $$2 "-linux-armv7 " $$3 }'); \
		tools/stage-app.sh $(APPS_TREE) "$$app" $$binary || exit 1; \
	done
	build/ndrelease-host pack-apps --tree $(APPS_TREE) --out $(APPS_ZIP_DIR)
	touch dist/.stamp

# Signs the dist tree. Needs the private key made once by `build/ndrelease-host keygen`;
# that also writes the public key appsctl embeds, so keygen comes before the first release build.
release:
	@if [ "$(origin VERSION)" != "command line" ]; then echo "Usage: make release VERSION=x.y.z"; exit 1; fi
	@if [ ! -f "$(PRIVATE_KEY)" ]; then \
		echo "No signing key at $(PRIVATE_KEY)."; \
		echo "Run once: go run ./cmd/ndrelease keygen   (then commit spec/release-ed25519.pub)"; \
		exit 1; \
	fi
	$(MAKE) check build dist VERSION=$(VERSION)
	build/ndrelease-host sign --dist $(DIST_ONBOARD) --apps-dir $(APPS_ZIP_DIR) --version $(VERSION)
	build/ndrelease-host verify

clean:
	rm -rf build dist
