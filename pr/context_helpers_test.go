package pr

import (
	"context"
	"testing"
)

func TestContextWithProvider(t *testing.T) {
	mock := &MockProvider{}
	ctx := ContextWithProvider(context.Background(), mock)

	retrieved := ProviderFromContext(ctx)
	if retrieved == nil {
		t.Fatal("ProviderFromContext returned nil")
	}

	// Verify it's the same instance
	if retrieved != mock {
		t.Error("retrieved provider is not the same instance")
	}
}

func TestProviderFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	retrieved := ProviderFromContext(ctx)
	if retrieved != nil {
		t.Errorf("expected nil, got %v", retrieved)
	}
}

func TestProviderFromContext_WrongType(t *testing.T) {
	// Store something else with the key to ensure type assertion works
	ctx := context.WithValue(context.Background(), prProviderKey, "not a provider")

	retrieved := ProviderFromContext(ctx)
	if retrieved != nil {
		t.Errorf("expected nil for wrong type, got %v", retrieved)
	}
}

func TestMustProviderFromContext_Success(t *testing.T) {
	mock := &MockProvider{}
	ctx := ContextWithProvider(context.Background(), mock)

	// Should not panic
	retrieved := MustProviderFromContext(ctx)
	if retrieved != mock {
		t.Error("retrieved provider is not the same instance")
	}
}

func TestMustProviderFromContext_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got none")
		}
	}()

	ctx := context.Background()
	MustProviderFromContext(ctx)
}

func TestContextWithProvider_OverwritesPrevious(t *testing.T) {
	mock1 := &MockProvider{}
	mock2 := &MockProvider{}

	ctx := ContextWithProvider(context.Background(), mock1)
	ctx = ContextWithProvider(ctx, mock2)

	retrieved := ProviderFromContext(ctx)
	if retrieved != mock2 {
		t.Error("expected second provider, got first")
	}
}
