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

	dbConn, err := sql.Open("d1", d1BindingName)
	if err != nil {
		log.Fatalf("Failed to sql.Open D1: %v", err)
	}
	defer dbConn.Close()
	if err := dbConn.Ping(); err != nil {
		log.Fatalf("Failed to ping D1: %v", err)
	}
	log.Println("Successfully connected to D1 database via binding:", d1BindingName)

	refreshTokenKV, err := kv.NewNamespace(refreshTokenKVBindingName)
	if err != nil {
		log.Fatalf("Failed to get KV namespace: %v", err)
	}
	log.Printf("Successfully bound to KV namespace: %s", refreshTokenKVBindingName)

	menuRepo := infraD1.NewMenuRepository(dbConn)
	deviceRepo := infraD1.NewD1DeviceRepository(dbConn)
	refreshTokenRepo := infraKV.NewKVRefreshTokenRepository(*refreshTokenKV)

	jwtService, err := infraAuth.NewJWTService(cfg.JWTPrivateKeyPEM, cfg.JWTPublicKeyPEM, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		log.Fatalf("Failed to initialize JWT service: %v", err)
	}

	// --- Application Layer Handlers/Services ---
	activateDeviceHandler := appCmd.NewActivateDeviceHandler(deviceRepo, jwtService, refreshTokenRepo, cfg.RefreshTokenTTL)
	refreshTokenHandler := appCmd.NewRefreshTokenHandler(jwtService, refreshTokenRepo, cfg.RefreshTokenTTL)
	logoutHandler := appCmd.NewLogoutHandler(jwtService, refreshTokenRepo)
	listMenusQuery := appQuery.NewListMenusQueryService(menuRepo)
	createMenuCmdHandler := appCmd.NewCreateMenuHandler(menuRepo)
	pingService := appQuery.NewPingQueryService()

	// --- HTTP Handlers ---
	menuHttpHandler := appHttpHandler.NewMenuHandler(listMenusQuery, createMenuCmdHandler)
	pingHttpHandler := appHttpHandler.NewPingHandler(pingService)
	authHttpHandler := appHttpHandler.NewAuthHandler(activateDeviceHandler, refreshTokenHandler, logoutHandler)

	// --- Middlewares ---
	loggingMiddlewareFunc := middleware.LoggingMiddleware
	corsMiddlewareFunc := middleware.CORS
	authMiddlewareFunc := middleware.RequireAuth(jwtService)

	globalMiddlewares := []middleware.Middleware{
		loggingMiddlewareFunc,
		corsMiddlewareFunc,
	}

	// --- Routes Definition (Use RouterDependencies for main routes now) ---
	routes := []httpRouter.Route{
		// Other custom routes can be defined here if needed
		// {
		// 	Path:        "/v1/some_other_path",
		// 	Handler:     someOtherHandler,
		// 	Middlewares: []middleware.Middleware{authMiddlewareFunc},
		// },
	}

	// --- Configure Router Dependencies ---
	routerDeps := httpRouter.RouterDependencies{
		Routes:                routes,
		GlobalMiddlewares:     globalMiddlewares,
		AuthHandler:           authHttpHandler,
		PingHandler:           pingHttpHandler,
		MenuHandler:           menuHttpHandler,
		RequireAuthMiddleware: authMiddlewareFunc,
	}

	finalRouter := httpRouter.NewRouter(routerDeps)

	log.Println("Starting server with sqlc-based device repository...")
	workers.Serve(finalRouter)
}
