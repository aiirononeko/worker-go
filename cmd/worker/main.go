package main

import (
	"net/http"

	"github.com/aiirononeko/bulktrack-api/internal/app/query"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/handler"
	"github.com/syumai/workers"
)

func main() {
	// 依存性の注入 (Dependency Injection)
	pingQueryService := query.NewPingQueryService()
	pingHandler := handler.NewPingHandler(pingQueryService)

	http.Handle("/ping", pingHandler)

	workers.Serve(nil) // use http.DefaultServeMux
}
