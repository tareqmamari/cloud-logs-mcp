// Package auth provides authentication functionality for IBM Cloud API access.
package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"go.uber.org/zap"
)

// JWTClaims represents the claims from an IBM Cloud IAM JWT token
type JWTClaims struct {
	Subject    string `json:"sub"`        // User/Service ID (e.g., "iam-ServiceId-...")
	IAMId      string `json:"iam_id"`     // IAM ID
	AccountID  string `json:"account"`    // IBM Cloud account ID
	RealmID    string `json:"realmid"`    // Realm ID
	Identifier string `json:"identifier"` // Service ID or user identifier
	Name       string `json:"name"`       // Human-readable name
	Email      string `json:"email"`      // User email (if applicable)
	IssuedAt   int64  `json:"iat"`        // Issued at timestamp
	ExpiresAt  int64  `json:"exp"`        // Expiration timestamp
}

// Authenticator handles IBM Cloud authentication
type Authenticator struct {
	authenticator core.Authenticator
	logger        *zap.Logger
}

// New creates a new authenticator using IBM SDK
func New(apiKey string, iamURL string, logger *zap.Logger) (*Authenticator, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Create IBM Cloud IAM authenticator
	authenticator := &core.IamAuthenticator{
		ApiKey: apiKey, // pragma: allowlist secret
	}

	// Set custom IAM URL if provided (for staging/dev environments)
	// Production uses default: https://iam.cloud.ibm.com
	// Staging uses: https://iam.test.cloud.ibm.com
	if iamURL != "" {
		authenticator.URL = iamURL
		logger.Info("Using custom IAM endpoint", zap.String("iam_url", iamURL))
	}

	// Validate the authenticator
	if err := authenticator.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate authenticator: %w", err)
	}

	logger.Info("IBM Cloud IAM authenticator initialized successfully")

	return &Authenticator{
		authenticator: authenticator,
		logger:        logger,
	}, nil
}

// Authenticate adds authentication to an HTTP request
func (a *Authenticator) Authenticate(req *http.Request) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Use IBM SDK to authenticate the request
	// This automatically handles bearer token generation and refresh
	err := a.authenticator.Authenticate(req)
	if err != nil {
		// Log authentication failure with sanitized context (no sensitive data)
		a.logger.Warn("Authentication failed",
			zap.Error(err),
			zap.String("target_host", req.URL.Host),
			zap.String("method", req.Method),
			zap.String("path", req.URL.Path),
		)
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Log successful authentication at debug level (no sensitive data)
	a.logger.Debug("Request authenticated successfully",
		zap.String("target_host", req.URL.Host),
		zap.String("method", req.Method),
	)

	return nil
}

// GetToken retrieves the current bearer token (for debugging/monitoring)
func (a *Authenticator) GetToken() (string, error) {
	// This is useful for health checks and monitoring
	if iamAuth, ok := a.authenticator.(*core.IamAuthenticator); ok {
		token, err := iamAuth.RequestToken()
		if err != nil {
			return "", fmt.Errorf("failed to get token: %w", err)
		}
		return token.AccessToken, nil
	}
	return "", fmt.Errorf("unsupported authenticator type")
}

// ValidateToken validates that we can obtain a valid token
func (a *Authenticator) ValidateToken() error {
	_, err := a.GetToken()
	if err != nil {
		a.logger.Warn("Token validation failed", zap.Error(err))
		return err
	}
	a.logger.Debug("Token validation successful")
	return nil
}

// GetUserIdentity extracts user identity from the JWT token
// Returns the subject claim which uniquely identifies the user/service
func (a *Authenticator) GetUserIdentity() (string, error) {
	claims, err := a.GetTokenClaims()
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", fmt.Errorf("token has no subject claim")
	}
	return claims.Subject, nil
}

// GetTokenClaims extracts all claims from the current JWT token
func (a *Authenticator) GetTokenClaims() (*JWTClaims, error) {
	token, err := a.GetToken()
	if err != nil {
		return nil, err
	}
	return parseJWTClaims(token)
}

// parseJWTClaims parses the claims from a JWT token without validation
// (validation is handled by IBM IAM)
func parseJWTClaims(token string) (*JWTClaims, error) {
	// JWT format: header.payload.signature
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format")
	}

	// Decode the payload (second part)
	payload := parts[1]
	// Add padding if needed for base64 decoding
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Try standard base64 if URL encoding fails
		decoded, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode token payload: %w", err)
		}
	}

	var claims JWTClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	return &claims, nil
}
