# Root Makefile — delegates to backend/
.PHONY: build test lint ci distributions help

distributions:
	@if command -v pwsh >/dev/null 2>&1; then \
		pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/build-distributions.ps1; \
	else \
		bash scripts/build-distributions.sh; \
	fi

%:
	$(MAKE) -C backend $@
