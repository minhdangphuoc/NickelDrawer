# Build the ndhello NickelHook mod. Run from the repo root:
#   make -C mod/hello docker-build
# Outputs: build/mod/libndhello.so and build/mod/KoboRoot-ndhello.tgz

REPO_ROOT       := $(abspath $(CURDIR)/../..)
NICKELTC_IMAGE  ?= ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840
OUTPUT_DIR      := $(REPO_ROOT)/build/mod
NICKELHOOK_DIR  := ../../third_party/NickelHook
GENERATED_IN_VENDOR := nhplugin.json nhplugin.moc nhplugin.moc.o nh.o

ifneq ($(IN_NICKELTC),)
# --- inside the container: NickelHook build system ---
override VERSION := 0.1.0
include $(NICKELHOOK_DIR)/NickelHook.mk

override LIBRARY  := libndhello.so
override SOURCES  += hello.cc toast.cc
override MOCS     += toast.h
override CFLAGS   += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden
override CXXFLAGS += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden -fvisibility-inlines-hidden -fno-exceptions -frtti
override CPPFLAGS += -I$(NICKELHOOK_DIR)

include $(NICKELHOOK_DIR)/NickelHook.mk
endif

# --- on the host ---
.PHONY: docker-build docker-clean
docker-build:
	mkdir -p $(OUTPUT_DIR)
	docker run --rm --platform linux/amd64 -v "$(REPO_ROOT):$(REPO_ROOT)" -w "$(CURDIR)" \
		-e IN_NICKELTC=1 --entrypoint make $(NICKELTC_IMAGE) all koboroot
	cp libndhello.so $(OUTPUT_DIR)/libndhello.so
	cp KoboRoot.tgz $(OUTPUT_DIR)/KoboRoot-ndhello.tgz
	$(MAKE) docker-clean

# Removes intermediate files, including those NickelHook generates inside its vendored folder.
docker-clean:
	rm -f *.o *.moc *.so KoboRoot.tgz
	cd $(NICKELHOOK_DIR) && rm -f $(GENERATED_IN_VENDOR)
