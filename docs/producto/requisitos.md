# Requisitos

> Fuente primaria: [producto.md](../../../producto.md) §5–§8. Etiquetas de fase: **[MVP]** semanas 1–3 · **[F2]** retención · **[F3]** senior/monetización · **[F4]** viralidad · **[F5]** futuro.
> Última actualización: 2026-10-27

## MVP — "el loop mínimo creible" (3 semanas)

### Semana 1 — Fundamento

| ID | Requisito | Criterio de aceptación |
|---|---|---|
| R-01 | Auth por teléfono (OTP) | Registro/login con OTP por WhatsApp/SMS (Twilio Verify); sesión JWT; sin contraseñas |
| R-02 | Perfil | Nombre, hora de check-in, ventana de escalada configurables |
| R-03 | Contactos (1–5) | Alta con nombre + teléfono; mensaje de bienvenida por WhatsApp/SMS; **el contacto nunca se registra** |
| R-04 | Home = el botón | "Estoy OK" en 1 tap + estado ("Siguiente check-in: hoy 21:00") |

### Semana 2 — El motor (esto ES el producto)

| ID | Requisito | Criterio de aceptación |
|---|---|---|
| R-05 | Recordatorio diario push | Push a la hora elegida (FCM/APNs) |
| R-06 | Check-in por WhatsApp | Responder "ok" al bot marca el check-in; sin abrir la app |
| R-07 | Motor de escalada | Escalera default: suave → +4h urgente → +8h contacto 1 → +12h o 2h sin confirmación → círculo. Estados auditable (ver [motor-escalada](../arquitectura/motor-escalada.md)) |
| R-08 | Alerta accionable | "Ana no ha confirmado desde ayer 22:00. Última señal: hoy 08:12. ¿Puedes comprobarlo?" + botones **Voy yo / No puedo** por link web, sin app |
| R-09 | Ciclo de vida del incidente | Un contacto confirma → se avisa al resto y se cierra; sin duplicar llamadas |
| R-10 | Botón SOS | Alerta inmediata al círculo con countdown cancelable de 10s |
| R-11 | Modo pausa/viaje | Congela la escalera, protege la racha, auto-resume por fecha |

### Semana 3 — Retención mínima + cobro

| ID | Requisito | Criterio de aceptación |
|---|---|---|
| R-12 | Racha | Contador de días OK + freeze en modo viaje |
| R-13 | Paywall | RevenueCat, 14 días de prueba, desde el día 1 (planes Gratis/Premium/Cuidar) |
| R-14 | Onboarding pulido | El momento "añade a tu mamá" es el más cuidado del producto (loop viral) |
| R-15 | Beta cerrada | 10 familias reales; medir y corregir falsas alarmas antes de escalar |

## Fuera del MVP (y por qué)

- Mascota, chat, diario, estado de ánimo → retención extra; medir antes el loop básico (F2/F4).
- Llamadas IA, notas de voz, modo senior → requiere Twilio Voice + diseño de voz (F3).
- Widget de lock-screen → depende de APIs nativas; F4 cuando haya usuarios que lo vean.
- Testamento digital → riesgo legal, no diferenciador (F5).
- Auto check-in por ubicación → fricción de permisos en onboarding; F2.
- B2B/aseguradoras → F5.

## Roadmap de funcionalidades por fase (resumen)

Ver tabla completa en [producto.md §5](../../../producto.md). Resumen de responsables técnicos:

- **F2**: widget check-in (Glance/WidgetKit), geofence on-device, check-in por SMS, línea de vida, grupo familiar, orden de prioridad, zona horaria en viaje, portal del cuidador (inicio).
- **F3**: modo senior completo (UI gigante, config remota), nota de voz + transcripción, llamada IA, señal por reloj, escalada con llamada IA, portal del cuidador, cifrado de notas de voz.
- **F4**: widget social de lock-screen (Locket-style), complicación Apple Watch, mascota, tarjeta compartible, referidos, mensajes walkie-talkie, lifetime de lanzamiento.
- **F5**: pre-señal por actividad, integraciones servicios/país, testamento digital, B2B.

## Requisitos no funcionales

| Área | Requisito |
|---|---|
| Disponibilidad | 99.9% API; el motor de escalada es el componente más crítico (SLO de retraso < 2 min, ver [observabilidad](../observabilidad/observabilidad.md)) |
| Latencia | API p99 < 300 ms; check-in confirmado en UI < 1 s |
| Fiabilidad del motor | At-least-once con idempotencia; toda transición de estado queda en log de auditoría append-only |
| Privacidad | Mínimos datos (nombre, teléfono, contactos); geofence 100% on-device, nunca se guarda ubicación; borrado total de cuenta en 1 tap |
| Legal | Disclaimer visible: "servicio de notificación entre personas, no un servicio de emergencia; no sustituye al 911/112". Sin claims médicos. Logs auditable como defensa "best effort" |
| Idiomas | MVP: español. Preparado para i18n (es/en/pt) desde el día 1 (copy externalizado) |
| Accesibilidad | WCAG 2.2 AA; modo senior: texto ≥ 24 pt, targets ≥ 48 dp / 44 pt, cero navegación |
| Plataformas | Android (Kotlin) y iOS (Swift) nativos; contactos y cuidadores acceden por web sin app |

## Riesgos y mitigaciones (del spec §8)

| Riesgo | Mitigación |
|---|---|
| Falsas alarmas destruyen la confianza | Gracia, confirmación en 2 pasos, escalera larga default, modo viaje, beta cerrada primero |
| Responsabilidad legal | Disclaimer "best effort", no claims de emergencia, T&C claros, logs auditable |
| Aprobación WhatsApp Business API (Meta) | Verificar empresa cuanto antes; sandbox Twilio en beta; SMS fallback |
| Review de Apple (apps de "seguridad") | Claims honestos, evitar "emergencia médica" |
| Churn post-novedad | Racha + coste social del círculo |
| Copia por rivales | Ventaja regional (WhatsApp, español, LATAM) + widget social + voz al hijo; velocidad |
