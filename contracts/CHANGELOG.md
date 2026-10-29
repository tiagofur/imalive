# Contracts changelog

> `info.version` in `openapi.yaml` matches the latest entry here
> (api-guidelines §Versionado). Additive changes only; a break requires a
> `contract-break` PR and a new entry describing it.

## [Unreleased]

## [0.1.0] — 2026-10-28 (issue #2, week 1)

- Auth: `POST /v1/auth/otp/request` (204, `429` + `Retry-After`),
  `POST /v1/auth/otp/verify` (tokens without PII), `POST /v1/auth/refresh`
  (pair rotation). Business codes: `PHONE_INVALID`, `OTP_INVALID`,
  `OTP_EXPIRED`.
- Profile: `GET`/`PATCH /v1/me` (`name`, `checkinTime` HH:MM, IANA
  `timezone`, `escalationWindowMinutes` default 60).
- Contacts: `GET`/`POST /v1/contacts` (no pagination, ≤5),
  `DELETE /v1/contacts/{id}`. Codes: `CONTACT_LIMIT_REACHED` (422),
  `CONTACT_NOT_FOUND` (404).
- Check-ins: `POST /v1/checkins` (idempotent: 201 first, 200 replay,
  `Idempotency-Key`), `GET /v1/checkins/today` (`pending`/`confirmed`/
  `paused`; no streak yet — R-12 week 3, additive later).
- Errors: RFC 9457 `Problem` with stable business `code`; security:
  global `bearerAuth`, auth endpoints public.
- Tooling: Spectral 6.17.0 ruleset (incl. `no-location-fields` privacy
  gate) + oasdiff v1.35.0 breaking-change check; CI `contracts` job.
