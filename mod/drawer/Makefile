# Build the NickelDrawer NickelHook mod. From the repo root:
#   make -C mod/drawer docker-build     # build/mod/libnickeldrawer.so, build/mod/KoboRoot-nickeldrawer.tgz
#   make -C mod/drawer docker-test      # host unit tests of drawer_logic (container's native g++)
#   mod/drawer/check-symbols.sh         # static checks of the built library

REPO_ROOT       := $(abspath $(CURDIR)/../..)
NICKELTC_IMAGE  ?= ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840
OUTPUT_DIR      := $(REPO_ROOT)/build/mod
NICKELHOOK_DIR  := ../../third_party/NickelHook
GENERATED_IN_VENDOR := nhplugin.json nhplugin.moc nhplugin.moc.o nh.o
LIBRARY_NAME    := libnickeldrawer.so

ifneq ($(IN_NICKELTC),)
# --- inside the container: NickelHook build system ---
override VERSION := 0.1.0
include $(NICKELHOOK_DIR)/NickelHook.mk

override LIBRARY  := $(LIBRARY_NAME)
override SOURCES  += nickeldrawer.cc nickel_bridge.cc mod_switches.cc drawer_logic.cc fallback_strings.cc \
                     model_parser.cc appsctl_call.cc model_client.cc view_settings.cc press_area.cc \
                     painted_marks.cc item_widgets.cc drawer_view.cc plain_dialog.cc dialogs.cc \
                     wifi_connector.cc action_runner.cc drawer_session.cc drawer_controller.cc \
                     drawer_navigator.cc entry_row.cc
override MOCS     += nickel_bridge.h appsctl_call.h model_client.h press_area.h drawer_view.h plain_dialog.h \
                     dialogs.h wifi_connector.h action_runner.h drawer_session.h drawer_navigator.h entry_row.h
override PKGCONF  += Qt5Widgets
override CFLAGS   += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden
override CXXFLAGS += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden -fvisibility-inlines-hidden -fno-exceptions -fno-rtti
override CPPFLAGS += -I$(NICKELHOOK_DIR)

include $(NICKELHOOK_DIR)/NickelHook.mk

# Nickel dynamic_casts our controller (push/pop/prepareView), so its vtable
# needs typeinfo. The later flag wins over -fno-rtti.
drawer_controller.o: override CXXFLAGS += -frtti
endif

# --- on the host ---
TEST_BINARY := tests/drawer_logic_test.bin
TEST_FLAGS  := -std=c++11 -Wall -Wextra -Werror -fno-exceptions -fno-rtti -nostdinc++

.PHONY: docker-build docker-clean docker-test
docker-build:
	mkdir -p $(OUTPUT_DIR)
	docker run --rm --platform linux/amd64 -v "$(REPO_ROOT):$(REPO_ROOT)" -w "$(CURDIR)" \
		-e IN_NICKELTC=1 --entrypoint make $(NICKELTC_IMAGE) all koboroot
	cp $(LIBRARY_NAME) $(OUTPUT_DIR)/$(LIBRARY_NAME)
	cp KoboRoot.tgz $(OUTPUT_DIR)/KoboRoot-nickeldrawer.tgz
	$(MAKE) docker-clean

# -nostdinc++ proves the pure logic needs no C++ standard library headers.
docker-test:
	docker run --rm --platform linux/amd64 -v "$(REPO_ROOT):$(REPO_ROOT)" -w "$(CURDIR)" --entrypoint sh \
		$(NICKELTC_IMAGE) -c 'g++ $(TEST_FLAGS) -o $(TEST_BINARY) tests/drawer_logic_test.cc drawer_logic.cc && ./$(TEST_BINARY); status=$$?; rm -f $(TEST_BINARY); exit $$status'

# Removes intermediate files, including those NickelHook generates inside its vendored folder.
docker-clean:
	rm -f *.o *.moc *.so KoboRoot.tgz $(TEST_BINARY)
	cd $(NICKELHOOK_DIR) && rm -f $(GENERATED_IN_VENDOR)
