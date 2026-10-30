# Tranqui — iOS

App nativa iOS (ADR-0002): Swift 6 + SwiftUI + Observation (`@Observable`), MV con estado por feature ([estándar de capa](../../docs/stack/ios.md)).

## Requisitos

- Xcode (referencia local: 27.1; CI fija 26.3 en `macos-15`). Si `xcode-select` apunta a las Command Line Tools, usa `DEVELOPER_DIR`:

  ```bash
  export DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer
  xcodebuild -version   # debe imprimir Xcode 27.x / 26.x, no "xcodebuild requires..."
  ```

- [xcodegen](https://github.com/yonaskolb/XcodeGen) (`brew install xcodegen`; versión fijada en `docs/stack/ios.md`): `project.yml` es la fuente de verdad y **`Tranqui.xcodeproj` generado está commiteado** para que CI y otros agentes no necesiten la herramienta para buildear. Tras cambiar `project.yml` o añadir/mover archivos Swift:

  ```bash
  xcodegen generate
  ```

- Simulador "iPhone 16" (el destino matchea por nombre; crearlo una vez, sin descargar runtimes):

  ```bash
  xcrun simctl create "iPhone 16" \
    com.apple.CoreSimulator.SimDeviceType.iPhone-16 \
    com.apple.CoreSimulator.SimRuntime.iOS-27-0   # el runtime instalado local más nuevo
  ```

  Nota: el matcher de destinos de Xcode 27.1 por solo `name=` solo considera dispositivos del runtime más nuevo instalado; créalo ahí. En CI (Xcode 26.3) el matcheo por nombre funciona con cualquiera.

- SwiftLint + SwiftFormat (`brew install swiftlint swiftformat`; versiones fijadas en `docs/stack/ios.md`).

## Comandos

```bash
xcodegen generate                                        # regenera el proyecto (drift check en CI)
swiftlint lint --strict --quiet                          # lint (warnings bloquean)
swiftformat --lint .                                     # formato
xcodebuild build -scheme Tranqui \
  -destination 'platform=iOS Simulator,name=iPhone 16'
xcodebuild test -scheme Tranqui \
  -destination 'platform=iOS Simulator,name=iPhone 16'   # gate completo (lo que corre CI)
```

## Estructura

```
project.yml               # fuente de verdad del proyecto (bump de versiones aquí)
Tranqui.xcodeproj/        # generado por xcodegen, commiteado (incluye el scheme compartido)
Tranqui/Features/         # una carpeta por feature (checkin, circle, settings, sos)
Tranqui/Resources/        # Localizable.strings por idioma (es = desarrollo; en/pt)
Tests/                    # Swift Testing (modelos y repos)
UITests/                  # XCUITest de humo
```

## Estado (F0)

Scaffolding completo: home "Estoy OK" (botón 1 tap + estado estático, R-04) con copy externalizado (es/en/pt), Swift Testing + XCUITest y gates de calidad en CI. El consumo del backend (el botón hace POST /v1/checkins) llega con el MVP — el contrato vive en [`/contracts`](../../contracts/openapi.yaml).
