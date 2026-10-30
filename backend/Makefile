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
SQLC_VERSION := v1.31.1
GOLANGCI_LINT := $(BIN)/golangci-lint
GOFUMPT := $(BIN)/gofumpt

.PHONY: lint fmt test run worker migrate-up migrate-down migrate-version migrate-new sqlc sqlc-verify

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

worker: ## go run ./cmd/worker
	set -a; [ -f .env ] && . ./.env; set +a; \
	$(GO) run ./cmd/worker

migrate-up: ## docker compose up -d postgres && go run ./cmd/migrate up
	docker compose up -d postgres
	set -a; [ -f .env ] && . ./.env; set +a; \
	$(GO) run ./cmd/migrate up

migrate-down: ## roll back one migration (dev/CI only)
	docker compose up -d postgres
	set -a; [ -f .env ] && . ./.env; set +a; \
	$(GO) run ./cmd/migrate down

migrate-version: ## show the current schema version
	set -a; [ -f .env ] && . ./.env; set +a; \
	$(GO) run ./cmd/migrate version

migrate-new: ## create migrations/NNNNNN_NAME.{up,down}.sql (NAME=required)
	@test -n "$(NAME)" || { echo "usage: make migrate-new NAME=add_feature"; exit 2; }
	@next=$$(( $$( ls migrations/*.up.sql 2>/dev/null | sed 's/.*\/\([0-9]*\)_.*/\1/' | sort -n | tail -1 ) + 1 )); \
	next=$$( printf '%06d' $$next ); \
	printf -- '-- TODO: describe the change. PII mínima justificada aquí (producto.md §H).\n' > migrations/$${next}_$(NAME).up.sql; \
	printf -- '-- (dev/CI only; see up header for the policy)\n' > migrations/$${next}_$(NAME).down.sql; \
	echo "created migrations/$${next}_$(NAME).{up,down}.sql"

sqlc: ## regenerate the typed stores from the queries (version pinned above)
	$(GO) run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

sqlc-verify: sqlc ## fail if the committed stores drifted from the queries
	git diff --exit-code -- internal/circle/store internal/checkin/store

$(GOLANGCI_LINT):
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GOFUMPT):
	$(GO) install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
