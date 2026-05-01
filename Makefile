# Root Makefile — delegates to backend/
.PHONY: build test lint ci help

%:
	$(MAKE) -C backend $@
