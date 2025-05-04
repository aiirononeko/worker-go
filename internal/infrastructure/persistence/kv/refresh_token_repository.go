//go:build js && wasm

package kv

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainAuth "github.com/aiirononeko/bulktrack-api/internal/domain/auth"
	"github.com/syumai/workers/cloudflare/kv" // KV ライブラリ
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
	ttlInSeconds := int64(ttl.Seconds())
	if ttlInSeconds <= 0 {
		return errors.New("TTL must be positive")
	}

	err := r.kvNamespace.PutString(jti, uid, &kv.PutOptions{ExpirationTTL: int(ttlInSeconds)})
	if err != nil {
		return fmt.Errorf("failed to put refresh token to KV (jti: %s): %w", jti, err)
	}
	return nil
}

// Validate は KV ストアで jti を検証します。
func (r *kvRefreshTokenRepository) Validate(ctx context.Context, jti string, expectedUID string) error {
	storedUID, err := r.kvNamespace.GetString(jti, nil)

	if err != nil {
		return fmt.Errorf("failed to get refresh token from KV (jti: %s): %w", jti, err)
	}

	if storedUID == "" {
		return domainAuth.ErrRefreshTokenNotFound
	}

	if storedUID != expectedUID {
		return domainAuth.ErrMismatchedUID
	}

	return nil
}

// Delete は KV ストアから jti をキーとするエントリを削除します。
func (r *kvRefreshTokenRepository) Delete(ctx context.Context, jti string) error {
	err := r.kvNamespace.Delete(jti)
	if err != nil {
		return fmt.Errorf("KV Delete operation error: %w", err)
	}
	return nil
}
