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
	httpRouter "github.com/aiirononeko/bulktrack-api/internal/interface/http"
	appHttpHandler "github.com/aiirononeko/bulktrack-api/internal/interface/http/handler"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware"

	"github.com/syumai/workers"
)

const (
	d1BindingName             = "DB"
	refreshTokenKVBindingName = "REFRESH_TOKENS_KV"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := sql.Open("d1", d1BindingName)
	if err != nil {
		log.Fatalf("Failed to sql.Open D1: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping D1: %v", err)
	}
	log.Println("Successfully connected to D1 database via binding:", d1BindingName)

	refreshTokenKV, err := kv.NewNamespace(refreshTokenKVBindingName)
	if err != nil {
		log.Fatalf("Failed to get KV namespace: %v", err)
	}
	log.Printf("Successfully bound to KV namespace: %s", refreshTokenKVBindingName)

	menuRepo := infraD1.NewMenuRepository(db)
	deviceRepo := infraD1.NewD1DeviceRepository(db)
	refreshTokenRepo := infraKV.NewKVRefreshTokenRepository(*refreshTokenKV)

	jwtService, err := infraAuth.NewJWTService(cfg.JWTPrivateKeyPEM, cfg.JWTPublicKeyPEM, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		log.Fatalf("Failed to initialize JWT service: %v", err)
	}

	activateDeviceHandler := appCmd.NewActivateDeviceHandler(deviceRepo, jwtService, refreshTokenRepo, cfg.RefreshTokenTTL)
	refreshTokenHandler := appCmd.NewRefreshTokenHandler(jwtService, refreshTokenRepo, cfg.RefreshTokenTTL)
	logoutHandler := appCmd.NewLogoutHandler(jwtService, refreshTokenRepo)
	listMenusService := appQuery.NewListMenusQueryService(menuRepo)
	pingService := appQuery.NewPingQueryService()

	listMenusHttpHandler := appHttpHandler.NewListMenusHandler(listMenusService)
	pingHttpHandler := appHttpHandler.NewPingHandler(pingService)
	authHttpHandler := appHttpHandler.NewAuthHandler(activateDeviceHandler, refreshTokenHandler, logoutHandler)

	// --- ミドルウェアの定義 ---
	loggingMiddlewareFunc := middleware.LoggingMiddleware
	corsMiddlewareFunc := middleware.CORS
	authMiddlewareFunc := middleware.RequireAuth(jwtService)

	globalMiddlewares := []middleware.Middleware{
		loggingMiddlewareFunc,
		corsMiddlewareFunc,
	}

	// --- ルートの定義 ---
	routes := []httpRouter.Route{
		{
			Path:        "/v1/menus",
			Handler:     listMenusHttpHandler,
			Middlewares: []middleware.Middleware{authMiddlewareFunc},
		},
	}

	// ルーターの依存関係を設定
	routerDeps := httpRouter.RouterDependencies{
		Routes:            routes,
		GlobalMiddlewares: globalMiddlewares,
		AuthHandler:       authHttpHandler,
		PingHandler:       pingHttpHandler,
		ListMenusHandler:  listMenusHttpHandler,
	}

	finalRouter := httpRouter.NewRouter(routerDeps)

	log.Println("Starting server with new router and middleware configuration...")
	workers.Serve(finalRouter)
}
