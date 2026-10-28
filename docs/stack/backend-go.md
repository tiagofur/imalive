# Estándares de backend — Go

> Última actualización: 2026-10-27. Stack autorizado en [stack.md](stack.md); arquitectura en [arquitectura/overview.md](../arquitectura/overview.md).

## Estructura del proyecto

```
/backend
  cmd/
    api/          # main del servidor HTTP
    worker/       # main del motor de escalada + dispatcher outbox
    webpages/     # main de páginas web (alerta/portal) — o paquete montado en api
  internal/
    auth/         # OTP, JWT, magic links de contactos
    checkin/      # señales de vida: app, whatsapp, sms, voz, geofence (dominio)
    circle/       # usuarios, contactos, círculos
    escalation/   # MOTOR: daily cycle, incidentes, jobs (el corazón)
    notification/ # outbox, dispatcher, adapters Twilio/FCM/APNs
    billing/      # entitlements, webhooks RevenueCat
    web/          # handlers de páginas server-rendered
    platform/     # infra compartida: clock, db, redis, config, logging, telemetry
  migrations/     # SQL versionado (golang-migrate)
  api/            # handlers HTTP generados/validados contra /contracts/openapi.yaml
```

- **Dominio primero**: la lógica de negocio vive en `internal/*` sin depender de HTTP ni de SQL; los handlers y repos son adaptadores.
- `pkg/` está prohibido salvo utilidades realmente genéricas; todo lo de negocio va en `internal/`.
- Un paquete por concepto, no por capa (`escalation` contiene modelos + lógica + repo de ese concepto).

## Convenciones de código

- **Formato**: `gofumpt`. **Lint**: `golangci-lint` (config en repo: errcheck, govet, staticcheck, revive, gocritic, rowserrcheck, noctx…).
- **Nombres**: inglés; cortos y según idioma Go (`getUserByID`, `Incident`, `notifyCircle`). Sin húngaro ni abreviaturas raras.
- **Errores**:
  - `fmt.Errorf("abrir incidente %s: %w", id, err)` — siempre envolver con contexto.
  - Errores centinela en el dominio (`var ErrUserPaused = errors.New(...)`) y `errors.Is/As` para clasificar.
  - Nada de `panic` fuera de arranque; los errores de negocio tienen problema RFC 9457 asociado.
- **Context**: primer parámetro siempre `ctx context.Context`; toda operación de E/S lo propaga. Timeouts por petición en middleware.
- **Config**: struct tipado validada al arranque (`ConfigFromEnv()`); fail-fast si falta algo.
- **Logging**: `slog` JSON; campos estructurados (`slog.String("user_id", …)`); **nunca** loguear PII de más (ver seguridad); request-id en todo log de petición.

## HTTP (API)

- Router: `net/http` con patrones `GET /v1/checkins/{id}` (o chi si el middleware lo pide).
- Cadena de middleware: request-id → logging → recovery → telemetry → rate limit → auth (según ruta). Recovery va por **dentro** del request-logger: así el registro `request` lleva el 500 real de un pánico y ambos registros (`request` y `panic recovered`) llevan `request_id`.
- Handlers finos: parsean/validan entrada, llaman al dominio, mapean respuesta/errores (RFC 9457). Cero lógica de negocio en handlers.
- Contrato: los handlers se validan contra `/contracts/openapi.yaml` en CI (ADR-0003).
- Webhooks (Twilio, RevenueCat): verificación de **firma obligatoria** antes de parsear; idempotencia por id externo.

## Datos

- **sqlc** para queries SQL tipadas; SQL explícito versionado en `migrations/` (una migración = un cambio, solo hacia adelante).
- Transacciones en el repositorio; **el outbox se escribe en la misma transacción** que el cambio de estado (ADR-0004).
- Todo timestamp en UTC (`timestamptz`); la hora de check-in del usuario se guarda con su zona (`text` IANA + offset resuelto en dominio).
- IDs: UUID v7 (ordenables por tiempo).
- Sin ORM; sin query builders mágicos — SQL explícito es parte de la auditabilidad.

## Testing

- **Table-driven tests** como idioma por defecto (`tests/` junto al código, mismo paquete o `_test` externo para API pública).
- **Reloj falso** (`platform.Clock` interfaz) en todo el motor de escalada; ver escenarios obligatorios en [arquitectura/motor-escalada.md](../arquitectura/motor-escalada.md).
- **testcontainers-go** para integración con Postgres/Redis real (migraciones incluidas).
- Fakes de Twilio/FCM/APNs detrás de las interfaces de `internal/notification` — nunca llamadas reales en tests.
- Comando único: `make test` → `go test -race -cover ./...`. CI falla con cualquier race.

## Comandos (Makefile)

```makefile
run:        ## docker-compose up -d postgres redis && go run ./cmd/api
worker:     ## go run ./cmd/worker
test:       ## go test -race -cover ./...
lint:       ## golangci-lint run && gofumpt -l -w .
migrate:    ## golang-migrate up
gen:        ## openapi-generator (clientes) + validación del contrato
```

## Reglas de revisión backend (checklist de PR)

1. ¿El dominio queda libre de HTTP/SQL? 2. ¿Errores envueltos con contexto y clasificados? 3. ¿Toda E/S con ctx y timeout? 4. ¿Outbox transaccional donde cambia estado? 5. ¿Job idempotente y reintentable? 6. ¿Tests con reloj fijo cubren los estados nuevos? 7. ¿Log sin PII? 8. ¿Migración solo-adelante y compatible?
