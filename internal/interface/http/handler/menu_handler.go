package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/aiirononeko/bulktrack-api/internal/app/query" // Application レイヤーのサービスIF
)

// ListMenusHandler はメニュー一覧取得リクエストを処理します。
type ListMenusHandler struct {
	listMenusService query.ListMenusQueryService
}

// NewListMenusHandler は ListMenusHandler の新しいインスタンスを生成します。
func NewListMenusHandler(lms query.ListMenusQueryService) *ListMenusHandler {
	return &ListMenusHandler{
		listMenusService: lms,
	}
}

// ServeHTTP は GET /v1/menus リクエストを処理します。
func (h *ListMenusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// --- サービスの呼び出し ---
	menuDTOs, err := h.listMenusService.Execute(r.Context())
	if err != nil {
		// エラーロギング
		// TODO: エラーの種類に応じてステータスコードを変える (例: Not Found など)
		// サービス層で返されるエラーが fmt.Errorf でラップされているため、
		// errors.Is や errors.As を使って特定のドメインエラーやアプリケーションエラーを判定できる。
		log.Printf("Error fetching menus: %v", err)
		http.Error(w, "Failed to retrieve menus", http.StatusInternalServerError)
		return
	}

	// --- 成功レスポンス (JSON) ---
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(menuDTOs); err != nil {
		log.Printf("Error encoding menus to JSON: %v", err)
		return
	}
}
