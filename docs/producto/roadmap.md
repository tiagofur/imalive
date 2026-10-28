# Roadmap

> Última actualización: 2026-10-27. Las fases siguen `producto.md` §5; F0 es añadido de ingeniería (fábrica de software).

## F0 — Fundación de la fábrica (actual)

Objetivo: poder construir el MVP a máxima velocidad con agentes, sin deuda de fundación.

- [x] Spec de producto (`producto.md`).
- [x] Documentación de ingeniería (`docs/`) y ADRs fundacionales.
- [ ] Decisión de nombre + compra de dominio (bloquea branding, no bloquea backend).
- [ ] Crear cuenta Twilio + **iniciar verificación Meta Business** (el trámite más lento — arrancar ya).
- [ ] ADR-0006 (pendiente): hosting/infra (recomendación inicial: contenedores gestionados — Fly.io/Railway/ECS; decidir con datos de coste).
- [ ] ADR-0007 (pendiente): facturación web/lifetime F4 (Stripe vs RevenueCat web).
- [ ] Scaffolding: `/backend` (Go + docker-compose Postgres/Redis + CI), `/apps/android`, `/apps/ios`.
- [ ] CI/CD base por stack ([procesos/ci-cd](../procesos/ci-cd.md)).
- [ ] Opcional (semana 0): smoke test del loop con bot de WhatsApp puro sobre 10 familias para medir check-in rate D7 y falsas alarmas sin app.

## Fase MVP (semanas 1–3)

Alcance exacto: [requisitos.md](requisitos.md) (R-01..R-15). Orden de construcción sugerido:

1. **Semana 1**: auth OTP, perfil, contactos, home-botón.
2. **Semana 2**: recordatorio push, check-in WhatsApp, **motor de escalada** (lo más crítico), alerta web accionable, incidente, SOS, pausa/viaje.
3. **Semana 3**: racha, paywall, onboarding pulido, beta cerrada con 10 familias.

**Gate de salida del MVP** (no se escala sin cumplirlos): D7 ≥ 60% · falsas alarmas < 1% · ≥ 40% invita contacto en 1ª sesión · conversión trial→pago ≥ 5% a 30 días.

## F2 — Retención (semanas 4–6)

- Widget de check-in directo (home/lock screen) — Android Glance + iOS WidgetKit.
- Auto check-in por geofence on-device ("llegué a casa" = OK; **sin** guardar historial de ubicación).
- Check-in por SMS (flip phones).
- Línea de vida (calendario de señales).
- Estado de ánimo (1 tap) + nota de 1 línea.
- Orden de prioridad de escalada, grupo familiar (varios hijos cuidando a un padre).
- Detección de zona horaria en viaje; borrado de cuenta en 1 tap.

## F3 — Modo Senior / monetización grande (mes 2–3)

- UI gigante (un botón, alto contraste, cero navegación) + **configuración remota** por el hijo (P2).
- Nota de voz (10s) + transcripción diaria al círculo (el vínculo emocional que ningún rival tiene).
- Llamada diaria de IA (Twilio Voice + LLM/TTS) para seniors sin smartphone.
- Señal de vida pasiva por reloj (Apple Watch / Wear OS).
- Escalada con llamada IA hasta confirmación ("broadcast hasta confirmación").
- Portal web del cuidador (estado, historial, resúmenes — sin instalar nada).
- Cifrado de notas de voz/transcripciones.

## F4 — Viralidad (mes 2–4)

- Widget familiar de lock-screen estilo Locket (cada usuario = anunciante pasivo ante los que más se preocupan).
- Complicación Apple Watch.
- Mascota/planta virtual (solo modo joven) + tarjeta compartible de racha.
- Mensajes walkie-talkie ("¿alguien despierto?") + check-in conjunto programado.
- Referidos (1 mes premium por amigo activo) + lifetime de lanzamiento ($79, primeros 500).

## F5 — Futuro

Pre-señal por actividad del teléfono (retrasar alertas, menos falsas alarmas) · integraciones con servicios (vecino designado, teleasistencia municipal, ubicación exacta solo durante incidente abierto) · contacto de emergencia profesional por incidente · testamento digital (con asesoría legal) · detección de deterioro por análisis de voz (framing "señal", nunca diagnóstico) · B2B (residencias, aseguradoras).

## Dependencias externas críticas (trámite lento — arrancar en F0)

1. Verificación **Meta Business** para WhatsApp Business API (semanas).
2. Cuentas: Twilio, Google Play Console, Apple Developer, RevenueCat, Sentry.
3. Revisión de App Store para apps de "seguridad" → claims honestos desde la primera submission.
