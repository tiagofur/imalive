# Backend Go — comandos definitivos (docs/stack/backend-go.md).
#
# Herramientas de calidad pinneadas a versión fija e instaladas con `go install`
# (fallback del plan técnico del issue #1: la directiva `tool` de go.mod añadiría
# ~1.200 líneas generadas de go.sum/go.mod al diff, fuera del presupuesto de
# revisión de una PR).

GO ?= go
BIN := $(or $(shell $(GO) env GOBIN),$(shell $(GO) env GOPATH)/bin)

GOLANGCI_LINT_VERSION := v2.15.0
GOFUMPT_VERSION := v0.12.0
GOLANGCI_LINT := $(BIN)/golangci-lint
GOFUMPT := $(BIN)/gofumpt

.PHONY: lint fmt test run

lint: $(GOLANGCI_LINT) ## golangci-lint run (versión pinneada arriba)
	$(GOLANGCI_LINT) run

fmt: $(GOFUMPT) ## gofumpt -l -w . (versión pinneada arriba)
	$(GOFUMPT) -l -w .

test: ## go test -race -cover ./...
	$(GO) test -race -cover ./...

run: ## docker compose up -d postgres redis && go run ./cmd/api
	docker compose up -d postgres redis
	set -a; [ -f .env ] && . ./.env; set +a; \
	$(GO) run ./cmd/api

$(GOLANGCI_LINT):
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GOFUMPT):
	$(GO) install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
