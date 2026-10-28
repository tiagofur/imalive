# Backend Go

API + worker de imalive ([ADR-0001](../docs/arquitectura/adr/ADR-0001-backend-go.md)).
Estado actual: **scaffolding** — estructura, salud (`/healthz`), calidad y
logging; cero endpoints de producto (el contrato OpenAPI llega con el issue #2).

## Requisitos

- Go (versión fijada en [`go.mod`](go.mod)).
- Docker y docker compose (para el entorno local).

## Entorno local

```bash
cp .env.example .env        # config local de compose y de la api; .env está gitignoreado
make run                    # postgres + redis (compose) y la api en :8080
curl localhost:8080/healthz # {"status":"ok","version":"dev"}
```

`make run` levanta `postgres:16` y `redis:7` con healthchecks
([docker-compose.yml](docker-compose.yml)) y arranca la api en el host con la
config de `.env` (el target lo carga antes de ejecutar `go run ./cmd/api`; sin
`.env`, exporta las variables a mano). Para parar: `docker compose down` (los
datos de Postgres persisten en el volumen `postgres-data`).

## Comandos

```bash
make lint    # golangci-lint run (versión pinneada en el Makefile)
make fmt     # gofumpt -l -w .
make test    # go test -race -cover ./...
make run     # docker compose up -d postgres redis && go run ./cmd/api
```

Las herramientas de calidad se instalan solas, a la versión fijada en el
`Makefile`, la primera vez que se invocan (el plan técnico descartó las
directivas `tool` de go.mod para no inflar el diff con ~1.200 líneas de
`go.sum` generadas).

## API

Hoy solo existe `GET /healthz` — liveness del proceso con su versión, endpoint
operacional fuera de `/v1` (no toca el contrato OpenAPI, ADR-0003). La cadena
de middleware por petición: `recovery → request-id`
([`internal/platform/httpmw`](internal/platform/httpmw/)); el request-id va en
el header de respuesta `X-Request-ID`. El request-log por petición y readiness
con ping a Postgres/Redis llegan en las siguientes PRs de la cadena.

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
Campos estructurados; **sin PII, sin headers ni cuerpos ni query strings de
petición** (privacidad día 0). El log por petición (método, ruta, status,
duración, request-id) llega con el request-log middleware en la siguiente PR
de la cadena.
