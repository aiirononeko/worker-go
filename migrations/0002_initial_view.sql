-- views.sql

-- View 1: Add workout context (timestamp, device_id) to workout sets
CREATE VIEW IF NOT EXISTS vw_workout_set_details AS
SELECT
    ws.id AS set_id,
    ws.workout_id,
    ws.set_order,
    ws.weight,
    ws.reps,
    ws.volume,      -- The generated volume column
    ws.interval,
    ws.created_at AS set_created_at,
    ws.updated_at AS set_updated_at,
    w.performed_at,
    w.device_id     -- Keep device_id for potential filtering/aggregation
FROM
    workout_sets ws
JOIN
    workouts w ON ws.workout_id = w.id;

-- View 2: Join set details with exercises via menus
CREATE VIEW IF NOT EXISTS vw_exercise_volumes AS
SELECT
    vwsd.set_id,
    vwsd.workout_id,
    vwsd.performed_at,
    vwsd.device_id,
    vwsd.set_order,
    vwsd.volume,
    ex.id AS exercise_id,
    ex.name AS exercise_name
    -- Add other columns from exercises or menus if needed for dashboard display
FROM
    vw_workout_set_details vwsd
JOIN
    workouts w ON vwsd.workout_id = w.id -- Need workouts again to get menu_id
JOIN
    menus m ON w.menu_id = m.id
JOIN
    menu_exercises me ON m.id = me.menu_id
JOIN
    exercises ex ON me.exercise_id = ex.id;

-- View 3: Join exercise volumes with muscle groups
CREATE VIEW IF NOT EXISTS vw_muscle_volumes AS
SELECT
    vev.*, -- Select all columns from vw_exercise_volumes
    mu.id AS muscle_id,
    mu.name AS muscle_name
FROM
    vw_exercise_volumes vev
JOIN
    exercise_muscles em ON vev.exercise_id = em.exercise_id
JOIN
    muscles mu ON em.muscle_id = mu.id;
