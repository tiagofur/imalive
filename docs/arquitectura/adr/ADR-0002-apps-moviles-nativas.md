# ADR-0002: Apps móviles nativas (Kotlin/Compose y Swift/SwiftUI)

- **Estado**: Aceptado
- **Fecha**: 2026-10-27
- **Decisores**: Fundador (decisión fija del proyecto)

## Contexto

`producto.md` §5.K recomendaba **Flutter o React Native** para tener una codebase iOS+Android en 3 semanas. El fundador fija la restricción contraria: **apps Android e iOS nativas**. La clave del conflicto: el producto depende de superficies profundamente nativas (widgets de home/lock-screen F2/F4, notificaciones sensibles, geofencing F2, háptica, futuras complicaciones de reloj), donde los frameworks multiplataforma terminan exigiendo código nativo de todos modos.

## Decisión

- **Android**: Kotlin + Jetpack Compose + Material 3. Estándares en [stack/android.md](../../stack/android.md).
- **iOS**: Swift + SwiftUI + Observation. Estándares en [stack/ios.md](../../stack/ios.md).
- **Prohibido** introducir Flutter, React Native, KMP, .NET MAUI u otro framework multiplataforma sin un ADR nuevo que supersede este.
- La duplicidad de UI se mitiga con: contrato OpenAPI único con clientes generados (ADR-0003), tokens de diseño compartidos (`/design/tokens.json`), y specs de flujo únicos ([diseno/ux-movil.md](../../diseno/ux-movil.md)).

## Consecuencias

- **Positivas**:
  - Widgets (Glance/WidgetKit), lock-screen widgets (F4), APNs/FCM con API completas y sin puentes.
  - Mejor rendimiento, accesibilidad (Dynamic Type, TalkBack/VoiceOver) y "feel" nativo — crítico para el modo senior.
  - Beta/producción sin sorpresas de revisión de tiendas por runtimes embebidos.
- **Negativas / costes asumidos**:
  - Dos UIs que construir y mantener (~2x en capa de presentación).
  - Menos velocidad inicial que una codebase única.
- **Mitigaciones**: lógica de negocio 100% en backend (las apps son clientes finos); diseño compartido por tokens; agentes especializados por plataforma trabajando en paralelo sobre specs únicos.

## Alternativas consideradas

- **Flutter** (recomendación original del spec): una sola codebase, pero lock-screen widgets/háptica/geofence exigen nativo; se pierde calidad nativa y se añade un runtime.
- **React Native**: idem, más el coste del puente y una superficie de notificaciones/widgets más frágil.
- **KMP compartiendo lógica**: mantiene UI nativa pero complica el build por poco beneficio; la lógica ya vive en el backend.

> Nota para mantener: marcar la recomendación Flutter de `producto.md` §5.K como desaprobada en la próxima revisión del spec.
