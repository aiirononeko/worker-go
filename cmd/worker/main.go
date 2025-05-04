//go:build js && wasm

package main

import (
	"database/sql"
	"log"

	_ "github.com/syumai/workers/cloudflare/d1"

	"github.com/aiirononeko/bulktrack-api/config"                      // 設定パッケージを import
	appCmd "github.com/aiirononeko/bulktrack-api/internal/app/command" // Command パッケージ
	appQuery "github.com/aiirononeko/bulktrack-api/internal/app/query"
	infraAuth "github.com/aiirononeko/bulktrack-api/internal/infrastructure/auth" // Infra Auth パッケージ
	infraD1 "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1"
	appRouter "github.com/aiirononeko/bulktrack-api/internal/interface/http" // エイリアス appRouter を使用
	appHttp "github.com/aiirononeko/bulktrack-api/internal/interface/http/handler"

	"github.com/syumai/workers"
)

const d1BindingName = "DB" // wrangler.jsonc で設定した binding 名

func main() {
	// --- 設定読み込み ---
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// JWTSecretKey が空でないことは config.Load 内でチェック済み

	// --- データベース接続 (D1) ---
	db, err := sql.Open("d1", d1BindingName)
	if err != nil {
		log.Fatalf("Failed to sql.Open D1 with driver 'd1' and binding '%s': %v", d1BindingName, err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping D1 database: %v", err)
	}
	log.Println("Successfully connected to D1 database via binding:", d1BindingName)

	// --- 依存性注入 (DI) ---

	// Repositories
	menuRepo := infraD1.NewMenuRepository(db)
	deviceRepo := infraD1.NewD1DeviceRepository(db) // Device Repository を初期化

	// Services
	jwtService, err := infraAuth.NewJWTService(cfg.JWTPrivateKeyPEM, cfg.JWTPublicKeyPEM, cfg.AccessTokenTTL, cfg.RefreshTokenTTL) // EdDSA Service を初期化
	if err != nil {
		log.Fatalf("Failed to initialize JWT service: %v", err)
	}

	// Application Handlers (Commands & Queries)
	activateDeviceHandler := appCmd.NewActivateDeviceHandler(deviceRepo, jwtService) // ActivateDevice Handler を初期化
	listMenusService := appQuery.NewListMenusQueryService(menuRepo)
	pingService := appQuery.NewPingQueryService()

	// Interface Handlers (HTTP)
	listMenusHandler := appHttp.NewListMenusHandler(listMenusService)
	pingHandler := appHttp.NewPingHandler(pingService)
	authHandler := appHttp.NewAuthHandler(activateDeviceHandler) // Auth Handler を初期化

	// Router を初期化して取得
	routerDeps := appRouter.RouterDependencies{ // エイリアスを使用
		AuthHandler:      authHandler,
		PingHandler:      pingHandler,
		ListMenusHandler: listMenusHandler,
	}
	router := appRouter.NewRouter(routerDeps) // エイリアスを使用

	log.Println("Starting server...")
	workers.Serve(router) // mux の代わりに router を渡す
}
