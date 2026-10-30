-- 000010_pause_since.up.sql
-- Racha (R-12): el freeze necesita conocer la ventana de pausa [desde, hasta]
-- para saltar (no romper) los días caídos dentro de ella al caminar hacia
-- atrás. pause_until solo existía; pause_since completa la ventana. Se escribe
-- en SetPause y se limpia en CancelPause (operativa, sin PII nueva).
-- Política de down: herramienta de dev/CI; en producción solo hacia adelante.

ALTER TABLE users ADD COLUMN pause_since timestamptz NULL;
