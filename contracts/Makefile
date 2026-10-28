# Contracts (OpenAPI) — definitive commands (docs/api/api-guidelines.md).
#
# Quality tooling pinned to fixed versions and installed on demand
# (same pattern as backend/Makefile): Spectral via npx, oasdiff via `go install`.

SPEC := openapi.yaml
BASE ?= origin/main

SPECTRAL_VERSION := 6.17.0
OASDIFF_VERSION := v1.35.0

NPX ?= npx
GO ?= go
BIN := $(or $(shell $(GO) env GOBIN),$(shell $(GO) env GOPATH)/bin)
OASDIFF := $(BIN)/oasdiff

.PHONY: lint diff check

lint: ## spectral lint with the repo ruleset (pinned version above)
	$(NPX) --yes @stoplight/spectral-cli@$(SPECTRAL_VERSION) lint -r .spectral.yaml $(SPEC)

diff: $(OASDIFF) ## breaking-change check of the spec against BASE (default origin/main)
	@if git cat-file -e "$(BASE):contracts/$(SPEC)" 2>/dev/null; then \
		base=$$(mktemp -t openapi-base-XXXXXX.yaml); \
		trap 'rm -f "$$base"' EXIT; \
		git show "$(BASE):contracts/$(SPEC)" > "$$base"; \
		$(OASDIFF) breaking "$$base" "$(SPEC)"; \
	else \
		echo "contracts: no base spec at $(BASE) — skipping breaking-change check (first contracts PR)."; \
	fi

check: lint diff ## lint + breaking-change check

$(OASDIFF):
	$(GO) install github.com/oasdiff/oasdiff@$(OASDIFF_VERSION)
