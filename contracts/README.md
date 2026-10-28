# Contracts — Tranqui OpenAPI (contract-first)

Single contract `openapi.yaml` (OpenAPI 3.1, ADR-0003). Every endpoint is
described here first, reviewed in PR, and only then implemented; the
Kotlin/Swift clients are generated from it.

## Layout

| File | Purpose |
|---|---|
| `openapi.yaml` | The contract (single file per ADR-0003) |
| `.spectral.yaml` | Lint ruleset: `spectral:oas` + factory rules |
| `Makefile` | `lint` · `diff` · `check` (tooling pinned, installed on demand) |
| `CHANGELOG.md` | Contract changelog (mandatory per api-guidelines) |

## Commands

```bash
make lint   # Spectral lint (pinned @stoplight/spectral-cli)
make diff   # oasdiff breaking-change check vs BASE (default origin/main)
make check  # lint + diff
```

`make diff` fails on breaking changes. Escape hatch for an intentional
break: label the PR `contract-break` (api-guidelines) and add a
`CHANGELOG.md` entry. The check is skipped when the base has no spec yet
(first contracts PR).

## Lint rules (beyond `spectral:oas`)

- `operation-description`: every operation documents what it does.
- `error-responses-use-problem`: 4xx/5xx responses use RFC 9457 `Problem`.
- `no-location-fields`: **privacy gate** (producto.md §H) — no property may
  look like a location field; violations fail CI.

## References

- [ADR-0003](../docs/arquitectura/adr/ADR-0003-contract-first-openapi.md)
- [api-guidelines](../docs/api/api-guidelines.md)
