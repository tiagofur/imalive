---
name: triage
description: "Trigger: triage, clasificar issues, preparar backlog, procesar status:nuevo. Convierte issues crudos en especificaciones listas para implementar (DoR) sin escribir código."
metadata:
  version: "1.0"
---

## Activation Contract

Load this skill when processing issues with `status:nuevo`, when preparing the backlog, or when an issue fails DoR during issue-to-pr.

## Hard Rules

- `producto.md` y los ADRs son la fuente de verdad. Un issue que los contradice es un bloqueo de producto (pregunta + `needs-human`), nunca una decisión del triage.
- Todo issue que salga de triage tiene: `type:`, `priority:`, `size:` y ≥ 1 criterio de aceptación verificable.
- "Verificable" = se puede comprobar con un comando, una acción de usuario concreta o un estado observable. Rechaza criterios vagos ("funciona bien", "sea usable").
- Deduplica: antes de preparar, busca issues abiertos con el mismo objetivo y propón el cierre del duplicado.
- Respeta la fase activa del roadmap ([docs/producto/roadmap.md](../../../docs/producto/roadmap.md)): lo de fases futuras no se marca `status:ready`.
- El triage no escribe código ni modifica ramas.

## Decision Gates

| Condición | Acción |
|---|---|
| Issue claro y completo tras editarlo | `status:ready` |
| Falta decisión humana (dinero, credenciales, cuentas, producto, marca) | Comentario con pregunta concreta + recomendación; `status:bloqueado` + `needs-human` |
| Duplicado de otro issue | Comentar y cerrar citando el que queda |
| Feature `size:m+` sin plan técnico | `status:ready` + nota "necesita plan del architect" |
| Bug reportado sin pasos | Comentar pidiendo los pasos mínimos; `status:bloqueado` |
| Issue de una fase futura del roadmap | Queda `status:nuevo` con comentario de por qué espera |

## Execution Steps

1. `gh issue list --label status:nuevo --state open`.
2. Por issue: leerlo completo, buscar duplicados, clasificar `type:`.
3. Editar el cuerpo: problema/valor, alcance, no-objetivos, criterios de aceptación numerados y verificables.
4. Asignar `priority:` (p0 bloquea el loop del producto · p1 esta semana · p2 cuando quepa) y `size:` (xs <1h · s <4h · m <2d · l >2d).
5. Aplicar el estado según decision gates (quitar `status:nuevo` cuando corresponda).
6. Resumen final: listos, bloqueados (con qué falta exactamente), duplicados cerrados.

## Output Contract

Return: por cada issue procesado — decisión (ready/bloqueado/cerrado/en espera), criterios de aceptación escritos, type/priority/size asignados, y la lista de bloqueos humanos pendientes.

## References

- [docs/procesos/flujo-desarrollo.md](../../../docs/procesos/flujo-desarrollo.md) — estados, labels y DoR
- `producto.md` y `docs/producto/roadmap.md` — qué se construye y cuándo
