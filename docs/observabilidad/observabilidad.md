# Observabilidad

> Última actualización: 2026-10-27. Si el motor de escalada se retrasa y nadie se entera, el producto falla en silencio. La observabilidad no es opcional: es parte del requisito.

## Pilares

| Pilar | Herramienta | Dónde |
|---|---|---|
| Logs estructurados | `log/slog` JSON (backend) | stdout → agregación del hosting |
| Trazas | OpenTelemetry (backend); correlación por request-id | OTLP a proveedor (definir en F0) |
| Métricas | Prometheus (backend) | scraping o push según hosting |
| Crashes móviles | **Sentry** (Android + iOS) | cuentas a crear en F0 |
| Uptime externo | checker 1-min contra `/healthz` y staging | proveedor simple |

## Convenciones de logging (Go)

- `slog` JSON: `time`, `level`, `msg`, `request_id`, `user_id`, `incident_id`, `job_id`, duración.
- Niveles: `debug` (solo local), `info` (eventos de negocio: check-in recibido, incidente cerrado), `warn` (reintentos, webhook inválido), `error` (fallo real con contexto).
- **Sin PII** (ver [seguridad](../seguridad/seguridad-privacidad.md)): teléfonos solo últimos 4, nunca nombres ni contenido de notas.
- Eventos de negocio siempre logueados con el mismo nombre canónico (`checkin.received`, `incident.opened`, `incident.claimed`, `incident.closed`, `notification.sent/failed`) para poder graficar el loop completo.

## Métricas de negocio — los KPIs del spec como métricas

Los [gates del MVP](../producto/vision.md) se exponen como métricas y dashboard desde el día 1:

| Métrica | Cálculo |
|---|---|
| Check-in rate D7 | % de usuarios con ≥ 1 check-in los 7 días tras registro |
| Falsas alarmas | incidentes cerrados por señal tardía / usuarios activos / mes |
| Tasa de invitación | % de nuevas cuentas que crean ≥ 1 contacto en la 1ª sesión |
| Conversión trial→pago | entitlements de pago / trials a 30 días (RevenueCat) |
| Canal de check-in | distribución app/WhatsApp/SMS (valida la apuesta WhatsApp-first) |

## Métricas técnicas y SLOs

| SLO | Objetivo | Alerta |
|---|---|---|
| Disponibilidad API | 99.9% | < 99.5% semanal |
| Latencia API | p99 < 300 ms | p99 > 1 s sostenido 5 min |
| **Retraso del motor** (`now - run_at` de jobs) | p95 < 60 s | **crítico > 5 min** (heartbeat: todo job vencido sin procesar) |
| Jobs `failed` | 0 | crítico al primer failed con incidente abierto |
| Edad del outbox | < 1 min | crítico > 5 min |
| Entrega Twilio (errores / intentos) | < 1% | > 5% en 10 min |
| Webhooks fallidos (firma/parseo) | 0 | warning al 1º, crítico si sostenido |
| Crashes móviles libres | ≥ 99.5% sesiones | caída brusca tras release |

- El **heartbeat del motor** es la alerta más importante del sistema: "hay trabajo vencido sin procesar" detecta worker caído, BD bloqueada o tick muerto. Nada más importa si esto suena.
- Sentry en apps: release tracking (cada build etiquetada), crash-free sessions, breadcrumbs de red (sin PII).

## Dashboards mínimos (staging y prod)

1. **Loop del producto**: check-ins por canal → ciclos abiertos → escaladas → incidentes por estado → cierres. La salud del negocio en una pantalla.
2. **Motor**: jobs pendientes por edad, retraso p95, failed, outbox age, workers activos.
3. **API**: tráfico, latencia, errores por problema (`code` RFC 9457), rate limiting.
4. **Integraciones**: Twilio (entregas/fallos por canal), FCM/APNs, RevenueCat webhooks.

## Alertas → a quién

- F0 (solo fundador): críticos por push/WhatsApp (Twilio ya está integrado — la fábrica avisa por su propio producto cuando sea posible 🙂) y email.
- Nada de alertas sin runbook: cada alerta crítica tiene su acción documentada (link desde el dashboard).
- Guardar silencio: si una alerta dispara dos veces sin acción real, se ajusta el umbral en la misma semana (fatiga de alertas = alerta perdida).
