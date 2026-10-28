# Tranqui (nombre de trabajo) — check-in diario "Estoy vivo / Estoy OK"

Un botón al día confirma que estás bien; si no lo confirmas, tu gente lo sabe de inmediato.

## Puntos de entrada

| Documento | Qué es |
|---|---|
| [producto.md](producto.md) | Spec de producto v0.1 — visión, mercado, funcionalidades, MVP |
| [docs/](docs/README.md) | Documentación de ingeniería y diseño (fuente de verdad operativa) |
| [AGENTS.md](AGENTS.md) | Reglas para agentes de la fábrica de software |

## Estructura del repositorio (monorepo, ver ADR-0005)

```
/backend        API + motor de escalada (Go)
/apps/android   App nativa (Kotlin + Jetpack Compose)
/apps/ios       App nativa (Swift + SwiftUI)
/design         Tokens de diseño y assets compartidos
/docs           Documentación de ingeniería
/producto.md    Spec de producto (documento vivo)
```

> Nota: `producto.md` §5.K recomendaba Flutter/RN. Esa recomendación quedó **desaprobada** por
> decisión del fundador: apps nativas + backend Go. Ver [ADR-0002](docs/arquitectura/adr/ADR-0002-apps-moviles-nativas.md).

## Estado

- **Fase actual:** F0 — fundación de la fábrica de software (docs, ADRs, scaffolding pendiente).
- Roadmap: [docs/producto/roadmap.md](docs/producto/roadmap.md)
