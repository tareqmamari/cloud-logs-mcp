package auth

import (
	"fmt"
	"net/http"

	"github.com/IBM/go-sdk-core/v5/core"
	"go.uber.org/zap"
)

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
		ApiKey: apiKey,
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
		a.logger.Error("Authentication failed", zap.Error(err))
		return fmt.Errorf("authentication failed: %w", err)
	}

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
	return err
}
