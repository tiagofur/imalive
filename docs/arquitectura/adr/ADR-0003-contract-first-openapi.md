# ADR-0003: API REST con OpenAPI contract-first

- **Estado**: Aceptado
- **Fecha**: 2026-10-27
- **Decisores**: Equipo de fábrica

## Contexto

Tres clientes consumen el backend: app Android, app iOS y web ligera de contactos/cuidadores. El desarrollo es simultáneo y por agentes: necesitan un contrato estable para trabajar en paralelo sin integraciones sorpresa.

## Decisión

- La API es **REST + JSON** bajo `/v1`, descrita por un contrato **OpenAPI 3.1** único en `/contracts/openapi.yaml`.
- **Contract-first**: el contrato se edita primero, se revisa en PR, y de él se generan los clientes Kotlin y Swift (`openapi-generator`) y se validan los handlers en CI.
- Errores en formato **RFC 9457 (`application/problem+json`)** con códigos de negocio; convenciones completas en [api/api-guidelines.md](../../api/api-guidelines.md).

## Consecuencias

- **Positivas**: paralelismo real entre plataformas; detección de breaking changes en CI; documentación viva; clientes consistentes.
- **Negativas**: disciplina adicional (todo endpoint nace en el YAML); generación requiere mantenimiento del pipeline.
- **Mitigaciones**: CI valida que handlers Go y contrato no divergen; los cambios incompatibles exigen plan de versión (guidelines).

## Alternativas consideradas

- **gRPC**: mejor para interno; peor para la web ligera y la generación multiplataforma móvil con plugins. REST es suficiente a esta escala.
- **GraphQL**: flexibilidad innecesaria para un producto con flujos muy acotados; más superficie de caché/seguridad que gestionar.
- **Schema-last (generar OpenAPI desde Go)**: el contrato deja de ser la negociación entre plataformas y diverge del cliente.
