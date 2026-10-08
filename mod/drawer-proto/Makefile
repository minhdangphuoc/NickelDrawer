# Build the nddrawer NickelHook mod (gate-G2 prototypes A and C). From the repo root:
#   make -C mod/drawer-proto docker-build
#   mod/drawer-proto/check-symbols.sh
# Outputs: build/mod/libnddrawer.so and build/mod/KoboRoot-nddrawer.tgz

REPO_ROOT       := $(abspath $(CURDIR)/../..)
NICKELTC_IMAGE  ?= ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840
OUTPUT_DIR      := $(REPO_ROOT)/build/mod
NICKELHOOK_DIR  := ../../third_party/NickelHook
GENERATED_IN_VENDOR := nhplugin.json nhplugin.moc nhplugin.moc.o nh.o

ifneq ($(IN_NICKELTC),)
# --- inside the container: NickelHook build system ---
override VERSION := 0.1.0-proto
include $(NICKELHOOK_DIR)/NickelHook.mk

override LIBRARY  := libnddrawer.so
override SOURCES  += nddrawer.cc nickel_api.cc device_state.cc more_row.cc drawer_view.cc drawer_launcher.cc drawer_controller.cc
override MOCS     += more_row.h drawer_view.h drawer_launcher.h
override PKGCONF  += Qt5Widgets
override CFLAGS   += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden
override CXXFLAGS += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden -fvisibility-inlines-hidden -fno-exceptions -fno-rtti
override CPPFLAGS += -I$(NICKELHOOK_DIR)

include $(NICKELHOOK_DIR)/NickelHook.mk

# Nickel dynamic_casts our controller (push/pop/topController), so its vtable
# needs typeinfo. Later flag wins over -fno-rtti.
drawer_controller.o: override CXXFLAGS += -frtti
endif

# --- on the host ---
.PHONY: docker-build docker-clean
docker-build:
	mkdir -p $(OUTPUT_DIR)
	docker run --rm --platform linux/amd64 -v "$(REPO_ROOT):$(REPO_ROOT)" -w "$(CURDIR)" \
		-e IN_NICKELTC=1 --entrypoint make $(NICKELTC_IMAGE) all koboroot
	cp libnddrawer.so $(OUTPUT_DIR)/libnddrawer.so
	cp KoboRoot.tgz $(OUTPUT_DIR)/KoboRoot-nddrawer.tgz
	$(MAKE) docker-clean

# Removes intermediate files, including those NickelHook generates inside its vendored folder.
docker-clean:
	rm -f *.o *.moc *.so KoboRoot.tgz
	cd $(NICKELHOOK_DIR) && rm -f $(GENERATED_IN_VENDOR)
