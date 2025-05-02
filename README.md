# BulkTrack API – Backend (Go × Cloudflare Workers)

**BulkTrack** は "トレーニングボリューム" にフォーカスした筋トレ記録アプリのバックエンドです。  
iOS (Swift) と Web クライアントが **同一 REST API** を呼び出します。  
本リポジトリでは **Go (公式 Wasm compiler)** と **Cloudflare Workers** をベースに、Clean Architecture / DDD を最小構成で適用しています。

---

## 🗺️ High-Level Architecture

```
Client (iOS / Web)
        │  HTTPS (REST)
        ▼
┌────────────────────────────┐
│ Go Edge Worker (Wasm)      │  ← Cloudflare Workers
│ ─ Router + Middleware      │
│ ─ Command / Query (CQRS)   │
│ ─ Domain Logic             │
└────────┬───────────────────┘
         │ pooled TCP
         ▼
┌──────────────┐
│ Hyperdrive   │  ← Connection pool at PoP
└────────┬─────┘
         ▼
┌─────────────────┐
│ Neon Postgres   │  ← Serverless PG
└─────────────────┘
```

* **Data Store** Neon (serverless PostgreSQL) — Hyperdrive 経由で低レイテンシ接続  
* **Auth** Clerk (＋Apple Sign-In統合) — Edge Workerで JWT 検証のみ  
* **Cache** `Cache-Control: private` ＋ Cloudflare Cache API (短期: 10–30 s)  
* **API** 純粋 REST／OpenAPI 定義 `api/openapi.yaml`
* **Schema Definition** Atlas HCL (`schema.hcl`) — DBスキーマの唯一の信頼できる情報源

---

## 🏗️ Layered Design

| Layer | Package | Responsibility |
|-------|---------|----------------|
| **Domain** | `internal/domain/...` | Entities, VOs, Domain Services (**pure Go**) |
| **Application** | `internal/app/{command,query}` | Use-case orchestration, Tx boundary, DTO ↔ Entity |
| **Interface / Adapter** | `internal/interface/http` | HTTP Router, DTO marshaling, Auth/CORS middleware |
| **Infrastructure** | `internal/infrastructure/...` | Neon repo, Clerk client, mail, feature-flag, etc. |

依存方向は **Domain → Application → Interface**。Go の import も同方向のみ。  
`go vet -unusedresult` と `go test ./...` を CI で循環チェック。

---

## 📁 Directory Skeleton

```
bulktrack-api/
├── cmd/
│   └── worker/
│       └── main.go          # workers.Serve(entrypoint)
├── api/
│   └── openapi.yaml         # REST 定義 (Swagger UI 用)
├── config/
│   ├── config.go            # envparse + defaults
│   └── config_test.go
├── internal/
│   ├── domain/
│   │   ├── workout/
│   │   │   ├── entity.go
│   │   │   ├── value_volume.go
│   │   │   └── service_pr.go
│   │   └── user/…
│   ├── app/
│   │   ├── command/         # CQ ハンドラ
│   │   ├── query/
│   │   ├── dto/             # Request/Response Model
│   │   └── di/wire.go       # Provider set
│   ├── interface/
│   │   └── http/
│   │       ├── router.go
│   │       ├── middleware/
│   │       │   └── auth.go
│   │       └── handler/
│   │           └── workout_handler.go
│   │── infrastructure/
│   │   ├── persistence/
│   │   │   └── postgres/
│   │   │       ├── repo.go          # Implements domain repository
│   │   │       ├── db.go            # Open(), migrations
│   │   │       ├── sqlc.yaml
│   │   │       └── sql/             # sqlc-generated
│   │   └── auth/
│   │       └── clerk.go
│   └── platform/            # 共通ユーティリティ
│       ├── logger/
│       └── errors/
├── migrations/              # atlas migration files
├── scripts/                 # CI helper scripts
├── wrangler.toml
├── go.mod
├── README.md
├── Makefile
└── test/
    ├── unit/
    ├── usecase/
    └── integration/
```

