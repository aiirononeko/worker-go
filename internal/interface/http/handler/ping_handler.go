package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/aiirononeko/bulktrack-api/internal/app/query"
)

// PingHandler は /ping エンドポイントのリクエストを処理します。
type PingHandler struct {
	pingService query.PingQueryService
}

// NewPingHandler は PingHandler の新しいインスタンスを生成します。
func NewPingHandler(ps query.PingQueryService) *PingHandler {
	return &PingHandler{pingService: ps}
}

// ServeHTTP は GET /ping リクエストを処理します。
func (h *PingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: Received Ping request. Path: %s", r.URL.Path)

	result := h.pingService.Execute(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("ERROR: Failed to encode Ping response: %v. Path: %s", err, r.URL.Path)
	}
	log.Printf("INFO: Successfully processed Ping request. Path: %s", r.URL.Path)
}
