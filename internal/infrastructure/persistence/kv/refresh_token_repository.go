//go:build js && wasm

package kv

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	domainAuth "github.com/aiirononeko/bulktrack-api/internal/domain/auth"
	"github.com/syumai/workers/cloudflare/kv"
)

// kvRefreshTokenRepository は RefreshTokenRepository の KV 実装です。
type kvRefreshTokenRepository struct {
	kvNamespace kv.Namespace // ポインタ型から値型に戻す
}

// NewKVRefreshTokenRepository は新しい kvRefreshTokenRepository を初期化します。
func NewKVRefreshTokenRepository(ns kv.Namespace /* 値型に戻す */) domainAuth.RefreshTokenRepository {
	return &kvRefreshTokenRepository{kvNamespace: ns}
}

// Save は jti をキー、uid を値として KV に TTL 付きで保存します。
func (r *kvRefreshTokenRepository) Save(ctx context.Context, jti string, uid string, ttl time.Duration) error {
	log.Printf("INFO: Attempting to save refresh token to KV. JTI: %s, UID: %s, TTL: %s", jti, uid, ttl)

	ttlInSeconds := int64(ttl.Seconds())
	if ttlInSeconds <= 0 {
		log.Printf("ERROR: Invalid TTL for refresh token JTI %s: %s. Must be positive.", jti, ttl)
		return errors.New("TTL must be positive")
	}

	err := r.kvNamespace.PutString(jti, uid, &kv.PutOptions{ExpirationTTL: int(ttlInSeconds)})
	if err != nil {
		log.Printf("ERROR: Failed to put refresh token to KV for JTI %s, UID %s: %v", jti, uid, err)
		return fmt.Errorf("failed to put refresh token to KV (jti: %s): %w", jti, err)
	}
	log.Printf("INFO: Successfully saved refresh token to KV. JTI: %s, UID: %s", jti, uid)
	return nil
}

// Validate は KV ストアで jti を検証します。
func (r *kvRefreshTokenRepository) Validate(ctx context.Context, jti string, expectedUID string) error {
	log.Printf("INFO: Attempting to validate refresh token in KV. JTI: %s, Expected UID: %s", jti, expectedUID)

	storedUID, err := r.kvNamespace.GetString(jti, nil)
	if err != nil {
		log.Printf("ERROR: Failed to get refresh token from KV for JTI %s: %v", jti, err)
		return fmt.Errorf("failed to get refresh token from KV (jti: %s): %w", jti, err)
	}

	if storedUID == "" {
		log.Printf("WARN: Refresh token not found in KV for JTI: %s. Expected UID: %s", jti, expectedUID)
		return domainAuth.ErrRefreshTokenNotFound
	}

	if storedUID != expectedUID {
		log.Printf("ERROR: Mismatched UID for refresh token JTI %s. Stored UID: %s, Expected UID: %s", jti, storedUID, expectedUID)
		return domainAuth.ErrMismatchedUID
	}

	log.Printf("INFO: Successfully validated refresh token in KV. JTI: %s, UID: %s", jti, storedUID)
	return nil
}

// Delete は KV ストアから jti をキーとするエントリを削除します。
func (r *kvRefreshTokenRepository) Delete(ctx context.Context, jti string) error {
	log.Printf("INFO: Attempting to delete refresh token from KV. JTI: %s", jti)
	err := r.kvNamespace.Delete(jti)
	if err != nil {
		// 冪等性を考慮し、削除失敗が「見つからない」ことに起因する場合は致命的エラーとしないことも考えられるが、
		// 現状の syumai/workers/cloudflare/kv の Delete はエラーを返さない場合もあるため、一旦エラーは全てログ＆リターン。
		log.Printf("ERROR: Failed to delete refresh token from KV for JTI %s: %v", jti, err)
		return fmt.Errorf("KV Delete operation error for JTI %s: %w", jti, err)
	}
	log.Printf("INFO: Successfully deleted (or ensured deletion of) refresh token from KV. JTI: %s", jti)
	return nil
}
