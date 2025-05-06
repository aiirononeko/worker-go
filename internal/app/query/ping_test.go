package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/aiirononeko/bulktrack-api/internal/app/query"
)

func TestPingQueryService_Execute(t *testing.T) {
	// Arrange (準備)
	service := query.NewPingQueryService()
	ctx := context.Background()
	expectedMessage := "pong"

	// Act (実行)
	result := service.Execute(ctx)

	// Assert (検証)
	if result.Message != expectedMessage {
		t.Errorf("Execute() returned wrong message: got %q, want %q", result.Message, expectedMessage)
	}

	// Timestamp should be recent, check if it's within a reasonable range (e.g., last 5 seconds)
	if time.Since(result.Timestamp) > 5*time.Second {
		t.Errorf("Execute() returned a timestamp that is too old: got %v", result.Timestamp)
	}
	if result.Timestamp.IsZero() {
		t.Errorf("Execute() returned a zero timestamp")
	}
}
