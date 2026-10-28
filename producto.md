Spec de producto v0.1 — App de check-in diario "Estoy vivo / Estoy OK"
Nombre de trabajo: Tranqui (candidatos al final). Documento vivo. Fecha: 2026-10-27

1. Visión en una frase
"Paz mental diaria para quien vive solo y para quien se preocupa por alguien que vive solo": un botón al día confirma que estás bien; si no lo confirmas, tu gente lo sabe de inmediato.

2. Por qué ahora (evidencia de mercado)
Validación de viralidad: la app china de check-in diario para jóvenes que viven solos fue cubierta por BBC News (enero 2026) como fenómeno viral.
Competencia occidental fragmentada o muerta:
imalivetoday.com (clon directo, $12.99/año, escalera 24h/40h/48h) — fuera de línea ahora mismo.
assureokay.com ($4.99/$8.99 mensual) — el más completo: app + SMS + llamada IA + smartwatch + testamento digital. Nada de WhatsApp, sin ubicación, sin widget social.
seniorcheckin.app — llamadas IA a seniors, piloto gratuito solo en Perth (20 plazas). Confirma el modelo "el hijo adulto paga".
feedfluffy.com — dead man's switch web, gratis hasta 2 contactos.
Demografía: ~28% de hogares son unipersonales en EE.UU. (~37M personas), 125M+ en China, tendencia alcista en todo el mundo. En LATAM el ritual "avísame que llegaste" es diario.
Patrón Cal AI: MyFitnessPal adquirió Cal AI (marzo 2026) — las apps simples de un solo propósito con demo compartible se consolidan rápido. Este nicho aún no tiene dueño en Occidente.
Huecos detectados (nadie los cubre):

WhatsApp-first (crítico en LATAM/España/India — todos los rivales usan SMS/email/llamada).
Check-in automático por geolocalización (llegar a casa = check-in hecho).
Widget de lock-screen familiar estilo Locket ("tu familia ve que estás OK").
Check-in por nota de voz con transcripción para la familia (modo senior).
Mercado hispanohablante completo: nadie lo atiende.
3. Personas
Persona	Edad	Rol	Dolor	Qué paga
P1 — El que vive solo	20-35	Usuario viral	Soledad, seguridad, "mi mamá me pregunta a diario si llegué bien"	Quizá (freemium)
P2 — El hijo remoto	35-55	El pagador principal	Vive lejos de su padre/madre mayor, ansiedad diaria, culpa	Sí, seguro ($8-10/mes)
P3 — El senior	65+	Usuario final de modo senior	No domina apps, vive solo, miedo a "pasar algo y que nadie se entere"	No paga él (paga P2)
P4 — Casos secundarios	—	Viajero solo, persona con condición crónica, single parent, pareja LDR	"¿Y si me pasa algo y nadie lo nota?"	Freemium → premium
Insight clave del mercado: en P2 el que paga NO es el usuario final. El valor no es el botón, es la tranquilidad del que se preocupa. El producto vende "deja de revisar el 'visto por última vez'" (sale del círculo. La app china viral probó que P1 también lo adopta solo, por el ángulo emocional de la soledad — contenido enormemente compartible en TikTok).

4. El loop central del producto

text
   [Check-in diario]                [Confirmación pasiva]
   botón / ubicación /     ─────►   widget familiar "OK ✓"
   WhatsApp / voz / reloj           (la familia no pregunta más)
          │
          │ no ocurre
          ▼
   [Recordatorio suave]  ──►  [Recordatorio urgente]
          │ sigue sin ocurrir
          ▼
   [Alerta a contacto 1]  ──►  [Incidente: contacto confirma "voy yo"]
          │ sin confirmación            │
          ▼                             ▼
   [Alerta a todo el círculo]    [Incidente cerrado, todos avisados]
          │
          ▼
   [Racha intacta → salir cuesta]  ← retención
Cada usuario necesita invitar 1-3 contactos para que el producto valga → loop viral incorporado: invitar contactos ES usar el producto.

