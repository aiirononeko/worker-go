-- ----------------------------------------------------------
-- 0. OPTIONAL: device 一覧
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS devices (
    id           TEXT PRIMARY KEY,                 -- deviceId (= Keychain UUID)
    user_id      TEXT,                             -- nullable
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ----------------------------------------------------------
-- 1. マスタ
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS muscles (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

-- ----------------------------------------------------------
-- 2. エクササイズ
--    device_id = NULL なら「公式種目」
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS exercises (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    device_id   TEXT NULL,                         -- NULL = 公式
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (device_id) REFERENCES devices(id)
);

CREATE INDEX IF NOT EXISTS idx_exercise_device_id ON exercises(device_id);

-- ----------------------------------------------------------
-- 3. メニュー（ユーザー／デバイスごと）
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS menus (
    id          TEXT PRIMARY KEY,
    device_id   TEXT NOT NULL,                     -- パーティションキー
    name        TEXT NOT NULL,
    description TEXT,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (device_id) REFERENCES devices(id),
    UNIQUE (device_id, name)
);

CREATE INDEX IF NOT EXISTS idx_menu_device_id ON menus(device_id);

-- ----------------------------------------------------------
-- 4. ワークアウト（実施記録）
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS workouts (
    id           TEXT PRIMARY KEY,          -- UUID for the workout session
    device_id    TEXT NOT NULL,             -- Foreign key to devices table
    menu_id      TEXT NOT NULL,             -- Foreign key to menus table
    performed_at TEXT NOT NULL,             -- Timestamp when the workout was performed (ISO8601 format recommended)
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')), -- Use ISO8601 format
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')), -- Use ISO8601 format
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE, -- Delete workouts if device is deleted
    FOREIGN KEY (menu_id) REFERENCES menus(id) ON DELETE RESTRICT     -- Prevent deleting menus if workouts reference them (or use ON DELETE SET NULL/DEFAULT)
);

CREATE INDEX IF NOT EXISTS idx_workouts_device_id ON workouts(device_id);
CREATE INDEX IF NOT EXISTS idx_workouts_menu_id ON workouts(menu_id); -- If querying by menu is common

-- ----------------------------------------------------------
-- 5. セット（1 ワークアウト内の複数セット）
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS workout_sets (
    id            TEXT PRIMARY KEY,          -- UUID for the workout set
    workout_id    TEXT NOT NULL,             -- Foreign key to workouts table
    set_order     INTEGER NOT NULL,          -- Order of the set within the workout (1-based)
    weight        REAL NOT NULL,             -- Weight used (use REAL for floating point)
    reps          INTEGER NOT NULL,          -- Repetitions performed
    volume        REAL GENERATED ALWAYS AS (weight * reps) STORED, -- Volume generated column (STORED recommended)
    interval      INTEGER,                   -- Rest interval *after* this set in seconds (nullable).
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE -- Delete sets if workout is deleted
);

CREATE INDEX IF NOT EXISTS idx_workout_sets_workout_id ON workout_sets(workout_id);

-- Trigger to update workouts.updated_at when workout_sets are modified (Optional but good practice)
-- Note: Cloudflare D1 might have limitations on complex triggers. Basic update trigger should work.
CREATE TRIGGER trigger_workout_sets_update_workouts_updated_at
AFTER UPDATE ON workout_sets
FOR EACH ROW
BEGIN
    UPDATE workouts SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = OLD.workout_id;
END;

-- Consider a similar trigger for INSERT/DELETE on workout_sets if needed

-- ----------------------------------------------------------
-- 6. Join テーブル
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS menu_exercises (
    menu_id     TEXT NOT NULL REFERENCES menus(id)     ON DELETE CASCADE,
    exercise_id TEXT NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (menu_id, exercise_id)
);

CREATE INDEX IF NOT EXISTS idx_menu_exercise_menu_id     ON menu_exercises(menu_id);
CREATE INDEX IF NOT EXISTS idx_menu_exercise_exercise_id ON menu_exercises(exercise_id);

CREATE TABLE IF NOT EXISTS exercise_muscles (
    exercise_id TEXT NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    muscle_id   TEXT NOT NULL REFERENCES muscles(id)   ON DELETE CASCADE,
    PRIMARY KEY (exercise_id, muscle_id)
);

CREATE INDEX IF NOT EXISTS idx_exercise_muscle_exercise_id ON exercise_muscles(exercise_id);
CREATE INDEX IF NOT EXISTS idx_exercise_muscle_muscle_id   ON exercise_muscles(muscle_id);

-- ----------------------------------------------------------
-- 7. Views for Dashboard Aggregation
-- ----------------------------------------------------------

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
    vwsd.weight,
    vwsd.reps,
    ex.id AS exercise_id,
    ex.name AS exercise_name
    -- Add other columns from exercises or menus if needed for dashboard display
FROM
    vw_workout_set_details vwsd
JOIN
    workouts w ON vwsd.workout_id = w.id
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
