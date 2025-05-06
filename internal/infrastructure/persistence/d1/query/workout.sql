-- name: CreateWorkout :one
INSERT INTO workouts (id, device_id, menu_id, performed_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: CreateWorkoutSet :one
INSERT INTO workout_sets (id, workout_id, set_order, weight, reps, interval, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListWorkoutsByDeviceId :many
SELECT * FROM workouts
WHERE device_id = ?
ORDER BY performed_at DESC;

-- name: ListWorkoutSetsByWorkoutId :many
SELECT * FROM workout_sets
WHERE workout_id = ?
ORDER BY set_order ASC;

-- name: GetWorkout :one
SELECT * FROM workouts
WHERE id = ?;
