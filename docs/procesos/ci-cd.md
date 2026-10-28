# CI/CD y releases

> Última actualización: 2026-10-27. Monorepo (ADR-0005): cada pipeline se dispara por rutas.

## GitHub Actions — workflows

| Workflow | Disparador | Rutas | Pasos |
|---|---|---|---|
| `backend.yml` | PR/push | `/backend`, `/contracts` | golangci-lint → gofumpt --check → `go test -race` → validación contrato (schemas/errores) → build imagen Docker |
| `contract.yml` | PR | `/contracts` | lint del YAML → detección de **breaking changes** → regeneración de clientes con diff limpio |
| `android.yml` | PR/push | `/apps/android` | detekt+ktlint+lint → `testDebugUnitTest` → assemble → (main) instrumentados en emulador |
| `ios.yml` | PR/push | `/apps/ios` | SwiftLint → build + Swift Testing en simulador → (main) XCUITest humo |
| `deploy.yml` | push a `main` + tags | `/backend` | push imagen a registry → deploy staging (auto) → deploy prod (manual/gated) |
| `release-mobile.yml` | tag `app-android/vX.Y.Z` / `app-ios/vX.Y.Z` | apps | build firmado → Play internal testing / TestFlight |

Secretos en GitHub Actions + gestor del proveedor (nunca en repo). Cachés de Go/Gradle/SPM para velocidad.

## Entornos

| Entorno | Qué corre | Datos | Deploy |
|---|---|---|---|
| local | docker-compose: postgres, redis, api, worker | ficticios | manual |
| staging | api + worker + web; Twilio **sandbox**; RevenueCat sandbox | beta (10 familias) | automático al merge en `main` |
| producción | api + worker + web; Twilio prod; stores | usuarios reales | **gated** (aprobación humana) |

## Backend

- Imagen Docker multi-stage (build → distroless/slim), binario Go estático.
- Migraciones corren **antes** del deploy (job separado); solo-adelante, compatibles con la versión N-1.
- Deploy: contenedores gestionados (hosting exacto en ADR-0006, pendiente); healthchecks; rollback = redeploy del tag anterior (imágenes inmutables).
- Post-deploy: smoke checks automatizados (login sandbox, check-in sandbox, job del motor corre).

## Mobile

- **Versionado**: SemVer por app (`app-android/v1.2.3`, `app-ios/v1.2.3`); `versionCode`/build number incrementales automáticos en CI.
- **Android**: CI firma con keystore secreto → Play **internal testing** (CI) → **closed testing** (beta 10 familias) → **producción por etapas** (staged rollout 10%→50%→100%). R8 on, mapping a Sentry.
- **iOS**: CI firma + sube a **TestFlight** (beta interna y externa) → App Store review → release. dSYMs a Sentry. Claims honestos desde la primera submission (riesgo de review de Apple, ver [requisitos](../producto/requisitos.md)).
- Cadencia: builds continuos a internal/TestFlight; release de producción por fase del roadmap, no por calendario.

## Feature flags

- Flags **server-driven**: la API expone configuración de producto (p. ej. toggles por fase); las apps no recompilan para activar features (clave en revisión de stores).
- MVP: tabla/config simple en backend + Redis; evaluar librería de flags en F2 solo si hace falta.

## Hotfix path

1. Rama `fix/…` desde `main` (o revert directo si es seguro).
2. Backend: merge → deploy staging → aprobación → prod. Mobile: build urgente a internal/TestFlight con bump de patch; la revisión express de Apple solo si es crítica.

## Checklist de release backend (humano/agente)

- [ ] Migraciones compatibles con N-1 · [ ] Changelog del contrato al día · [ ] Smoke checks verdes en staging · [ ] Alertas de observabilidad activas (motor SLO) · [ ] Rollback probado o plan claro
