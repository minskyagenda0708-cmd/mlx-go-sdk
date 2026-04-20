package mlx

import (
	"testing"
)

func TestNewFromEnvRequiresToken(t *testing.T) {
	t.Setenv(EnvToken, "")

	_, err := NewFromEnv()
	if err != ErrMissingToken {
		t.Fatalf("expected ErrMissingToken, got %v", err)
	}
}

func TestNewConfiguresServices(t *testing.T) {
	client, err := New(WithToken("test-token"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if client.Profiles == nil || client.Launcher == nil || client.Folders == nil || client.Transfers == nil || client.Archives == nil || client.Cookies == nil || client.Workflows == nil {
		t.Fatalf("expected all core services to be initialized")
	}
}
