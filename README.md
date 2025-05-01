# BulkTrack API – Backend (Go × Cloudflare Workers)

**BulkTrack** は “トレーニングボリューム” にフォーカスした筋トレ記録アプリのバックエンドです。  
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

### 4. Apply DB migrations

```bash
atlas migrate apply \
  -u "postgres://user:pass@localhost:5432/bulktrack?sslmode=disable"
```

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
