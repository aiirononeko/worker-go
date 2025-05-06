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
    id           TEXT PRIMARY KEY,
    device_id    TEXT NOT NULL,                    -- パーティションキー
    menu_id      TEXT NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    performed_at TEXT NOT NULL DEFAULT (datetime('now')),
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (device_id) REFERENCES devices(id)
);

CREATE INDEX IF NOT EXISTS idx_workout_device_id   ON workouts(device_id);
CREATE INDEX IF NOT EXISTS idx_workout_performed   ON workouts(performed_at DESC);

-- ----------------------------------------------------------
-- 5. セット（1 ワークアウト内の複数セット）
-- ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS workout_sets (
    id          TEXT PRIMARY KEY,
    workout_id  TEXT NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    exercise_id TEXT NOT NULL REFERENCES exercises(id),
    weight      REAL    NOT NULL CHECK (weight >= 0),
    reps        INTEGER NOT NULL CHECK (reps   >= 0),
    rpe         REAL    CHECK (rpe BETWEEN 1 AND 10),
    rir         INTEGER CHECK (rir BETWEEN 0 AND 5)
);

CREATE INDEX IF NOT EXISTS idx_set_workout_id  ON workout_sets(workout_id);
CREATE INDEX IF NOT EXISTS idx_set_exercise_id ON workout_sets(exercise_id);

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
