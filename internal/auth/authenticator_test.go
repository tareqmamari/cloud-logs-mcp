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
		iamURL  string
		wantErr bool
	}{
		{
			name:    "valid API key",
			apiKey:  "test-api-key-12345", //nolint:gosec // test value, not a real secret
			iamURL:  "",
			wantErr: false,
		},
		{
			name:    "valid API key with custom IAM URL",
			apiKey:  "test-api-key-12345", //nolint:gosec // test value, not a real secret
			iamURL:  "https://iam.test.cloud.ibm.com",
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			iamURL:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := New(tt.apiKey, tt.iamURL, logger)
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
	auth, err := New("test-api-key", "", logger)
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
	auth, err := New("test-api-key", "", logger)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	err = auth.Authenticate(nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}
}

func TestParseJWTClaims(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		wantSubject string
		wantErr     bool
	}{
		{
			name: "valid JWT token",
			// This is a test token with claims: {"sub": "iam-ServiceId-12345", "iam_id": "iam-12345", "account": "account123"}
			token:       "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJpYW0tU2VydmljZUlkLTEyMzQ1IiwiaWFtX2lkIjoiaWFtLTEyMzQ1IiwiYWNjb3VudCI6ImFjY291bnQxMjMifQ.signature", //nolint:gosec // pragma: allowlist secret
			wantSubject: "iam-ServiceId-12345",
			wantErr:     false,
		},
		{
			name: "valid JWT with user email",
			// Claims: {"sub": "IBMid-123456789", "email": "user@example.com", "name": "Test User"}
			token:       "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJJQk1pZC0xMjM0NTY3ODkiLCJlbWFpbCI6InVzZXJAZXhhbXBsZS5jb20iLCJuYW1lIjoiVGVzdCBVc2VyIn0.sig", //nolint:gosec // pragma: allowlist secret
			wantSubject: "IBMid-123456789",
			wantErr:     false,
		},
		{
			name:    "invalid token format - no dots",
			token:   "invalidtoken",
			wantErr: true,
		},
		{
			name:    "invalid token format - only two parts",
			token:   "header.payload",
			wantErr: true,
		},
		{
			name:    "invalid base64 payload",
			token:   "header.!!!invalid!!!.signature",
			wantErr: true,
		},
		{
			name:    "invalid JSON in payload",
			token:   "header.bm90anNvbg.signature", // "notjson" base64 encoded, pragma: allowlist secret
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := parseJWTClaims(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJWTClaims() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && claims.Subject != tt.wantSubject {
				t.Errorf("parseJWTClaims() subject = %v, want %v", claims.Subject, tt.wantSubject)
			}
		})
	}
}

func TestJWTClaimsFields(t *testing.T) {
	// Token with full claims structure
	// Claims: {"sub": "iam-ServiceId-test", "iam_id": "iam-id-123", "account": "acc-456", "realmid": "realm", "identifier": "svc-id", "name": "Test Service", "email": "", "iat": 1700000000, "exp": 1700003600}
	token := "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJpYW0tU2VydmljZUlkLXRlc3QiLCJpYW1faWQiOiJpYW0taWQtMTIzIiwiYWNjb3VudCI6ImFjYy00NTYiLCJyZWFsbWlkIjoicmVhbG0iLCJpZGVudGlmaWVyIjoic3ZjLWlkIiwibmFtZSI6IlRlc3QgU2VydmljZSIsImVtYWlsIjoiIiwiaWF0IjoxNzAwMDAwMDAwLCJleHAiOjE3MDAwMDM2MDB9.sig" //nolint:gosec // pragma: allowlist secret

	claims, err := parseJWTClaims(token)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if claims.Subject != "iam-ServiceId-test" {
		t.Errorf("Subject mismatch: %s", claims.Subject)
	}
	if claims.IAMId != "iam-id-123" {
		t.Errorf("IAMId mismatch: %s", claims.IAMId)
	}
	if claims.AccountID != "acc-456" {
		t.Errorf("AccountID mismatch: %s", claims.AccountID)
	}
	if claims.RealmID != "realm" {
		t.Errorf("RealmID mismatch: %s", claims.RealmID)
	}
	if claims.Identifier != "svc-id" {
		t.Errorf("Identifier mismatch: %s", claims.Identifier)
	}
	if claims.Name != "Test Service" {
		t.Errorf("Name mismatch: %s", claims.Name)
	}
	if claims.IssuedAt != 1700000000 {
		t.Errorf("IssuedAt mismatch: %d", claims.IssuedAt)
	}
	if claims.ExpiresAt != 1700003600 {
		t.Errorf("ExpiresAt mismatch: %d", claims.ExpiresAt)
	}
}
