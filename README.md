# BulkTrack API – Backend (Go × Cloudflare Workers)

**BulkTrack** は "トレーニングボリューム" にフォーカスした筋トレ記録アプリのバックエンドです。  
iOS (Swift) と Web クライアントが **同一 REST API** を呼び出します。  
本リポジトリでは **Go (公式 Wasm compiler)** と **Cloudflare Workers**, **Cloudflare D1** をベースに、Clean Architecture / DDD を最小構成で適用しています。

---

## 🗺️ High-Level Architecture

```mermaid
graph TD
    Client[Client (iOS / Web)] -->|HTTPS (REST)| Worker
    Worker[Go Edge Worker (Wasm) <br/> - Router + Middleware <br/> - Command / Query (CQRS) <br/> - Domain Logic <br/> ← Cloudflare Workers] -->|D1 Binding| D1
    D1[Cloudflare D1 <br/> ← Serverless SQLite] 
```

* **Data Store** Cloudflare D1 (serverless SQLite) — Workers から直接低レイテンシ接続
* **Auth** Clerk (＋Apple Sign-In統合) — Edge Workerで JWT 検証のみ
* **Cache** `Cache-Control: private` ＋ Cloudflare Cache API (短期: 10–30 s)
* **API** 純粋 REST／OpenAPI 定義 `api/openapi.yaml`
* **Schema Definition** SQL (`schema.sql`) — DBスキーマの唯一の信頼できる情報源 (Atlas & sqlc 兼用、**SQLite 互換**)

---

## 🏗️ Layered Design

| Layer | Package | Responsibility |
|-------|---------|----------------|
| **Domain** | `internal/domain/...` | Entities, VOs, Domain Services (**pure Go**) |
| **Application** | `internal/app/{command,query}` | Use-case orchestration, Tx boundary, DTO ↔ Entity |
| **Interface / Adapter** | `internal/interface/http` | HTTP Router, DTO marshaling, Auth/CORS middleware |
| **Infrastructure** | `internal/infrastructure/persistence/d1` | **D1** repo implementation, Clerk client, etc. |

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
│   │   │   └── d1/            # ← D1 リポジトリ実装
│   │   │       ├── repo.go
│   │   │       ├── sql/         # sqlc-generated Go code
│   │   │       └── query/       # sqlc source queries (*.sql)
│   │   │       └── sqlc.yaml
│   │   └── auth/
│   │       └── clerk.go
│   └── platform/            # 共通ユーティリティ
│       ├── logger/
│       └── errors/
├── migrations/              # D1 マイグレーションファイル (*.sql)
├── scripts/                 # CI helper scripts
├── wrangler.jsonc           # wrangler 設定 (D1 バインディング含む)
├── go.mod
├── go.sum
├── schema.sql               # DB スキーマ定義 (SQLite)
├── README.md
├── Makefile
└── test/
    ├── unit/
    ├── usecase/
    └── integration/
```

* **1 file = 1 responsibility**（ユースケース or 型）で小分け
* インフラ差し替え用に `//go:build test` タグでメモリ実装を用意 (現状なし)

---

## 📦 Data Modeling

**注意:** スキーマの唯一の信頼できる情報源 (Single Source of Truth) は `schema.sql` ファイル (SQLite 互換) です。以下の Mermaid 図は視覚的な理解を助けるための参考情報であり、常に最新の状態を反映しているとは限りません。

