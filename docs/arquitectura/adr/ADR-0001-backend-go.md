# ADR-0001: El backend es Go

- **Estado**: Aceptado
- **Fecha**: 2026-10-27
- **Decisores**: Fundador (decisión fija del proyecto)

## Contexto

El producto es, sobre todo, un **motor de escalada fiable** (timers, colas, webhooks, notificaciones) más una API REST simple. El spec (`producto.md` §5.K) no fijaba lenguaje de backend. El fundador establece la restricción: backend siempre en Go.

## Decisión

Todo el backend se desarrolla en **Go** (versión estable más reciente; fijar en `go.mod` en el scaffolding). Incluye API, worker del motor, servidores de páginas web e integraciones. Estándares en [stack/backend-go.md](../../stack/backend-go.md).

## Consecuencias

- **Positivas**:
  - Concurrency de primera clase (goroutines/channels): ideal para workers, tickers y colas del motor.
  - Binario único estático → despliegue simple y barato; el MVP es sensible a coste.
  - Tipado fuerte + compilación rápida → buen encaje con desarrollo autónomo por agentes (errores antes del runtime).
  - Excelente tooling: `go test -race`, profiling, `govulncheck`.
- **Negativas / costes asumidos**:
  - Menos "magia" que frameworks de Node/Rails: más código explícito. Aceptado: la explicitud es un valor para auditoría.
  - Ecosistema web ligero (server-rendered) algo más artesanal que otros lenguajes.
- **Mitigaciones**: estándares estrictos de estructura y errores (stack/backend-go.md); librerías mínimas y estables (pgx, sqlc, chi o stdlib).

## Alternativas consideradas

- **Node/TypeScript**: buen DX para APIs, pero el motor de timers/colas gana mucho con el modelo de concurrencia de Go.
- **Python**: rápido de iterar, menos adecuado para un scheduler de larga vida y despliegues pequeños.
- **Elixir**: la OTP encaja conceptualmente con la escalada, pero el pool de talento/tooling del proyecto y la decisión del fundador hacen a Go la elección.

> Esta decisión desaproba cualquier recomendación de stack de `producto.md` §5.K que contradiga este ADR o [ADR-0002](ADR-0002-apps-moviles-nativas.md).
