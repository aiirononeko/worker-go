package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/app/command"
	"github.com/aiirononeko/bulktrack-api/internal/app/dto"
	"github.com/aiirononeko/bulktrack-api/internal/app/query"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware"
	"github.com/go-playground/validator/v10"
)

// バリデーターインスタンス (シングルトンまたはハンドラー初期化時に生成)
var validate = validator.New()

// --- DTOs (common for Menu endpoints) --- //

// CreateMenuRequest mirrors the OpenAPI schema `CreateMenuRequest`
type CreateMenuRequest struct {
	Name        string  `json:"name" validate:"required,gte=1,lte=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,lte=500"`
	SortOrder   int     `json:"sort_order" validate:"gte=0"`
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
	updateMenuExHandler  *command.UpdateMenuExercisesHandler
}

// NewMenuHandler は MenuHandler の新しいインスタンスを生成します。
func NewMenuHandler(
	lms query.ListMenusQueryService,
	createCmdHandler *command.CreateMenuHandler,
	updateExCmdHandler *command.UpdateMenuExercisesHandler,
) *MenuHandler {
	return &MenuHandler{
		listMenusQuery:       lms,
		createMenuCmdHandler: createCmdHandler,
		updateMenuExHandler:  updateExCmdHandler,
	}
}

// ServeHTTP は /v1/menus へのリクエストをメソッドに応じて処理します。
func (h *MenuHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFromContext(ctx)
	r = r.WithContext(context.WithValue(ctx, middleware.LoggerKey, logger))

	basePath := "/v1/menus"
	fullPath := r.URL.Path

	logger.DebugContext(ctx, "MenuHandler ServeHTTP called", "method", r.Method, "path", fullPath)

	if strings.HasPrefix(fullPath, basePath+"/") && strings.HasSuffix(fullPath, "/exercises") {
		trimmedPath := strings.TrimPrefix(fullPath, basePath+"/")
		menuIDStr := strings.TrimSuffix(trimmedPath, "/exercises")

		if menuIDStr == "" || strings.Contains(menuIDStr, "/") {
			logger.WarnContext(ctx, "Invalid path format for menu exercises", "path", fullPath)
			SendJSONError(w, logger, "Not Found", http.StatusNotFound, "Invalid path")
			return
		}

		if r.Method == http.MethodPut {
			h.updateMenuExercises(w, r, menuIDStr)
			return
		}
		logger.WarnContext(ctx, "Method not allowed for menu exercises path", "method", r.Method, "path", fullPath)
		SendJSONError(w, logger, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed, "")
		return
	}

	if fullPath == basePath {
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
			return
		case http.MethodPost:
			h.create(w, r)
			return
		}
		logger.WarnContext(ctx, "Method not allowed for /v1/menus", "method", r.Method)
		SendJSONError(w, logger, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed, "")
		return
	}

	logger.WarnContext(ctx, "Path not found in MenuHandler", "path", fullPath)
	SendJSONError(w, logger, "Not Found", http.StatusNotFound, "The requested menu operation is not supported.")
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

	// バリデーションの実行
	if err := validate.Struct(req); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			// エラーメッセージを整形 (詳細は省略可能)
			// Example: Format validation errors into a readable string or structure
			var errorMsgs []string
			for _, fe := range validationErrors {
				// ここで fe.Tag(), fe.Field(), fe.Param() などを使って詳細なメッセージを生成できる
				errorMsgs = append(errorMsgs, fmt.Sprintf("Field '%s' failed validation on '%s' tag", fe.Field(), fe.Tag()))
			}
			details := strings.Join(errorMsgs, "; ")
			appErr := apperror.NewErrBadRequest("Input validation failed", details)
			logger.WarnContext(ctx, "Input validation failed for create menu", slog.Any("validation_errors", details), slog.Any("error", appErr))
			SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, appErr.Details) // Use formatted details
		} else {
			// バリデーションライブラリ自体のエラーなど、予期せぬケース
			appErr := apperror.NewErrInternal("Error during input validation", err)
			logger.ErrorContext(ctx, "Unexpected error during validation for create menu", slog.Any("error", appErr))
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Validation check failed unexpectedly")
		}
		return
	}

	// バリデーション成功後、コマンドを作成して実行
	cmd := command.CreateMenuCommand{
		DeviceID:    uid,
		Name:        req.Name, // バリデーション済みの値を使用
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

// UpdateMenuExercises handles PUT /v1/menus/{menuId}/exercises
func (h *MenuHandler) updateMenuExercises(w http.ResponseWriter, r *http.Request, menuIDStr string) {
	ctx := r.Context()
	logger := middleware.LoggerFromContext(ctx).With(slog.String("menu_id_path", menuIDStr))
	logger.InfoContext(ctx, "Processing update menu exercises request")

	uidWithPrefix, ok := middleware.GetDeviceIDFromContext(ctx)
	if !ok {
		logger.WarnContext(ctx, "Authorization error: UID not found in context for update menu exercises")
		SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, "Device ID not found in context")
		return
	}
	deviceIDStr := strings.TrimPrefix(uidWithPrefix, "device:")
	logger = logger.With(slog.String("device_id_token", deviceIDStr))

	var reqDTO dto.UpdateMenuExercisesRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		logger.WarnContext(ctx, "Failed to decode request body for update menu exercises", slog.Any("error", err))
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	if err := validate.StructCtx(ctx, reqDTO); err != nil {
		var validationErrors validator.ValidationErrors
		var errorMsgs []string
		if errors.As(err, &validationErrors) {
			for _, fe := range validationErrors {
				errorMsgs = append(errorMsgs, fmt.Sprintf("Field '%s' failed on '%s' tag", fe.Field(), fe.Tag()))
			}
		} else {
			errorMsgs = append(errorMsgs, err.Error())
		}
		details := strings.Join(errorMsgs, "; ")
		logger.WarnContext(ctx, "Input validation failed for update menu exercises", slog.String("validation_errors", details))
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, "Validation failed: "+details)
		return
	}

	cmd := command.UpdateMenuExercisesCommand{
		MenuID:    menuIDStr,
		DeviceID:  deviceIDStr,
		Exercises: reqDTO.Exercises,
	}

	result, cmdErr := h.updateMenuExHandler.Handle(ctx, cmd)
	if cmdErr != nil {
		logger.WarnContext(ctx, "UpdateMenuExercises command failed", slog.Any("error", cmdErr))

		var nfErr *apperror.ErrNotFound
		var badReqErr *apperror.ErrBadRequest
		var forbiddenErr *apperror.ErrForbidden
		var internalErr *apperror.ErrInternal

		switch {
		case errors.As(cmdErr, &nfErr):
			SendJSONError(w, logger, "Not Found", http.StatusNotFound, nfErr.Error())
		case errors.As(cmdErr, &badReqErr):
			SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, badReqErr.Error())
		case errors.As(cmdErr, &forbiddenErr):
			SendJSONError(w, logger, "Forbidden", http.StatusForbidden, forbiddenErr.Error())
		case errors.As(cmdErr, &internalErr):
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, internalErr.Error())
		default:
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to update menu exercises: "+cmdErr.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result.Menu); err != nil {
		logger.ErrorContext(ctx, "Failed to encode update menu exercises response", slog.Any("error", err))
	}
	logger.InfoContext(ctx, "Successfully processed update menu exercises request", slog.String("menu_id", result.Menu.ID))
}
