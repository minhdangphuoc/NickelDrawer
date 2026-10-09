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
# -frtti for every file: Nickel runs __dynamic_cast on our objects (IconButtonBase::tapGesture on
# the row's children, push/pop on the controller), and a vtable built with -fno-rtti has a null
# typeinfo slot that crashes it (re/notes/protoA-crash.md). check-symbols.sh checks every vtable.
override VERSION := 0.1.0-proto
include $(NICKELHOOK_DIR)/NickelHook.mk

override LIBRARY  := libnddrawer.so
override SOURCES  += nddrawer.cc ../common/debug_log.cc nickel_api.cc device_state.cc more_row.cc drawer_view.cc drawer_launcher.cc drawer_controller.cc
override MOCS     += more_row.h drawer_view.h drawer_launcher.h
override PKGCONF  += Qt5Widgets
override CFLAGS   += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden
override CXXFLAGS += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden -fvisibility-inlines-hidden -fno-exceptions -frtti
override CPPFLAGS += -I$(NICKELHOOK_DIR) -I../common

include $(NICKELHOOK_DIR)/NickelHook.mk

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
	rm -f *.o *.moc *.so KoboRoot.tgz ../common/*.o
	cd $(NICKELHOOK_DIR) && rm -f $(GENERATED_IN_VENDOR)
