package handler

import (
	"net/http"

	"github.com/aiirononeko/bulktrack-api/internal/app/query"
)

// PingHandler は /ping エンドポイントのリクエストを処理します。
type PingHandler struct {
	pingQueryService query.PingQueryService // Application レイヤーのサービスに依存
}

// NewPingHandler は PingHandler の新しいインスタンスを生成します。
func NewPingHandler(pingQueryService query.PingQueryService) *PingHandler {
	return &PingHandler{pingQueryService: pingQueryService}
}

func (h *PingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// サービスメソッドを呼び出す (Context を渡す)
	// r.Context() でリクエストに紐づく Context を取得できる
	msg, err := h.pingQueryService.Ping(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// レスポンスを書き込む
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(msg))
}
