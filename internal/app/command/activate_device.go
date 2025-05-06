package command

import (
	"context"
	"fmt"
	"log"
	"time"

	domainAuth "github.com/aiirononeko/bulktrack-api/internal/domain/auth"
	"github.com/aiirononeko/bulktrack-api/internal/domain/device"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
)

// ActivateDeviceCommand はデバイスアクティベートの入力データを保持します。
type ActivateDeviceCommand struct {
	DeviceID string
}

// ActivateDeviceResult はデバイスアクティベートの結果を保持します。
type ActivateDeviceResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time // Access token の有効期限
}

// ActivateDeviceHandler はデバイスアクティベートのユースケースを実行します。
type ActivateDeviceHandler interface {
	Handle(ctx context.Context, cmd ActivateDeviceCommand) (*ActivateDeviceResult, error)
}

// --- Implementation ---

type activateDeviceHandler struct {
	deviceRepo       device.DeviceRepository
	jwtService       domainAuth.JWTService
	refreshTokenRepo domainAuth.RefreshTokenRepository
	refreshTokenTTL  time.Duration
}

// NewActivateDeviceHandler は新しい activateDeviceHandler を初期化します。
func NewActivateDeviceHandler(deviceRepo device.DeviceRepository, jwtService domainAuth.JWTService, refreshTokenRepo domainAuth.RefreshTokenRepository, refreshTokenTTL time.Duration) ActivateDeviceHandler {
	return &activateDeviceHandler{
		deviceRepo:       deviceRepo,
		jwtService:       jwtService,
		refreshTokenRepo: refreshTokenRepo,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

func (h *activateDeviceHandler) Handle(ctx context.Context, cmd ActivateDeviceCommand) (*ActivateDeviceResult, error) {
	log.Printf("INFO: Starting device activation for DeviceID: %s", cmd.DeviceID)

	// Convert string cmd.DeviceID to entity.DeviceID for repository and domain use
	deviceIDValue, err := entity.NewDeviceID(cmd.DeviceID)
	if err != nil {
		log.Printf("ERROR: Invalid device ID '%s' in ActivateDeviceCommand: %v", cmd.DeviceID, err)
		// Consider returning a more specific validation error if NewDeviceID provides it
		return nil, fmt.Errorf("invalid device ID '%s': %w", cmd.DeviceID, err)
	}

	log.Printf("INFO: Finding or creating device for DeviceID: %s", deviceIDValue.String())
	existingDevice, err := h.deviceRepo.FindByID(ctx, deviceIDValue)
	if err != nil {
		// Assuming FindByID might return a specific error for not found, or (nil, nil)
		// If it's a generic error, log and return
		log.Printf("ERROR: Failed to find device by ID %s: %v", deviceIDValue.String(), err)
		return nil, fmt.Errorf("failed to find device: %w", err)
	}

	var dev *device.Device
	if existingDevice == nil {
		log.Printf("INFO: No existing device found for DeviceID %s, creating new one.", deviceIDValue.String())
		dev, err = device.NewDevice(cmd.DeviceID, nil) // cmd.DeviceID is string, NewDevice handles conversion
		if err != nil {
			log.Printf("ERROR: Failed to create new device instance for DeviceID %s: %v", cmd.DeviceID, err)
			return nil, fmt.Errorf("failed to instantiate device: %w", err)
		}
	} else {
		log.Printf("INFO: Existing device found for DeviceID %s, updating last seen.", deviceIDValue.String())
		dev = existingDevice
		dev.UpdateLastSeen()
	}

	log.Printf("INFO: Saving device information for DeviceID: %s (User ID: %v)", dev.ID.String(), dev.UserID)
	if err := h.deviceRepo.Save(ctx, dev); err != nil {
		log.Printf("ERROR: Failed to save device for DeviceID %s: %v", dev.ID.String(), err)
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	// Use the original string device ID from the command for the UID prefix logic if needed,
	// or ensure dev.ID.String() is used consistently if it should be the canonical one.
	// For JWT UID, it's common to use the canonical (potentially normalized) ID.
	uid := fmt.Sprintf("device:%s", dev.ID.String()) // Use dev.ID.String()
	log.Printf("INFO: Generating JWT for UID: %s", uid)
	accessToken, refreshToken, refreshTokenJTI, expiresAt, err := h.jwtService.GenerateTokens(ctx, uid)
	if err != nil {
		log.Printf("ERROR: Failed to generate tokens for UID %s: %v", uid, err)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	log.Printf("INFO: Saving refresh token to KV for JTI: %s, UID: %s", refreshTokenJTI, uid)
	if err := h.refreshTokenRepo.Save(ctx, refreshTokenJTI, uid, h.refreshTokenTTL); err != nil {
		log.Printf("ERROR: Failed to save refresh token to KV for JTI %s, UID %s: %v", refreshTokenJTI, uid, err)
		return nil, fmt.Errorf("failed to save refresh token state: %w", err)
	}

	log.Printf("INFO: Successfully activated device and generated tokens for DeviceID: %s (UID: %s)", cmd.DeviceID, uid)
	return &ActivateDeviceResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
