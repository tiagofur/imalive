# Estándares de Android

> Última actualización: 2026-10-27. ADR-0002 (nativo). Stack autorizado en [stack.md](stack.md).

## Base técnica

- **Kotlin** (2.x estable), **Jetpack Compose** + **Material 3** para toda la UI.
- minSdk 26+ (afinar en scaffolding), targetSdk/compileSdk = última estable.
- Build: **Gradle con KTS** + version catalog (`libs.versions.toml`); dependencias solo del catálogo.
- Proyecto en `/apps/android`:

```
/app
  src/main/kotlin/…/feature/   # una carpeta por feature (checkin, circle, settings, sos)
  src/main/kotlin/…/core/      # designsystem (tokens generados), network (cliente fino del contrato), session, notification, data
  src/test/                    # unit tests
  src/androidTest/             # instrumented / Compose UI tests
```

## Arquitectura

- **MVVM con flujo unidireccional**: `ViewModel` expone un `StateFlow<UiState>` inmutable; la UI envía eventos hacia arriba. Nada de estado mutable en Composables fuera de VM.
- Capas: `UI (Compose)` → `ViewModel` → `Repository` → `network` (cliente fino del contrato) / `datastore` (solo preferencias).
- **La lógica de negocio vive en el backend**: los repos orquestan llamadas API y no contienen reglas del producto (la escalera es del servidor).
- DI: **Hilt**. Concurrencia: **Coroutines/Flow**; `viewModelScope` para estado, `Dispatchers.IO` solo en repos.
- Errores: los de red/negocio se mapean a estados de UI (`UiState.Error` con acción de reintento); nada de excepciones cruzando a la UI.

## Networking

- **Cliente fino hand-written** sobre `/contracts/openapi.yaml` (Retrofit + OkHttp + kotlinx.serialization, `core/network/TranquiApi.kt`): DTOs nombrados 1:1 con los schemas del contrato y los `examples` del spec como fixtures de test. El openapi-generator degrada OpenAPI 3.1 a 3.0; **el cliente generado vuelve cuando la superficie supere ~15 operaciones** (se reevalúa por issue).
- OkHttp interceptor para JWT (Bearer del access token) + `Authenticator` de refresh **single-flight** (401 → una rotación vía `POST /auth/refresh`; refresh 401 → sesión limpiada) y un interceptor de logging **solo en debug, nivel BASIC y con `Authorization` redactada** (cero PII en logs: teléfono y OTP viajan en el body, que BASIC nunca imprime).
- Errores: respuestas `application/problem+json` (RFC 9457) se mapean a `ApiError(code de negocio, retryAfterSeconds?, httpStatus)`; `429` sin `Retry-After` válido → cooldown por defecto de la capa de UI (60 s).
- Base URL por BuildConfig (`API_BASE_URL`); flavors `dev` (local/sandbox Twilio), `staging`, `prod` cuando exista backend desplegable.

## Notificaciones y superficies nativas

- **FCM**: `FirebaseMessagingService` guarda el token en el backend al login/rotación.
- Canales de notificación separados: `reminders`, `urgent`, `incidents` (alta prioridad), `marketing` (off por defecto) — ver [ux-movil](../diseno/ux-movil.md).
- **Widgets (F2)**: check-in directo con **Glance**; autenticación vía token persistido o intent con token firmado.
- **Geofencing (F2)**: `GeofencingClient` 100% on-device; al entrar al geofence la app llama a la API de check-in. **Nunca** se guarda ni envía historial de ubicación (solo el evento de check-in).
- SOS: intent explícito + servicio en primer plano solo si es imprescindible (revisar coste de permisos).

## Datos locales

- **DataStore (Preferences o Proto)** para: token de sesión, config de notificaciones, última pantalla. Cero datos de negocio (el backend es la fuente de verdad).

## Calidad

- **detekt + ktlint + Android Lint** (incluye Compose lint) con config versionada; CI bloquea warnings nuevos.
- Tests (ver [estrategia-testing](../calidad/estrategia-testing.md)):
  - Unit: ViewModels con coroutines-test + Turbine.
  - Compose UI: flows críticos (onboarding, check-in, SOS countdown) con `createComposeRule`.
  - Screenshot tests (Paparazzi) de los componentes del design system.
- Accessibility scanner en las pantallas del MVP; Dynamic Type (`sp`) en todo.

## Releases (resumen; detalle en [ci-cd](../procesos/ci-cd.md))

- Versionado SemVer + `versionCode` incremental automático en CI.
- Firmas con keystore gestionado en CI (secreto), Play App Signing.
- Pistas Play: internal testing (CI) → closed testing (beta 10 familias) → producción por etapas.
- ProGuard/R8 activado en release; mapping subido a Sentry.
