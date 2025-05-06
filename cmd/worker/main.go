//go:build js && wasm

package main

import (
	"database/sql"
	"log"

	_ "github.com/syumai/workers/cloudflare/d1"
	"github.com/syumai/workers/cloudflare/kv"

	"github.com/aiirononeko/bulktrack-api/config"
	appCmd "github.com/aiirononeko/bulktrack-api/internal/app/command"
	appQuery "github.com/aiirononeko/bulktrack-api/internal/app/query"
	infraAuth "github.com/aiirononeko/bulktrack-api/internal/infrastructure/auth"
	infraD1 "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1"
	infraKV "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/kv"
	appRouter "github.com/aiirononeko/bulktrack-api/internal/interface/http"
	appHttp "github.com/aiirononeko/bulktrack-api/internal/interface/http/handler"

	"github.com/syumai/workers"
)

const (
	d1BindingName             = "DB"
	refreshTokenKVBindingName = "REFRESH_TOKENS_KV"
)

func main() {
	// --- 設定読み込み ---
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

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

	// --- KV 名前空間バインディング取得 ---
	refreshTokenKV, err := kv.NewNamespace(refreshTokenKVBindingName)
	if err != nil {
		log.Fatalf("Failed to get KV namespace binding '%s': %v", refreshTokenKVBindingName, err)
	}
	log.Printf("Successfully bound to KV namespace: %s", refreshTokenKVBindingName)

	// --- 依存性注入 (DI) ---

	// Repositories
	menuRepo := infraD1.NewMenuRepository(db)
	deviceRepo := infraD1.NewD1DeviceRepository(db)
	refreshTokenRepo := infraKV.NewKVRefreshTokenRepository(*refreshTokenKV) // ポインタをデリファレンスして値を渡す

	// Services
	jwtService, err := infraAuth.NewJWTService(cfg.JWTPrivateKeyPEM, cfg.JWTPublicKeyPEM, cfg.AccessTokenTTL, cfg.RefreshTokenTTL) // EdDSA Service を初期化
	if err != nil {
		log.Fatalf("Failed to initialize JWT service: %v", err)
	}

	// Application Handlers (Commands & Queries)
	activateDeviceHandler := appCmd.NewActivateDeviceHandler(deviceRepo, jwtService, refreshTokenRepo, cfg.RefreshTokenTTL)
	refreshTokenHandler := appCmd.NewRefreshTokenHandler(jwtService, refreshTokenRepo, cfg.RefreshTokenTTL)
	logoutHandler := appCmd.NewLogoutHandler(jwtService, refreshTokenRepo) // LogoutHandler を初期化
	listMenusService := appQuery.NewListMenusQueryService(menuRepo)
	pingService := appQuery.NewPingQueryService()

	// Interface Handlers (HTTP)
	listMenusHandler := appHttp.NewListMenusHandler(listMenusService)
	pingHandler := appHttp.NewPingHandler(pingService)
	authHandler := appHttp.NewAuthHandler(activateDeviceHandler, refreshTokenHandler, logoutHandler) // Auth Handler に logoutHandler を渡す

	// Router を初期化して取得
	routerDeps := appRouter.RouterDependencies{
		AuthHandler:      authHandler,
		PingHandler:      pingHandler,
		ListMenusHandler: listMenusHandler,
	}
	router := appRouter.NewRouter(routerDeps)

	log.Println("Starting server...")
	workers.Serve(router) // mux の代わりに router を渡す
}