5. Universo de funcionalidades
Etiquetas: [MVP] semana 1-3 · [F2] fase 2 (retención, semana 4-6) · [F3] fase 3 (senior/monetización grande, mes 2-3) · [F4] fase 4 (viralidad, mes 2-4) · [F5] futuro

A. Check-in (el núcleo)
[MVP] Botón "Estoy OK" — 1 tap en la app.
[MVP] Check-in por WhatsApp — responder "ok" al recordatorio del bot (clave: el check-in NO requiere abrir la app).
[MVP] Recordatorio diario push a la hora elegida.
[F2] Widget de check-in directo en home/lock screen (tap sin abrir app).
[F2] Auto check-in por geolocalización — geofence on-device "llegué a casa" = check-in. Sin historial de ubicación guardado jamás.
[F2] Check-in por SMS (responder al mensaje; funciona en flip phones — lo tiene AssureOkay, validado).
[F3] Check-in por nota de voz — grabar 10s en vez de tocar (modo senior) + transcripción enviada al círculo.
[F3] Llamada diaria de IA (voz) como método de check-in — para seniors sin smartphone (patrón seniorcheckin.app + AssureOkay Plus).
[F3] Señal de vida pasiva por reloj — Apple Watch/Wear OS: pasos/batería como check-in automático (AssureOkay lo tiene).
[F5] Actividad del teléfono como pre-señal — si el teléfono estuvo activo hace poco, retrasar la alerta (reduce falsas alarmas).
B. Escalada e incidentes (el corazón de la seguridad)
[MVP] Escalera configurable (default): hora X recordatorio suave → +4h recordatorio urgente → +8h alerta a contacto 1 → +12h o sin confirmación en 2h → alerta a todo el círculo.
[MVP] Mensaje de alerta accionable: "Ana no ha confirmado desde ayer 22:00. Última señal de vida: hoy 08:12. ¿Puedes comprobarlo?" + botones "Voy yo" / "No puedo" (link web, sin app).
[MVP] Ciclo de vida del incidente: un contacto confirma → se notifica al resto y se cierra. Sin duplicar llamadas.
[MVP] Botón SOS — alerta inmediata al círculo con countdown cancelable de 10s (estilo botón de pánico).
[MVP] Modo pausa/viaje — congela la escalera y protege la racha, con auto-resume por fecha.
[F2] Detección de zona horaria en viaje (no alertar de madrugada por jet lag).
[F3] Escalada con llamada de IA al contacto hasta que alguien confiera ("broadcast hasta confirmación" — AssureOkay lo cobra en Plus, nosotros en plan Cuidar).
[F5] Integración con servicios: opción de alertar a un vecino designado, teleasistencia municipal, o (según país) compartir ubicación exacta solo DURANTE un incidente abierto.
[F5] Contacto de emergencia "profesional" pagado por incidente (estilo vigilancia).
C. Círculo de contactos
[MVP] 1-5 contactos con nombre + teléfono; nunca necesitan la app (reciben WhatsApp/SMS + links web).
[MVP] Mensaje de bienvenida al contacto ("María te añadió como persona de confianza…").
[F2] Orden de prioridad de la escalada.
[F2] Grupo familiar — varios hijos cuidando a un padre: quién confirmó, historial compartido.
[F3] Portal web del cuidador — el hijo ve estado, historial y resúmenes sin instalar nada (patrón seniorcheckin).
[F5] Contactos con app ven el estado "OK hoy ✓" de todo su círculo (bucle pasivo).
D. Retención / gamificación
[MVP] Racha de días OK con contador.
[MVP] Freeze de racha en modo viaje.
[F2] Línea de vida — calendario visual de señales de vida (mes/vista anual).
[F4] Mascota/planta virtual que vive de tus check-ins (patrón Finch — extremo probado en retención; opcional, solo modo joven).
[F4] Tarjeta compartible de racha ("365 días OK 🌱") — imagen para redes.
E. Widget social de lock-screen (el motor viral)
[F4] Widget familiar estilo Locket: tu círculo ve en SU pantalla de bloqueo tu estado "OK ✓" (+ foto opcional). Convierte a cada usuario en anunciante pasivo ante las 1-3 personas que más se preocupan por él.
[F4] Complicación de Apple Watch con el botón.
F. Modo Senior (la monetización grande)
[F3] UI gigante — un botón, alto contraste, textos grandes, cero navegación.
[F3] Configuración remota — el hijo (P2) configura TODO desde su app/portal; el senior solo ve el botón o recibe la llamada.
[F3] Nota de voz → transcripción diaria al hijo ("hoy fui al médico…") — esto convierte el check-in en vínculo emocional diario, no solo seguridad. Ningún rival lo hace.
[F3] Detección de deterioro (F5): análisis de voz/patrones de la llamada IA (habla más lenta, confusión) → alerta temprana al hijo. Cuidado con claims médicos — framing de "señal", no diagnóstico.
G. Capa emocional / anti-soledad (lo que hizo viral a la original)
[F2] Estado de ánimo en 1 tap (emoji) al check-in.
[F2] Nota de 1 línea del día (diario mínimo).
[F4] Mensajes al círculo ("¿alguien despierto?") — chat mínimo tipo walkie-talkie.
[F4] "Cena compartida" / check-in conjunto programado con un contacto (ver una película juntos a distancia, etc.) — espacio a explorar.
H. Privacidad y confianza (crítico en este dominio)
[MVP] Principio: mínimos datos. Teléfono, nombre, contactos. Cero tracking de ubicación continua — geofence solo on-device.
[MVP] Disclaimer honesto: "servicio de notificación entre personas, no un servicio de emergencia médica; no sustituye al 911/112". Protege legalmente y genera confianza.
[F2] Borrado total de cuenta en 1 tap.
[F3] Cifrado de notas de voz/transcripciones.
[F5] Testamento digital — mensajes/notas liberadas a contactos tras 7 días sin respuesta (AssureOkay lo valida en su plan Plus; con asesoría legal).
I. Monetización
Plan	Precio	Qué incluye
Gratis	$0	1 contacto, check-in + WhatsApp, escalada básica, racha, SOS
Premium	$3.99/mes · $29/año	5 contactos, auto check-in por ubicación, widget familiar, historial/línea de vida, pausa viaje, SOS avanzado
Cuidar (senior)	$9.99/mes	Todo + modo senior completo, llamadas/nota de voz IA, transcripciones diarias al hijo, portal del cuidador
Anclas del mercado: AssureOkay $4.99/$8.99/mes · I'm Alive Today $12.99/año.
[F5] B2B: residencias, teleasistencia municipal, aseguradoras (AssureOkay ya factura a organizaciones).
[F4] Lifetime de lanzamiento ($79) para los primeros 500 usuarios — genera caja y embajadores.
J. Crecimiento
Loop natural: invitar contactos = usar el producto (K-factor estructural).
TikTok: el ángulo "vivir solo" / "mi mamá deja de preguntarme" es género enorme; el demo de 15s es perfecto (botón → widget de la familia).
[F4] Referido: 1 mes premium por amigo activo.
SEO local: "app para saber si mi madre mayor está bien" — intención de búsqueda real y desatendida en español.
K. Stack técnico
Capa	Recomendación	Por qué
App	Flutter (o React Native)	Una codebase iOS+Android; los widgets de lock-screen (F4) exigen nativo de todos modos
Bot WhatsApp	Twilio WhatsApp Business API (sandbox para dev)	Canal #1 en LATAM; SMS fallback con Twilio
Push	FCM + APNs	—
Escalada	Cron/queue en el backend (estados: OK → reminder → alerted → incident → closed)	El corazón del producto; debe ser robusto y auditable
Auth	OTP por teléfono (Firebase Auth o Twilio Verify)	Sin contraseñas, los contactos nunca se registran
Pagos	RevenueCat (móvil)	Paywall desde el día 1
Llamadas IA (F3)	Twilio Voice + LLM con voz (ElevenLabs/PlayHT)	~$0.03-0.08/min → margen sano en plan $9.99
Atajo de validación (opcional, semana 0): probar el loop SOLO con un bot de WhatsApp (sin app nativa) con 10 familias → mide check-in rate D7 y falsas alarmas casi gratis antes de escribir la app.

