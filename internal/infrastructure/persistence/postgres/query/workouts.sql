-- name: GetWorkoutByID :one
SELECT * FROM workouts WHERE id = $1 LIMIT 1;

-- name: ListWorkoutsByUserID :many
SELECT * FROM workouts WHERE user_id = $1 ORDER BY performed_at DESC;
