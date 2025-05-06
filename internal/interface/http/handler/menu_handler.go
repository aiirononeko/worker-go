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
	log.Printf("INFO: Received ListMenus request. Path: %s", r.URL.Path)

	menuDTOs, err := h.listMenusService.Execute(r.Context())
	if err != nil {
		log.Printf("ERROR: Failed to execute ListMenus service: %v. Path: %s", err, r.URL.Path)
		http.Error(w, "Failed to retrieve menus", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(menuDTOs); err != nil {
		log.Printf("ERROR: Failed to encode ListMenus response: %v. Path: %s", err, r.URL.Path)
	}
	log.Printf("INFO: Successfully processed ListMenus request. Path: %s. Returned %d menus.", r.URL.Path, len(menuDTOs))
}
