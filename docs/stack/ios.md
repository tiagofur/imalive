# Estándares de iOS

> Última actualización: 2026-10-27. ADR-0002 (nativo). Stack autorizado en [stack.md](stack.md).

## Base técnica

- **Swift 6** (strict concurrency activada por defecto) + **SwiftUI** + **Observation** (`@Observable`).
- Deployment target: iOS 17+ (revisar cobertura en scaffolding).
- Gestión de dependencias: **Swift Package Manager** exclusivamente. Mínimo de dependencias externas (el SDK cubre casi todo).
- Proyecto en `/apps/ios` (o paquete por feature con un app target fino):

```
/Tranqui
  /Features      # Onboarding, CheckIn, Circle, SOS, Settings — una carpeta por feature
  /Core          # DesignSystem (tokens generados), Network (cliente OpenAPI), Notifications
  /Widgets       # Target WidgetKit (F2/F4)
  /Tests         # Swift Testing
  /UITests       # XCUITest humo
```

## Arquitectura

- **MV con SwiftUI + Observation**: `@Observable` model por feature expone estado; la vista declara UI y envía intents al model. ViewModels tradicionales solo si un flujo lo exige.
- **Swift Concurrency**: `async/await` en toda la red; `@MainActor` en modelos de UI; `Sendable` verificado por el compilador (Swift 6 estricto). Nada de `DispatchQueue` manual.
- Capas: `View (SwiftUI)` → `FeatureModel` → `Repository` → `Network` (cliente OpenAPI generado) / `UserDefaults` (solo preferencias).
- **La lógica de negocio vive en el backend** (igual que Android); repos sin reglas del producto.
- Errores: `Result`/`throws` mapeados a estado de UI (`ErrorState` con reintento); sin `try!` ni `fatalError` en código de producto.

## Networking

- Cliente **generado** desde `/contracts/openapi.yaml` (openapi-generator → URLSession async/await) + `AuthInterceptor` con refresh de JWT.
- Base URL por configuración de build (`Debug` → staging/sandbox; `Release` → prod).

## Notificaciones y superficies nativas

- **APNs** vía `UNUserNotificationCenter`; token al backend en login/rotación.
- Categorías por severidad: recordatorio (normal), urgente/incidente (`UNNotificationInterruptionLevel.timeSensitive`), marketing (off).
- **Widgets (F2/F4)**: **WidgetKit** — check-in directo en home y **lock screen** (F4, el motor viral); actualización por timeline + push (iOS 17+).
- **Geofencing (F2)**: `CLCircularGeofence`/`CLLocationManager` 100% on-device; el check-in por API es el único dato que sale del teléfono. **Nunca** historial de ubicación.
- App Intents (F5+): "Estoy OK" por Siri/Shortcuts.

## Datos locales

- `UserDefaults`/SwiftData solo para preferencias y token; el estado de negocio vive en el backend.

## Calidad

- **SwiftLint + SwiftFormat** con config versionada; CI bloquea warnings nuevos.
- Tests (ver [estrategia-testing](../calidad/estrategia-testing.md)):
  - **Swift Testing** (`#expect`, parametrizados) para modelos y repos (con stubs de red).
  - SwiftUI Previews como validación visual constante (todos los estados del design system).
  - XCUITest de humo: onboarding, check-in, SOS.
- Accessibility Inspector en pantallas MVP; Dynamic Type obligatorio.

## Releases (resumen; detalle en [ci-cd](../procesos/ci-cd.md))

- Versionado SemVer + build number incremental en CI (`xcodebuild -settings CURRENT_PROJECT_VERSION`).
- TestFlight para beta (10 familias); producción vía App Store con revisión — **claims honestos** (no "emergencia médica"; ver riesgos en [requisitos](../producto/requisitos.md)).
- Firmas y perfiles gestionados en CI (secretos); símbolos de crash subidos a Sentry (dSYMs).
