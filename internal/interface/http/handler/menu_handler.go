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
	// --- 認証: リクエストコンテキストから UserID を取得 ---
	// userID, ok := middleware.GetUserIDFromContext(r.Context()) // 仮の関数
	// if !ok {
	// 	http.Error(w, "Unauthorized: User ID not found in context", http.StatusUnauthorized)
	// 	return
	// }
	// ★★★ 実際のユーザーID取得方法は既存の認証実装に合わせる必要があります ★★★
	// ↓↓↓ とりあえずテスト用に固定のIDを使う例 ↓↓↓
	userID := "user_test_123" // TODO: 実際の認証から取得する

	// --- サービスの呼び出し ---
	menuDTOs, err := h.listMenusService.Execute(r.Context(), userID)
	if err != nil {
		// エラーロギング
		log.Printf("Error fetching menus for user %s: %v", userID, err) // 例
		// TODO: エラーの種類に応じてステータスコードを変える (例: Not Found など)
		http.Error(w, "Failed to retrieve menus", http.StatusInternalServerError)
		return
	}

	// --- 成功レスポンス (JSON) ---
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(menuDTOs); err != nil {
		// JSONエンコード失敗時のエラーハンドリング (クライアントには Internal Server Error を返すことが多い)
		log.Printf("Error encoding menus to JSON for user %s: %v", userID, err)
		// http.Error は既にヘッダが書き込まれていると警告を出す場合があるので注意
		// ここでは致命的なエラーとしてログに残す程度が良いかもしれない
		return
	}
}
