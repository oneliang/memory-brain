package privacy

import (
	"regexp"
	"strings"
)

// PrivateDataPatterns defines regex patterns for sensitive data
var privateDataPatterns = []string{
	// API keys, tokens, secrets
	`(?i)(api[_-]?key|token|secret|password|auth|credential)["']?\s*[:=]\s*["']?[^"'}\s]+`,
	// Bearer tokens
	`(?i)Bearer\s+[A-Za-z0-9\-._~+/]+=*`,
	// JWT tokens
	`(?i)eyJ[A-Za-z0-9\-._~+/]+=*\.eyJ[A-Za-z0-9\-._~+/]+=*\.[A-Za-z0-9\-._~+/]+=*`,
	// AWS keys
	`(?i)AKIA[0-9A-Z]{16}`,
	// Private tags
	`(?i)<private>.*?</private>`,
	// Email addresses (optional)
	// `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
}

// compiledPatterns caches compiled regex patterns
var compiledPatterns []*regexp.Regexp

func init() {
	for _, pattern := range privateDataPatterns {
		compiledPatterns = append(compiledPatterns, regexp.MustCompile(pattern))
	}
}

// StripPrivateData removes sensitive information from JSON string
func StripPrivateData(data string) string {
	result := data
	for _, re := range compiledPatterns {
		result = re.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}

// StripPrivateDataFromMap removes sensitive information from a map
func StripPrivateDataFromMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		switch val := v.(type) {
		case string:
			result[k] = StripPrivateData(val)
		case map[string]interface{}:
			result[k] = StripPrivateDataFromMap(val)
		default:
			result[k] = v
		}
	}
	return result
}

// IsPrivateData checks if a string contains private data
func IsPrivateData(data string) bool {
	for _, re := range compiledPatterns {
		if re.MatchString(data) {
			return true
		}
	}
	return false
}

// Truncate truncates data to specified length (borrowed from agentmemory)
func Truncate(data string, maxLen int) string {
	if len(data) <= maxLen {
		return data
	}
	return data[:maxLen] + "...[truncated]"
}

// FilterJSON filters private data from JSON string
func FilterJSON(jsonStr string) string {
	// Simple approach: regex replace on the raw JSON string
	return StripPrivateData(jsonStr)
}

// Common field names that might contain sensitive data
var sensitiveFields = []string{
	"password", "passwd", "pwd",
	"secret", "api_key", "apikey", "api-key",
	"token", "access_token", "auth_token",
	"credential", "private_key",
}

// IsSensitiveField checks if a field name is sensitive
func IsSensitiveField(field string) bool {
	fieldLower := strings.ToLower(field)
	for _, sf := range sensitiveFields {
		if strings.Contains(fieldLower, strings.ToLower(sf)) {
			return true
		}
	}
	return false
}