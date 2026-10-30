-- Racha (R-12): los hechos ya persistidos son suficientes — fechas de ciclo
-- con check-in y la ventana de pausa. La lógica (contar, saltar pausados,
-- cortar en hueco) es una función pura en Go, testeable sin BD.

-- name: ListRecentCheckinDates :many
SELECT cycle_date FROM checkins
WHERE user_id = $1
ORDER BY cycle_date DESC
LIMIT $2;

-- name: GetPauseWindow :one
SELECT pause_since, pause_until FROM users
WHERE id = $1;
