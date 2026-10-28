# Seguridad y privacidad

> Última actualización: 2026-10-27. La confianza ES el producto: si la gente duda de Tranqui, no la usa. Este documento es vinculante para todo PR (ver regla P0 en [flujo-desarrollo](../procesos/flujo-desarrollo.md)).

## Principios no negociables

1. **Mínimos datos**: nombre, teléfono, contactos del círculo. Nada más. Cada campo nuevo de PII se justifica en la PR.
2. **Cero historial de ubicación**: el geofence (F2) vive 100% on-device; al entrar, la app envía un check-in normal. Nunca se persiste ni transmite posición.
3. **No somos emergencias**: disclaimer visible ("servicio de notificación entre personas; no sustituye al 911/112"), sin claims médicos ni promesas de vigilancia. Protege legalmente y genera confianza.
4. **Auditabilidad**: los incidentes quedan en log append-only (evidencia de la promesa "best effort", riesgo legal del spec §8).
5. **Borrado real**: eliminar la cuenta borra PII de prod en < 30 días (incluidos backups por expiración) y desactiva inmediatamente el motor.

## Autenticación y sesiones

- **OTP por teléfono** (Twilio Verify) como única identidad: sin contraseñas. OTP de 6 dígitos, expiración 10 min, un intento por código, rate limit por teléfono e IP (Redis).
- **JWT de acceso 15 min** + refresh rotativo (detección de reuso → revocación de la familia de tokens).
- **Contactos sin cuenta**: tokens firmados (HMAC), de un solo uso, expiración corta, ligados al incidente. Reusado o expirado → página de "enlace caducado" + opción de pedir nuevo al usuario.
- Webhooks (Twilio, RevenueCat): verificación de **firma obligatoria** antes de parsear nada + dedupe por id externo.

## Autorización

- El usuario accede solo a sus recursos; los contactos solo a la página del incidente para el que tienen token.
- El círculo se consulta por `user_id` en cada query (sin IDs predecibles en rutas).
- Configuración remota del modo senior (F3): solo el rol "cuidador-configurador" autorizado por el usuario senior puede cambiar su política.

## Datos personales (inventario del MVP)

| Dato | Dónde | Retención |
|---|---|---|
| Teléfono, nombre, zona horaria, hora check-in | Postgres (prod) | vida de la cuenta |
| Contactos (nombre + teléfono) | Postgres | vida de la cuenta |
| Check-ins (origen, timestamp, mood/nota opcional) | Postgres | vida de la cuenta; agregados anónimos para métricas |
| Incidentes + `audit_events` | Postgres, append-only | retención legal-defensiva (definir plazo con asesoría; por defecto 2 años) |
| Tokens de push, plan (entitlement) | Postgres | vida de la cuenta |
| Notas de voz/transcripciones (F3) | Object storage **cifradas** + claves en KMS | configurable; borrado con la cuenta |

- Cifrado: TLS 1.2+ en tránsito; cifrado en reposo del proveedor de BD/storage + campo sensible (F3) con cifrado aplicativo.
- **Nunca** en logs: teléfonos completos (últimos 4), nombres, contenido de notas. El request-id correlaciona sin exponer PII.

## Duridades por superficie (OWASP)

- **Backend**: ASVS nivel 2 como guía de revisión; input validation en el dominio; prepared statements (sqlc); errores RFC 9457 sin stacktraces; rate limit global y por endpoint sensible (auth, SOS, webhooks).
- **Móvil (MASVS)**: sin secretos en el binario; JWT en Keychain (iOS) / EncryptedSharedPreferences-Keystore (Android); certificate pinning decidido en F2 (mientras: TLS + detección de entorno comprometido básica); sin PII en notificaciones (solo "Tienes un recordatorio", el detalle tras unlock o en la web).
- **Dependencias**: `govulncheck` en CI (Go), Dependabot/Renovate en todo el repo; versiones fijadas ([stack.md](../stack/stack.md)).

## Secretos

- Nunca en el repo (`.gitignore` desde F0); en local `.env` gitignoreado; en CI GitHub secrets; en producción el gestor del proveedor.
- Rotación: Twilio/APNs/JWT signing key rotables con dual-key window; procedimiento documentado en F0.
- Scopes mínimos: el worker envía notificaciones pero no lee PII más allá de lo necesario por evento (el payload del outbox se construye en dominio).

## Respuesta a incidentes (borrador F0)

1. Detectar (alertas de [observabilidad](../observabilidad/observabilidad.md)) → 2. Contener (rotar credenciales, apagar canal afectado, pausar worker si es el motor) → 3. Comunicar al fundador → 4. Erradicar y restaurar → 5. Post-mortem sin culpables con acción preventiva → 6. Notificación regulatoria si aplica (GDPR 72 h / ley local).

## Cumplimiento

- GDPR (España/EU) y leyes LATAM relevantes por país de lanzamiento; DPA con Twilio/RevenueCat/hosting; política de privacidad y T&C con el disclaimer honesto antes del primer check-in.
- Eliminación de cuenta en 1 tap (F2 del spec, pero el endpoint `DELETE /me` existe desde el MVP).
- Cualquier feature que amplíe datos (ubicación, voz, análisis de deterioro F5) exige **revisión de esta doc + posible ADR + asesoría legal** antes de implementarse.
