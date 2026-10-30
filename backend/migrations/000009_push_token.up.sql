-- 000009_push_token.up.sql
-- Push (R-05 salida): el token FCM del dispositivo para los recordatorios
-- diarios. PII mínima operativa: un token revocable por usuario, sin
-- historial, sin ubicación. Re-registrar lo reemplaza (un dispositivo activo
-- por usuario en el MVP).
-- Política de down: herramienta de dev/CI; en producción solo hacia adelante.

ALTER TABLE users ADD COLUMN push_token text NULL;
