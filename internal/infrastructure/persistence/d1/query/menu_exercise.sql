-- name: DeleteMenuExercisesByMenuID :exec
DELETE FROM menu_exercises
WHERE menu_id = ?;

-- name: CreateMenuExercise :exec
INSERT INTO menu_exercises (menu_id, exercise_id, position)
VALUES (?, ?, ?);

-- name: ListMenuExercisesByMenuID :many
SELECT
    me.exercise_id,
    e.name AS exercise_name,
    me.position
FROM
    menu_exercises me
JOIN
    exercises e ON me.exercise_id = e.id
WHERE
    me.menu_id = ?
ORDER BY
    me.position ASC;
