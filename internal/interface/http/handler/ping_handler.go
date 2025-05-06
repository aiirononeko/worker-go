package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aiirononeko/bulktrack-api/internal/app/query"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware"
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
	logger := middleware.LoggerFromContext(r.Context())
	logger.Info("Received Ping request", slog.String("path", r.URL.Path))

	result := h.pingService.Execute(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		logger.Error("Failed to encode Ping response", slog.String("path", r.URL.Path), slog.Any("err", err))
	}
	logger.Info("Successfully processed Ping request", slog.String("path", r.URL.Path))
}
