package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"              // apperror をインポート
	appCmd "github.com/aiirononeko/bulktrack-api/internal/app/command"        // Application 層を import
	"github.com/aiirononeko/bulktrack-api/internal/interface/http/middleware" // middleware をインポート
)

// AuthHandler は認証関連のエンドポイントを扱います。
type AuthHandler struct {
	activateDeviceCmd appCmd.ActivateDeviceHandler
	refreshTokenCmd   appCmd.RefreshTokenHandler
	logoutCmd         appCmd.LogoutHandler // LogoutHandler への依存を追加
}

// NewAuthHandler は新しい AuthHandler を初期化します。
func NewAuthHandler(activateDeviceCmd appCmd.ActivateDeviceHandler, refreshTokenCmd appCmd.RefreshTokenHandler, logoutCmd appCmd.LogoutHandler) *AuthHandler {
	return &AuthHandler{
		activateDeviceCmd: activateDeviceCmd,
		refreshTokenCmd:   refreshTokenCmd,
		logoutCmd:         logoutCmd, // 依存を初期化
	}
}

// ActivateDevice はデバイスをアクティベートし、トークンを返します。
// POST /v1/auth/device
func (h *AuthHandler) ActivateDevice(w http.ResponseWriter, r *http.Request) {
	logger := middleware.LoggerFromContext(r.Context())
	deviceID := r.Header.Get("X-Device-Id")
	logger.Info("Received ActivateDevice request", slog.String("path", r.URL.Path), slog.String("device_id_header", deviceID))

	if deviceID == "" {
		err := apperror.NewErrBadRequest("X-Device-Id header is required", "")
		logger.Warn("ActivateDevice request failed", slog.String("path", r.URL.Path), slog.Any("error", err))
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, err.Error())
		return
	}

	logger = logger.With(slog.String("device_id", deviceID))

	cmd := appCmd.ActivateDeviceCommand{DeviceID: deviceID}
	result, cmdErr := h.activateDeviceCmd.Handle(r.Context(), cmd)
	if cmdErr != nil {
		var badRequestErr *apperror.ErrBadRequest
		var unauthorizedErr *apperror.ErrUnauthorized

		if errors.As(cmdErr, &badRequestErr) {
			logger.Warn("ActivateDevice command failed with bad request", slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, badRequestErr.Error())
		} else if errors.As(cmdErr, &unauthorizedErr) {
			logger.Warn("ActivateDevice command failed with unauthorized", slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, unauthorizedErr.Error())
		} else {
			logger.Error("Failed to handle ActivateDevice command", slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to activate device")
		}
		return
	}

	// レスポンスを構築
	resp := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"` // Access token の有効期間 (秒)
	}{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    int64(time.Until(result.ExpiresAt).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode ActivateDevice response", slog.String("path", r.URL.Path), slog.String("device_id", deviceID), slog.Any("err", err))
	}
	logger.Info("Successfully processed ActivateDevice request", slog.String("path", r.URL.Path), slog.String("device_id", deviceID))
}

// RefreshTokenRequest はリフレッシュトークンリクエストのボディを表します。
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse はリフレッシュ成功時のレスポンスを表します。
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access token の有効期間 (秒)
}

// RefreshToken は受け取ったリフレッシュトークンを検証し、新しいトークンペアを返します。
// POST /v1/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	logger := middleware.LoggerFromContext(r.Context())
	logger.Info("Received RefreshToken request", slog.String("path", r.URL.Path))

	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := apperror.NewErrBadRequest("Invalid request body", err.Error())
		logger.Warn("RefreshToken request failed", slog.String("path", r.URL.Path), slog.Any("error", appErr))
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, appErr.Error())
		return
	}
	if req.RefreshToken == "" {
		appErr := apperror.NewErrBadRequest("refresh_token is required", "")
		logger.Warn("RefreshToken request failed", slog.String("path", r.URL.Path), slog.Any("error", appErr))
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, appErr.Error())
		return
	}

	cmd := appCmd.RefreshTokenCommand{RefreshToken: req.RefreshToken}
	result, cmdErr := h.refreshTokenCmd.Handle(r.Context(), cmd)
	if cmdErr != nil {
		var badRequestErr *apperror.ErrBadRequest
		var unauthorizedErr *apperror.ErrUnauthorized

		if errors.As(cmdErr, &unauthorizedErr) {
			logger.Warn("RefreshToken command failed with unauthorized", slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, unauthorizedErr.Error())
		} else if errors.As(cmdErr, &badRequestErr) {
			logger.Warn("RefreshToken command failed with bad request", slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, badRequestErr.Error())
		} else {
			logger.Error("Failed to handle RefreshToken command", slog.String("path", r.URL.Path), slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to refresh token")
		}
		return
	}

	// 3. 成功レスポンスを返す
	resp := RefreshTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    int64(time.Until(result.ExpiresAt).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode RefreshToken response", slog.String("path", r.URL.Path), slog.Any("err", err))
	}
	logger.Info("Successfully processed RefreshToken request", slog.String("path", r.URL.Path))
}

// LogoutRequest はログアウトリクエストのボディを表します。
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout は受け取ったリフレッシュトークンを無効化します。
// POST /v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	logger := middleware.LoggerFromContext(r.Context())
	logger.Info("Received Logout request", slog.String("path", r.URL.Path))

	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := apperror.NewErrBadRequest("Invalid request body", err.Error())
		logger.Warn("Logout request failed", slog.String("path", r.URL.Path), slog.Any("error", appErr))
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, appErr.Error())
		return
	}
	if req.RefreshToken == "" {
		appErr := apperror.NewErrBadRequest("refresh_token is required", "")
		logger.Warn("Logout request failed", slog.String("path", r.URL.Path), slog.Any("error", appErr))
		SendJSONError(w, logger, "Bad Request", http.StatusBadRequest, appErr.Error())
		return
	}

	cmd := appCmd.LogoutCommand{RefreshToken: req.RefreshToken}
	if cmdErr := h.logoutCmd.Handle(r.Context(), cmd); cmdErr != nil {
		var unauthorizedErr *apperror.ErrUnauthorized
		if errors.As(cmdErr, &unauthorizedErr) {
			logger.Warn("Logout command failed with unauthorized", slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Unauthorized", http.StatusUnauthorized, unauthorizedErr.Error())
		} else {
			logger.Error("Failed to handle Logout command", slog.String("path", r.URL.Path), slog.Any("error", cmdErr))
			SendJSONError(w, logger, "Internal Server Error", http.StatusInternalServerError, "Failed to logout")
		}
		return
	}

	// 3. 成功レスポンス (No Content)
	w.WriteHeader(http.StatusNoContent)
	logger.Info("Successfully processed Logout request", slog.String("path", r.URL.Path))
}
