//go:build js && wasm

package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/syumai/workers/cloudflare/d1"

	"github.com/aiirononeko/bulktrack-api/internal/app/query"
	infra "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/handler"

	"github.com/syumai/workers"
)

const d1BindingName = "DB" // wrangler.jsonc で設定した binding 名

func main() {
	// --- データベース接続 (D1) ---
	db, err := sql.Open("d1", d1BindingName)
	if err != nil {
		log.Fatalf("Failed to sql.Open D1 with driver 'd1' and binding '%s': %v", d1BindingName, err)
	}
	defer db.Close()

	// DB接続をテスト
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping D1 database: %v", err)
	}
	log.Println("Successfully connected to D1 database via binding:", d1BindingName)

	// --- 依存性注入 (DI) ---
	menuRepo := infra.NewMenuRepository(db)
	listMenusService := query.NewListMenusQueryService(menuRepo)
	listMenusHandler := handler.NewListMenusHandler(listMenusService)

	pingService := query.NewPingQueryService()
	pingHandler := handler.NewPingHandler(pingService)

	// --- ルーティング設定 ---
	mux := http.NewServeMux()
	mux.Handle("/ping", pingHandler)
	mux.Handle("/v1/menus", listMenusHandler)
	log.Println("Starting server...")
	workers.Serve(mux)
}
