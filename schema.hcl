table "exercise_muscles" {
  schema  = schema.public
  comment = "エクササイズとターゲット部位の関連付け"
  column "exercise_id" {
    null = false
    type = uuid
  }
  column "muscle_id" {
    null = false
    type = uuid
  }
  primary_key {
    columns = [column.exercise_id, column.muscle_id]
  }
  foreign_key "exercise_muscles_exercise_id_fkey" {
    columns     = [column.exercise_id]
    ref_columns = [table.exercises.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "exercise_muscles_muscle_id_fkey" {
    columns     = [column.muscle_id]
    ref_columns = [table.muscles.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "idx_exercise_muscle_exercise_id" {
    columns = [column.exercise_id]
  }
  index "idx_exercise_muscle_muscle_id" {
    columns = [column.muscle_id]
  }
}
table "exercises" {
  schema  = schema.public
  comment = "エクササイズ種目"
  column "id" {
    null = false
    type = uuid
  }
  column "name" {
    null = false
    type = text
  }
  column "user_id" {
    null    = true
    type    = text
    comment = "NULLの場合は公式提供の種目"
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("CURRENT_TIMESTAMP")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("CURRENT_TIMESTAMP")
  }
  primary_key {
    columns = [column.id]
  }
}
table "menu_exercises" {
  schema  = schema.public
  comment = "メニューとエクササイズの関連付け"
  column "menu_id" {
    null = false
    type = uuid
  }
  column "exercise_id" {
    null = false
    type = uuid
  }
  column "position" {
    null    = false
    type    = integer
    default = 0
    comment = "メニュー内でのエクササイズの順序"
  }
  primary_key {
    columns = [column.menu_id, column.exercise_id]
  }
  foreign_key "menu_exercises_exercise_id_fkey" {
    columns     = [column.exercise_id]
    ref_columns = [table.exercises.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "menu_exercises_menu_id_fkey" {
    columns     = [column.menu_id]
    ref_columns = [table.menus.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "idx_menu_exercise_exercise_id" {
    columns = [column.exercise_id]
  }
  index "idx_menu_exercise_menu_id" {
    columns = [column.menu_id]
  }
}
table "menus" {
  schema  = schema.public
  comment = "トレーニングメニュー"
  column "id" {
    null = false
    type = uuid
  }
  column "user_id" {
    null    = false
    type    = text
    comment = "ClerkのユーザーID"
  }
  column "name" {
    null = false
    type = text
  }
  column "description" {
    null = true
    type = text
  }
  column "sort_order" {
    null    = false
    type    = integer
    default = 0
    comment = "メニューの表示順"
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("CURRENT_TIMESTAMP")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("CURRENT_TIMESTAMP")
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_menu_user_id" {
    columns = [column.user_id]
  }
}
table "muscles" {
  schema  = schema.public
  comment = "部位"
  column "id" {
    null = false
    type = uuid
  }
  column "name" {
    null    = false
    type    = text
    comment = "部位名"
  }
  primary_key {
    columns = [column.id]
  }
  unique "muscles_name_key" {
    columns = [column.name]
  }
}
table "workout_sets" {
  schema  = schema.public
  comment = "ワークアウト内の具体的なセット記録"
  column "id" {
    null = false
    type = uuid
  }
  column "workout_id" {
    null    = false
    type    = uuid
    comment = "所属するワークアウトID"
  }
  column "exercise_id" {
    null    = false
    type    = uuid
    comment = "実行したエクササイズ種目ID"
  }
  column "weight" {
    null    = false
    type    = numeric
    comment = "使用重量 (kg)"
  }
  column "reps" {
    null    = false
    type    = integer
    comment = "レップ数"
  }
  column "volume" {
    null    = true
    type    = numeric
    comment = "ボリューム (重量 * レップ数)"
    as {
      expr = "(weight * (reps)::numeric)"
      type = STORED
    }
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "workout_sets_exercise_id_fkey" {
    columns     = [column.exercise_id]
    ref_columns = [table.exercises.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "workout_sets_workout_id_fkey" {
    columns     = [column.workout_id]
    ref_columns = [table.workouts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "idx_workout_set_exercise_id" {
    columns = [column.exercise_id]
  }
  index "idx_workout_set_workout_id" {
    columns = [column.workout_id]
  }
  check "workout_sets_reps_check" {
    expr = "(reps >= 0)"
  }
  check "workout_sets_weight_check" {
    expr = "(weight >= (0)::numeric)"
  }
}
table "workouts" {
  schema  = schema.public
  comment = "特定の日のワークアウト記録"
  column "id" {
    null = false
    type = uuid
  }
  column "user_id" {
    null    = false
    type    = text
    comment = "ClerkのユーザーID"
  }
  column "menu_id" {
    null    = false
    type    = uuid
    comment = "実行したメニューのID"
  }
  column "performed_at" {
    null    = false
    type    = timestamptz
    default = sql("CURRENT_TIMESTAMP")
    comment = "ワークアウト実行日時"
  }
  column "rpe" {
    null    = true
    type    = numeric(3,1)
    comment = "主観的運動強度 (1-10)"
  }
  column "rir" {
    null    = true
    type    = smallint
    comment = "レップス・イン・リザーブ (0-5)"
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("CURRENT_TIMESTAMP")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("CURRENT_TIMESTAMP")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "workouts_menu_id_fkey" {
    columns     = [column.menu_id]
    ref_columns = [table.menus.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "idx_workout_performed_at" {
    on {
      desc   = true
      column = column.performed_at
    }
  }
  index "idx_workout_user_id" {
    columns = [column.user_id]
  }
  check "workouts_rir_check" {
    expr = "((rir >= 0) AND (rir <= 5))"
  }
  check "workouts_rpe_check" {
    expr = "((rpe >= (1)::numeric) AND (rpe <= (10)::numeric))"
  }
}
schema "public" {
  comment = "standard public schema"
}
