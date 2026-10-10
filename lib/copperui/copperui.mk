# CopperUI as a static library inside the NickelTC build. Include this file after the final
# `include NickelHook.mk` of a mod's Makefile (inside the IN_NICKELTC block):
#
#   COPPERUI_DIR := ../../lib/copperui
#   include $(COPPERUI_DIR)/copperui.mk
#
# It builds $(COPPERUI_DIR)/libcopperui.a with the mod's compiler and flags (so the same -frtti,
# -fPIC and -fvisibility settings apply) and makes $(LIBRARY) link it. Only the objects the mod
# uses are pulled out of the archive. The sources need no C++ standard library and no libm.
# `make docker-clean` of the mod removes the objects and the archive (COPPERUI_GENERATED).

COPPERUI_SOURCES  := $(addprefix $(COPPERUI_DIR)/src/,pack.cc pager.cc button.cc list_row.cc divider.cc empty_state.cc)
COPPERUI_OBJECTS  := $(COPPERUI_SOURCES:.cc=.o)
COPPERUI_ARCHIVE  := $(COPPERUI_DIR)/libcopperui.a
COPPERUI_GENERATED := $(COPPERUI_OBJECTS) $(COPPERUI_ARCHIVE)

$(COPPERUI_OBJECTS): %.o: %.cc
	$(CXX) $(CPPFLAGS) $(CXXFLAGS) -c $< -o $@

$(COPPERUI_ARCHIVE): $(COPPERUI_OBJECTS)
	rm -f $@
	$(AR) rcs $@ $^

$(LIBRARY): $(COPPERUI_ARCHIVE)
