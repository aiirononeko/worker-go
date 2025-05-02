-- Create initial schema based on README data modeling

-- === Core Tables ===

CREATE TABLE "muscles" (
    "id" UUID PRIMARY KEY,
    "name" TEXT NOT NULL UNIQUE
);

COMMENT ON TABLE "muscles" IS '部位';
COMMENT ON COLUMN "muscles"."name" IS '部位名';

CREATE TABLE "exercises" (
    "id" UUID PRIMARY KEY,
    "name" TEXT NOT NULL,
    "user_id" TEXT NULL, -- NULL = official exercise
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE "exercises" IS 'エクササイズ種目';
COMMENT ON COLUMN "exercises"."user_id" IS 'NULLの場合は公式提供の種目';

CREATE TABLE "menus" (
    "id" UUID PRIMARY KEY,
    "user_id" TEXT NOT NULL, -- Clerk User ID
    "name" TEXT NOT NULL,
    "description" TEXT,
    "sort_order" INT NOT NULL DEFAULT 0,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE "menus" IS 'トレーニングメニュー';
COMMENT ON COLUMN "menus"."user_id" IS 'ClerkのユーザーID';
COMMENT ON COLUMN "menus"."sort_order" IS 'メニューの表示順';


CREATE TABLE "workouts" (
    "id" UUID PRIMARY KEY,
    "user_id" TEXT NOT NULL,
    "menu_id" UUID NOT NULL REFERENCES "menus"("id") ON DELETE CASCADE,
    "performed_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "rpe" NUMERIC(3,1) NULL CHECK ("rpe" >= 1 AND "rpe" <= 10), -- Rate of Perceived Exertion
    "rir" SMALLINT NULL CHECK ("rir" >= 0 AND "rir" <= 5), -- Reps In Reserve
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE "workouts" IS '特定の日のワークアウト記録';
COMMENT ON COLUMN "workouts"."user_id" IS 'ClerkのユーザーID';
COMMENT ON COLUMN "workouts"."menu_id" IS '実行したメニューのID';
COMMENT ON COLUMN "workouts"."performed_at" IS 'ワークアウト実行日時';
COMMENT ON COLUMN "workouts"."rpe" IS '主観的運動強度 (1-10)';
COMMENT ON COLUMN "workouts"."rir" IS 'レップス・イン・リザーブ (0-5)';

CREATE TABLE "workout_sets" (
    "id" UUID PRIMARY KEY,
    "workout_id" UUID NOT NULL REFERENCES "workouts"("id") ON DELETE CASCADE,
    "exercise_id" UUID NOT NULL REFERENCES "exercises"("id"), -- Do not cascade delete exercise itself
    "weight" NUMERIC NOT NULL CHECK ("weight" >= 0),
    "reps" INT NOT NULL CHECK ("reps" >= 0),
    "volume" NUMERIC GENERATED ALWAYS AS ("weight" * "reps") STORED
);

COMMENT ON TABLE "workout_sets" IS 'ワークアウト内の具体的なセット記録';
COMMENT ON COLUMN "workout_sets"."workout_id" IS '所属するワークアウトID';
COMMENT ON COLUMN "workout_sets"."exercise_id" IS '実行したエクササイズ種目ID';
COMMENT ON COLUMN "workout_sets"."weight" IS '使用重量 (kg)';
COMMENT ON COLUMN "workout_sets"."reps" IS 'レップ数';
COMMENT ON COLUMN "workout_sets"."volume" IS 'ボリューム (重量 * レップ数)';

-- === Join Tables ===

CREATE TABLE "menu_exercises" (
    "menu_id" UUID NOT NULL REFERENCES "menus"("id") ON DELETE CASCADE,
    "exercise_id" UUID NOT NULL REFERENCES "exercises"("id") ON DELETE CASCADE,
    "position" INT NOT NULL DEFAULT 0,
    PRIMARY KEY ("menu_id", "exercise_id")
);

COMMENT ON TABLE "menu_exercises" IS 'メニューとエクササイズの関連付け';
COMMENT ON COLUMN "menu_exercises"."position" IS 'メニュー内でのエクササイズの順序';


CREATE TABLE "exercise_muscles" (
    "exercise_id" UUID NOT NULL REFERENCES "exercises"("id") ON DELETE CASCADE,
    "muscle_id" UUID NOT NULL REFERENCES "muscles"("id") ON DELETE CASCADE,
    PRIMARY KEY ("exercise_id", "muscle_id")
);

COMMENT ON TABLE "exercise_muscles" IS 'エクササイズとターゲット部位の関連付け';

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
