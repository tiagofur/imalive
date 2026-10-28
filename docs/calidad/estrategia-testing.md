# Estrategia de testing

> Última actualización: 2026-10-27. Este proyecto no tiene QA manual en equipo: **los tests son el QA**. Todo lo que no esté probado, no existe.

## Filosofía

- El riesgo dominante del producto es **el motor de escalada** (notificaciones perdidas = coste humano) y las **falsas alarmas** (coste de confianza). Los esfuerzos de testing se concentran ahí.
- La cobertura es un indicador, no una meta; en el **motor de escalada** sí es obligatoria (≥ 90% del paquete `internal/escalation`).
- Todo bug corregido entra con un test que lo reproduce (regresión mínima).

## Backend (Go)

| Nivel | Qué | Herramientas | Gate |
|---|---|---|---|
| Unit (dominio) | Reglas: ciclos, escalera, incidentes, plans | `testing` table-driven + **Clock falso** | bloqueante |
| Integración | Repos + migraciones + transacciones/outbox con Postgres y Redis reales | testcontainers-go | bloqueante |
| Motor (escenarios) | Días completos simulados: ver lista obligatoria abajo | testcontainers + reloj simulado | bloqueante |
| Contrato | Handlers cumplen `/contracts/openapi.yaml` (schemas + errores RFC 9457) | validación en CI (kin-openapi u oapi-codegen) | bloqueante |
| Webhooks | Firma Twilio/RevenueCat válida, inválida, duplicada | unit | bloqueante |

**Escenarios obligatorios del motor** (deben existir como tests desde el scaffolding, ver [motor-escalada §Cómo probarlo](../arquitectura/motor-escalada.md)):
1. check-in a tiempo cierra el día · 2. ausencia total → suave → urgente → contacto 1 → círculo · 3. "Voy yo" cierra con notificación al círculo · 4. "No puedo" escala al siguiente · 5. señal tardía cierra como falsa alarma · 6. pausa/viaje congela y auto-resume · 7. SOS cancelado y no cancelado · 8. job falla 3 veces → `failed` + alerta · 9. doble entrega de job no duplica notificación · 10. cambio de zona horaria (F2).

- Ejecución: `go test -race ./...` en CI (las races son bloqueantes).

## Android

| Nivel | Qué | Herramientas |
|---|---|---|
| Unit | ViewModels y repos (fakes de red) | JUnit + coroutines-test + Turbine |
| UI de componentes | Pantallas críticas: onboarding, home, SOS countdown | Compose testing (`createComposeRule`) |
| Screenshot | Design system (tokens claro/oscuro) | Paparazzi |
| Humo E2E | Login OTP (sandbox), check-in, SOS contra staging | instrumented (`androidTest`) |

## iOS

| Nivel | Qué | Herramientas |
|---|---|---|
| Unit | Feature models y repos (stubs) | **Swift Testing** (`#expect`, parametrizado) |
| Preview | Todos los estados del design system (claro/oscuro/Dynamic Type) | SwiftUI Previews en PR |
| Humo E2E | Login, check-in, SOS | XCUITest |

## Web de contactos (alerta "Voy yo / No puedo")

- Tests de integración en Go: token válido/expirado/reusado, doble tap, incidente ya reclamado (idempotencia del cierre).
- La página se prueba también como render (html/template snapshot) — es la cara de confianza del producto.

## Datos de prueba

- Fábricas/constructores de dominio con defaults sensatos (`NewTestUser(withContact("María"))`), no fixtures sueltas.
- Nunca datos reales de usuarios en tests ni en staging (ver [seguridad](../seguridad/seguridad-privacidad.md)).
- Credenciales de integraciones: solo sandbox (Twilio sandbox, RevenueCat sandbox).

## Los tests en el flujo de la fábrica

- Todo PR ejecuta la batería completa de su stack (ver [ci-cd](../procesos/ci-cd.md)); sin verde no hay merge.
- Un agente **no da por terminada** una tarea sin tests nuevos/actualizados cubriendo el cambio (parte del DoD, ver [fabrica-software](../agentes/fabrica-software.md)).
- Beta cerrada (10 familias, R-15) es el test de producto: medir falsas alarmas y check-in rate antes de escalar.
