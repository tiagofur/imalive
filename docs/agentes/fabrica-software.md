# Fábrica de software — guía operativa para agentes

> Última actualización: 2026-10-27. Este documento describe cómo trabaja un agente en este repo y conecta la documentación con la infraestructura de la fábrica ya montada.

## Qué es esta fábrica

El desarrollo es realizado por agentes autónomos supervisados por el fundador. La fábrica combina:

| Pieza | Ubicación | Qué hace |
|---|---|---|
| Agentes de rol | `.zcode/agents/` (`triage`, `architect`, `implementer`, `reviewer`) | Cuatro roles especializados; cada uno con su protocolo |
| Skills de flujo | `.agents/skills/` (`triage`, `issue-to-pr`, `pr-review`) | Procedimientos ejecutables issue→PR→revisión |
| Flujo de trabajo | [procesos/flujo-desarrollo.md](../procesos/flujo-desarrollo.md) | **La regla madre**: sin issue no hay código; issue→PR→merge; labels, DoR/DoD, assertividad, ronda de fábrica |
| Plantillas | `.github/ISSUE_TEMPLATE/`, `PULL_REQUEST_TEMPLATE.md` | Issues con type/priority/size; PRs con evidencia de verificación |
| CI semilla | `.github/workflows/ci.yml` | Lint+test por stack, activado por presencia del código (monorepo en F0) |
| Labels | `scripts/setup-labels.sh` | Taxonomía `status:`, `priority:`, `size:`, `needs-human` |
| Reglas inamovibles | [AGENTS.md](../../AGENTS.md) | Decisores fijos del proyecto (Go, nativas, OpenAPI, privacidad) |

## Jerarquía de verdad

1. **ADRs** (`arquitectura/adr/`) → 2. **`docs/`** → 3. **`producto.md`** (recomendaciones técnicas §5.K desaprobadas; producto vigente) → 4. código existente.

## Ciclo de trabajo de un agente (cada sesión)

1. **Orientarse**: leer `AGENTS.md`, este documento y el [roadmap](../producto/roadmap.md) (fase activa).
2. **Tomar trabajo**: ejecutar la **ronda de fábrica** ([flujo-desarrollo §Ronda](../procesos/flujo-desarrollo.md)): triage → arquitectura (plan para size m+) → implementación del mejor issue `status:ready` → revisión de PRs abiertos → cierre.
3. **Antes de implementar**: leer el estándar de la capa que toca (`stack/backend-go.md`, `stack/android.md`, `stack/ios.md`, `api/api-guidelines.md`, `diseno/*`) y, si el issue toca check-in/escalada/notificaciones, **obligatoriamente** [arquitectura/motor-escalada.md](../arquitectura/motor-escalada.md).
4. **Implementar** por work units (comportamiento + tests + docs en el mismo commit).
5. **Verificar** con los comandos por stack (abajo) y registrar el resultado exacto en el PR.
6. **Documentar**: si tomaste una decisión técnica no trivial → ADR con la plantilla antes del PR; si la implementación contradice la docs → se corrige la docs en la misma PR.

## Verificación por stack

| Capa | Comandos |
|---|---|
| Backend (`/backend`) | `make lint` · `make test` (`go test -race ./...`) |
| Android (`/apps/android`) | `./gradlew :app:lint testDebugUnitTest` |
| iOS (`/apps/ios`) | `xcodebuild test -scheme Tranqui -destination 'platform=iOS Simulator,name=iPhone 16'` |
| Contrato (`/contracts`) | lint OpenAPI + diff compatible (sin breaking sin etiqueta `contract-break`) |
| Local full | `docker-compose up -d postgres redis` + `go run ./cmd/api` + `go run ./cmd/worker` |

## Definition of Done (resumen; completo en [flujo-desarrollo](../procesos/flujo-desarrollo.md) y [calidad](../calidad/estrategia-testing.md))

- CI verde del stack tocado; criterios de aceptación con evidencia en el PR; tests y docs en los mismos commits; PR ≤ 400 líneas (o chained-pr); issue cerrado con `Closes #N`.

## Guardarraíles (duras)

1. **Decisiones inamovibles** de `AGENTS.md` (Go, nativas, OpenAPI-first, motor Postgres+outbox, monorepo, privacidad mínima): no se re-debaten; cambiarlas exige ADR nuevo + aprobación humana.
2. **Privacidad como gate P0**: un PR que persista ubicación o amplíe PII se rechaza aunque funcione.
3. **Solo la fase activa** del roadmap; trabajo detectado → issue nuevo, nunca se mezcla en el PR.
4. **Bloqueo honesto**: credenciales, dinero, legal o ambigüedad de producto → `status:bloqueado` + `needs-human` con recomendación; **nunca** simular con datos inventados ni "a ver si compila".
5. Secretos nunca en repo; credenciales reales nunca en tests (solo sandbox).
6. `main` solo recibe merges de PR con CI verde; nadie pushea a `main`.

## Guardarraíles ejecutables (solo lectura salvo indicación)

| Script | Uso |
|---|---|
| `python3 scripts/factory_preflight.py` | Arranque de sesión: verifica archivos de la fábrica y estado de git. No corre tests ni autoriza escrituras. `--require-clean` antes de entregar. |
| `python3 scripts/factory_handoff.py --base origin/main --head <sha> --issue <N>` | Paquete de revisión con contexto limpio: SHAs fijados, digest, diff y triggers de revisión dual. Única entrada del revisor. |
| `scripts/setup-labels.sh` | Taxonomía de labels (incluye `contract-break`, `judgment-day`, `size:exception`). |
| `python3 -m unittest discover -s tests` | Tests de los guardarraíles. |

## Antes de terminar una sesión

- [ ] Issues movidos de estado con comentarios explicando el resultado.
- [ ] PRs abiertos con evidencia de verificación y self-review hecha.
- [ ] Docs/ADRs actualizados si hubo decisiones.
- [ ] Si todo está bloqueado o no hay `status:ready`: terminar reportando bloqueos — **nunca inventar trabajo**.
