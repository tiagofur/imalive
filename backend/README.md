# Backend Go

API + worker de imalive ([ADR-0001](../docs/arquitectura/adr/ADR-0001-backend-go.md)).
Estado actual: **scaffolding** — estructura, calidad y logging; cero endpoints de
producto (el contrato OpenAPI llega con el issue #2).

## Requisitos

- Go (versión fijada en [`go.mod`](go.mod)).
- Docker y docker compose (para el entorno local, cuando exista el compose).

## Comandos

```bash
make lint    # golangci-lint run (versión pinneada en el Makefile)
make fmt     # gofumpt -l -w .
make test    # go test -race -cover ./...
```

Las herramientas de calidad se instalan solas, a la versión fijada en el
`Makefile`, la primera vez que se invocan (el plan técnico descartó las
directivas `tool` de go.mod para no inflar el diff con ~1.200 líneas de
`go.sum` generadas).

## Layout

Ver [`doc.go`](doc.go) y el estándar de capa
[docs/stack/backend-go.md](../docs/stack/backend-go.md): los binarios viven en
`cmd/` (`api`, `worker`), el dominio y la infraestructura compartida en
`internal/` (`internal/platform/` para infra transversal).

## Configuración (variables de entorno)

Validada al arranque, fail-fast ([`internal/platform/config`](internal/platform/config/config.go)):

| Variable | Obligatoria | Default | Valores |
|---|---|---|---|
| `APP_ENV` | sí | — | `local` · `staging` · `production` |
| `PORT` | no | `8080` | puerto TCP válido |
| `LOG_LEVEL` | no | `info` | `debug` · `info` · `warn` · `error` |
| `DATABASE_URL` | declarada¹ | — | PostgreSQL ([ADR-0004](../docs/arquitectura/adr/ADR-0004-motor-escalada-postgres-outbox.md)) |
| `REDIS_URL` | declarada¹ | — | Redis |

¹ Se exigirán cuando exista su primer consumidor de datos. `VERSION` se inyecta
con `-ldflags "-X main.version=..."`, nunca por entorno.

## Logging

`slog` JSON a stdout ([`internal/platform/logging`](internal/platform/logging/logging.go)).
Campos estructurados; **sin PII, sin headers ni cuerpos de petición** (privacidad
día 0). El request-id llegará como gancho de correlación con el middleware HTTP.
