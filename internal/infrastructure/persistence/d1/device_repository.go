package d1

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/domain/device"
)

// d1DeviceRepository は DeviceRepository の D1 実装です。
type d1DeviceRepository struct {
	db *sql.DB // Workers 環境では D1 binding に置き換わる想定
}

// NewD1DeviceRepository は新しい d1DeviceRepository を初期化します。
func NewD1DeviceRepository(db *sql.DB) device.DeviceRepository {
	return &d1DeviceRepository{db: db}
}

const deviceSqliteTimeFormat = "2006-01-02 15:04:05"

// Save はデバイス情報を D1 に保存します (INSERT ON CONFLICT UPDATE)。
func (r *d1DeviceRepository) Save(ctx context.Context, d *device.Device) error {
	log.Printf("INFO: Attempting to save device. DeviceID: %s, UserID: %v", d.ID, d.UserID)

	query := `
	INSERT INTO devices (id, user_id, created_at, last_seen_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
	  user_id = excluded.user_id,
	  last_seen_at = excluded.last_seen_at;
	`

	// *string を sql.NullString に変換してドライバの互換性を高める
	var userID sql.NullString
	if d.UserID != nil {
		userID = sql.NullString{String: *d.UserID, Valid: true}
	} // Valid のデフォルトは false なので、nil の場合は何もしなくて良い

	// time.Time を SQLite が解釈できる文字列形式 (YYYY-MM-DD HH:MM:SS) に変換
	createdAtStr := d.CreatedAt.UTC().Format(deviceSqliteTimeFormat)   // UTC に変換してからフォーマット
	lastSeenAtStr := d.LastSeenAt.UTC().Format(deviceSqliteTimeFormat) // UTC に変換してからフォーマット

	_, err := r.db.ExecContext(ctx, query,
		d.ID,
		userID,
		createdAtStr,  // 文字列形式で渡す
		lastSeenAtStr, // 文字列形式で渡す
	)

	if err != nil {
		log.Printf("ERROR: Failed to save device %s to D1: %v", d.ID, err)
		return fmt.Errorf("d1 exec failed for device %s: %w", d.ID, err)
	}
	log.Printf("INFO: Successfully saved device. DeviceID: %s", d.ID)
	return nil
}

// FindByID は D1 から ID でデバイスを検索します。
func (r *d1DeviceRepository) FindByID(ctx context.Context, id string) (*device.Device, error) {
	log.Printf("INFO: Attempting to find device by ID: %s", id)
	query := `
	SELECT id, user_id, created_at, last_seen_at
	FROM devices
	WHERE id = ?;
	`
	d := &device.Device{}
	var userID sql.NullString // NULL 可能カラムの受け皿
	var createdAtStr string   // created_at を文字列として受け取る
	var lastSeenAtStr string  // last_seen_at を文字列として受け取る

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID,
		&userID,
		&createdAtStr,  // 文字列変数でスキャン
		&lastSeenAtStr, // 文字列変数でスキャン
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("INFO: No device found for ID: %s", id)
			return nil, nil // 見つからない
		}
		log.Printf("ERROR: Failed to find device by ID %s from D1: %v", id, err)
		return nil, fmt.Errorf("d1 scan failed for device ID %s: %w", id, err)
	}

	// sql.NullString から *string へ変換
	if userID.Valid {
		d.UserID = &userID.String
	} else {
		d.UserID = nil
	}

	// 文字列から time.Time へパース
	createdAt, err := time.Parse(deviceSqliteTimeFormat, createdAtStr)
	if err != nil {
		log.Printf("WARN: Failed to parse created_at string '%s' for Device ID %s: %v", createdAtStr, id, err)
		return nil, fmt.Errorf("failed to parse created_at for device ID %s: %w", id, err)
	}
	lastSeenAt, err := time.Parse(deviceSqliteTimeFormat, lastSeenAtStr)
	if err != nil {
		log.Printf("WARN: Failed to parse last_seen_at string '%s' for Device ID %s: %v", lastSeenAtStr, id, err)
		return nil, fmt.Errorf("failed to parse last_seen_at for device ID %s: %w", id, err)
	}
	d.CreatedAt = createdAt
	d.LastSeenAt = lastSeenAt

	log.Printf("INFO: Successfully found device by ID: %s. UserID: %v", id, d.UserID)
	return d, nil
}
