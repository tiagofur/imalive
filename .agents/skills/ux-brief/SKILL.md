---
name: ux-brief
description: "Trigger: ux:shape, brief UX, shape desatendido, brief de diseño para un issue. Contrato de handoff de la fábrica imalive: corre impeccable shape sin interacción y publica el brief como comentario del issue."
metadata:
  version: "1.0"
---

## Activation Contract

Load this skill when an issue carries the `ux:shape` label and needs its UX brief, or when asked for the UX brief of a screen/flow. The design method is the `impeccable` skill (shape); this skill is the factory handoff contract around it. This skill never writes product code.

## Hard Rules

- El método vive en `impeccable`: resuelve su SKILL.md instalada, ejecuta `<skill-base-dir>/scripts/impeccable context` una vez por sesión desde la raíz del repo, y sigue su `reference/shape.md`. Esta skill no duplica ese método.
- Desatendido: sin preguntas al usuario. Todos los supuestos se marcan como supuestos en el brief.
- Cero código: ni implementación, ni comps, ni DESIGN.md. Entregable único = comentario en el issue.
- El brief se publica con `gh issue comment N --body-file`, nunca solo en el chat.
- No toques `PRODUCT.md` ni `producto.md`: el registro de producto de impeccable es un prerrequisito pendiente (colisión case-insensitive). Trabaja con `producto.md`, `docs/diseno/*` y el código existente como contexto.
- Respeta `docs/diseno/ux-movil.md` (Material 3 / HIG, copy en español, prohibiciones) y `docs/diseno/design-system.md`.
- Si la superficie exige fundar un mundo visual nuevo (sin DESIGN.md ni brief vigente, y superficie nueva o que reemplaza identidad): entrega el brief con supuestos igualmente y declara que el mundo visual requiere `new-work` en sesión atendida.

## Decision Gates

| Condición | Acción |
|---|---|
| Hay brief de superficie vigente para esa pantalla/flujo | Extiéndelo; la adición hereda el mundo, sin ejercicio de identidad |
| Superficie nueva dentro de un mundo establecido | Shape completo; el mundo se hereda fijo |
| El issue contradice `docs/diseno/*` o `producto.md` | No diseñes; comentario con la contradicción + `status:bloqueado` |
| Falta contexto para decidir algo material | Supuesto marcado, nunca invención silenciosa |

## Execution Steps

1. Leer: issue + comentarios, `producto.md` relevante, `docs/diseno/ux-movil.md`, `docs/diseno/design-system.md`, brief de superficie vigente si existe, código de la pantalla si existe.
2. `impeccable context` (una vez por sesión; cwd en la raíz del repo).
3. Ejecutar `shape` en modo desatendido: descubrimiento desde el contexto, brief mínimo en sus secciones.
4. Publicar el brief como comentario del issue con encabezado `## Brief UX (ux-brief)` y sección final `### Supuestos` (lista explícita).
5. Responder al orquestador: enlace al comentario + resumen en 3 líneas.

## Output Contract

Return: comment URL, resumen de 3 líneas (modo de visitante, tesis de composición, momento memorable), lista de supuestos, y si se requiere new-work atendido (sí/no con motivo).

## References

- Skill `impeccable` (método de diseño; `reference/shape.md`) — user-level, no vive en este repo
- [docs/diseno/ux-movil.md](../../../docs/diseno/ux-movil.md), [docs/diseno/design-system.md](../../../docs/diseno/design-system.md)
- [docs/procesos/flujo-desarrollo.md](../../../docs/procesos/flujo-desarrollo.md) — el brief corre antes del plan del architect
