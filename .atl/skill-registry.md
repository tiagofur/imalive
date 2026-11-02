# Registro de skills de la fábrica (P2, issue #28)

> Fuente canónica de qué skill existe, cuándo se dispara y qué entrega.
> El **leader** de una ronda es `issue-to-pr`: toda la fábrica se orquesta
> alrededor del flujo issue → PR → revisión; las demás skills le insumen
> (triage, ux-brief) o le cierran el ciclo (pr-review). Adaptado del ASF
> (presupuesto ~500 líneas; divergencias: sin skills web/Tauri).

## La skill leader

**`issue-to-pr`** (`.agents/skills/issue-to-pr/SKILL.md`) — posee el flujo
end-to-end de una entrega: toma un issue `status:ready`, rama, work-unit
commits, verificación por CHECKPOINTS (docs/procesos/CHECKPOINTS.md) y PR
con metadata completa. Las rondas se narran como su loop; ningún otro
skill sustituye su rol de dueño del entregable.

## El registro

| Skill | Dispara | Entrada | Entrega | Rol en la ronda |
|---|---|---|---|---|
| `triage` | issues `status:nuevo` | issue crudo | issue `status:ready` con DoR (labels, criterios, dependencias) | insume del leader; convierte crudo en especificación |
| `ux-brief` | issues con `ux:shape` | issue + brief del designer | brief UX publicado como comentario | insume del leader y del architect (se absorbe en el plan) |
| `issue-to-pr` (leader) | issues `status:ready` | issue + plan | PR con CI verde y metadata completa | **dueño del flujo**: rama, work units, checkpoints, PR |
| `pr-review` | PRs abiertos | paquete de `factory_handoff.py` | verdict APPROVE/REQUEST_CHANGES con hallazgos P0-P3 | cierra el ciclo; su entrada es el paquete de contexto limpio (P1) |

## Los agentes de rol (`.zcode/agents/`, no son skills pero comparten registro)

| Agente | Produce | Momento |
|---|---|---|
| `triage` | especificación DoR | arranque de ronda |
| `designer` | brief UX (`ux:shape`) | antes del plan |
| `architect` | plan técnico + `delivery_strategy`/`chain_strategy` | issues `size:m`+ |
| `implementer` | rama + work units + PR (sin merge) | el leader lo orquesta |
| `reviewer` | verdict con evidencia | PR abierto; `judgment-day` escala |

## Reglas del registro

1. Una skill nueva entra aquí **y** en `REQUIRED_FILES` de
   `factory_preflight.py` si su archivo es parte del andamiaje obligatorio.
2. El leader no cambia por sesión: cambiarlo es una decisión del fundador
   (el flujo issue→PR es la columna de la fábrica).
3. Las skills viven en `.agents/skills/<nombre>/SKILL.md`; los agentes en
   `.zcode/agents/<rol>.md`. El registro mapea, nunca duplica.
