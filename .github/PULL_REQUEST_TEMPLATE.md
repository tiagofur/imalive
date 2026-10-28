## Issue

Closes #

## Qué y por qué

<!-- 2-4 frases: qué cambia y por qué es lo que el issue pide -->

## Cómo probarlo

1. ...

## Evidencia visual (solo PRs con `ux:shape`; borrar la sección en los demás)

<!-- Capturas del emulador/simulador por clase de dispositivo enviada (mínimo un teléfono Android y uno iOS si el PR toca ambas apps). Android: `adb exec-out screencap -p > evidencia/<pantalla>-<clase>.png`; iOS: captura del simulador con `xcrun simctl io booted screenshot`. Nombra qué archivo prueba qué, y qué brief UX (comentario del issue) se está honrando. Indica si la evidencia viene de emulador o hardware real. -->

- `evidencia/...` — ...
- Brief UX honrado: <enlace al comentario del issue>

## Commits (work units)

- `<sha corto> feat(...): ...` — propósito de la unidad

## Verificación

- Suite completa: `<comando>` → `<resultado exacto>`
- Lint: `<comando>` → `<resultado exacto>`
- Test enfocado por unidad: registrado en cada commit

## Criterios de aceptación

<!-- Uno por criterio del issue, con la evidencia de que se cumple -->

- [ ] Criterio 1 — evidencia: ...

## Decisiones tomadas sin preguntar

<!-- Defaults del proceso aplicados donde el issue no especificaba; una línea por decisión con su porqué -->

## Rollback

<!-- Archivos/comportamiento que se pueden revertir sin tocar trabajo ajeno -->

## Checklist

- [ ] CI verde en los jobs del stack tocado
- [ ] Evidencia visual adjunta (solo PRs con `ux:shape`; N/A explícito en los demás)
- [ ] Tests incluidos con el comportamiento que verifican
- [ ] Docs actualizadas en la misma PR que el cambio
- [ ] ≤ 400 changed lines (o chain de `chained-pr` / `size:exception` justificado)
- [ ] Alcance = solo este issue (follow-ups en issues nuevos)
- [ ] Privacidad: cero historial de ubicación, mínimos datos, incidentes append-only
- [ ] Sin cambios rompientes en el contrato OpenAPI (ADR-0003)
- [ ] Self-review del diff completo hecha
