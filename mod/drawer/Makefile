# Build the NickelDrawer NickelHook mod. From the repo root:
#   make -C mod/drawer docker-build     # build/mod/libnickeldrawer.so, build/mod/KoboRoot-nickeldrawer.tgz
#   make -C mod/drawer docker-test      # host unit tests of the Qt-free units (container's native g++)
#   mod/drawer/check-symbols.sh         # static checks of the built library

REPO_ROOT       := $(abspath $(CURDIR)/../..)
NICKELTC_IMAGE  ?= ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840
OUTPUT_DIR      := $(REPO_ROOT)/build/mod
NICKELHOOK_DIR  := ../../third_party/NickelHook
GENERATED_IN_VENDOR := nhplugin.json nhplugin.moc nhplugin.moc.o nh.o
LIBRARY_NAME    := libnickeldrawer.so

ifneq ($(IN_NICKELTC),)
# --- inside the container: NickelHook build system ---
# -frtti for every file: Nickel runs __dynamic_cast on our objects (IconButtonBase::tapGesture on
# the row's children, push/pop on the controller), and a vtable built with -fno-rtti has a null
# typeinfo slot that crashes it (re/notes/protoA-crash.md). check-symbols.sh checks every vtable.
override VERSION := 0.1.0
include $(NICKELHOOK_DIR)/NickelHook.mk

override LIBRARY  := $(LIBRARY_NAME)
override SOURCES  += nickeldrawer.cc ../common/debug_log.cc ../common/status_bar_inset.cc nickel_bridge.cc mod_switches.cc drawer_logic.cc fallback_strings.cc \
                     model_parser.cc appsctl_call.cc model_client.cc view_settings.cc ../common/press_area.cc ../common/page_header.cc \
                     painted_marks.cc item_widgets.cc drawer_view.cc plain_dialog.cc dialogs.cc open_popups.cc \
                     wifi_connector.cc action_runner.cc drawer_session.cc drawer_controller.cc \
                     drawer_navigator.cc entry_row.cc screen_logic.cc qr_encoder.cc ../common/nickel_fonts.cc \
                     screen_layout.cc screen_canvas.cc screen_view.cc field_editor.cc screen_session.cc \
                     navigation_stack.cc navigation.cc
override MOCS     += nickel_bridge.h appsctl_call.h model_client.h ../common/press_area.h ../common/page_header.h drawer_view.h plain_dialog.h \
                     dialogs.h open_popups.h wifi_connector.h action_runner.h drawer_session.h drawer_navigator.h entry_row.h \
                     screen_canvas.h screen_view.h field_editor.h screen_session.h
override PKGCONF  += Qt5Widgets
override CFLAGS   += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden
override CXXFLAGS += -Wall -Wextra -Werror -Wno-missing-field-initializers -fvisibility=hidden -fvisibility-inlines-hidden -fno-exceptions -frtti
override CPPFLAGS += -I$(NICKELHOOK_DIR) -I../common

include $(NICKELHOOK_DIR)/NickelHook.mk

endif

# --- on the host ---
# Each Qt-free unit and its test: tests/<unit>_test.cc + <unit>.cc.
TEST_UNITS  := drawer_logic screen_logic qr_encoder navigation_stack
TEST_BINARY := tests/unit_test.bin
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
		$(NICKELTC_IMAGE) -c 'status=0; for unit in $(TEST_UNITS); do \
			g++ $(TEST_FLAGS) -o $(TEST_BINARY) tests/$${unit}_test.cc $${unit}.cc && ./$(TEST_BINARY) || status=1; \
		done; rm -f $(TEST_BINARY); exit $$status'

# Removes intermediate files, including those NickelHook generates inside its vendored folder.
docker-clean:
	rm -f *.o *.moc *.so KoboRoot.tgz ../common/*.o ../common/*.moc $(TEST_BINARY)
	cd $(NICKELHOOK_DIR) && rm -f $(GENERATED_IN_VENDOR)
