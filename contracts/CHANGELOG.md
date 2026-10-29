# Contracts changelog

> `info.version` in `openapi.yaml` matches the latest entry here
> (api-guidelines §Versionado). Additive changes only; a break requires a
> `contract-break` PR and a new entry describing it.

## [Unreleased]

- Tooling skeleton: `Problem` (RFC 9457), `bearerAuth`, shared `401`/`429`
  responses; Spectral ruleset with the `no-location-fields` privacy gate;
  `make lint|diff|check`; CI `contracts` job.
