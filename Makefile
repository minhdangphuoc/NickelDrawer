# koboapps build and packaging. `make check build dist` before tools/install-to-kobo.sh.
VERSION ?= 0.1.0
ARM_ENV = CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7
GO_BUILD = go build -trimpath -ldflags='-s -w'
DIST_ONBOARD = dist/onboard
HOST_DIR = $(DIST_ONBOARD)/.adds/koboapps
APPS_DIR = $(DIST_ONBOARD)/.adds/apps

.PHONY: check build dist clean

check:
	@unformatted=$$(gofmt -l cmd internal); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	go test ./...

build:
	mkdir -p build
	$(ARM_ENV) $(GO_BUILD) -o build/appsctl-linux-armv7 ./cmd/appsctl
	$(ARM_ENV) $(GO_BUILD) -o build/ghsync-linux-armv7 ./cmd/ghsync
	cd third_party/localgo && $(ARM_ENV) $(GO_BUILD) -o ../../build/localgo-linux-armv7 ./cmd/localgo
	go build -o build/appsctl-host ./cmd/appsctl
	go build -o build/nmstrip-host ./cmd/nmstrip

# dist/.stamp is what tools/install-to-kobo.sh compares against the sources.
dist: build
	rm -rf dist
	mkdir -p $(HOST_DIR)/bin $(HOST_DIR)/lib $(HOST_DIR)/locales $(APPS_DIR)/localsend/bin $(APPS_DIR)/ghsync/bin $(APPS_DIR)/ghsync/data
	cp build/appsctl-linux-armv7 $(HOST_DIR)/bin/appsctl
	cp host/lib/common.sh $(HOST_DIR)/lib/common.sh
	cp host/locales/*.json $(HOST_DIR)/locales/
	cp apps/localsend/app.json $(APPS_DIR)/localsend/app.json
	cp apps/localsend/bin/*.sh $(APPS_DIR)/localsend/bin/
	mkdir -p $(APPS_DIR)/localsend/locales && cp apps/localsend/locales/*.json $(APPS_DIR)/localsend/locales/
	cp build/localgo-linux-armv7 $(APPS_DIR)/localsend/bin/localgo
	cp apps/ghsync/app.json $(APPS_DIR)/ghsync/app.json
	cp apps/ghsync/bin/*.sh $(APPS_DIR)/ghsync/bin/
	mkdir -p $(APPS_DIR)/ghsync/locales && cp apps/ghsync/locales/*.json $(APPS_DIR)/ghsync/locales/
	cp build/ghsync-linux-armv7 $(APPS_DIR)/ghsync/bin/ghsync
	cp apps/ghsync/data/config.example.json $(APPS_DIR)/ghsync/data/config.example.json
	chmod 755 $(HOST_DIR)/bin/appsctl $(APPS_DIR)/localsend/bin/* $(APPS_DIR)/ghsync/bin/*
	chmod 644 $(HOST_DIR)/lib/common.sh $(HOST_DIR)/locales/*.json $(APPS_DIR)/*/locales/*.json $(APPS_DIR)/*/app.json $(APPS_DIR)/ghsync/data/config.example.json
	cd dist && COPYFILE_DISABLE=1 zip -qr koboapps-$(VERSION).zip onboard
	touch dist/.stamp

clean:
	rm -rf build dist
