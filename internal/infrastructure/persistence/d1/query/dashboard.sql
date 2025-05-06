-- name: GetWeeklyVolumeSummary :many
-- Get total volume per week for a given device within a date range.
-- Weeks start on Monday ('%Y-%W'). Use '%Y-%w' for Sunday start (0=Sunday).
SELECT
    strftime('%Y-%W', performed_at) AS year_week, -- Format: YYYY-WW (e.g., '2024-01')
    SUM(volume) AS total_volume
FROM
    vw_workout_set_details -- Use View 1
WHERE
    device_id = ? -- $1: device_id
    AND performed_at >= ? -- $2: start_date (inclusive, e.g., 'YYYY-MM-DD')
    AND performed_at < ?  -- $3: end_date (exclusive, e.g., 'YYYY-MM-DD')
GROUP BY
    year_week
ORDER BY
    year_week ASC;

-- name: GetExerciseVolumeSummary :many
-- Get total volume per exercise for a given device within a date range.
SELECT
    exercise_id,
    exercise_name,
    SUM(volume) AS total_volume
FROM
    vw_exercise_volumes -- Use View 2
WHERE
    device_id = ? -- $1: device_id
    AND performed_at >= ? -- $2: start_date (inclusive)
    AND performed_at < ?  -- $3: end_date (exclusive)
GROUP BY
    exercise_id, exercise_name
ORDER BY
    total_volume DESC, exercise_name ASC;

-- name: GetMuscleVolumeSummary :many
-- Get total volume per muscle group for a given device within a date range.
SELECT
    muscle_id,
    muscle_name,
    SUM(volume) AS total_volume
FROM
    vw_muscle_volumes -- Use View 3
WHERE
    device_id = ? -- $1: device_id
    AND performed_at >= ? -- $2: start_date (inclusive)
    AND performed_at < ?  -- $3: end_date (exclusive)
GROUP BY
    muscle_id, muscle_name
ORDER BY
    total_volume DESC, muscle_name ASC;

-- name: GetTotalVolumeAveragePerWeek :one
-- Calculate the overall average weekly volume for a device over a specified period.
-- This first calculates weekly sums, then averages those sums.
WITH WeeklyVolumes AS (
    SELECT
        strftime('%Y-%W', performed_at) AS year_week,
        SUM(volume) AS weekly_total_volume
    FROM
        vw_workout_set_details
    WHERE
        device_id = ? -- $1: device_id
        AND performed_at >= ? -- $2: start_date (inclusive)
        AND performed_at < ?  -- $3: end_date (exclusive)
    GROUP BY
        year_week
)
SELECT
    AVG(weekly_total_volume) AS average_weekly_volume
FROM
    WeeklyVolumes;