6. El MVP exacto (3 semanas) — "el loop mínimo creíble"
Semana 1 — Fundamento

Auth por teléfono (OTP).
Perfil: nombre, hora de check-in, ventana de escalada.
Añadir contactos → mensaje de bienvenida por WhatsApp/SMS (sin registro para ellos).
Pantalla home = el botón + estado ("Siguiente check-in: hoy 21:00").
Semana 2 — El motor (esto ES el producto) 5. Recordatorio push diario a la hora elegida + check-in respondiendo al WhatsApp. 6. Motor de escalada (cron con estados, auditable): suave → urgente → contacto 1 → círculo. 7. Alerta con última señal de vida + botones "Voy yo / No puedo" por link (sin app). 8. Cierre de incidente notificando al círculo. 9. Botón SOS con countdown cancelable. 10. Modo pausa/viaje.

Semana 3 — Retención mínima + cobro 11. Racha + freeze. 12. Paywall (14 días de prueba, RevenueCat) desde el día 1. 13. Onboarding pulido (el momento "añade a tu mamá" debe ser hermoso: es el loop viral). 14. Beta con 10 familias reales → medir y corregir falsas alarmas ANTES de escalar.

Criterios de salida del MVP (gate):

Check-in rate D7 ≥ 60%
Falsas alarmas < 1% por usuario-activo/mes
≥ 40% de usuarios invita ≥ 1 contacto en la primera sesión
Conversión trial→pagado ≥ 5% a 30 días
7. Qué NO entra al MVP (y por qué)
Mascota virtual, chat, diario, estado de ánimo → retención extra sin la cual el producto no existe; medir primero el loop básico.
Llamadas de IA, notas de voz, modo senior → es la FASE 3 porque es la monetización grande, pero exige integración Twilio Voice + diseño de voz; validar antes el mercado P1/P2 barato.
Widget de lock-screen → depende de API nativa; hacer en F4 cuando haya usuarios que lo vean.
Testamento digital → riesgo legal, validado como feature Plus por rival, no diferenciador.
Auto check-in por ubicación → permisos de ubicación generan fricción en el onboarding; F2.
B2B/aseguradoras → F5.
8. Riesgos y mitigación
Riesgo	Mitigación
Falsas alarmas destruyen la confianza	Períodos de gracia, confirmación en 2 pasos, escalera larga por default, modo viaje, beta cerrada primero
Responsabilidad legal ("no avisaron a tiempo")	Disclaimer "best effort", no claims de servicio de emergencia, T&C claros, logs auditable
Aprobación de WhatsApp Business API (Meta)	Verificar empresa cuanto antes; sandbox Twilio para la beta; SMS como fallback
Review de Apple para apps de "seguridad"	Claims honestos, no promesas médicas; evitar la palabra "emergencia médica"
Churn tras la novedad	Racha + círculo: salir rompe una señal visible para tu familia (coste social real)
AssureOkay copia nuestras features	Ventaja regional (WhatsApp, español, LATAM) + widget social + voz al hijo; velocidad
9. Nombres candidatos
Tranqui — cálido, memorable, español; "¿estás tranqui?" es la esencia del producto.
EstoyOK / OkDiario — descriptivo, bilingüe fácil.
Salvo — corto, emotivo.
StillHere / DailyOK — para versión global EN.
Nota: imalivetoday.com (el clon caído) podría estar en venta — comprar el dominio/nombre daría tráfico heredado.
10. Próximos pasos
Elegir nombre y comprar dominio.
Decidir ruta: (A) app Flutter completa (recomendada, 3 semanas) o (B) smoke test con bot de WhatsApp puro (1 semana).
Crear cuenta Twilio + iniciar verificación Meta Business (el trámite más lento — arrancar ya).
Scaffolding del backend del motor de escalada (lo más crítico técnicamente).
Diseño del onboarding "invita a tu mamá" (lo más crítico de crecimiento).