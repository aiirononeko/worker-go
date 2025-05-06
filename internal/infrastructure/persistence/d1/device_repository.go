package d1

import (
	"context"
	"database/sql"
	"errors" // For errors.Is
	"fmt"
	"log/slog"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/apperror"
	"github.com/aiirononeko/bulktrack-api/internal/domain/device"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	db "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1/sql"
)

// d1DeviceRepository は DeviceRepository の D1 実装です。
type d1DeviceRepository struct {
	db db.DBTX // Use DBTX interface from sqlc
}

// NewD1DeviceRepository は新しい d1DeviceRepository を初期化します。
func NewD1DeviceRepository(database db.DBTX) device.DeviceRepository {
	return &d1DeviceRepository{db: database}
}

const deviceSqliteTimeFormat = "2006-01-02 15:04:05" // Keep this for time formatting/parsing

// Save はデバイス情報を D1 に保存します (sqlc の UpsertDevice を使用)。
func (r *d1DeviceRepository) Save(ctx context.Context, d *device.Device) error {
	q := db.New(r.db)
	slog.InfoContext(ctx, "Attempting to save device via sqlc",
		slog.String("deviceID", d.ID.String()),
		slog.Any("userID", d.UserID),
	)

	var userID sql.NullString
	if d.UserID != nil {
		userID = sql.NullString{String: *d.UserID, Valid: true}
	}

	params := db.UpsertDeviceParams{
		ID:         d.ID.String(),
		UserID:     userID,
		CreatedAt:  d.CreatedAt.UTC().Format(deviceSqliteTimeFormat),
		LastSeenAt: d.LastSeenAt.UTC().Format(deviceSqliteTimeFormat),
	}

	err := q.UpsertDevice(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to sqlc UpsertDevice",
			slog.String("deviceID", d.ID.String()),
			slog.Any("error", err),
		)
		// Consider mapping specific DB errors (like constraint violations if any) to domain errors here
		return apperror.NewErrInternal(fmt.Sprintf("Failed to save device %s to DB", d.ID.String()), err)
	}

	slog.InfoContext(ctx, "Successfully saved device via sqlc", slog.String("deviceID", d.ID.String()))
	return nil
}

// FindByID は D1 から ID でデバイスを検索します (sqlc の GetDevice を使用)。
func (r *d1DeviceRepository) FindByID(ctx context.Context, id entity.DeviceID) (*device.Device, error) {
	q := db.New(r.db)
	slog.InfoContext(ctx, "Attempting to find device by ID via sqlc", slog.String("deviceID", id.String()))

	row, err := q.GetDevice(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.InfoContext(ctx, "sqlc GetDevice: No device found", slog.String("deviceID", id.String()))
			// Return specific NotFound error instead of (nil, nil)
			return nil, apperror.NewErrNotFound("device", id.String())
		}
		slog.ErrorContext(ctx, "Failed to sqlc GetDevice by ID",
			slog.String("deviceID", id.String()),
			slog.Any("error", err),
		)
		return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to get device %s from DB", id.String()), err)
	}

	// Map sqlc row to domain entity
	deviceID, err := entity.NewDeviceID(row.ID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse DeviceID from DB (sqlc result)",
			slog.String("db_id", row.ID),
			slog.String("target_device_id", id.String()), // For context, which device were we trying to hydrate
			slog.Any("error", err),
		)
		return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to parse device ID '%s' from DB for device %s", row.ID, id.String()), err)
	}

	d := &device.Device{ID: deviceID}

	if row.UserID.Valid {
		d.UserID = &row.UserID.String
	} else {
		d.UserID = nil
	}

	createdAt, err := time.Parse(deviceSqliteTimeFormat, row.CreatedAt)
	if err != nil {
		slog.WarnContext(ctx, "Failed to parse created_at from DB (sqlc result)",
			slog.String("deviceID", id.String()),
			slog.String("db_created_at", row.CreatedAt),
			slog.Any("error", err),
		)
		return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to parse created_at for device %s from DB", id.String()), err)
	}
	lastSeenAt, err := time.Parse(deviceSqliteTimeFormat, row.LastSeenAt)
	if err != nil {
		slog.WarnContext(ctx, "Failed to parse last_seen_at from DB (sqlc result)",
			slog.String("deviceID", id.String()),
			slog.String("db_last_seen_at", row.LastSeenAt),
			slog.Any("error", err),
		)
		return nil, apperror.NewErrInternal(fmt.Sprintf("Failed to parse last_seen_at for device %s from DB", id.String()), err)
	}
	d.CreatedAt = createdAt
	d.LastSeenAt = lastSeenAt

	slog.InfoContext(ctx, "Successfully found device by ID via sqlc",
		slog.String("deviceID", id.String()),
		slog.Any("userID", d.UserID),
	)
	return d, nil
}
