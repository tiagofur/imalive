# Stack oficial

> Última actualización: 2026-10-29. Añadir una tecnología no listada aquí exige justificarlo en la PR (y ADR si es arquitectónica).

## Decisiones por capa

| Capa | Elección | Por qué / notas |
|---|---|---|
| **Backend** | **Go** | ADR-0001. API + worker motor + páginas web |
| **API** | REST + JSON, OpenAPI 3.1 contract-first | ADR-0003. Contrato en `/contracts/openapi.yaml` |
| **Android** | **Kotlin + Jetpack Compose + Material 3** | ADR-0002 |
| **iOS** | **Swift + SwiftUI + Observation** | ADR-0002 |
| **Base de datos** | PostgreSQL 16+ (gestionado) | Fuente de verdad + cola del motor (ADR-0004) |
| **Caché/ephemeral** | Redis 7+ | OTP, rate limit, feature flags. Descartable por diseño |
| **Auth** | OTP por teléfono con **Twilio Verify**; JWT corto + refresh | Sin contraseñas; contactos nunca se registran (magic links firmados) |
| **WhatsApp/SMS** | **Twilio** (WhatsApp Business API; sandbox en dev, SMS fallback) | Canal #1 en LATAM; iniciar verificación Meta Business ya |
| **Push** | **FCM** (Android) + **APNs** (iOS) | Canales separados por severidad |
| **Pagos (móvil)** | **RevenueCat** + webhooks | Paywall desde el día 1; plans Gratis/Premium/Cuidar |
| **Voz IA (F3)** | Twilio Voice + LLM/TTS (ElevenLabs u otro) | Post-MVP; coste ~$0.03–0.08/min |
| **Web contactos/portal** | Server-rendered en Go (html/template) | Links de alerta sin app (R-08) y portal cuidador (F3) |
| **Observabilidad** | `slog` JSON + OpenTelemetry + Prometheus + **Sentry** (apps) | Ver [observabilidad](../observabilidad/observabilidad.md) |
| **CI/CD** | GitHub Actions | Ver [procesos/ci-cd](../procesos/ci-cd.md) |
| **Contenedores** | Docker + docker-compose (local); imagen distroless/slim | Hosting: pendiente ADR-0006 |

## Backend Go — librerías autorizadas (mínimas)

| Uso | Librería |
|---|---|
| Router HTTP | `net/http` stdlib (patrones de métodos/rutas Go 1.22+) o `go-chi/chi` si hace falta middleware fino |
| Postgres | `jackc/pgx/v5` + **sqlc** para queries tipadas |
| Migraciones | `golang-migrate` (SQL puro, versionado en repo) |
| Config | Variables de entorno (`envconfig` o stdlib), validadas al arranque |
| Logging | `log/slog` (JSON) con request-id y user-id |
| Métricas/tracing | OpenTelemetry SDK + Prometheus |
| Validación | Validadores propios en el dominio (preferido) o `go-playground/validator` |
| Test | stdlib `testing` (table-driven) + `testcontainers-go` + `stretchr/testify` (asserts) |

## Android — librerías autorizadas

Jetpack: Compose, Material 3, Lifecycle/ViewModel, Navigation-Compose, Hilt, Coroutines/Flow, DataStore. Red: OkHttp + Retrofit + kotlinx.serialization con cliente fino del contrato (el cliente generado openapi-generator vuelve cuando la superficie supere ~15 operaciones; umbral en [android.md](android.md)). Notificaciones: FCM. Widgets (F2+): Glance. Build: Gradle KTS + version catalog (`libs.versions.toml`). Calidad: detekt + ktlint + lint de Compose. Detalles en [android.md](android.md).

## iOS — librerías autorizadas

SwiftUI, Observation (`@Observable`), Swift Concurrency (async/await, actors), UNUserNotificationCenter/APNs, URLSession (+ cliente generado OpenAPI), Swift Package Manager (único gestor de dependencias), WidgetKit (F2+). Calidad: SwiftLint + SwiftFormat. Detalles en [ios.md](ios.md).

## Herramientas transversales

| Uso | Herramienta |
|---|---|
| Generación del proyecto iOS | `xcodegen` (2.46.0 vía brew; `xcodegen generate` en `apps/ios`) — `project.yml` es la fuente de verdad y el `.xcodeproj` generado se comitea para que CI y agentes no necesiten instalar la herramienta para buildear (solo para regenerar; drift check en CI: `xcodegen generate && git diff --exit-code`). Descartado pbxproj manual (frágil e irrevieweable por agentes) y app solo como Swift Package (no produce app target iOS con scheme/Info.plist/UI tests). |
| Generación de clientes | `openapi-generator-cli` (Kotlin → OkHttp/retrofit2; Swift → URLSession async/await) |
| Lint del contrato OpenAPI | `Spectral` (`@stoplight/spectral-cli`, versión pinneada en `contracts/Makefile`, instalado on-demand vía `npx`) — gate contract-first de ADR-0003: valida el spec, exige `operationId`/descripciones/examples coherentes y aplica la regla de privacidad `no-location-fields` (producto.md §H). Descartado Redocly CLI (su bundle multi-archivo no aporta con un spec único). |
| Diff del contrato (breaking changes) | `oasdiff` (binario Go, versión pinneada en `contracts/Makefile`, instalado con `go install`) — compara cada PR contra `main` y falla ante cambios incompatibles (válvula: label `contract-break`). Descartado `openapi-diff` (centrado en OpenAPI 3.0, runtime Java). |
| Lint backend | `golangci-lint` (+ `gofumpt`) |
| Vulnerabilidades | `govulncheck` (Go), Dependabot/Renovate en todo el repo |
| Tests E2E humo | Postman/Newman o script Go contra staging |
| Gestión de secretos | Gestor del proveedor de hosting (o Doppler/1Password Teams) — nunca en el repo |
| Datos locales app | Android: DataStore; iOS: UserDefaults/SwiftData (solo preferencias; el estado vive en el backend) |

## Política de versiones

- Usar la **versión estable más reciente** de cada lenguaje/tooling en el momento del scaffolding y **fijarla** (go.mod, version catalog, `.xcodeenv`/Gemfile, CI matrix).
- Revisión trimestral de versiones (o al empezar cada fase del roadmap); las actualizaciones menores son PRs normales, las mayores van con PR dedicada y tests.
- minSdk Android: 26+ (decidir exacto en scaffolding según cobertura de dispositivos del público LATAM); iOS deployment target: 17+ (revisar cobertura al iniciar).
