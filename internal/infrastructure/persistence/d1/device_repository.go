package d1

import (
	"context"
	"database/sql"
	"errors" // For errors.Is
	"fmt"
	"log"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/domain/device"
	"github.com/aiirononeko/bulktrack-api/internal/domain/entity"
	db "github.com/aiirononeko/bulktrack-api/internal/infrastructure/persistence/d1/sql" // Import sqlc generated code
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
	log.Printf("INFO: Attempting to save device via sqlc. DeviceID: %s, UserID: %v", d.ID.String(), d.UserID)

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
		log.Printf("ERROR: Failed to sqlc UpsertDevice %s: %v", d.ID.String(), err)
		// Consider mapping specific DB errors (like constraint violations if any) to domain errors here
		return fmt.Errorf("sqlc upsert device failed for %s: %w", d.ID.String(), err)
	}

	log.Printf("INFO: Successfully saved device via sqlc. DeviceID: %s", d.ID.String())
	return nil
}

// FindByID は D1 から ID でデバイスを検索します (sqlc の GetDevice を使用)。
func (r *d1DeviceRepository) FindByID(ctx context.Context, id entity.DeviceID) (*device.Device, error) {
	q := db.New(r.db)
	log.Printf("INFO: Attempting to find device by ID via sqlc: %s", id.String())

	row, err := q.GetDevice(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("INFO: sqlc GetDevice: No device found for ID: %s", id.String())
			return nil, nil // Not found, return nil, nil as per interface comment
		}
		log.Printf("ERROR: Failed to sqlc GetDevice by ID %s: %v", id.String(), err)
		return nil, fmt.Errorf("sqlc get device failed for ID %s: %w", id.String(), err)
	}

	// Map sqlc row to domain entity
	deviceID, err := entity.NewDeviceID(row.ID)
	if err != nil {
		log.Printf("ERROR: Failed to parse DeviceID '%s' from DB (sqlc result): %v", row.ID, err)
		return nil, fmt.Errorf("failed to parse device id '%s' from DB: %w", row.ID, err)
	}

	d := &device.Device{ID: deviceID}

	if row.UserID.Valid {
		d.UserID = &row.UserID.String
	} else {
		d.UserID = nil
	}

	createdAt, err := time.Parse(deviceSqliteTimeFormat, row.CreatedAt)
	if err != nil {
		log.Printf("WARN: Failed to parse created_at string '%s' from DB (sqlc result) for Device ID %s: %v", row.CreatedAt, id.String(), err)
		return nil, fmt.Errorf("failed to parse created_at for device ID %s: %w", id.String(), err)
	}
	lastSeenAt, err := time.Parse(deviceSqliteTimeFormat, row.LastSeenAt)
	if err != nil {
		log.Printf("WARN: Failed to parse last_seen_at string '%s' from DB (sqlc result) for Device ID %s: %v", row.LastSeenAt, id.String(), err)
		return nil, fmt.Errorf("failed to parse last_seen_at for device ID %s: %w", id.String(), err)
	}
	d.CreatedAt = createdAt
	d.LastSeenAt = lastSeenAt

	log.Printf("INFO: Successfully found device by ID via sqlc: %s. UserID: %v", id.String(), d.UserID)
	return d, nil
}
