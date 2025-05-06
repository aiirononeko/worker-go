# BulkTrack API – Backend (Go × Cloudflare Workers)

**BulkTrack** は *"トレーニングボリューム"* にフォーカスした筋トレ記録アプリのバックエンドです。

- **iOS (Swift / SwiftUI / Core Data & Transferable)** と将来の **Web クライアント** が **同じ REST API** を呼び出します。
- バックエンドは **Go (公式 Wasm compiler) + Cloudflare Workers + Cloudflare D1**。
- クリーンアーキテクチャ / DDD を *最小構成* で適用しています。

---

## 🔑 User Strategy & Auth / AuthZ

| フェーズ | 仕組み | ユーザー操作 | 認可キー | ストレージ |
|----------|--------|--------------|-----------|------------|
| **① 初回起動** | `deviceId`(UUID) を Keychain に払い出し | 操作ゼロ | `device:<uuid>` | — |
| **② 普段の同期** | デバイス署名付き *stateless JWT* (15 min)<br>Refresh Token = KV (TTL ≤ 30 d) | 操作ゼロ | `device:<uuid>` | Workers KV |
| **③ データ移行 (機種変更など)** | **iOS 17+ Transferable API**<br>(AirDrop / QR コード) | • 送信: 「データを移行」をタップ<br>• 受信: Face ID 承認のみ | 旧 `device:` → 新 `device:` へサーバーが所有権更新 | D1 Tx |
| **④ オプション**<br>Appleでサインイン | Quick Login 1 タップ | 任意 | `user:<uuid>` | Users Table (D1) |

> **強制ログアウト**: KV の失効リスト更新 → アクセストークンは 60 s 以内に無効化。

### 行レベルセキュリティ
- **デフォルト** : `device_id` カラムでパーティショニング。
- **アップグレード後** : `user_id` カラムへマイグレーションし、複数 `device_id` を同一 `user_id` に束ねる。
- クエリ層で必ず `WHERE device_id = :caller OR user_id = :caller` の形を自動付与。

---

## 🗺️ High‑Level Architecture

```mermaid
graph TD
    Client[Client (iOS / Web)] -->|HTTPS (REST)| Worker
    Worker[Go Edge Worker (Wasm)\n- Router + Middleware\n- Command / Query (CQRS)\n- Domain Logic\n← Cloudflare Workers] -->|D1 Binding| D1
    D1[Cloudflare D1\n← Serverless SQLite]
```

* **Data Store**   Cloudflare D1 (serverless SQLite) — Workers から低レイテンシ直接接続
* **Auth**        DeviceID stateless JWT (+ optional Apple ID) — Edge Worker で署名検証のみ
* **Token Store** Refresh Token & revocation list → Workers KV (TTL 制御)
* **Sync Transfer** iOS 17+ Transferable Framework でデバイス間手渡し
* **API**        純粋 REST／OpenAPI 定義 `api/openapi.yaml`
* **Schema Definition** `schema.sql` が Single Source of Truth (SQLite 互換)

---

## 🏗️ Layered Design

| Layer | Package | Responsibility |
|-------|---------|----------------|
| **Domain** | `internal/domain/...` | Entities, VOs, Domain Services (pure Go) |
| **Application** | `internal/app/{command,query}` | Use‑case orchestration, Tx boundary, DTO ↔ Entity |
| **Interface / Adapter** | `internal/interface/http` | HTTP Router, DTO marshal, Auth/CORS middleware |
| **Infrastructure** | `internal/infrastructure/persistence/d1` | D1 repo impl, KV client, JWT utils |

依存方向は **Domain → Application → Interface**。Go import も同方向のみ。

## 🔌 Middleware Architecture

The application utilizes a middleware chaining approach for handling cross-cutting concerns like CORS, logging, and authentication.

- **Global Middlewares**: Applied to most routes.
    - `CORS`: Handles Cross-Origin Resource Sharing headers (allows all origins in development). Defined in `internal/interface/http/middleware/cors.go`.
    - `Logging`: Logs basic HTTP request/response details (method, path, status, duration). Defined in `internal/interface/http/middleware/logging.go`.
- **Route-Specific Middlewares**: Applied only to specific routes requiring them.
    - `RequireAuth`: Verifies the `Authorization: Bearer <token>` header using `JWTService` and injects the `uid` (e.g., `device:<uuid>`) into the request context. Defined in `internal/interface/http/middleware/auth.go`.
- **Chaining**: Implemented using a helper function `middleware.Chain` in `internal/interface/http/middleware/middleware.go`.
- **Router Configuration**: Defined in `internal/interface/http/router.go`, which allows specifying middlewares per route. Global middlewares (CORS, Logging) are applied automatically to most endpoints, while `RequireAuth` is applied selectively to protected routes (e.g., `/v1/menus`) in `cmd/worker/main.go`. Endpoints under `/v1/auth/*` and `/ping` currently bypass the `RequireAuth` middleware but include CORS and Logging.

---

## 📁 Directory Skeleton

```
bulktrack-api/
├── cmd/
│   └── worker/
│       └── main.go          # workers.Serve(entrypoint)
├── api/
│   └── openapi.yaml         # REST 定義 (Swagger UI 用)
├── config/
│   ├── config.go            # envparse + defaults
│   └── config_test.go
├── internal/
│   ├── domain/
│   ├── app/
│   ├── interface/
│   │   └── http/
│   │       ├── router.go
│   │       ├── middleware/
│   │       │   └── auth.go
│   │       └── handler/
│   │   ├── infrastructure/
│   │   │   ├── persistence/
│   │   │   │   └── d1/
│   │   │   └── auth/
│   │   └── platform/
│   └── migrations/
├── scripts/
├── wrangler.jsonc
├── schema.sql               # ← デバイスIDベーススキーマ
├── README.md
└── …
```

