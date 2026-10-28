---
name: issue-to-pr
description: "Trigger: implementar issue, issue a PR, cerrar issue, trabajar un issue, start work on issue. Pipeline completo de la fábrica imalive: issue ready → rama → work-unit commits → verificación → PR → self-review."
metadata:
  version: "1.0"
---

## Activation Contract

Load this skill when starting work on a factory issue (implementación de un issue `status:ready`), when asked to turn an issue into a PR, or when resuming an in-progress issue.

## Hard Rules

- One issue = one PR. El alcance del PR es exactamente el del issue (más lo que su plan técnico autorice).
- Rama: `feat|fix|chore/NN-slug`, nacida de `main` actualizada (`git pull --ff-only`).
- Cada commit es una work unit: comportamiento + sus tests + sus docs juntos, conventional commit en inglés (skill `work-unit-commits`). El repo queda coherente tras cada commit.
- PR ≤ 400 changed lines. Si se excede: una sola pasada de slicing honesta con la skill `chained-pr`; si aun así no cabe, entregar la mejor partición y recomendar `size:exception`.
- Verifica antes de reportar: registra comando y resultado exacto de la suite completa + lint del stack tocado. Sin verificación no hay PR.
- Los criterios de aceptación del issue son la definition of done del PR, con evidencia por criterio en el cuerpo.
- `Closes #N` en el cuerpo del PR para que el merge cierre el issue.
- Nunca push a `main`; nunca merge; nunca saltarse tests o linters.
- Privacidad (`producto.md` §H, ADR-0004): cero historial de ubicación, geofence solo on-device, mínimos datos, incidentes append-only.

## Decision Gates

| Condición | Acción |
|---|---|
| El issue no pasa DoR (sin criterios, sin size) | Ejecuta la skill `triage` sobre él antes de empezar |
| El issue depende de un humano (credenciales, dinero, producto ambiguo) | Comenta qué falta + recomendación, aplica `status:bloqueado`, termina con el reporte del bloqueo |
| Existe plan del architect en los comentarios | Síguelo; desviaciones justificadas en el PR |
| Diff proyectado > 400 líneas | Slicing con `chained-pr` (stacked o chain con tracker) |
| Verificación por stack | backend: `make test && make lint` · android: `./gradlew :app:lint testDebugUnitTest` · iOS: `xcodebuild test -scheme Tranqui ...` · contracts: lint de OpenAPI. Usa el primero que exista para el código que tocas |
| Aparece trabajo fuera del alcance | Issue nuevo; nota en el PR "follow-up: #M" |
| El trabajo contradice un ADR o AGENTS.md | Bloqueo: exige ADR nuevo + aprobación humana; no lo implementes |

## Execution Steps

1. **Leer**: `gh issue view NN --comments` (incluye plan técnico si lo hay). Confirmar DoR y que el issue está en la fase activa del roadmap.
2. **Etiquetas**: `status:in-progress` al issue.
3. **Rama**: `git switch main && git pull --ff-only && git switch -c feat/NN-slug`.
4. **Plan corto**: 5 líneas máximo (qué, dónde, en qué orden de work units).
5. **Implementar por work units**: cada unidad = commit conventional con su test y su doc. Ejecutar el test enfocado de la unidad antes de cada commit.
6. **Verificar**: suite completa + lint del stack tocado. Registrar comandos y resultados exactos.
7. **Self-review**: leer `git diff main...HEAD` entero; corregir lo que tú mismo señalarías.
8. **PR**: push y abrir con `gh pr create` siguiendo el template de `.github/PULL_REQUEST_TEMPLATE.md` (qué/por qué, cómo probarlo, work units, verificación con resultados, decisiones sin preguntar, criterios con evidencia, rollback, checklist). Título con prefijo de capa: `[backend]` / `[android]` / `[ios]` / `[contracts]` / `[docs]` / `[fábrica]`.
9. **Etiqueta final**: `status:in-review`. CI en verde (arreglar si no).

## Output Contract

Return: PR URL, criterios de aceptación con evidencia, comandos de verificación con resultados exactos, work units (commits) con su propósito, decisiones tomadas sin preguntar, y bloqueos si los hubo.

## References

- [docs/procesos/flujo-desarrollo.md](../../../docs/procesos/flujo-desarrollo.md) — estados, DoR/DoD, ronda de fábrica
- Skill `work-unit-commits` — estructura de commits
- Skill `chained-pr` — PRs grandes
- Skill `go-testing` — patrones de test en Go
- `docs/arquitectura/adr/` — decisiones vinculantes
