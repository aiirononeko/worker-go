package d1

import (
	"context"
	"database/sql"
	"fmt"
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

// Save はデバイス情報を D1 に保存します (INSERT ON CONFLICT UPDATE)。
func (r *d1DeviceRepository) Save(ctx context.Context, d *device.Device) error {
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
	const sqliteTimeFormat = "2006-01-02 15:04:05"
	createdAtStr := d.CreatedAt.UTC().Format(sqliteTimeFormat)   // UTC に変換してからフォーマット
	lastSeenAtStr := d.LastSeenAt.UTC().Format(sqliteTimeFormat) // UTC に変換してからフォーマット

	_, err := r.db.ExecContext(ctx, query,
		d.ID,
		userID,
		createdAtStr,  // 文字列形式で渡す
		lastSeenAtStr, // 文字列形式で渡す
	)

	// TODO: エラーハンドリング (制約違反など)
	if err != nil {
		return fmt.Errorf("d1 exec failed: %w", err)
	}
	return nil
}

// FindByID は D1 から ID でデバイスを検索します。
func (r *d1DeviceRepository) FindByID(ctx context.Context, id string) (*device.Device, error) {
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
			return nil, nil // 見つからない
		}
		// TODO: その他のDBエラーハンドリング
		return nil, fmt.Errorf("d1 scan failed: %w", err)
	}

	// sql.NullString から *string へ変換
	if userID.Valid {
		d.UserID = &userID.String
	} else {
		d.UserID = nil
	}

	// 文字列から time.Time へパース
	const sqliteTimeFormat = "2006-01-02 15:04:05"
	createdAt, err := time.Parse(sqliteTimeFormat, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}
	lastSeenAt, err := time.Parse(sqliteTimeFormat, lastSeenAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last_seen_at: %w", err)
	}
	d.CreatedAt = createdAt
	d.LastSeenAt = lastSeenAt

	return d, nil
}
