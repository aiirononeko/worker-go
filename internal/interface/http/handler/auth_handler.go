package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/command" // Application 層を import
)

// AuthHandler は認証関連のエンドポイントを扱います。
type AuthHandler struct {
	activateDeviceCmd command.ActivateDeviceHandler // ActivateDeviceHandler への依存を追加
}

// NewAuthHandler は新しい AuthHandler を初期化します。
func NewAuthHandler(activateDeviceCmd command.ActivateDeviceHandler) *AuthHandler { // 引数で依存を受け取る
	return &AuthHandler{
		activateDeviceCmd: activateDeviceCmd,
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