---

## 📦 Data Modeling (抜粋)

デフォルトパーティションは **`device_id`**。

```sql
CREATE TABLE devices (
  id           TEXT PRIMARY KEY,          -- deviceId (Keychain UUID)
  user_id      TEXT,                      -- nullable (Apple ID 統合後)
  created_at   TEXT NOT NULL DEFAULT (datetime('now')),
  last_seen_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE workouts (
  id          TEXT PRIMARY KEY,
  device_id   TEXT NOT NULL,
  user_id     TEXT,                       -- nullable
  menu_id     TEXT NOT NULL,
  performed_at TEXT NOT NULL,
  -- …
  FOREIGN KEY (device_id) REFERENCES devices(id)
);

CREATE TABLE IF NOT EXISTS menus (
    id          TEXT PRIMARY KEY,
    device_id   TEXT NOT NULL,                     -- パーティションキー (NOTE: Currently, queries are based on device_id only)
    name        TEXT NOT NULL,
    description TEXT,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (device_id) REFERENCES devices(id)
);
```

マイグレーションポリシーは README 冒頭の戦略セクションを参照。

---

## ✅ 初回〜リフレッシュのフロー確認

| ステップ | エンドポイント | クライアント送信 | Worker 処理 | メモ |
|----------|---------------|-----------------|-------------|------|
| **① アクティベート** | `POST /v1/auth/device` (認証不要) | ヘッダ `X-Device-Id: <uuid>` | `INSERT OR IGNORE` で `devices` 登録 → **access JWT (15 min)** と **refresh token (30 d)** を返す | 初回起動時のみ |
| **② API 呼び出し** | 任意 | ヘッダ `Authorization: Bearer <access>` | 署名検証のみ（stateless） | Token は Keychain などに保持 |
| **③ リフレッシュ** | `POST /v1/auth/refresh` | JSON `{ "refresh_token": "…" }` | Workers KV で `jti` 検証 → 新しい access＋refresh を発行（**回転**）し KV を更新 | スライディングウィンドウで 30 日延長 |
| **④ ログアウト** | `POST /v1/auth/logout` | JSON `{ "refresh_token": "…" }` | KV のキー削除＋`revoked:<jti>` を TTL=access残存秒で書き込み | 失効伝搬 ≤ 60 s |

### デバイス ID 生成方針
1. **Keychain に `deviceId` が無ければ** `UUID().uuidString` を生成して保存  
2. `UIDevice.identifierForVendor` は再インストールで変わるため **不採用**  
3. 以後は Keychain の値をそのまま送信

### クライアント保管場所
- **access token**: メモリ or Keychain（どちらでも。15 分ごとに更新）
- **refresh token**: Keychain 専用。アプリ外へ送らない

---

## 🛠️ Build & Local Development

> 環境変数 `JWT_PRIVATE_KEY` / `JWT_PUBLIC_KEY` を `wrangler secret` で登録して下さい。

1. **Prerequisites** (Go ≥ 1.23, wrangler ≥ 4, sqlc, …)
2. `npx wrangler d1 create bulktrack-db`
3. `npx wrangler d1 migrations apply bulktrack-db --local`
4. `sqlc generate -f internal/infrastructure/persistence/d1/sqlc.yaml`
5. `npm start`

---

## 🪵 Logging Strategy (PoC Phase)

- **Goal**: Facilitate error investigation during the Proof of Concept phase.
- **Approach**: Utilize the standard Go `log` package (`log.Printf`, etc.).
    - Logs are automatically directed to Cloudflare Workers' logging system (`wrangler tail`).
    - Focus on logging detailed error information (operation, relevant IDs, error messages) primarily within service and repository layers where errors occur.
    - Minimal INFO level logging for key process steps (e.g., request handling start/end, successful operations).
- **Current Implementation**:
    - Logging middleware (`internal/interface/http/middleware/logging.go`) records basic HTTP request/response info (method, path, status, duration, user-agent).
    - Key handlers, services, and repositories include `log.Printf` calls for errors and significant events.
- **Future Considerations**: Introduce structured logging (e.g., `slog`) and context-based tracing (request IDs) as the application complexity grows.

---

## 🚚 Data Transfer with iOS Transferable

1. 送信端末: `Settings → Data Export` で AirDrop / QR を起動 → Transferable payload (= signed backup JSON) を生成
2. 受信端末: AirDrop/QR 読み込み → `Transferable.import()` → バックアップ JSON を POST `/v1/import` ( Authorization なし )
3. Worker で新しい `device_id` を払い出し、全テーブルの `device_id` を移行後 JWT 発行

これにより **Apple ID に縛られず** シームレスな機種変更を実現。

---

## ✅ Test / CI Pipeline

| Stage | What |
|-------|------|
| **Unit** | Domain logic — table‑driven, `-race -cover` |
| **Use‑Case** | App layer with memory repo DI |
| **Contract** | Dredd / Prism vs OpenAPI |
| **Integration** | GitHub Actions: `wrangler d1` local + Worker |
