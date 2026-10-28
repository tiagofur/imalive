# UX móvil — guías de comportamiento

> Última actualización: 2026-10-27. Complemento de [design-system.md](design-system.md): el "cómo se comporta" de las apps Android e iOS.

## Regla de oro de plataformas

**Nativo de verdad.** Material 3 en Android (navegación, componentes, gestos), Human Interface Guidelines en iOS. Lo compartido es el diseño de producto (flujos, copy, tokens), no la piel. Prohibido emular el look del otro sistema.

## Flujos críticos del MVP

### 1. Onboarding — "el momento añade a tu mamá" (R-14, loop viral)

1. Bienvenida con la promesa emocional (no features): "Un toque al día para que los que te quieren sepan que estás bien".
2. OTP por teléfono (autofill de código en ambas plataformas).
3. Perfil mínimo: nombre, hora de check-in sugerida (21:00 por defecto, editable).
4. **El momento clave**: añadir el primer contacto con preview del mensaje que recibirá ("María te añadió como persona de confianza…") y envío por WhatsApp con un tap. Debe ser hermoso y fricción-cero: es el loop viral.
5. Al home, con la racha en 0 y el primer check-in cerca.

- Máximo 5 pasos; progreso visible; ninguna pregunta que no sea imprescindible.
- Permisos de notificación se piden con pre-permiso explicativo (just-in-time), nunca en frío al abrir.

### 2. Check-in diario (debe durar < 3 s)

- Push a la hora elegida → tap → botón → OK + haptic + confirmación de racha.
- **Sin app**: responder "ok" al WhatsApp del bot (R-06) marca el check-in igual; la UI debe reflejarlo ("Hoy confirmaste por WhatsApp a las 21:04").
- Estados del día: pendiente → confirmado → en gracia (recordatorio suave) → urgente. Siempre explícito en home.

### 3. Alerta al contacto (web, sin app — R-08)

Link en WhatsApp/SMS → página web móvil ligera (server-rendered por el backend Go, ver [arquitectura/overview](../arquitectura/overview.md)):

- Mensaje accionable con contexto: quién, última señal de vida, cuánto hace.
- Dos botones grandes: **"Voy yo"** y **"No puedo"**. Un tap = confirmación; segundo paso ligero de confirmación (evita taps accidentales, R-riesgo falsas alarmas).
- Al confirmar: pantalla de calma ("Gracias — avisamos al resto del círculo") y el incidente se cierra notificando a todos (R-09). Sin app, sin registro, tokens firmados de un solo uso.

### 4. SOS (R-10)

- Botón visible pero protegido: tap → countdown de 10 s con cancelación gigante → envío inmediato al círculo.
- Haptic fuerte; nunca autoenvío sin countdown.

### 5. Modo pausa/viaje (R-11)

- Accesible desde home en 2 taps; pide fecha de vuelta (auto-resume); explica que protege la racha y congela alertas. Estados claros mientras está activo (banner permanente).

### 6. Modo Senior (F3, se diseña ya para no romper después)

- Una sola pantalla: fecha/hora grande + el botón. Cero navegación, cero menús.
- Alternativas de check-in visibles: botón, nota de voz (F3), o simplemente atender la llamada de IA (F3).
- Todo se configura **remotamente** por el hijo (P2): el senior jamás entra a settings.

## Notificaciones y permisos

- Canales Android separados: recordatorios (default), urgentes/incidentes (alta, sin silenciar), marketing (off). En iOS, Time Sensitive para la escalada urgente.
- Nada de notificaciones decorativas; cada notificación tiene una acción.
- Recordatorios suaves con tono cálido ("¿Tranqui? Tu círculo espera tu señal de hoy") — el urgente cambia de tono y de canal.

## Offline y errores

- El check-in debe tolerar red intermitente: envío con reintento en background; la UI confirma cuando el backend responde (o optimistic + reconciliación).
- Sin conexión: home muestra el último estado real y "se enviará cuando haya red"; nunca estados falsos.
- Errores siempre con acción (reintentar / cambiar canal), tono humano.

## Háptica y movimiento

- Check-in: háptica suave de confirmación; SOS: háptica de alerta.
- Animaciones cortas (< 300 ms), con significado (confirmación, countdown); respetan reduced-motion. Nada de animaciones de espera largas en el flujo principal.

## Copy y localización

- Tono: familiar, español natural, "tú". Tranquilizador sin infantilizar. Ejemplos: "Estoy OK", "Tu racha sigue viva 🌱", "No pudieron comprobar. ¿Lo intentas tú?".
- Prohibido: "emergencia médica", promesas de seguridad, tecnicismos ("sincronizar", "payload").
- Todo el copy externalizado desde el día 1 (i18n es/en/pt) — MVP solo español.

## Listas de verificación de UI por plataforma

**Android**: navegación con Compose M3 (Navigation Bar/Barra de gestos), ripple states, edge-to-edge, back predictivo, FCM tokens gestionados, widgets con Glance (F2+).
**iOS**: NavigationStack/sheet con HIG, safe areas, Dynamic Type en todo, APNs con UNUserNotificationCenter, widgets con WidgetKit + lock-screen (F2/F4), App Intents (Siri, F5+).
