-- name: CreateWorkout :exec
INSERT INTO workouts (id, device_id, menu_id, performed_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: CreateWorkoutSet :exec
INSERT INTO workout_sets (id, workout_id, exercise_id, weight, reps, rpe, rir)
VALUES (?, ?, ?, ?, ?, ?, ?);
