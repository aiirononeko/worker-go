package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/command"
	"github.com/aiirononeko/bulktrack-api/internal/app/query"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware"
)

// --- DTOs (common for Menu endpoints) --- //

// CreateMenuRequest mirrors the OpenAPI schema `CreateMenuRequest`
type CreateMenuRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	SortOrder   int     `json:"sort_order"`
}

// MenuDTO mirrors the OpenAPI schema `MenuDTO` for the response
type MenuDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// --- Menu Handler (Handles multiple methods for /v1/menus) --- //

// MenuHandler は /v1/menus に対するリクエストを処理します。
type MenuHandler struct {
	listMenusQuery       query.ListMenusQueryService // Changed name for clarity
	createMenuCmdHandler *command.CreateMenuHandler
}

// NewMenuHandler は MenuHandler の新しいインスタンスを生成します。
func NewMenuHandler(lms query.ListMenusQueryService, cmdHandler *command.CreateMenuHandler) *MenuHandler {
	return &MenuHandler{
		listMenusQuery:       lms,
		createMenuCmdHandler: cmdHandler,
	}
}

// ServeHTTP は /v1/menus へのリクエストをメソッドに応じて処理します。
func (h *MenuHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListMenus(w, r)
	case http.MethodPost:
		h.handleCreateMenu(w, r)
	default:
		log.Printf("WARN: Method not allowed for %s: %s", r.URL.Path, r.Method)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

// handleListMenus は GET /v1/menus リクエストを処理します。
func (h *MenuHandler) handleListMenus(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: Received ListMenus request. Path: %s", r.URL.Path)

	// ★ Note: query.ListMenusQueryService should ideally take deviceID explicitly
	// For now, it might be getting it from context internally, which couples layers.
	menuDomainModels, err := h.listMenusQuery.Execute(r.Context()) // Assume it returns domain models now
	if err != nil {
		log.Printf("ERROR: Failed to execute ListMenus query: %v. Path: %s", err, r.URL.Path)
		http.Error(w, "Failed to retrieve menus", http.StatusInternalServerError)
		return
	}

	// Map domain models to DTOs
	menuDTOs := make([]MenuDTO, 0, len(menuDomainModels))
	for _, m := range menuDomainModels {
		menuDTOs = append(menuDTOs, MenuDTO{
			ID:          m.ID.String(),
			Name:        m.Name,
			Description: m.Description,
			SortOrder:   m.SortOrder,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(menuDTOs); err != nil {
		log.Printf("ERROR: Failed to encode ListMenus response: %v. Path: %s", err, r.URL.Path)
	}
	log.Printf("INFO: Successfully processed ListMenus request. Path: %s. Returned %d menus.", r.URL.Path, len(menuDTOs))
}

// handleCreateMenu は POST /v1/menus リクエストを処理します。
func (h *MenuHandler) handleCreateMenu(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: Received CreateMenu request. Path: %s", r.URL.Path)

	// 1. Get DeviceID from context (set by RequireAuth middleware)
	deviceID, ok := middleware.GetDeviceIDFromContext(r.Context())
	if !ok {
		log.Printf("ERROR: Device ID not found in context for CreateMenu. Path: %s", r.URL.Path)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// 2. Decode request body
	var req CreateMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ERROR: Failed to decode CreateMenu request body: %v. Path: %s", err, r.URL.Path)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 3. Basic Validation
	if req.Name == "" {
		log.Printf("WARN: Validation failed for CreateMenu: Name is required. Path: %s", r.URL.Path)
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// 4. Create and execute the application command
	cmd := command.CreateMenuCommand{
		DeviceID:    deviceID,
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	createdMenu, err := h.createMenuCmdHandler.Handle(r.Context(), cmd)
	if err != nil {
		log.Printf("ERROR: Failed to execute CreateMenu command: %v. Path: %s", err, r.URL.Path)
		http.Error(w, "Failed to create menu", http.StatusInternalServerError)
		return
	}

	// 5. Map domain entity to DTO for response
	respDTO := MenuDTO{
		ID:          createdMenu.ID.String(),
		Name:        createdMenu.Name,
		Description: createdMenu.Description,
		SortOrder:   createdMenu.SortOrder,
		CreatedAt:   createdMenu.CreatedAt,
		UpdatedAt:   createdMenu.UpdatedAt,
	}

	// 6. Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	if err := json.NewEncoder(w).Encode(respDTO); err != nil {
		log.Printf("ERROR: Failed to encode CreateMenu response: %v. Path: %s", err, r.URL.Path)
	}
	log.Printf("INFO: Successfully processed CreateMenu request. Path: %s. Created Menu ID: %s", r.URL.Path, respDTO.ID)
}
