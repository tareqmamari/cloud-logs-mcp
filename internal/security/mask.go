// Package security provides security utilities for the MCP server.
package security

import (
	"regexp"
	"strings"
)

// MaskAPIKey masks an API key, showing only the first 4 and last 4 characters
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

// MaskBearerToken masks a bearer token for safe logging
func MaskBearerToken(token string) string {
	if len(token) <= 10 {
		return "***REDACTED***"
	}
	return token[:6] + "..." + token[len(token)-4:]
}

// MaskSensitiveHeaders masks sensitive values in HTTP headers
func MaskSensitiveHeaders(headers map[string][]string) map[string]string {
	masked := make(map[string]string)
	sensitiveHeaders := map[string]bool{ // pragma: allowlist secret
		"authorization":    true,
		"x-api-key":        true,
		"api-key":          true,
		"apikey":           true, // pragma: allowlist secret
		"x-auth-token":     true,
		"cookie":           true,
		"set-cookie":       true,
		"x-csrf-token":     true,
		"x-request-id":     false, // Not sensitive, don't mask
		"x-trace-id":       false,
		"x-correlation-id": false,
	}

	for key, values := range headers {
		keyLower := strings.ToLower(key)
		if sensitiveHeaders[keyLower] {
			masked[key] = "***REDACTED***"
		} else if len(values) > 0 {
			masked[key] = values[0]
			if len(values) > 1 {
				masked[key] += "..."
			}
		}
	}

	return masked
}

// SensitivePatterns contains regex patterns for sensitive data
var SensitivePatterns = []*regexp.Regexp{
	// API keys (various formats)
	regexp.MustCompile(`(?i)(api[_-]?key|apikey)[=:]["']?([a-zA-Z0-9_-]{20,})["']?`),
	// Bearer tokens
	regexp.MustCompile(`(?i)(bearer\s+)([a-zA-Z0-9_.-]{20,})`),
	// IBM Cloud API keys (specific format)
	regexp.MustCompile(`(?i)([a-zA-Z0-9]{44})`),
	// Passwords in URLs or config
	regexp.MustCompile(`(?i)(password|passwd|pwd)[=:]["']?([^"'\s&]+)["']?`),
	// Secrets
	regexp.MustCompile(`(?i)(secret|token)[=:]["']?([a-zA-Z0-9_-]{16,})["']?`),
}

// MaskSensitiveData masks sensitive data in a string using pattern matching
func MaskSensitiveData(data string) string {
	result := data

	for _, pattern := range SensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Keep the key name, mask the value
			parts := pattern.FindStringSubmatch(match)
			if len(parts) >= 3 {
				return parts[1] + "***REDACTED***"
			}
			return "***REDACTED***"
		})
	}

	return result
}

// MaskURL masks sensitive query parameters in URLs
func MaskURL(rawURL string) string {
	sensitiveParams := []string{
		"api_key", "apikey", "api-key",
		"token", "access_token", "auth_token",
		"password", "passwd", "pwd",
		"secret", "key",
	}

	result := rawURL
	for _, param := range sensitiveParams {
		// Match param=value pattern
		pattern := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(param) + `=)([^&\s]+)`)
		result = pattern.ReplaceAllString(result, "${1}***REDACTED***")
	}

	return result
}

// IsSensitiveField checks if a field name indicates sensitive data
func IsSensitiveField(fieldName string) bool {
	sensitiveNames := []string{
		"password", "passwd", "pwd",
		"secret", "token", "key", "apikey", "api_key",
		"authorization", "auth", "credential",
		"private", "ssh", "certificate", "cert",
	}

	fieldLower := strings.ToLower(fieldName)
	for _, name := range sensitiveNames {
		if strings.Contains(fieldLower, name) {
			return true
		}
	}

	return false
}

// SanitizeError removes sensitive data from error messages
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return MaskSensitiveData(err.Error())
}
