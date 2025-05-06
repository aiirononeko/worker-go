package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
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

// ErrorResponse struct and sendJSONError func are now in response_util.go

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
	ctx := r.Context()
	logger := middleware.LoggerFromContext(ctx)

	r = r.WithContext(context.WithValue(ctx, middleware.LoggerKey, logger))

	logger.DebugContext(ctx, "MenuHandler ServeHTTP called", slog.String("method", r.Method), slog.String("path", r.URL.Path))
	switch r.Method {
	case http.MethodGet:
		logger.DebugContext(ctx, "Routing to list handler")
		h.list(w, r)
	case http.MethodPost:
		logger.DebugContext(ctx, "Routing to create handler")
		h.create(w, r)
	default:
		logger.WarnContext(ctx, "Method not allowed for menu handler", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		SendJSONError(w, logger, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed, "")
	}
}

// list は GET /v1/menus リクエストを処理します。
func (h *MenuHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFromContext(ctx)
	logger.InfoContext(ctx, "Processing list menus request")

	uid, ok := middleware.GetDeviceIDFromContext(ctx)
	if !ok {
		err := apperror.NewErrUnauthorized("UID not found in context")
		logger.WarnContext(ctx, "Authorization error in list menus", slog.Any("error", err))
		SendJSONError(w, logger, err.Error(), http.StatusUnauthorized, "")
		return
	}

	logger = logger.With(slog.String("uid", uid))

	if !strings.HasPrefix(uid, "device:") {
		err := apperror.NewErrForbidden("Operation not supported for this token type for list menus")
		logger.WarnContext(ctx, "Forbidden error in list menus", slog.Any("error", err))
		SendJSONError(w, logger, err.Error(), http.StatusForbidden, "")
		return
	}
	stringPureDeviceID := strings.TrimPrefix(uid, "device:")
	domainDeviceID, err := entity.NewDeviceID(stringPureDeviceID)
	if err != nil {
		appErr := apperror.NewErrBadRequest("Invalid device identifier in token", err.Error())
		logger.WarnContext(ctx, "Failed to parse pure DeviceID from UID for list menus", slog.Any("error", appErr))
		SendJSONError(w, logger, appErr.Error(), http.StatusBadRequest, appErr.Details)
		return
	}

	logger = logger.With(slog.String("device_id", domainDeviceID.String()))

	returnedDTOs, queryErr := h.listMenusQuery.Execute(ctx, domainDeviceID)
	if queryErr != nil {
		var nfErr *apperror.ErrNotFound
		var internalErr *apperror.ErrInternal

		if errors.As(queryErr, &nfErr) {
			logger.InfoContext(ctx, "ListMenus query returned not found", slog.Any("error", queryErr))
			SendJSONError(w, logger, nfErr.Error(), http.StatusNotFound, "")
		} else if errors.As(queryErr, &internalErr) {
			logger.ErrorContext(ctx, "Internal error from menu repository while listing menus", slog.Any("error", queryErr))
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to retrieve menus")
		} else {
			logger.ErrorContext(ctx, "Unexpected error from menu repository while listing menus", slog.Any("original_error", queryErr.Error()))
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to retrieve menus")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(returnedDTOs); err != nil {
		logger.ErrorContext(ctx, "Failed to encode list menus response", slog.Any("error", err))
	}
	logger.InfoContext(ctx, "Successfully processed list menus request", slog.Int("menu_count", len(returnedDTOs)))
}

// create は POST /v1/menus リクエストを処理します。
func (h *MenuHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFromContext(ctx)
	logger.InfoContext(ctx, "Processing create menu request")

	uid, ok := middleware.GetDeviceIDFromContext(ctx)
	if !ok {
		err := apperror.NewErrUnauthorized("UID not found in context")
		logger.WarnContext(ctx, "Authorization error in create menu", slog.Any("error", err))
		SendJSONError(w, logger, err.Error(), http.StatusUnauthorized, "")
		return
	}
	logger = logger.With(slog.String("uid", uid))

	var req CreateMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := apperror.NewErrBadRequest("Invalid request body", err.Error())
		logger.WarnContext(ctx, "Failed to decode request body for create menu", slog.Any("error", appErr))
		SendJSONError(w, logger, appErr.Error(), http.StatusBadRequest, appErr.Details)
		return
	}

	cmd := command.CreateMenuCommand{
		DeviceID:    uid,
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	createdMenu, cmdErr := h.createMenuCmdHandler.Handle(ctx, cmd)
	if cmdErr != nil {
		var valErr *command.ErrValidation
		var conflictErr *command.ErrMenuNameConflict
		var appNotFoundErr *apperror.ErrNotFound
		var appUnauthorizedErr *apperror.ErrUnauthorized
		var appForbiddenErr *apperror.ErrForbidden

		if errors.As(cmdErr, &valErr) {
			logger.WarnContext(ctx, "CreateMenu validation failed", slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, valErr.Error())
		} else if errors.As(cmdErr, &conflictErr) {
			logger.WarnContext(ctx, "CreateMenu conflict detected", slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Conflict", http.StatusConflict, conflictErr.Error())
		} else if errors.As(cmdErr, &appUnauthorizedErr) {
			logger.WarnContext(ctx, "CreateMenu unauthorized by application logic", slog.Any("error", cmdErr))
			SendJSONError(w, logger, appUnauthorizedErr.Error(), http.StatusUnauthorized, "")
		} else if errors.As(cmdErr, &appForbiddenErr) {
			logger.WarnContext(ctx, "CreateMenu forbidden by application logic", slog.Any("error", cmdErr))
			SendJSONError(w, logger, appForbiddenErr.Error(), http.StatusForbidden, "")
		} else if errors.As(cmdErr, &appNotFoundErr) {
			logger.InfoContext(ctx, "CreateMenu command resulted in not found", slog.Any("error", cmdErr))
			SendJSONError(w, logger, appNotFoundErr.Error(), http.StatusNotFound, "")
		} else {
			logger.ErrorContext(ctx, "CreateMenu command failed with internal error", slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to create menu")
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
		logger.ErrorContext(ctx, "Failed to encode create menu response", slog.Any("error", err))
	}
	logger.InfoContext(ctx, "Successfully processed create menu request", slog.String("menu_id", respDTO.ID))
}
