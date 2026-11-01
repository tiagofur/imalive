# Estándares de iOS

> Última actualización: 2026-11-01. ADR-0002 (nativo). Stack autorizado en [stack.md](stack.md).

## Versiones (fijadas en el scaffolding F0)

| Componente | Versión | Dónde se fija |
|---|---|---|
| Swift (language mode) | 6 (strict concurrency) | `SWIFT_VERSION` en `apps/ios/project.yml` |
| Xcode (referencia local) | 27.1 (Build 27A9275) | `DEVELOPER_DIR` — ver README de `apps/ios` |
| Xcode (CI, `macos-15`) | 26.3 | `DEVELOPER_DIR` del job `ios` (fallback documentado: 16.4, default de la imagen) |
| Deployment target | iOS 17 | `apps/ios/project.yml` (cobertura LATAM por revisar antes del MVP; placeholder F0) |
| Generación del proyecto | xcodegen 2.46.0 | `brew`; `xcodegen generate` (pbxproj commiteado) |
| Lint | SwiftLint 0.65.1 · SwiftFormat 0.64.4 | `brew` (última estable; bump = PR normal) |

## Base técnica

- **Swift 6** (strict concurrency activada por defecto) + **SwiftUI** + **Observation** (`@Observable`).
- Deployment target: iOS 17 (cobertura LATAM por revisar antes del MVP).
- Gestión de dependencias: **Swift Package Manager** exclusivamente. Mínimo de dependencias externas (el SDK cubre casi todo).
- Proyecto en `/apps/ios`, **generado con xcodegen** (`project.yml` = fuente de verdad; el `.xcodeproj` generado se comitea; ver fila en [stack.md](stack.md)) (o paquete por feature con un app target fino):

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
- Capas: `View (SwiftUI)` → `FeatureModel` → `Repository` → `Network` (cliente fino del contrato) / `UserDefaults` (solo preferencias).
- **La lógica de negocio vive en el backend** (igual que Android); repos sin reglas del producto.
- Errores: `Result`/`throws` mapeados a estado de UI (`ErrorState` con reintento); sin `try!` ni `fatalError` en código de producto.

## Networking

- **Cliente fino hand-written** sobre `/contracts/openapi.yaml` (`URLSession` + async/await, `Tranqui/Core/Network/TranquiAPI.swift`): DTOs `Codable` nombrados 1:1 con los schemas del contrato y los `examples` del spec como fixtures de test. El openapi-generator degrada OpenAPI 3.1 a 3.0 (`type: [string, 'null']`, igual que en Android); **el cliente generado vuelve cuando la superficie supere ~15 operaciones** (se reevalúa por issue).
- Errores: `application/problem+json` (RFC 9457) → `ApiError` con el `code` de negocio y el `Retry-After` parseado; los fallos de transporte quedan como `URLError` (offline ≠ respuesta de negocio).
- **Bearer attach** en cada llamada protegida (desde la caché de sesión, nunca un hit de Keychain por request) y **refresh single-flight** sobre `POST /auth/refresh`: 401 del refresh invalida la sesión; un refresh inalcanzable (offline) la conserva.
- Base URL por configuración de build (`Debug` → staging/sandbox; `Release` → prod).

## Notificaciones y superficies nativas

- **APNs** vía `UNUserNotificationCenter`; token al backend en login/rotación.
- Categorías por severidad: recordatorio (normal), urgente/incidente (`UNNotificationInterruptionLevel.timeSensitive`), marketing (off).
- **Widgets (F2/F4)**: **WidgetKit** — check-in directo en home y **lock screen** (F4, el motor viral); actualización por timeline + push (iOS 17+).
- **Geofencing (F2)**: `CLCircularGeofence`/`CLLocationManager` 100% on-device; el check-in por API es el único dato que sale del teléfono. **Nunca** historial de ubicación.
- App Intents (F5+): "Estoy OK" por Siri/Shortcuts.

## Datos locales

- `UserDefaults`/SwiftData solo para preferencias (p. ej. el paso actual del onboarding); los **tokens de sesión viven en el Keychain** (`kSecClassGenericPassword`, `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`): nunca en UserDefaults ni en logs. El estado de negocio vive en el backend.

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
