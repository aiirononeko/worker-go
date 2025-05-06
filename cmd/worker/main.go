//go:build js && wasm

package main

import (
	"database/sql"
	"log/slog"
	"os"

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

	db "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1/sql" // sqlc generated code
	"github.com/go-playground/validator/v10"
	"github.com/syumai/workers"
)

const (
	d1BindingName             = "DB"
	refreshTokenKVBindingName = "REFRESH_TOKENS_KV"
)

func main() {
	// Setup structured logger (slog)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger) // Set as default logger for the application

	// Replace standard log fatal with slog error and exit
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic recovered", slog.Any("recover_info", r))
			// Optionally re-panic or os.Exit(1)
		}
	}()

	cfg, err := config.Load()
	if err != nil {
		// Use slog for fatal errors now
		slog.Error("Failed to load config", slog.Any("error", err))
		os.Exit(1) // Exit after logging fatal error
	}

	dbConn, err := sql.Open("d1", d1BindingName)
	if err != nil {
		slog.Error("Failed to sql.Open D1", slog.Any("error", err))
		os.Exit(1)
	}
	defer dbConn.Close()
	if err := dbConn.Ping(); err != nil {
		slog.Error("Failed to ping D1", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Successfully connected to D1 database", slog.String("binding", d1BindingName))

	refreshTokenKV, err := kv.NewNamespace(refreshTokenKVBindingName)
	if err != nil {
		slog.Error("Failed to get KV namespace", slog.String("binding", refreshTokenKVBindingName), slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Successfully bound to KV namespace", slog.String("binding", refreshTokenKVBindingName))

	validate := validator.New()

	// --- Repositories ---
	menuRepo := infraD1.NewMenuRepository(dbConn)
	deviceRepo := infraD1.NewD1DeviceRepository(dbConn)
	refreshTokenRepo := infraKV.NewKVRefreshTokenRepository(*refreshTokenKV)
	workoutRepo := infraD1.NewD1WorkoutRepository(dbConn)

	// SQLC Querier (implements query methods)
	querier := db.New(dbConn)

	// --- Services & Handlers ---
	jwtService, err := infraAuth.NewJWTService(cfg.JWTPrivateKeyPEM, cfg.JWTPublicKeyPEM, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		slog.Error("Failed to initialize JWT service", slog.Any("error", err))
		os.Exit(1)
	}

	// Application Layer Handlers/Services
	activateDeviceHandler := appCmd.NewActivateDeviceHandler(deviceRepo, jwtService, refreshTokenRepo, cfg.RefreshTokenTTL)
	refreshTokenHandler := appCmd.NewRefreshTokenHandler(jwtService, refreshTokenRepo, cfg.RefreshTokenTTL)
	logoutHandler := appCmd.NewLogoutHandler(jwtService, refreshTokenRepo)
	listMenusQuery := appQuery.NewListMenusQueryService(menuRepo)
	createMenuCmdHandler := appCmd.NewCreateMenuHandler(menuRepo)
	pingService := appQuery.NewPingQueryService()
	createWorkoutHandler := appCmd.NewCreateWorkoutHandler(workoutRepo)
	dashboardQueryService := appQuery.NewDashboardQueryService(querier)

	// HTTP Handlers
	menuHttpHandler := appHttpHandler.NewMenuHandler(listMenusQuery, createMenuCmdHandler)
	pingHttpHandler := appHttpHandler.NewPingHandler(pingService)
	authHttpHandler := appHttpHandler.NewAuthHandler(activateDeviceHandler, refreshTokenHandler, logoutHandler)
	workoutHttpHandler := appHttpHandler.NewWorkoutHandler(createWorkoutHandler, validate)
	dashboardHttpHandler := appHttpHandler.NewDashboardHandler(dashboardQueryService)

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
		WorkoutHandler:        workoutHttpHandler,
		DashboardHandler:      dashboardHttpHandler,
		RequireAuthMiddleware: authMiddlewareFunc,
	}

	finalRouter := httpRouter.NewRouter(routerDeps)

	slog.Info("Starting server", slog.String("config_source", "env/defaults"))
	workers.Serve(finalRouter)
}
