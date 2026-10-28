---
name: pr-review
description: "Trigger: revisar PR, code review, revisión de PR, approve PR. Protocolo de revisión de la fábrica: checklist, severidades P0-P3, verdict APPROVE/REQUEST_CHANGES."
metadata:
  version: "1.0"
---

## Activation Contract

Load this skill when reviewing a factory PR, when asked for a second review, or before merging (to confirm the PR passes).

## Hard Rules

- La barra de aceptación son los criterios del issue que el PR cierra, no el gusto del revisor.
- Read-only sobre el código del PR: leer, correr tests/lint, escribir la revisión. Nada de edits.
- Privacidad primero (`producto.md` §H, ADR-0004): persistencia de ubicación o historial de ubicaciones, ampliación de datos personales, incidentes no append-only, o claims de "emergencia médica" son P0 aunque funcionen.
- Presupuesto de revisión ≤ 400 changed lines; un PR mayor se revisa igual y se recomienda chain o `size:exception`.
- Sin P0/P1 abiertos no se piden cambios: `APPROVE` con P2/P3 anotados.
- La revisión dual adversarial (skill `judgment-day`) la invoca el usuario explícitamente **o** la dispara la sección "Revisión dual automática" de esta skill; en ambos casos reemplaza este protocolo para ese target.

## Decision Gates

| Hallazgo | Severidad | Verdict |
|---|---|---|
| Bug, seguridad, privacidad, criterio incumplido, decisión inamovible violada | P0 | REQUEST_CHANGES |
| Comportamiento incorrecto o comportamiento nuevo sin tests | P1 | REQUEST_CHANGES |
| Mejora clara sin romper nada (naming, estructura) | P2 | APPROVE con nota |
| Nit (estilo, typos) | P3 | APPROVE, sin ruido |

## Execution Steps

1. Contexto: `gh pr view`, `gh pr diff` y el issue cerrado por el PR (criterios = barra).
2. Ejecutar la verificación del stack tocado (backend `make test && make lint` · android gradle · iOS xcodebuild · contracts lint OpenAPI) y registrar comandos/resultados exactos.
3. Checklist:
   - [ ] Cada commit es una work unit coherente (tests/docs con el comportamiento)
   - [ ] Los criterios de aceptación están cubiertos y con evidencia en el PR
   - [ ] Alcance cerrado: nada fuera del issue (follow-ups referenciados, no mezclados)
   - [ ] Privacidad respetada (cero historial de ubicación, mínimos datos, append-only en incidentes)
   - [ ] Contrato OpenAPI: sin cambios rompientes (ADR-0003)
   - [ ] `Closes #N` correcto; CI en verde; no rompe `main`
4. Comentar el verdict en el PR: tabla de hallazgos (severidad, archivo:línea, qué, fix concreto) + estado de los criterios de aceptación.
5. Si `REQUEST_CHANGES`: el PR vuelve a implementación (quien implementó corrige y re-pide revisión).

## Revisión dual automática

Antes de revisar en simple, comprueba estos triggers. Si se cumple **cualquiera**, no revises en simple: ejecuta `judgment-day` sobre el PR y su veredicto reemplaza este protocolo:

1. El diff toca motor de escalada, auth/OTP, pagos/entitlements, notificaciones a contactos o datos personales.
2. El PR es un chain (o parte de uno) o recomienda `size:exception`.
3. Este PR ya recibió dos `REQUEST_CHANGES` (desacuerdo repetido implementador↔revisor → jueces ciegos en vez de un tercer round simple).
4. El usuario la pidió explícitamente para este target.

Si `judgment-day` termina `ESCALATED`, el PR queda bloqueado para el humano: nada de merges autónomos, nada de tercer intento en simple.

## Output Contract

Return: verdict (`APPROVE`/`REQUEST_CHANGES`), tabla de hallazgos con severidades, estado de los criterios de aceptación, y comandos de verificación con resultados.

## References

- [docs/procesos/flujo-desarrollo.md](../../../docs/procesos/flujo-desarrollo.md) — DoD y ronda de fábrica
- Skill `judgment-day` — solo bajo petición explícita del usuario