```mermaid
-- (Mermaid 図は変更なし、ただしデータ型は実際には SQLite 互換になっている)
erDiagram
    %% === Core Tables ===
    menus {
        TEXT id PK
        TEXT user_id FK "Clerk ID"
        TEXT name
        TEXT description
        INTEGER sort_order
        TEXT created_at
        TEXT updated_at
    }

    exercises {
        TEXT id PK
        TEXT name
        TEXT user_id NULL "NULL = official"
        TEXT created_at
        TEXT updated_at
    }

    muscles {
        TEXT id PK
        TEXT name
    }

    workouts {
        TEXT id PK
        TEXT user_id FK
        TEXT menu_id FK
        TEXT performed_at
        REAL rpe NULL
        INTEGER rir NULL
        TEXT created_at
        TEXT updated_at
    }

    workout_sets {
        TEXT id PK
        TEXT workout_id FK
        TEXT exercise_id FK
        REAL weight
        INTEGER reps
        REAL volume "GENERATED ALWAYS AS (weight * reps) STORED"
    }

    %% === Join Tables ===
    menu_exercises {
        TEXT menu_id FK
        TEXT exercise_id FK
        INTEGER  position
        PK  (menu_id, exercise_id)
    }

    exercise_muscles {
        TEXT exercise_id FK
        TEXT muscle_id FK
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

* Go `>= 1.23`
* `wrangler >= 4` (or compatible version)
* Node.js / npm (for `wrangler` and build scripts)
* `sqlc` (for Go code generation from SQL)
* (Optional) `atlas` (for schema management assistance)

### 2. Create Cloudflare D1 Database (First time only)

Cloudflare アカウントにログインし、以下のコマンドで D1 データベースを作成します (例: `bulktrack-db`)。

```bash
npx wrangler d1 create bulktrack-db
```

出力された `database_id` を `wrangler.jsonc` の `d1_databases` セクションに設定します。

```jsonc
// wrangler.jsonc
{
  // ... other settings
  "d1_databases": [
    {
      "binding": "DB", // Worker から参照する名前
      "database_name": "bulktrack-db", // 作成したDB名
      "database_id": "YOUR_DATABASE_ID_HERE"
    }
  ]
}
```

### 3. Apply DB Migrations (Local)

ローカル開発用の D1 データベース (SQLite ファイル) にスキーマを適用します。
マイグレーションファイルは `migrations/` ディレクトリに配置します。

```bash
# migrations/ ディレクトリ内の未適用のマイグレーションをローカルDBに適用
npx wrangler d1 migrations apply bulktrack-db --local 
```

### 4. Generate Go code from SQL

`sqlc` を使用して、`internal/infrastructure/persistence/d1/query/` 内の SQL クエリから Go のコードを生成します。

```bash
# sqlc.yaml の設定に従ってコードを生成
sqlc generate -f internal/infrastructure/persistence/d1/sqlc.yaml
# または make コマンドがあれば
make sqlc
```

### 5. Run Worker locally

```bash
# .dev.vars などで環境変数を設定 (必要であれば)
wrangler dev
```

Wasm ビルドは `wrangler dev` が内部で実行します (通常は `npm run build` 経由)。

---

## 🗂️ Schema & Migration Playbook (Declarative with Atlas & Wrangler D1)

この章では **Atlas** と **Wrangler D1 Migrations** を組み合わせ、`schema.sql` (SQLite 互換) を唯一の信頼できる情報源 (Source of Truth) とする宣言的なデータベーススキーマ管理と、**sqlc** による型安全なコード生成の運用手順をまとめます。

参考: 
* [Atlas Docs](https://atlasgo.io/)
* [Cloudflare D1 Migrations](https://developers.cloudflare.com/d1/platform/migrations/)
* [Declarative migrations for sqlc | Atlas](https://atlasgo.io/guides/frameworks/sqlc-declarative) (PostgreSQL の例ですが考え方は応用可能)

---

### 1 . インストール & 前提

| Tool | Version (例) | Install |
|------|--------------|---------|
| Go   | `>= 1.23` | <https://go.dev/doc/install> |
| Wrangler | `>= 4` | `npm install -g wrangler` |
| sqlc | `>= 1.26` | `brew install sqlc` |
| Atlas | `>= 0.17` | `brew install ariga/tap/atlas` |

### 2. スキーマ変更の手順 (`schema.sql` を更新)

1.  **`schema.sql` を編集:** SQLite 互換の構文でテーブル定義などを変更します。
2.  **(任意) Atlas で差分確認:** `atlas schema diff` を使って、現在の D1 スキーマ (ローカルまたはリモート) と `schema.sql` の差分からマイグレーション SQL を生成・確認できます。
    ```bash
    # ローカル D1 と schema.sql の差分からマイグレーションSQLを生成 (適用はしない)
    # ローカルDBのパスは環境により異なる可能性あり
    atlas schema diff \
      --from "sqlite://.wrangler/state/v3/d1/d17fb255-2ce1-4e3d-bd57-64de4fe0c57b/db.sqlite" \
      --to file://schema.sql \
      --dev-url "sqlite://dev.db?mode=memory" # 検証用インメモリDB
    ```
3.  **マイグレーションファイルの作成:** `migrations/` ディレクトリに新しいマイグレーションファイル (例: `0002_add_new_column.sql`) を作成し、スキーマ変更を行う SQL (ALTER TABLE など) を記述します。(Atlas が生成した SQL を参考にできます)
4.  **マイグレーションの適用 (ローカル):** `wrangler` を使ってローカル D1 にマイグレーションを適用し、動作確認します。
    ```bash
    npx wrangler d1 migrations apply bulktrack-db --local
    ```
5.  **`sqlc` コード再生成:** スキーマ変更に合わせて Go のコードを再生成します。
    ```bash
    sqlc generate -f internal/infrastructure/persistence/d1/sqlc.yaml
    ```
6.  **テスト:** コード変更と合わせてテストを実行します。
7.  **マイグレーションの適用 (リモート):** 問題がなければリモートの D1 にマイグレーションを適用します。
    ```bash
    npx wrangler d1 migrations apply bulktrack-db --remote
    ```
8.  **デプロイ:** `wrangler deploy` で Worker をデプロイします。

このフローにより、`schema.sql` を中心とした宣言的なスキーマ管理と、`wrangler` による安全なマイグレーション適用、`sqlc` による型安全なコード生成を両立できます。

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
