package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/command"
	"github.com/aiirononeko/bulktrack-api/internal/app/query"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware"
	"github.com/google/uuid"
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
	listMenusQuery       query.ListMenusQueryService
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
	log.Printf("DEBUG: MenuHandler.ServeHTTP called. Perceived Method: [%s], Path: [%s]", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodGet:
		log.Printf("DEBUG: Routing to handleListMenus for method [%s]", r.Method)
		h.handleListMenus(w, r)
	case http.MethodPost:
		log.Printf("DEBUG: Routing to handleCreateMenu for method [%s]", r.Method)
		h.handleCreateMenu(w, r)
	default:
		log.Printf("WARN: Method [%s] not allowed for path [%s]", r.Method, r.URL.Path)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

// handleListMenus は GET /v1/menus リクエストを処理します。
func (h *MenuHandler) handleListMenus(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: Received ListMenus request. Path: %s", r.URL.Path)

	ctxUID, ok := middleware.GetDeviceIDFromContext(r.Context())
	if !ok {
		log.Printf("ERROR: UID not found in context for ListMenus. Path: %s", r.URL.Path)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if !strings.HasPrefix(ctxUID, "device:") {
		log.Printf("ERROR: ListMenus currently only supports device context, got UID: %s. Path: %s", ctxUID, r.URL.Path)
		http.Error(w, "Operation not supported for this token type", http.StatusBadRequest)
		return
	}
	pureDeviceID := strings.TrimPrefix(ctxUID, "device:")
	if pureDeviceID == "" {
		log.Printf("ERROR: Empty pureDeviceID after trimming prefix from UID: %s. Path: %s", ctxUID, r.URL.Path)
		http.Error(w, "Invalid device identifier in token", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(pureDeviceID); err != nil {
		log.Printf("ERROR: Parsed pureDeviceID '%s' from UID '%s' is not a valid UUID: %v. Path: %s", pureDeviceID, ctxUID, err, r.URL.Path)
		http.Error(w, "Invalid device identifier format in token", http.StatusBadRequest)
		return
	}

	returnedDTOs, err := h.listMenusQuery.Execute(r.Context(), pureDeviceID)
	if err != nil {
		log.Printf("ERROR: Failed to execute ListMenus query for DeviceID %s: %v. Path: %s", pureDeviceID, err, r.URL.Path)
		http.Error(w, "Failed to retrieve menus", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(returnedDTOs); err != nil {
		log.Printf("ERROR: Failed to encode ListMenus response: %v. Path: %s", err, r.URL.Path)
	}
	log.Printf("INFO: Successfully processed ListMenus request for DeviceID %s. Path: %s. Returned %d menus.", pureDeviceID, r.URL.Path, len(returnedDTOs))
}

// handleCreateMenu は POST /v1/menus リクエストを処理します。
func (h *MenuHandler) handleCreateMenu(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: Received CreateMenu request. Path: %s", r.URL.Path)

	deviceIDWithPrefix, ok := middleware.GetDeviceIDFromContext(r.Context())
	if !ok {
		log.Printf("ERROR: Device ID not found in context for CreateMenu. Path: %s", r.URL.Path)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var req CreateMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ERROR: Failed to decode CreateMenu request body: %v. Path: %s", err, r.URL.Path)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		log.Printf("WARN: Validation failed for CreateMenu: Name is required. Path: %s", r.URL.Path)
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	cmd := command.CreateMenuCommand{
		DeviceID:    deviceIDWithPrefix,
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

	respDTO := MenuDTO{
		ID:          createdMenu.ID.String(),
		Name:        createdMenu.Name,
		Description: createdMenu.Description,
		SortOrder:   createdMenu.SortOrder,
		CreatedAt:   createdMenu.CreatedAt,
		UpdatedAt:   createdMenu.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(respDTO); err != nil {
		log.Printf("ERROR: Failed to encode CreateMenu response: %v. Path: %s", err, r.URL.Path)
	}
	log.Printf("INFO: Successfully processed CreateMenu request. Path: %s. Created Menu ID: %s", r.URL.Path, respDTO.ID)
}