* **1 file = 1 responsibility**（ユースケース or 型）で小分け  
* インフラ差し替え用に `//go:build test` タグでメモリ実装を用意
  ```bash
  make build
  test $(stat -c%s dist/worker.wasm) -lt 10000000
  ```

---

## 📦 Data Modeling

**注意:** スキーマの唯一の信頼できる情報源 (Single Source of Truth) は `schema.hcl` ファイルです。以下の Mermaid 図は視覚的な理解を助けるための参考情報であり、常に最新の状態を反映しているとは限りません。

```mermaid
erDiagram
    %% === Core Tables ===
    menus {
        UUID id PK
        TEXT user_id FK "Clerk ID"
        TEXT name
        TEXT description
        INT  sort_order
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    exercises {
        UUID id PK
        TEXT name
        TEXT user_id NULL "NULL = official"
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    muscles {
        UUID id PK
        TEXT name
    }

    workouts {
        UUID id PK
        TEXT user_id FK
        UUID menu_id FK
        TIMESTAMPTZ performed_at
        NUMERIC(3,1) rpe NULL
        SMALLINT rir NULL
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    workout_sets {
        UUID id PK
        UUID workout_id FK
        UUID exercise_id FK
        NUMERIC weight
        INT reps
        NUMERIC volume "GENERATED ALWAYS AS (weight * reps) STORED"
    }

    %% === Join Tables ===
    menu_exercises {
        UUID menu_id FK
        UUID exercise_id FK
        INT  position
        PK  (menu_id, exercise_id)
    }

    exercise_muscles {
        UUID exercise_id FK
        UUID muscle_id FK
        PK  (exercise_id, muscle_id)
    }

    %% === Relationships ===
    menus ||--o{ menu_exercises : includes
    exercises ||--o{ menu_exercises : "in menu"
    exercises ||--o{ exercise_muscles : targets
    muscles ||--o{ exercise_muscles : "< targeted by"
    menus ||--|{ workouts : "plans"
    workouts ||--o{ workout_sets : contains
    exercises ||--o{ workout_sets : performed
```

## 🛠️ Build & Local Development

### 1. Prerequisites

* Go `>= 1.22`
* `wrangler >= 3`
* Docker (Neon sandbox / atlas)

### 2. Start a Neon test instance + Hyperdrive stub

```bash
docker compose up neon-dev
```

### 3. Run Worker locally

```bash
wrangler dev --hyperdrive=stub --env=local
```

> `GOOS=wasip1 GOARCH=wasm go build -o dist/worker.wasm -trimpath -ldflags="-s -w" ./cmd/worker`

### 4. Apply DB migrations (Declarative)

ローカルDBにスキーマを適用します。

```bash
# schema.hcl の内容をローカルDBに適用
atlas schema apply \
  -u "postgres://postgres:password@localhost:5432/bulktrack?sslmode=disable" \
  --to file://schema.hcl

# 適用前に差分を確認し、問題なければ承認します。
# 自動で承認する場合は --auto-approve フラグを追加します。
```

---

## 🗂️ Schema & Migration Playbook (Declarative with Atlas HCL)

この章では **Atlas HCL (`schema.hcl`)** を用いた宣言的なデータベーススキーマ管理と、**sqlc** による型安全なコード生成の運用手順をまとめます。

---

### 1 . インストール & 前提

| Tool | Version (例) | Install |
|------|--------------|---------|
| Go   | `>= 1.22` | <https://go.dev/doc/install> |
| Atlas | `>= 0.17` | `brew install ariga/tap/atlas` |
| sqlc | `>= 1.26` | `brew install sqlc` |
| Docker (optional) | ― | Neon sandbox / postgres emulate |

```bash
# 初回のみ
brew install ariga/tap/atlas
brew install sqlc
```

環境変数 — `.envrc` などで永続化すると便利

```bash
export LOCAL_DATABASE_URL="postgres://postgres:password@localhost:5432/bulktrack?sslmode=disable"
export PRODUCTION_DATABASE_URL="postgres://user:pass@neon.tech/neondb?sslmode=require" # 例
```

