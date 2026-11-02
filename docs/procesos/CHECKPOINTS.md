# CHECKPOINTS — los puntos de control de cada entrega (P2, issue #28)

> Adaptado de software-factory-base a imalive (español, labels y comandos
> propios). Presupuesto: ~500 líneas. Divergencias intencionales del ASF:
> sin skills web/Tauri, sin ADRs ajenos, docs en español.

## Los cinco checkpoints (C1–C5)

Un checkpoint es un estado verificable con un comando, no una promesa. Una
entrega que no puede nombrar su checkpoint no sabe dónde está.

| # | Checkpoint | Qué verifica | Comando por stack (backend / android / ios / contracts) |
|---|---|---|---|
| C1 | **Compila** | El código del cambio construye entero | `go build ./...` · `./gradlew :app:assembleDebug` · `xcodebuild build-for-testing` · `yq parse openapi.yaml` |
| C2 | **Pruebas del cambio** | Los tests nuevos corren y la regresión del paquete tocado está verde | `go test ./paquete/...` · `./gradlew :app:testDebugUnitTest` · `xcodebuild test -only-testing:TranquiTests` · — |
| C3 | **Estáticos** | Linters del stack sin hallazgos nuevos | `make lint` · `:app:lintDebug`+`ktlintCheck`+`detekt` · `swiftlint --strict`+`swiftformat --lint` · `make -C contracts lint` |
| C4 | **Contrato y schema** | Lo que toca el schema/contrato no rompe ni deriva | `make sqlc-verify` (si queries) · `make -C contracts diff` (aditivo sin `contract-break`) · migraciones hacia adelante |
| C5 | **Cierre** | CI del runner verde en el stack tocado + revisión (self o reviewer-agente) + evidencia en el PR | el CI mismo · `factory_handoff.py` si trigger dual |

Regla dura: **ningún checkpoint se declara verde sin haber corrido su
comando en esta sesión**. "Debería compilar" no es un checkpoint.

## Los tres niveles de verificación (V0–V2)

El nivel lo decide el riesgo de la entrega, no la prisa:

| Nivel | Cuándo | Alcance |
|---|---|---|
| **V0 — en vuelo** | Cada work-unit commit (la disciplina de fábrica) | C1 + C2 del paquete tocado. Rápido y frecuente; no reemplaza a V1. |
| **V1 — pre-PR** | Antes de abrir el PR (el mínimo para abrirlo) | C1–C3 completos del stack tocado + C4 si el cambio toca schema/contrato. Los hallazgos que solo el CI puede correr (detekt en Android por el jvm-target del JBR, swiftlint por sourcekitd) se declaran "en manos del CI" y se confirman en V2. |
| **V2 — cierre** | El PR listo para revisión/merge | C1–C5: CI del runner verde en el stack tocado, self-review del diff completo publicada, evidencia exacta de comandos en el PR. |

## Mapeo a labels y triggers

| Situación | Nivel exigido | Nota |
|---|---|---|
| `size:xs` / `size:s` | V1 + C5 | V0 implícito en los work-unit commits |
| `size:m` / `size:l` | **V2** | plan del architect publicado antes (protocolo vigente) |
| `priority:p0` | **V2** siempre | el corazón del producto no se mergea con menos |
| `judgment-day` (trigger dual del handoff) | V2 + revisión dual ciega | el paquete de `factory_handoff.py` es la única entrada |
| `contract-break` | C4 **en rojo prohibido** — requiere versión nueva y ADR | el diff aditivo es la vía default |
| `size:exception` | V2 + motivo en el PR | el slicing honesto no cabe; el motivo se revisa, no se perdona |
| `ux:shape` | V2 + evidencia visual (capturas/emulador adjuntas) | la UI se verifica mirándola: la evidencia ha cazado desvíos reales |

## El checkpoint que no existe en imalive

No hay "checkpoint humano de aprobación" por defecto: la instrucción
standing del fundador ("si todo verde, mergea y seguimos") hace del CI +
self-review el gate de las entregas rutinarias. La revisión dual ciega
(`judgment-day`) y los issues `needs-human` son los dos caminos donde el
fundador vuelve a ser el gate.
