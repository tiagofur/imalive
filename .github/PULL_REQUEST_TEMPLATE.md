## Issue

Closes #

## Qué y por qué

<!-- 2-4 frases: qué cambia y por qué es lo que el issue pide -->

## Cómo probarlo

1. ...

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
- [ ] Tests incluidos con el comportamiento que verifican
- [ ] Docs actualizadas en la misma PR que el cambio
- [ ] ≤ 400 changed lines (o chain de `chained-pr` / `size:exception` justificado)
- [ ] Alcance = solo este issue (follow-ups en issues nuevos)
- [ ] Privacidad: cero historial de ubicación, mínimos datos, incidentes append-only
- [ ] Sin cambios rompientes en el contrato OpenAPI (ADR-0003)
- [ ] Self-review del diff completo hecha
