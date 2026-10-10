# Build the status-bar screenshot helper (a maintainer-only NickelHook mod, not part of dist/ or the
# releases). From the repo root:
#   make -C helpers/statusbar-screenshot docker-build   # build/helpers/libstatusbarshot.so, KoboRoot-statusbarshot.tgz
#   make -C helpers/statusbar-screenshot docker-test    # host unit tests of the Qt-free units (container's native g++)
#   helpers/statusbar-screenshot/check-symbols.sh       # static checks of the built library
#   helpers/statusbar-screenshot/install.sh             # back up and stage it on a mounted Kobo

REPO_ROOT       := $(abspath $(CURDIR)/../..)
NICKELTC_IMAGE  ?= ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840
OUTPUT_DIR      := $(REPO_ROOT)/build/helpers
NICKELHOOK_DIR  := ../../third_party/NickelHook
COMMON_DIR      := ../../mod/common
COPPERUI_INCLUDE := ../../lib/copperui/include
# NickelHook writes its plugin manifest (the unique key nh.<library name>) into the vendored folder,
# which the drawer build shares: a stale one would give this library the drawer's key.
GENERATED_IN_VENDOR := nhplugin.json nhplugin.moc nhplugin.moc.o nh.o
LIBRARY_NAME    := libstatusbarshot.so

ifneq ($(IN_NICKELTC),)
# --- inside the container: NickelHook build system ---
# -frtti for every file: Nickel runs __dynamic_cast on objects a mod gives it, and a vtable built
# with -fno-rtti has a null typeinfo slot (re/notes/protoA-crash.md). check-symbols.sh checks every vtable.
override VERSION := 0.1.0
include $(NICKELHOOK_DIR)/NickelHook.mk

override LIBRARY  := $(LIBRARY_NAME)
override SOURCES  += statusbarshot.cc nickel_bridge.cc screenshot_button.cc helper_flags.cc helper_log.cc \
                     helper_strings.cc icon_pixmaps.cc capture_check.cc $(COMMON_DIR)/device_gate.cc $(COMMON_DIR)/locale_chain.cc
override MOCS     += nickel_bridge.h screenshot_button.h
override PKGCONF  += Qt5Widgets
override CFLAGS   += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden
override CXXFLAGS += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden -fvisibility-inlines-hidden -fno-exceptions -frtti
override CPPFLAGS += -I$(NICKELHOOK_DIR) -I$(COMMON_DIR) -I$(COPPERUI_INCLUDE)

include $(NICKELHOOK_DIR)/NickelHook.mk
endif

# --- on the host ---
# Each Qt-free unit and its test: tests/<unit>_test.cc + the source after the "=".
TEST_UNITS  := helper_flags=helper_flags.cc capture_check=capture_check.cc locale_chain=$(COMMON_DIR)/locale_chain.cc
TEST_BINARY := tests/unit_test.bin
TEST_FLAGS  := -std=c++11 -Wall -Wextra -Werror -fno-exceptions -fno-rtti -nostdinc++ -I$(COMMON_DIR)

.PHONY: docker-build docker-clean docker-test
docker-build: docker-clean
	mkdir -p $(OUTPUT_DIR)
	docker run --rm --platform linux/amd64 -v "$(REPO_ROOT):$(REPO_ROOT)" -w "$(CURDIR)" \
		-e IN_NICKELTC=1 --entrypoint make $(NICKELTC_IMAGE) all koboroot
	cp $(LIBRARY_NAME) $(OUTPUT_DIR)/$(LIBRARY_NAME)
	cp KoboRoot.tgz $(OUTPUT_DIR)/KoboRoot-statusbarshot.tgz
	$(MAKE) docker-clean

# -nostdinc++ proves the pure logic needs no C++ standard library headers.
docker-test:
	docker run --rm --platform linux/amd64 -v "$(REPO_ROOT):$(REPO_ROOT)" -w "$(CURDIR)" --entrypoint sh \
		$(NICKELTC_IMAGE) -c 'status=0; for entry in $(TEST_UNITS); do \
			unit=$${entry%%=*}; source=$${entry#*=}; \
			g++ $(TEST_FLAGS) -o $(TEST_BINARY) tests/$${unit}_test.cc $$source && ./$(TEST_BINARY) || status=1; \
		done; rm -f $(TEST_BINARY); exit $$status'

# Removes intermediate files, including those NickelHook generates inside its vendored folder.
docker-clean:
	rm -f *.o *.moc *.so KoboRoot.tgz $(COMMON_DIR)/device_gate.o $(COMMON_DIR)/locale_chain.o $(TEST_BINARY)
	cd $(NICKELHOOK_DIR) && rm -f $(GENERATED_IN_VENDOR)
