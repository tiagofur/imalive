# ADR-0004: Motor de escalada sobre PostgreSQL (jobs + outbox)

- **Estado**: Aceptado
- **Fecha**: 2026-10-27
- **Decisores**: Equipo de fábrica

## Contexto

El motor de escalada (ver [motor-escalada.md](../motor-escalada.md)) necesita ejecutar acciones diferidas con fiabilidad: recordatorios, alertas, cierres. Es el componente más crítico: una notificación perdida tiene coste humano real. La carga inicial es pequeña (10 familias en beta, miles después), pero la **corrección y la auditabilidad** importan más que el throughput.

## Decisión

1. **PostgreSQL como cola**: tabla `escalation_jobs` con `FOR UPDATE SKIP LOCKED`; workers Go con tick de 30 s, reintentos con backoff, y estados pending/running/done/failed/cancelled.
2. **Outbox transaccional**: las notificaciones (Twilio/FCM/APNs) se graban como `outbox_event` en la misma transacción que el cambio de estado; un dispatcher las envía con idempotencia.
3. **Log de auditoría append-only** (`audit_events`) de toda transición de incidente.
4. El reloj es una interfaz inyectable (`Clock`) para probar el motor con tiempo simulado.

## Consecuencias

- **Positivas**:
  - Cola y estado en la misma BD → **transacciones reales** entre "cambió el estado del incidente" y "hay que notificar"; imposible el estado inconsistente clásico entre app y broker.
  - Cero infraestructura extra en el MVP (sin Kafka/RabbitMQ que operar) — coherente con el presupuesto.
  - Auditoría natural en la misma BD con backups transaccionales.
- **Negativas / costes asumidos**:
  - Throughput limitado frente a un broker dedicado (irrelevante a esta escala; poll de 30 s).
  - Las notificaciones dependen de que la BD y los workers estén sanos (igual que con un broker, pero con menos partes).
- **Mitigaciones**: SLO de retraso de jobs < 60 s con alerta crítica a los 5 min; reintentos con dead-letter (`failed`) visibles; si el producto escala a decenas de miles de incidentes concurrentes, migrar el transporte a un broker **manteniendo el outbox** (el patrón permite cambiar el medio de envío sin tocar el dominio).

## Alternativas consideradas

- **Redis (BullMQ-like / streams)**: rápido, pero Redis es descartable por diseño aquí (ver overview); no garantiza la transaccionalidad con el estado.
- **Kafka/RabbitMQ**: robustos, pero operar un broker en el MVP añade coste y complejidad sin beneficio a esta escala.
- **cron externo + HTTP**: el corazón del producto no puede depender de un scheduler externo sin estado ni reintentos.
