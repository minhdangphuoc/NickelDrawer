# koboapps build and packaging. `make check build dist` before tools/install-to-kobo.sh.
# `make release VERSION=x.y.z` additionally signs dist/ (maintainer only, see docs/UPDATES.md).
VERSION ?= 1.0.0-beta.1
ARM_ENV = CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7
GO_BUILD = go build -trimpath -ldflags='-s -w'
APPSCTL_LDFLAGS = -s -w -X main.version=$(VERSION)
PRIVATE_KEY = $(HOME)/.config/nickeldrawer/release-ed25519.key
DIST_ONBOARD = dist/onboard
HOST_DIR = $(DIST_ONBOARD)/.adds/koboapps
APPS_DIR = $(DIST_ONBOARD)/.adds/apps

.PHONY: check build dist release sync-embedded schema golden-test shellcheck shellcheck-docker clean

# go:embed cannot reach outside a package folder, so two source files are mirrored into the
# packages that embed them; tests fail if a mirror is stale. cmp first keeps mtimes stable.
sync-embedded:
	@cmp -s host/locales/en.json cmd/appsctl/embedded/en.json || cp host/locales/en.json cmd/appsctl/embedded/en.json
	@cmp -s spec/release-ed25519.pub internal/release/release-ed25519.pub || cp spec/release-ed25519.pub internal/release/release-ed25519.pub

# Rewrites spec/ui.schema.json from the component registry in internal/uikit.
schema:
	go test ./internal/uikit -run TestSchemaFileIsCurrent -update

# The drawer model and screen JSON that the mod parses, compared with cmd/appsctl/testdata/drawer.
# Rewrite them after a deliberate change: go test ./cmd/appsctl -run Golden -update
golden-test:
	go test ./cmd/appsctl -run Golden

check: sync-embedded
	@unformatted=$$(gofmt -l cmd internal); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	go test ./...
	@$(MAKE) --no-print-directory shellcheck

# shellcheck is optional: without it the check says so and moves on.
SHELL_SOURCES = tools/*.sh tools/lib/*.sh host/bin/*.sh host/lib/common.sh apps/*/bin/*.sh examples/*/bin/*.sh
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

# dist/.stamp is what tools/install-to-kobo.sh compares against the sources.
dist: build
	rm -rf dist
	mkdir -p $(HOST_DIR)/bin $(HOST_DIR)/lib $(HOST_DIR)/locales $(HOST_DIR)/config $(HOST_DIR)/screens $(APPS_DIR)/localsend/bin $(APPS_DIR)/ghsync/bin $(APPS_DIR)/wallpaper/bin $(APPS_DIR)/wallpaper/locales $(APPS_DIR)/library/bin $(APPS_DIR)/library/locales $(APPS_DIR)/todo/bin $(APPS_DIR)/todo/locales $(APPS_DIR)/uikit/bin $(APPS_DIR)/uikit/locales $(HOST_DIR)/icons/ui
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
	for app in localsend ghsync wallpaper library todo uikit; do cp apps/$$app/icon.png $(APPS_DIR)/$$app/icon.png; done
	cp apps/localsend/app.json apps/localsend/ui.json $(APPS_DIR)/localsend/
	cp apps/localsend/bin/*.sh $(APPS_DIR)/localsend/bin/
	mkdir -p $(APPS_DIR)/localsend/locales && cp apps/localsend/locales/*.json $(APPS_DIR)/localsend/locales/
	cp build/localgo-linux-armv7 $(APPS_DIR)/localsend/bin/localgo
	cp apps/ghsync/app.json apps/ghsync/ui.json $(APPS_DIR)/ghsync/
	cp apps/ghsync/bin/*.sh $(APPS_DIR)/ghsync/bin/
	mkdir -p $(APPS_DIR)/ghsync/locales && cp apps/ghsync/locales/*.json $(APPS_DIR)/ghsync/locales/
	cp build/ghsync-linux-armv7 $(APPS_DIR)/ghsync/bin/ghsync
	cp apps/ghsync/config.example.json $(APPS_DIR)/ghsync/config.example.json
	cp apps/wallpaper/app.json apps/wallpaper/ui.json $(APPS_DIR)/wallpaper/
	cp apps/wallpaper/bin/*.sh $(APPS_DIR)/wallpaper/bin/
	cp apps/wallpaper/locales/*.json $(APPS_DIR)/wallpaper/locales/
	cp build/wallpaper-linux-armv7 $(APPS_DIR)/wallpaper/bin/wallpaper
	cp apps/library/app.json apps/library/ui.json $(APPS_DIR)/library/
	cp apps/library/bin/*.sh $(APPS_DIR)/library/bin/
	cp apps/library/locales/*.json $(APPS_DIR)/library/locales/
	cp build/library-linux-armv7 $(APPS_DIR)/library/bin/library
	cp apps/todo/app.json apps/todo/ui.json $(APPS_DIR)/todo/
	cp apps/todo/bin/*.sh $(APPS_DIR)/todo/bin/
	cp apps/todo/locales/*.json $(APPS_DIR)/todo/locales/
	cp build/todo-linux-armv7 $(APPS_DIR)/todo/bin/todo
	cp apps/uikit/app.json apps/uikit/ui.json $(APPS_DIR)/uikit/
	cp apps/uikit/bin/*.sh $(APPS_DIR)/uikit/bin/
	cp apps/uikit/locales/*.json $(APPS_DIR)/uikit/locales/
	chmod 755 $(HOST_DIR)/bin/* $(APPS_DIR)/localsend/bin/* $(APPS_DIR)/ghsync/bin/* $(APPS_DIR)/wallpaper/bin/* $(APPS_DIR)/library/bin/* $(APPS_DIR)/todo/bin/* $(APPS_DIR)/uikit/bin/*
	chmod 644 $(HOST_DIR)/lib/common.sh $(HOST_DIR)/locales/*.json $(HOST_DIR)/config/routing.json $(HOST_DIR)/config/update.json $(HOST_DIR)/config/project.json $(HOST_DIR)/icons/*.png $(HOST_DIR)/icons/ui/*.png $(HOST_DIR)/screens/settings/ui.json $(HOST_DIR)/screens/settings/locales/*.json $(APPS_DIR)/*/icon.png $(APPS_DIR)/*/locales/*.json $(APPS_DIR)/*/app.json $(APPS_DIR)/localsend/ui.json $(APPS_DIR)/wallpaper/ui.json $(APPS_DIR)/library/ui.json $(APPS_DIR)/todo/ui.json $(APPS_DIR)/uikit/ui.json $(APPS_DIR)/ghsync/ui.json $(APPS_DIR)/ghsync/config.example.json
	if [ -d $(HOST_DIR)/mod ]; then chmod 644 $(HOST_DIR)/mod/*; fi
	cd dist && COPYFILE_DISABLE=1 zip -qr koboapps-$(VERSION).zip onboard
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
	build/ndrelease-host sign --dist $(DIST_ONBOARD) --version $(VERSION)
	build/ndrelease-host verify

clean:
	rm -rf build dist
