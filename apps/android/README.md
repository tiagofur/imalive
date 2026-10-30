# Tranqui — Android

App nativa Android (ADR-0002): Kotlin + Jetpack Compose + Material 3, MVVM con flujo unidireccional y Hilt ([estándar de capa](../../docs/stack/android.md)).

## Requisitos

- JDK 17 (toolchain de Gradle).
- Android SDK con `platforms;android-37` (compileSdk; AGP lo descarga si las licencias están aceptadas).

## Comandos

```bash
./gradlew :app:lint testDebugUnitTest detekt ktlintCheck  # gate completo (lo que corre CI)
./gradlew :app:assembleDebug                              # APK de debug
```

## Estructura

```
app/src/main/kotlin/com/imalive/tranqui/   # una carpeta por feature cuando crezca (checkin, circle, settings, sos)
app/src/main/kotlin/com/imalive/tranqui/core/   # designsystem, network (cliente OpenAPI), notification, data
app/src/test/                              # unit tests (ViewModels + Compose UI vía Robolectric)
app/src/androidTest/                       # instrumented
gradle/libs.versions.toml                  # única fuente de versiones (bump = PR dedicada)
```

## Estado (F0)

Scaffolding completo: home "Estoy OK" (botón 1 tap + estado estático, R-04) con test Compose bajo Robolectric, Hilt configurado, gates de calidad en CI. El consumo del backend (el botón hace POST /v1/checkins) llega con el MVP — el contrato vive en [`/contracts`](../../contracts/openapi.yaml).
