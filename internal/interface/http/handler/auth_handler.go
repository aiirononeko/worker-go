package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/command" // Application 層を import
)

// AuthHandler は認証関連のエンドポイントを扱います。
type AuthHandler struct {
	activateDeviceCmd command.ActivateDeviceHandler
	refreshTokenCmd   command.RefreshTokenHandler // RefreshTokenHandler への依存を追加
}

// NewAuthHandler は新しい AuthHandler を初期化します。
func NewAuthHandler(activateDeviceCmd command.ActivateDeviceHandler, refreshTokenCmd command.RefreshTokenHandler) *AuthHandler {
	return &AuthHandler{
		activateDeviceCmd: activateDeviceCmd,
		refreshTokenCmd:   refreshTokenCmd,
	}
}

// ActivateDevice はデバイスをアクティベートし、トークンを返します。
// POST /v1/auth/device
func (h *AuthHandler) ActivateDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.Header.Get("X-Device-Id")
	if deviceID == "" {
		http.Error(w, "X-Device-Id header is required", http.StatusBadRequest)
		return
	}

	// Application 層のコマンドを呼び出す
	cmd := command.ActivateDeviceCommand{DeviceID: deviceID}
	result, err := h.activateDeviceCmd.Handle(r.Context(), cmd)
	if err != nil {
		http.Error(w, "Failed to activate device", http.StatusInternalServerError)
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
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
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
	// 1. リクエストボディをデコード
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}

	// 2. Application 層の RefreshTokenCommand を呼び出す
	cmd := command.RefreshTokenCommand{RefreshToken: req.RefreshToken}
	result, err := h.refreshTokenCmd.Handle(r.Context(), cmd)
	if err != nil {
		http.Error(w, "Failed to refresh token", http.StatusInternalServerError)
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
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
