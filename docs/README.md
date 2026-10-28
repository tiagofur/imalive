# Documentación de ingeniería — Tranqui

Fuente de verdad operativa del proyecto. `producto.md` define **qué** construimos; `docs/` define **cómo** lo construimos.

## Índice

| Documento | Responde a | Cuándo leerlo |
|---|---|---|
| [producto/vision.md](producto/vision.md) | ¿Qué es el producto y para quién? | Siempre, al empezar cualquier trabajo |
| [producto/requisitos.md](producto/requisitos.md) | ¿Qué entra en el MVP y qué no? | Al planificar una fase |
| [producto/roadmap.md](producto/roadmap.md) | ¿En qué fase estamos y qué sigue? | Cada sesión |
| [diseno/design-system.md](diseno/design-system.md) | ¿Cómo se ve y se siente el producto? | Al construir UI |
| [diseno/ux-movil.md](diseno/ux-movil.md) | ¿Cómo se comportan las pantallas y flujos? | Al construir UI |
| [arquitectura/overview.md](arquitectura/overview.md) | ¿Cómo encaja el sistema? | Al tocar el backend o los contratos |
| [arquitectura/motor-escalada.md](arquitectura/motor-escalada.md) | ¿Cómo funciona el corazón del producto? | Antes de tocar check-in/escalada/notificaciones |
| [arquitectura/adr/](arquitectura/adr/) | ¿Qué decisiones se tomaron y por qué? | Antes de contradecir cualquiera |
| [stack/stack.md](stack/stack.md) | ¿Qué tecnologías usamos? | Al añadir cualquier dependencia |
| [stack/backend-go.md](stack/backend-go.md) | ¿Cómo se escribe Go aquí? | Al programar backend |
| [stack/android.md](stack/android.md) | ¿Cómo se escribe Android aquí? | Al programar Android |
| [stack/ios.md](stack/ios.md) | ¿Cómo se escribe iOS aquí? | Al programar iOS |
| [api/api-guidelines.md](api/api-guidelines.md) | ¿Cómo se diseña y evoluciona la API? | Al tocar el contrato OpenAPI |
| [calidad/estrategia-testing.md](calidad/estrategia-testing.md) | ¿Qué y cómo probamos? | Al escribir tests |
| [procesos/flujo-desarrollo.md](procesos/flujo-desarrollo.md) | ¿Cómo trabajamos (git, PRs, DoD)? | Al abrir ramas y PRs |
| [procesos/ci-cd.md](procesos/ci-cd.md) | ¿Cómo se construye, despliega y libera? | Al tocar CI/CD o versiones |
| [seguridad/seguridad-privacidad.md](seguridad/seguridad-privacidad.md) | ¿Qué riesgos protejo y cómo? | Al tocar datos, auth o integraciones |
| [observabilidad/observabilidad.md](observabilidad/observabilidad.md) | ¿Cómo sabemos que todo va bien? | Al añadir features o servicios |
| [agentes/fabrica-software.md](agentes/fabrica-software.md) | ¿Cómo trabaja un agente aquí? | Al empezar cada sesión |

## Convenciones de la documentación

- **Documento vivo**: toda página lleva fecha de última actualización y se edita en la misma PR que introduce el cambio.
- **Decisiones → ADR**: cualquier elección técnica no trivial se registra como ADR en `arquitectura/adr/` (plantilla incluida). Los ADRs no se editan una vez aceptados; se marcan como `Superseded` y se crea uno nuevo.
- **Placeholders marcados**: lo pendiente de decidir se escribe como `<!-- PENDIENTE: ... -->`, nunca se inventa.
- Idioma: español. Identificadores de código: inglés.