---

### 2 . ディレクトリ構成

```
bulktrack-api/
├── schema.hcl           # ← Atlas HCL スキーマ定義
├── internal/infrastructure/persistence/postgres/
│   ├── sqlc.yaml
│   ├── query/           # 手書き SQL (SELECT, INSERT, etc.)
│   └── sql/             # sqlc-generate 産物 (git add 可)
├── atlas.hcl            # 環境設定 (任意)
└── ... (その他)
```

`migrations/` ディレクトリは宣言的マイグレーションでは通常使用しません。

---

### 3 . **スキーマを変更する手順**

1.  **`schema.hcl` を編集**
    *   テーブル、カラム、インデックスなどの定義を直接編集します。
    *   Atlas HCL のシンタックスに従ってください。
    *   参考: [Atlas HCL Documentation](https://atlasgo.io/atlas-schema/hcl)
2.  **スキーマ定義の検証 (Lint)**
    ```bash
    # ローカルDBと比較して潜在的な問題を検出
    atlas schema lint --env local --dev-url file://schema.hcl
    ```
3.  **差分確認 (任意だが推奨)**
    ```bash
    # ローカルDBとの差分を表示
    atlas schema diff --env local --dev-url file://schema.hcl
    ```
4.  **ローカル DB へ適用 & 動作確認**
    ```bash
    # schema.hcl の内容をローカルDBに適用 (差分確認後に承認)
    atlas schema apply \
      -u $LOCAL_DATABASE_URL \
      --to file://schema.hcl

    # または、atlas.hcl で env "local" が設定されていれば:
    # atlas schema apply --env local

    # アプリケーションを起動し、変更が意図通りか確認
    ```
5.  **sqlc で型安全コードを更新**
    *   **重要:** sqlc は現時点 (2024年5月) で HCL を直接スキーマソースとして読み込めません。
    *   そのため、現在のスキーマを SQL 形式で `inspect` し、それを `sqlc.yaml` で参照する必要があります。
    ```bash
    # 現在のローカルDBスキーマをSQLファイルに出力 (sqlc用)
    atlas schema inspect -u $LOCAL_DATABASE_URL --format '{{ sql . }}' > internal/infrastructure/persistence/postgres/schema_for_sqlc.sql

    # sqlc.yaml で schema_for_sqlc.sql を参照するように設定
    # (例: schema: "./schema_for_sqlc.sql")

    # sqlc を実行
    sqlc generate
    ```
    *   `internal/infrastructure/persistence/postgres/query/*.sql` に必要なクエリを追記・修正してから `sqlc generate` を実行します。
6.  **ユニットテスト / 結合テスト** を通す
7.  **PR を作成**
    *   GitHub Actions: `schema.hcl` の変更と生成された Go コードを含む。
    *   CI では `atlas schema lint`, `atlas schema diff --env production --dev-url file://schema.hcl` (dry-run 的な確認), `sqlc generate`, `go test ./...` を実行。
8.  **マージ後、自動デプロイ**
    *   GitHub Actions が `atlas schema apply --env production --dev-url file://schema.hcl --auto-approve` を実行し、本番 DB へスキーマを適用。

---

### 4 . `atlas.hcl` サンプル (宣言的マイグレーション用)

```hcl
# schema.hcl をスキーマソースとして定義
schema {
  src = "file://schema.hcl"
}

# ローカル開発環境
env "local" {
  url = env("LOCAL_DATABASE_URL")
  # dev_url は schema {} ブロックで定義されたものが使われる
}

# 本番環境 (例: Neon)
env "production" {
  url = env("PRODUCTION_DATABASE_URL")
}

# lint や diff の設定 (任意)
lint {
  destructive {
    error = true # カラム削除などをエラーとするか
  }
}
diff {
  skip {
    # 差分検出時に無視する変更 (例: コメント変更)
    # change_comment = true
  }
}
```

---

### 5 . `sqlc.yaml` サンプル (宣言的マイグレーション用)

```yaml
version: "2"
sql:
  - # schema は atlas schema inspect で生成したSQLファイルを指定
    schema: "./internal/infrastructure/persistence/postgres/schema_for_sqlc.sql"
    queries: "./internal/infrastructure/persistence/postgres/query"
    engine: "postgresql"
    gen:
      go:
        out: "./internal/infrastructure/persistence/postgres/sql"
        package: "sql"
        emit_json_tags: true
        emit_interface: false # 必要に応じて true に
```

---

### 6 . Makefile Shortcut サンプル

```makefile
DB_URL ?= $(LOCAL_DATABASE_URL)
SCHEMA_HCL = schema.hcl
SQLC_SCHEMA_OUT = internal/infrastructure/persistence/postgres/schema_for_sqlc.sql

# スキーマ関連
db/lint:
	# HCLファイルの静的解析 (DB接続不要)
	atlas schema lint --dev-url file://$(SCHEMA_HCL)

db/diff:
	# ローカルDBとの差分を表示
	atlas schema diff \
	  -u $(DB_URL) \
	  --dev-url file://$(SCHEMA_HCL)

db/apply:
	# ローカルDBにスキーマを適用 (自動承認)
	atlas schema apply \
	  -u $(DB_URL) \
	  --to file://$(SCHEMA_HCL) \
	  --auto-approve

db/inspect-for-sqlc:
	@echo "Inspecting local DB schema to $(SQLC_SCHEMA_OUT) for sqlc..."
	atlas schema inspect -u $(DB_URL) --format '{{ sql . }}' > $(SQLC_SCHEMA_OUT)

# sqlc 関連
sqlc: db/inspect-for-sqlc
	@echo "Generating Go code with sqlc..."
	sqlc generate

.PHONY: db/lint db/diff db/apply db/inspect-for-sqlc sqlc
```

---

#### 🌟 開発フロー早見表 (宣言的)

| フェーズ         | コマンド                     | 目的                                         |
|------------------|------------------------------|----------------------------------------------|
| **スキーマ編集** | (手動で `schema.hcl` 編集) | スキーマ定義を変更                             |
| **検証**         | `make db/lint`               | スキーマ定義の問題点をチェック (HCL構文)       |
| **差分確認**     | `make db/diff`               | ローカル DB との差分を確認                   |
| **ローカル適用** | `make db/apply`              | ローカル DB にスキーマ変更を反映 (自動承認) |
| **コード生成**   | `make sqlc`                  | DB スキーマから Go コード (DAO) を生成/更新 |
| **本番適用**     | GitHub Actions / 手動 apply | 本番 DB にスキーマ変更を反映                 |

---

## 🔐 Auth Middleware (excerpt)

```go
func RequireAuth(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
    claims, err := clerkjwt.Verify(r.Context(), &clerkjwt.VerifyParams{
      Token: raw,
      Audience: []string{"https://api.bulktrack.app"},
    })
    if err != nil { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    ctx := context.WithValue(r.Context(), auth.UserIDKey, claims.Subject)
    next.ServeHTTP(w, r.WithContext(ctx))
  })
}
```

JWKS は 5 min TTL で Cloudflare KV キャッシュ。

---

## ✅ Test / CI Pipeline

| Stage | What |
|-------|------|
| **Unit** | Domain logic — table-driven, `-race -cover` |
| **Use-Case** | App layer with memory repo DI |
| **Contract** | Dredd / Prism vs OpenAPI |
| **Integration** | GitHub Actions: Neon preview branch + Hyperdrive stub |
| **Deploy** | `wrangler deploy --env=production` on tag push |

---

## ➡️ Next Steps

1. `wrangler hyperdrive create` で Neon へのプールをバインド  
2. `sqlc generate` → `GET /v1/workouts/:id` ハッピー-パス実装  
3. ビルドサイズ & p99 レイテンシ計測 → ボトルネック洗い出し  
4. Worker Cron で週次サマリー通知ジョブを追加

---
