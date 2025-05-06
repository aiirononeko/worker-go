package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/command"
	"github.com/aiirononeko/bulktrack-api/internal/app/query"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
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

// ErrorResponse is a generic JSON error response body.
type ErrorResponse struct {
	Error   string `json:"error"`             // A high-level error message string.
	Details string `json:"details,omitempty"` // More detailed error information, often the error.Error() string.
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
		sendJSONError(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed, "")
	}
}

// handleListMenus は GET /v1/menus リクエストを処理します。
func (h *MenuHandler) handleListMenus(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: Received ListMenus request. Path: %s", r.URL.Path)

	ctxUID, ok := middleware.GetDeviceIDFromContext(r.Context())
	if !ok {
		log.Printf("ERROR: UID not found in context for ListMenus. Path: %s", r.URL.Path)
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized, "UID not found in context")
		return
	}

	if !strings.HasPrefix(ctxUID, "device:") {
		log.Printf("ERROR: ListMenus currently only supports device context, got UID: %s. Path: %s", ctxUID, r.URL.Path)
		sendJSONError(w, "Bad Request", http.StatusBadRequest, "Operation not supported for this token type")
		return
	}
	stringPureDeviceID := strings.TrimPrefix(ctxUID, "device:")

	// Convert string to entity.DeviceID
	domainDeviceID, err := entity.NewDeviceID(stringPureDeviceID)
	if err != nil {
		log.Printf("ERROR: Parsed pureDeviceID '%s' from UID '%s' is not a valid UUID: %v. Path: %s", stringPureDeviceID, ctxUID, err, r.URL.Path)
		sendJSONError(w, "Bad Request", http.StatusBadRequest, fmt.Sprintf("Invalid device identifier format in token: %s. Error: %v", stringPureDeviceID, err))
		return
	}

	returnedDTOs, err := h.listMenusQuery.Execute(r.Context(), domainDeviceID)
	if err != nil {
		log.Printf("ERROR: Failed to execute ListMenus query for DeviceID %s: %v. Path: %s", domainDeviceID.String(), err, r.URL.Path)
		sendJSONError(w, "Internal Server Error", http.StatusInternalServerError, "Failed to retrieve menus")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(returnedDTOs); err != nil {
		log.Printf("ERROR: Failed to encode ListMenus response: %v. Path: %s", err, r.URL.Path)
	}
	log.Printf("INFO: Successfully processed ListMenus request for DeviceID %s. Path: %s. Returned %d menus.", domainDeviceID.String(), r.URL.Path, len(returnedDTOs))
}

// handleCreateMenu は POST /v1/menus リクエストを処理します。
func (h *MenuHandler) handleCreateMenu(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: Received CreateMenu request. Path: %s", r.URL.Path)

	deviceIDWithPrefix, ok := middleware.GetDeviceIDFromContext(r.Context())
	if !ok {
		log.Printf("ERROR: Device ID not found in context for CreateMenu. Path: %s", r.URL.Path)
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized, "Device ID not found in context")
		return
	}

	var req CreateMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ERROR: Failed to decode CreateMenu request body: %v. Path: %s", err, r.URL.Path)
		sendJSONError(w, "Bad Request", http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.Name == "" {
		log.Printf("WARN: Validation failed for CreateMenu: Name is required. Path: %s", r.URL.Path)
		sendJSONError(w, "Bad Request", http.StatusBadRequest, "Name is required")
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
		log.Printf("ERROR: CreateMenu command failed for menu name '%s': %v. Path: %s", req.Name, err, r.URL.Path)
		var valErr *command.ErrValidation
		var conflictErr *command.ErrMenuNameConflict

		if errors.As(err, &valErr) {
			sendJSONError(w, "Bad Request", http.StatusBadRequest, valErr.Error())
		} else if errors.As(err, &conflictErr) {
			sendJSONError(w, "Conflict", http.StatusConflict, conflictErr.Error())
		} else {
			sendJSONError(w, "Internal Server Error", http.StatusInternalServerError, "Failed to create menu")
		}
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(respDTO); err != nil {
		log.Printf("ERROR: Failed to encode CreateMenu response: %v. Path: %s", err, r.URL.Path)
	}
	log.Printf("INFO: Successfully processed CreateMenu request for menu name '%s'. Path: %s. Created Menu ID: %s", req.Name, r.URL.Path, respDTO.ID)
}

// sendJSONError はJSON形式でエラーレスポンスを送信するヘルパー関数です。
func sendJSONError(w http.ResponseWriter, message string, statusCode int, details string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Error:   message,
		Details: details,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback if JSON encoding fails, though client might have already received headers.
		// Log the original error and the encoding error.
		log.Printf("ERROR: Original error was for status %d, message: %s, details: %s. Additionally, failed to encode this error to JSON: %v", statusCode, message, details, err)
		// Avoid writing again if headers are sent, but if not, http.Error could be a last resort.
		// However, the WriteHeader above likely committed the headers.
	}
}
