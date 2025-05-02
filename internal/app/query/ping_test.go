package query_test

import (
	"context"
	"testing"

	"github.com/aiirononeko/bulktrack-api/internal/app/query"
)

func TestPingQueryService_Ping(t *testing.T) {
	// Arrange (準備)
	service := query.NewPingQueryService()
	ctx := context.Background()
	expected := "Pong"

	// Act (実行)
	msg, err := service.Ping(ctx)

	// Assert (検証)
	if err != nil {
		t.Fatalf("Ping() returned an unexpected error: %v", err)
	}
	if msg != expected {
		t.Fatalf("Ping() returned wrong message: got %q, want %q", msg, expected)
	}
}
