package api

import (
	"context"
	"errors"
	"testing"
)

func TestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // lol

	client := NewClient("no. 3-chan").WithContext(ctx)

	// This should fail.
	_, err := client.Me()
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatal("Unexpected error:", err)
	}
}
