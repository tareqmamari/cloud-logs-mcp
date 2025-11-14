package auth

import (
	"net/http"
	"testing"

	"go.uber.org/zap"
)

func TestNewAuthenticator(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "valid API key",
			apiKey:  "test-api-key-12345",
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := New(tt.apiKey, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && auth == nil {
				t.Error("Expected authenticator to be created")
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	t.Skip("Skipping test that requires valid IBM Cloud credentials")

	logger, _ := zap.NewDevelopment()
	auth, err := New("test-api-key", logger)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	req, _ := http.NewRequest("GET", "https://example.com", nil)

	// Note: This makes a real network call to IBM Cloud IAM
	// Skipped in unit tests, run in integration tests with real credentials
	err = auth.Authenticate(req)
	if err != nil {
		t.Errorf("Authenticate() failed: %v", err)
	}

	// Check that Authorization header was added
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		t.Error("Expected Authorization header to be set")
	}
}

func TestAuthenticateNilRequest(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	auth, err := New("test-api-key", logger)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	err = auth.Authenticate(nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}
}
