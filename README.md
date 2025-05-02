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
* **Schema Definition** SQL (`schema.sql`) — DBスキーマの唯一の信頼できる情報源 (Atlas & sqlc 兼用)

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

**注意:** スキーマの唯一の信頼できる情報源 (Single Source of Truth) は `schema.sql` ファイルです。以下の Mermaid 図は視覚的な理解を助けるための参考情報であり、常に最新の状態を反映しているとは限りません。

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

### 4. Apply DB migrations (Declarative with sqlc)

ローカルDBに `schema.sql` の内容を適用します。

```bash
# schema.sql の内容をローカルDBに適用
atlas schema apply \
  -u "postgres://postgres:password@localhost:5432/bulktrack?sslmode=disable" \
  --to file://schema.sql \
  --dev-url "docker://postgres/16/dev" # Atlas が検証用に一時DBコンテナを使用

# 適用前に差分を確認し、問題なければ承認します。
# 自動で承認する場合は --auto-approve フラグを追加します。
```

`--dev-url` を指定することで、Atlas は一時的な Docker コンテナ (`postgres:16` イメージの `dev` という名前のコンテナ) を起動し、適用計画の検証を行います。初回実行時は Docker イメージのプルに時間がかかることがあります。

---

## 🗂️ Schema & Migration Playbook (Declarative with Atlas & sqlc)

この章では **Atlas** と **sqlc** を組み合わせ、`schema.sql` を唯一の信頼できる情報源 (Source of Truth) とする宣言的なデータベーススキーマ管理と、型安全なコード生成の運用手順をまとめます。

参考: [Declarative migrations for sqlc | Atlas](https://atlasgo.io/guides/frameworks/sqlc-declarative)

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
├── schema.sql           # ← SQL スキーマ定義 (Atlas & sqlc 兼用)
├── internal/infrastructure/persistence/postgres/
│   ├── sqlc.yaml
│   ├── query/           # 手書き SQL (SELECT, INSERT, etc.)
│   └── sql/             # sqlc-generate 産物 (git add 可)
└── ... (その他)
```

`schema.hcl` や `migrations/` ディレクトリはこのワークフローでは使用しません。

---

### 3 . **スキーマを変更する手順**

1.  **`schema.sql` を編集**
    *   `CREATE TABLE`, `ALTER TABLE` (※注意: Atlas は差分から判断するため、通常 `CREATE` のみでOK) などの標準 SQL を使ってスキーマ定義を直接編集します。
2.  **`query.sql` を編集 (任意)**
    *   スキーマ変更に伴い、`internal/infrastructure/persistence/postgres/query/` 以下のクエリファイル (`.sql`) を必要に応じて修正します。
3.  **`sqlc generate` を実行**
    *   `schema.sql` または `query.sql` を変更したら、必ず `sqlc generate` を実行して Go の型定義やデータベースアクセスのコードを更新します。
    ```bash
    sqlc generate
    ```
4.  **差分確認 (任意だが推奨)**
    *   Atlas を使って、現在のローカル DB と `schema.sql` の差分を確認します。
    ```bash
    atlas schema diff \
      -u $LOCAL_DATABASE_URL \
      --dev-url file://schema.sql # 比較対象として schema.sql を指定
    ```
5.  **ローカル DB へ適用 & 動作確認**
    *   Atlas を使って `schema.sql` の内容をローカル DB に適用します。
    ```bash
    # schema.sql の内容をローカルDBに適用 (差分確認後に承認)
    atlas schema apply \
      -u $LOCAL_DATABASE_URL \
      --to file://schema.sql \
      --dev-url "docker://postgres/16/dev"

    # 自動承認する場合:
    # atlas schema apply -u $LOCAL_DATABASE_URL --to file://schema.sql --dev-url "docker://postgres/16/dev" --auto-approve
    ```
    *   アプリケーションを起動し、変更が意図通りか確認します。
6.  **ユニットテスト / 結合テスト** を通す
7.  **PR を作成**
    *   `schema.sql`, `query/*.sql` の変更と、`sqlc generate` で生成された Go コードを含む。
    *   CI でのチェック項目例:
        *   `sqlc generate` (差分がないこと)
        *   `atlas schema diff --url $PRODUCTION_DATABASE_URL --dev-url file://schema.sql` (本番との差分確認、Dry Run)
        *   `go test ./...`
8.  **マージ後、自動デプロイ**
    *   CI/CD パイプラインが `atlas schema apply --url $PRODUCTION_DATABASE_URL --to file://schema.sql --auto-approve` を実行し、本番 DB へスキーマを適用。

---

### 4 . `sqlc.yaml` サンプル

```yaml
version: "2"
sql:
  - # schema はプロジェクトルートの schema.sql を直接指定
    schema: "./schema.sql"
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

### 5 . Makefile Shortcut サンプル

```makefile
DB_URL ?= $(LOCAL_DATABASE_URL)
SCHEMA_SQL = schema.sql
ATLAS_DEV_DB = "docker://postgres/16/dev"

# sqlc
sqlc:
	@echo "Generating Go code with sqlc..."
	sqlc generate

# Atlas スキーマ関連
db/diff:
	@echo "Checking differences between DB and $(SCHEMA_SQL)..."
	atlas schema diff \
	  -u $(DB_URL) \
	  --dev-url file://$(SCHEMA_SQL)

db/apply:
	@echo "Applying $(SCHEMA_SQL) to the database (auto-approve)..."
	atlas schema apply \
	  -u $(DB_URL) \
	  --to file://$(SCHEMA_SQL) \
	  --dev-url $(ATLAS_DEV_DB) \
	  --auto-approve

db/apply-confirm:
	@echo "Applying $(SCHEMA_SQL) to the database (confirm required)..."
	atlas schema apply \
	  -u $(DB_URL) \
	  --to file://$(SCHEMA_SQL) \
	  --dev-url $(ATLAS_DEV_DB)

.PHONY: sqlc db/diff db/apply db/apply-confirm
```

---

#### 🌟 開発フロー早見表 (Atlas Declarative + sqlc)

| フェーズ         | コマンド                     | 目的                                            |
|------------------|------------------------------|-------------------------------------------------|
| **スキーマ編集** | (手動で `schema.sql` 編集) | スキーマ定義 (DDL) を変更                         |
| **クエリ編集**   | (手動で `query/*.sql` 編集) | スキーマ変更に合わせてクエリを修正 (任意)         |
| **コード生成**   | `make sqlc`                  | `schema.sql`/`query/*.sql` から Go コードを生成/更新 |
| **差分確認**     | `make db/diff`               | ローカル DB と `schema.sql` の差分を確認        |
| **ローカル適用** | `make db/apply-confirm`      | ローカル DB にスキーマ変更を反映 (確認あり)      |
| **テスト**       | `go test ./...`              | 変更後のコードとスキーマでテストを実行            |
| **本番適用**     | GitHub Actions / 手動 apply | 本番 DB に `schema.sql` の状態を反映            |

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
