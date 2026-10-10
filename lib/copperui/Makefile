# CopperUI library tasks. From the repo root:
#   make -C lib/copperui docker-test   # host unit tests of the pure code, in the NickelTC container
# The library is built into the mod by mod/drawer/Makefile (copperui.mk); there is no separate
# binary. Tests use the container's native g++ with -nostdinc++ to prove the code needs no C++
# standard library.

REPO_ROOT      := $(abspath $(CURDIR)/../..)
NICKELTC_IMAGE ?= ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840
TEST_NAMES     := pack components
TEST_SOURCES   := src/pack.cc src/pager.cc src/button.cc src/list_row.cc src/divider.cc
TEST_FLAGS     := -std=c++11 -Wall -Wextra -Werror -fno-exceptions -fno-rtti -nostdinc++ -Iinclude -Itests

.PHONY: docker-test test
docker-test:
	docker run --rm --platform linux/amd64 -v "$(REPO_ROOT):$(REPO_ROOT)" -w "$(CURDIR)" --entrypoint make \
		$(NICKELTC_IMAGE) test

test:
	@status=0; for name in $(TEST_NAMES); do \
		g++ $(TEST_FLAGS) -o tests/$$name.bin tests/$${name}_test.cc $(TEST_SOURCES) && tests/$$name.bin || status=1; \
		rm -f tests/$$name.bin; \
	done; exit $$status
