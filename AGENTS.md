# AGENTS.md — reglas para agentes de la fábrica de software

Este repositorio se desarrolla con agentes autónomos. Estas reglas son obligatorias.

## Jerarquía de verdad (en orden)

1. **ADRs** (`docs/arquitectura/adr/`) — decisiones tomadas. Un ADR activo no se contradice: se propone un ADR nuevo.
2. **`docs/`** — estándares, arquitectura, API, procesos. Si el código y la docs discrepan, la docs ganan o se corrige la docs.
3. **`producto.md`** — spec de producto. Sus recomendaciones técnicas (§5.K) están **desaprobadas** (ver ADR-0001 y ADR-0002); su contenido de producto sigue vigente.
4. **Código existente** — guíate por su estilo, pero nunca por encima de los puntos 1–3.

## Decisiones inamovibles (requieren ADR nuevo + aprobación humana para cambiar)

- **Backend siempre en Go** ([ADR-0001](docs/arquitectura/adr/ADR-0001-backend-go.md)).
- **Apps móviles nativas**: Android en Kotlin + Jetpack Compose, iOS en Swift + SwiftUI. Nada de Flutter, React Native, KMP ni multiplataforma ([ADR-0002](docs/arquitectura/adr/ADR-0002-apps-moviles-nativas.md)).
- **Contrato API OpenAPI-first** ([ADR-0003](docs/arquitectura/adr/ADR-0003-contract-first-openapi.md)).
- **Motor de escalada sobre PostgreSQL con outbox** ([ADR-0004](docs/arquitectura/adr/ADR-0004-motor-escalada-postgres-outbox.md)).
- **Monorepo** ([ADR-0005](docs/arquitectura/adr/ADR-0005-monorepo.md)).
- **Privacidad mínima**: nunca guardar historial de ubicación; geofence solo on-device; disclaimer "no es un servicio de emergencia".

## Flujo de trabajo de cada sesión de agente

1. Lee [docs/README.md](docs/README.md) y [docs/agentes/fabrica-software.md](docs/agentes/fabrica-software.md).
2. Identifica la fase activa en [docs/producto/roadmap.md](docs/producto/roadmap.md) y construye **solo** dentro de ella.
3. Antes de implementar: consulta el estándar de la capa (`docs/stack/`, `docs/api/`, `docs/diseno/`).
4. Implementa con tests (ver [docs/calidad/estrategia-testing.md](docs/calidad/estrategia-testing.md)).
5. Antes de terminar: Definition of Done en [docs/agentes/fabrica-software.md](docs/agentes/fabrica-software.md).

## Prohibido

- Commitear secretos, credenciales o tokens reales.
- Añadir dependencias no listadas en [docs/stack/stack.md](docs/stack/stack.md) sin justificarlo en la PR.
- Cambiar el contrato OpenAPI de forma incompatibilidad-rompiente sin versión nueva.
- Saltarse tests o linters para "que compile".
- Construir features fuera de la fase activa del roadmap.

## Comandos rápidos

```bash
# Backend (en /backend)
make test      # go test -race ./...
make lint      # golangci-lint run
make run       # api local con docker-compose (postgres + redis)

# Android (en /apps/android)
./gradlew :app:lint testDebugUnitTest

# iOS (en /apps/ios)
xcodebuild test -scheme Tranqui -destination 'platform=iOS Simulator,name=iPhone 16'
```

(Ajustar en cuanto exista el scaffolding; los comandos definitivos viven en cada `Makefile`/proyecto.)
