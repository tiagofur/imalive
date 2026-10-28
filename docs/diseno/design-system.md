# Design System

> Última actualización: 2026-10-27. Producto de trabajo: **Tranqui**. Este documento es el contrato visual compartido entre Android e iOS; los tokens viven en `/design/tokens.json` (única fuente) y se exportan a Compose y SwiftUI.

## Principios de diseño

1. **Calma**: el producto trata de tranquilidad; la UI respira (espacio, poco ruido, una acción protagonista por pantalla).
2. **Confianza**: estados siempre explícitos (nada ambiguo entre "OK" y "no sé"); honestidad sobre límites del servicio.
3. **Calidez**: cálido, humano, español natural ("tranqui"), sin jerga técnica ni tono de alarma salvo cuando toca.
4. **Doble público**: la misma app la usa un joven digital (P1) y un senior de 70+ (P3). El **modo senior** no es una variante de contraste: es una piel con navegación reducida a cero.
5. **Nativo, no genérico**: Material 3 en Android, HIG en iOS; el lenguaje compartido son los tokens y el tono, no componentes clónicos.

## Tokens (fuente única: `/design/tokens.json`)

Generadores: `tokens.json` → (script) → `Theme.kt` (Compose) y `Tokens.swift` (SwiftUI). Nada de colores/tamaños hardcodeados en apps.

### Color

| Token | Uso | Valor inicial (ajustar en F0 con el diseño) |
|---|---|---|
| `ok/primary` | El botón "Estoy OK", estados de éxito, racha | Verde calmado (~#2E7D5B), AA sobre blanco |
| `ok/surface` | Fondos con señal de vida positiva | Verde al 8–12% |
| `alert/soft` | Recordatorio suave | Ámbar (~#E8A13A) |
| `alert/urgent` | Recordatorio urgente | Ámbar profundo/naranja (~#D9730D) |
| `incident/critical` | Alerta al círculo, SOS | Rojo (~#C63B33) — reservado, **nunca** decorativo |
| `neutral/bg`, `neutral/surface`, `neutral/text`, `neutral/text-secondary` | Base | Neutros cálidos (ligeramente sepia, no grises puros) |
| `brand/primary` | Identidad, onboarding, links | A definir en F0 <!-- PENDIENTE: branding tras elegir nombre --> |

- Dark mode obligatorio en ambas apps (el check-in nocturno es un momento real del producto).
- El rojo crítico se usa **solo** en: SOS, alerta de incidente abierto, errores. Su valor semántico se degrada si se usa como acento.

### Tipografía

- Android: **Roboto** (system). iOS: **SF Pro** (system). Respetar **Dynamic Type / fontScale** siempre.
- Escala: display 32/40, headline 24/32, title 18/26, body 16/24, label 13/18.
- Modo senior: mínimo 24 pt en todo texto accionable, pesos más altos, sin abreviaturas.

### Espaciado, forma y elevación

- Rejilla de **4 dp/pt**; escala de espaciado: 4, 8, 12, 16, 24, 32, 48.
- Radios: `sm` 8, `md` 16, `pill` para el botón héroe.
- Elevación mínima: jerarquía por superficie y color, no por sombras; sombras solo en overlays (sheets, menús).
- Targets táctiles: ≥ **48 dp** (Android) / **44 pt** (iOS) en todo elemento interactivo.

## Componentes núcleo (inventario del MVP)

| Componente | Descripción | Estados |
|---|---|---|
| `OkButton` (héroe) | Botón "Estoy OK": grande, pill, tap único, haptic al confirmar | normal, confirmado (con hora), deshabilitado (pausa/viaje) |
| `NextCheckInCard` | "Siguiente check-in: hoy 21:00" + canal activo | on time, atrasado (con gracia), pausado |
| `StreakCard` | Racha de días OK + freeze aplicado | con racha, congelada, perdida (tono amable, nunca culpabiliza) |
| `SosButton` | SOS con countdown cancelable de 10 s, gesto de confirmación | armado (countdown animado), cancelado, enviado |
| `ContactRow` | Contacto del círculo: nombre, teléfono, estado de la última alerta | invitado, activo, respondió ("Voy yo"), no pudo |
| `IncidentBanner` | Incidente abierto: quién, desde cuándo, acción | abierto, reclamado, cerrado |
| `PauseSheet` | Modo pausa/viaje con fecha de auto-resume | configurado, activo |
| `SeniorHome` | Modo senior: un solo botón, fecha/hora grandes, sin navegación | igual que `OkButton` escalado |
| Estados genéricos | Loading (skeleton), empty (con acción sugerida), error (con reintento), offline | — |

En F2–F4 se añaden: `WidgetCheckIn`, `LifeLineCalendar`, `MoodPicker`, `VoiceNoteRecorder`, `FamilyLockScreenWidget`.

## Accesibilidad (WCAG 2.2 AA, obligatorio)

- Contraste AA (4.5:1 texto; 3:1 texto grande/órganos de UI) en claro y oscuro.
- Todo texto via Dynamic Type / `sp`; sin tamaños fijos en `dp` para texto.
- Labels de accesibilidad en todo control; `OkButton` anuncia "Estoy OK, botón, confirma tu estado de hoy".
- Sin dependencia del color para conveyir estado (siempre icono + texto).
- Reduced motion: respetar `prefers-reduced-motion` / `reduce motion`.

## Iconografía e imágenes

- Material Symbols (Android) y SF Symbols (iOS) como base nativa; iconos custom solo si el símbolo del sistema no existe.
- Ilustraciones de onboarding: cálido, plano, representativo del público hispanohablante. <!-- PENDIENTE: set de ilustraciones en F0 -->

## Assets y entregables

```
/design
  tokens.json          # única fuente de tokens
  icons/               # SVG exportados
  illustrations/       # onboarding y vacíos
  mockups/             # pantallas clave (onboarding, home, alerta web)
```

Pantallas a maquetar antes de empezar el MVP (bloquean desarrollo): onboarding completo (incluido el momento "añade a tu mamá"), home, alerta web "Voy yo / No puedo", SOS, paywall.
