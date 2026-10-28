# Guías de API

> Última actualización: 2026-10-27. Contrato: `/contracts/openapi.yaml` (OpenAPI 3.1, ADR-0003). Clientes Kotlin/Swift generados del contrato.

## Principios

1. **Contract-first**: el endpoint nace en el YAML, se revisa en PR, y después se implementa. CI valida handlers y clientes contra el contrato.
2. **Orientado al producto, no a la BD**: los recursos exponen conceptos (`checkins`, `incidents`), no tablas.
3. **Compatible hacia adelante por defecto**: añadir campos es seguro; romper exige versión.

## Convenciones generales

- Base: `https://api.<dominio>/v1`. JSON `camelCase`. Fechas: RFC 3339 en **UTC**.
- IDs: UUID v7 en texto.
- Autenticación: `Authorization: Bearer <JWT>` (15 min) + refresh. Contactos sin app: **tokens firmados de un solo uso** embebidos en el link de alerta (`/a/{token}`) — no tocan la API autenticada.
- Idempotencia: operaciones críticas (`POST /checkins`, `POST /sos`) aceptan `Idempotency-Key`; el webhook de Twilio deduplica por `MessageSid`.
- Paginación: por cursor (`?cursor=&limit=`) con `nextCursor` en la respuesta.
- Rate limiting por teléfono/IP en Redis; respuesta `429` con `Retry-After`.

## Errores — RFC 9457 (`application/problem+json`)

```json
{
  "type": "https://api.tranqui.app/problems/contact-limit-reached",
  "title": "Límite de contactos alcanzado",
  "status": 422,
  "detail": "El plan gratuito permite hasta 1 contacto. Actualiza a Premium.",
  "code": "CONTACT_LIMIT_REACHED",
  "instance": "/v1/contacts"
}
```

- `code` es el código de negocio estable (lo consumen las apps para copy y decisiones).
- Los errores de dominio se declaran una vez y se mapean en un único traductor problema↔error Go.

## Endpoints del MVP (borrador del contrato inicial)

| Método + ruta | Descripción |
|---|---|
| `POST /v1/auth/otp/request` | Envía OTP por WhatsApp/SMS (Twilio Verify) |
| `POST /v1/auth/otp/verify` | Verifica OTP → JWT + refresh |
| `POST /v1/auth/refresh` | Renueva JWT |
| `GET/PATCH /v1/me` | Perfil: nombre, hora check-in, ventana de escalada |
| `GET/POST /v1/contacts` · `DELETE /v1/contacts/{id}` | Círculo (1–5 según plan); alta dispara mensaje de bienvenida |
| `PUT /v1/escalation-policy` | Escalera personalizada (o defaults) |
| `POST /v1/checkins` | Señal de vida (app); soporta `Idempotency-Key` |
| `GET /v1/checkins/today` | Estado del ciclo de hoy (+ racha) |
| `POST /v1/sos` | SOS con countdown cancelable (`DELETE /v1/sos/{id}` cancela) |
| `POST /v1/pause` · `DELETE /v1/pause` | Modo pausa/viaje con auto-resume |
| `GET /v1/incidents` · `GET /v1/incidents/{id}` | Historial y detalle (línea de vida) |
| `POST /v1/devices` | Registro de token push (FCM/APNs) |
| `POST /webhooks/twilio` | Respuestas WhatsApp/SMS (firma verificada, R-06) |
| `POST /webhooks/revenuecat` | Entitlements (firma verificada) |
| `GET /a/{token}` + `POST /a/{token}/action` | **Web sin auth**: ver alerta y responder "Voy yo / No puedo" |

## Versionado y evolución

- Breaking change → nueva ruta `/v2` o campo nuevo con default; los campos viejos se deprecann con `deprecated: true` en el contrato y se eliminan en la versión siguiente.
- CI bloquea diffs de contrato no compatibles sin PR etiquetada `contract-break` (y changelog).
- El changelog del contrato vive en `contracts/CHANGELOG.md`.

## Reglas de revisión de API (checklist de PR)

1. ¿El recurso es un concepto del producto? 2. ¿Errores con `code` de negocio? 3. ¿Idempotente donde importa? 4. ¿Paginación por cursor si es lista? 5. ¿PII mínima en request/response? 6. ¿Contrato actualizado antes del código? 7. ¿Clientes regenerados en la misma PR?
