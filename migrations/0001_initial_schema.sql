-- Schema for SQLite (Cloudflare D1)

-- === Core Tables ===

CREATE TABLE "muscles" (
    "id" TEXT PRIMARY KEY,
    "name" TEXT NOT NULL UNIQUE
);

CREATE TABLE "exercises" (
    "id" TEXT PRIMARY KEY,
    "name" TEXT NOT NULL,
    "user_id" TEXT NULL, -- NULL = official exercise
    "created_at" TEXT NOT NULL DEFAULT (datetime('now')),
    "updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE "menus" (
    "id" TEXT PRIMARY KEY,
    "user_id" TEXT NOT NULL, -- Clerk User ID
    "name" TEXT NOT NULL,
    "description" TEXT,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TEXT NOT NULL DEFAULT (datetime('now')),
    "updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE "workouts" (
    "id" TEXT PRIMARY KEY,
    "user_id" TEXT NOT NULL,
    "menu_id" TEXT NOT NULL REFERENCES "menus"("id") ON DELETE CASCADE,
    "performed_at" TEXT NOT NULL DEFAULT (datetime('now')),
    "rpe" REAL NULL CHECK ("rpe" >= 1 AND "rpe" <= 10),
    "rir" INTEGER NULL CHECK ("rir" >= 0 AND "rir" <= 5),
    "created_at" TEXT NOT NULL DEFAULT (datetime('now')),
    "updated_at" TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE "workout_sets" (
    "id" TEXT PRIMARY KEY,
    "workout_id" TEXT NOT NULL REFERENCES "workouts"("id") ON DELETE CASCADE,
    "exercise_id" TEXT NOT NULL REFERENCES "exercises"("id"),
    "weight" REAL NOT NULL CHECK ("weight" >= 0),
    "reps" INTEGER NOT NULL CHECK ("reps" >= 0),
    "volume" REAL GENERATED ALWAYS AS ("weight" * "reps") STORED
);

-- === Join Tables ===

CREATE TABLE "menu_exercises" (
    "menu_id" TEXT NOT NULL REFERENCES "menus"("id") ON DELETE CASCADE,
    "exercise_id" TEXT NOT NULL REFERENCES "exercises"("id") ON DELETE CASCADE,
    "position" INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY ("menu_id", "exercise_id")
);

CREATE TABLE "exercise_muscles" (
    "exercise_id" TEXT NOT NULL REFERENCES "exercises"("id") ON DELETE CASCADE,
    "muscle_id" TEXT NOT NULL REFERENCES "muscles"("id") ON DELETE CASCADE,
    PRIMARY KEY ("exercise_id", "muscle_id")
);

-- === Indexes ===
CREATE INDEX idx_menu_user_id ON "menus" ("user_id");
CREATE INDEX idx_workout_user_id ON "workouts" ("user_id");
CREATE INDEX idx_workout_performed_at ON "workouts" ("performed_at" DESC);
CREATE INDEX idx_workout_set_workout_id ON "workout_sets" ("workout_id");
CREATE INDEX idx_workout_set_exercise_id ON "workout_sets" ("exercise_id");
CREATE INDEX idx_menu_exercise_menu_id ON "menu_exercises" ("menu_id");
CREATE INDEX idx_menu_exercise_exercise_id ON "menu_exercises" ("exercise_id");
CREATE INDEX idx_exercise_muscle_exercise_id ON "exercise_muscles" ("exercise_id");
CREATE INDEX idx_exercise_muscle_muscle_id ON "exercise_muscles" ("muscle_id");
