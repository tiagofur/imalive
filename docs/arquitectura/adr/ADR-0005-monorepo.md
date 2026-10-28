# ADR-0005: Monorepo (backend + apps + design + docs)

- **Estado**: Aceptado
- **Fecha**: 2026-10-27
- **Decisores**: Equipo de fábrica

## Contexto

El producto se desarrolla principalmente con agentes sobre un único dominio de negocio, con un contrato API compartido (ADR-0003) y tokens de diseño compartidos. El equipo es pequeño y las fases del roadmap cruzan capas (un check-in toca API + apps).

## Decisión

Un solo repositorio:

```
/backend  /apps/android  /apps/ios  /design  /docs  /contracts
```

- CI por rutas: cada pipeline solo construye lo tocado ([procesos/ci-cd.md](../../procesos/ci-cd.md)).
- Las apps y el backend tienen versionado y releases independientes (SemVer por carpeta).

## Consecuencias

- **Positivas**:
  - Un PR puede tocar contrato + backend + apps: cambios atómicos y verificables juntos.
  - La fábrica de agentes tiene un único lugar de verdad (docs + código + contratos).
  - Refactors cruzados (renombrar un campo de la API) son un commit, no una coordinación entre repos.
- **Negativas / costes asumidos**:
  - El repo crece; CI debe filtrar por rutas para no pagar builds de todo en cada PR.
  - Los permisos gruesos por carpeta no son posibles (irrelevante hoy).
- **Mitigaciones**: pipelines por ruta con detección de cambios; tooling por stack aislado (Make en backend, Gradle/Xcode en apps); CODEOWNERS por carpeta si entra más gente.

## Alternativas consideradas

- **Multi-repo (backend/android/ios)**: permisos y CI más simples, pero cambios atómicos entre contrato y clientes se vuelven coordinación entre repos — justo lo que la fábrica de agentes necesita evitar.
