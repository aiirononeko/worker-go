package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
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
	slog.InfoContext(ctx, "Starting device activation", slog.String("deviceID", cmd.DeviceID))

	deviceIDValue, err := entity.NewDeviceID(cmd.DeviceID)
	if err != nil {
		slog.WarnContext(ctx, "Invalid device ID in command", slog.String("deviceID", cmd.DeviceID), slog.Any("original_error", err.Error()))
		return nil, apperror.NewErrBadRequest(fmt.Sprintf("Invalid device ID format: %s", cmd.DeviceID), err.Error())
	}

	slog.InfoContext(ctx, "Finding device by ID", slog.String("deviceID", deviceIDValue.String()))
	existingDevice, err := h.deviceRepo.FindByID(ctx, deviceIDValue)

	var dev *device.Device
	if err != nil {
		var notFoundErr *apperror.ErrNotFound
		if errors.As(err, &notFoundErr) {
			slog.InfoContext(ctx, "No existing device found, creating new one", slog.String("deviceID", deviceIDValue.String()))
			dev, err = device.NewDevice(cmd.DeviceID, nil)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to create new device instance after NotFound", slog.String("deviceID", cmd.DeviceID), slog.Any("original_error", err.Error()))
				if strings.Contains(err.Error(), "invalid device ID format") {
					return nil, apperror.NewErrBadRequest(fmt.Sprintf("Failed to instantiate new device due to invalid ID: %s", cmd.DeviceID), err.Error())
				}
				return nil, apperror.NewErrInternal("Failed to instantiate new device", err)
			}
		} else {
			slog.ErrorContext(ctx, "Failed to find device by ID (non-NotFound error)", slog.String("deviceID", deviceIDValue.String()), slog.Any("original_error", err.Error()))
			return nil, apperror.NewErrInternal("Database error while finding device", err)
		}
	} else {
		slog.InfoContext(ctx, "Existing device found, updating last seen", slog.String("deviceID", deviceIDValue.String()))
		dev = existingDevice
		dev.UpdateLastSeen()
	}

	slog.InfoContext(ctx, "Saving device information", slog.String("deviceID", dev.ID.String()), slog.Any("userID", dev.UserID))
	if err := h.deviceRepo.Save(ctx, dev); err != nil {
		slog.ErrorContext(ctx, "Failed to save device", slog.String("deviceID", dev.ID.String()), slog.Any("original_error", err.Error()))
		return nil, apperror.NewErrInternal("Database error while saving device", err)
	}

	uid := fmt.Sprintf("device:%s", dev.ID.String())
	slog.InfoContext(ctx, "Generating JWT", slog.String("uid", uid))
	accessToken, refreshToken, refreshTokenJTI, expiresAt, err := h.jwtService.GenerateTokens(ctx, uid)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to generate tokens", slog.String("uid", uid), slog.Any("original_error", err.Error()))
		return nil, apperror.NewErrInternal("Failed to generate JWT tokens", err)
	}

	slog.InfoContext(ctx, "Saving refresh token to KV", slog.String("jti", refreshTokenJTI), slog.String("uid", uid))
	if err := h.refreshTokenRepo.Save(ctx, refreshTokenJTI, uid, h.refreshTokenTTL); err != nil {
		slog.ErrorContext(ctx, "Failed to save refresh token to KV", slog.String("jti", refreshTokenJTI), slog.String("uid", uid), slog.Any("original_error", err.Error()))
		return nil, apperror.NewErrInternal("Failed to save refresh token to KV store", err)
	}

	slog.InfoContext(ctx, "Successfully activated device and generated tokens", slog.String("deviceID", cmd.DeviceID), slog.String("uid", uid))
	return &ActivateDeviceResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
