package privacy

import (
	"strings"
	"testing"
)

// TestStripPrivateData_APIKey tests API key redaction
func TestStripPrivateData_APIKey(t *testing.T) {
	input := `{"api_key": "sk-1234567890abcdef", "data": "normal content"}`
	output := StripPrivateData(input)

	if strings.Contains(output, "sk-1234567890abcdef") {
		t.Error("API key should be redacted")
	}

	if !strings.Contains(output, "[REDACTED]") {
		t.Error("output should contain [REDACTED]")
	}

	if !strings.Contains(output, "normal content") {
		t.Error("normal content should remain")
	}
}

// TestStripPrivateData_Token tests token redaction
func TestStripPrivateData_Token(t *testing.T) {
	input := `"token": "Bearer abc123xyz"`
	output := StripPrivateData(input)

	if strings.Contains(output, "Bearer abc123xyz") {
		t.Error("Bearer token should be redacted")
	}

	if !strings.Contains(output, "[REDACTED]") {
		t.Error("output should contain [REDACTED]")
	}
}

// TestStripPrivateData_Password tests password redaction
func TestStripPrivateData_Password(t *testing.T) {
	// The regex pattern matches: password/key=value style patterns
	input := `"password": "secret123"`
	output := StripPrivateData(input)

	if strings.Contains(output, "secret123") {
		t.Error("password value should be redacted")
	}

	if !strings.Contains(output, "[REDACTED]") {
		t.Error("output should contain [REDACTED]")
	}
}

// TestStripPrivateData_JWT tests JWT token redaction
func TestStripPrivateData_JWT(t *testing.T) {
	input := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`
	output := StripPrivateData(input)

	if strings.Contains(output, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") {
		t.Error("JWT should be redacted")
	}

	if !strings.Contains(output, "[REDACTED]") {
		t.Error("output should contain [REDACTED]")
	}
}

// TestStripPrivateData_PrivateTags tests private tag removal
func TestStripPrivateData_PrivateTags(t *testing.T) {
	input := `{"content": "public <private>secret data</private> more content"}`
	output := StripPrivateData(input)

	if strings.Contains(output, "secret data") {
		t.Error("private tag content should be redacted")
	}

	if !strings.Contains(output, "public") || !strings.Contains(output, "more content") {
		t.Error("public content should remain")
	}
}

// TestStripPrivateData_AWSKey tests AWS key redaction
func TestStripPrivateData_AWSKey(t *testing.T) {
	input := `{"aws_key": "AKIAIOSFODNN7EXAMPLE"}`
	output := StripPrivateData(input)

	if strings.Contains(output, "AKIAIOSFODNN7EXAMPLE") {
		t.Error("AWS key should be redacted")
	}
}

// TestStripPrivateData_NoSensitiveData tests non-sensitive data
func TestStripPrivateData_NoSensitiveData(t *testing.T) {
	input := `{"name": "John", "age": 30, "city": "New York"}`
	output := StripPrivateData(input)

	if strings.Contains(output, "[REDACTED]") {
		t.Error("non-sensitive data should not be redacted")
	}

	if output != input {
		t.Errorf("non-sensitive data should remain unchanged: %s vs %s", input, output)
	}
}

// TestStripPrivateDataFromMap_Basic tests map filtering
func TestStripPrivateDataFromMap_Basic(t *testing.T) {
	// StripPrivateDataFromMap only filters string values, not field names
	// The password field value is a string, so it will be passed through StripPrivateData
	data := map[string]interface{}{
		"password": "api_key=secret123", // This contains a pattern that will be matched
		"content":  "normal content",
		"config":   "Bearer test-token", // This contains Bearer pattern
	}

	result := StripPrivateDataFromMap(data)

	// Values containing sensitive patterns should be redacted
	if !strings.Contains(result["password"].(string), "[REDACTED]") {
		t.Errorf("password value containing api_key pattern should be redacted, got: %v", result["password"])
	}

	if !strings.Contains(result["config"].(string), "[REDACTED]") {
		t.Errorf("config value containing Bearer pattern should be redacted, got: %v", result["config"])
	}

	if result["content"] != "normal content" {
		t.Errorf("normal content should remain, got: %v", result["content"])
	}
}

// TestStripPrivateDataFromMap_Nested tests nested map filtering
func TestStripPrivateDataFromMap_Nested(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name":     "John",
			"password": "api_key=nested_secret", // Contains matching pattern
		},
		"config": map[string]interface{}{
			"token": "Bearer nested_token", // Contains Bearer pattern
		},
	}

	result := StripPrivateDataFromMap(data)

	userMap, ok := result["user"].(map[string]interface{})
	if !ok {
		t.Fatal("user should be a map")
	}

	// The value contains api_key= pattern which matches the regex
	if !strings.Contains(userMap["password"].(string), "[REDACTED]") {
		t.Errorf("nested password containing pattern should be redacted, got: %v", userMap["password"])
	}

	configMap, ok := result["config"].(map[string]interface{})
	if !ok {
		t.Fatal("config should be a map")
	}

	if !strings.Contains(configMap["token"].(string), "[REDACTED]") {
		t.Errorf("nested token containing Bearer should be redacted, got: %v", configMap["token"])
	}
}

// TestStripPrivateDataFromMap_Empty tests empty map
func TestStripPrivateDataFromMap_Empty(t *testing.T) {
	data := map[string]interface{}{}
	result := StripPrivateDataFromMap(data)

	if len(result) != 0 {
		t.Errorf("empty map should remain empty, got %d items", len(result))
	}
}

// TestIsPrivateData tests private data detection
func TestIsPrivateData(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"api_key=sk-123456", true},
		{"Bearer abc123", true},
		{"AKIAIOSFODNN7EXAMPLE", true},
		{"<private>secret</private>", true},
		{"password=secret", true},
		{"normal content", false},
		{"Hello World", false},
		{"{\"name\": \"John\"}", false},
		// JWT needs full format with dots
		{"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dummy", true},
	}

	for _, tt := range tests {
		actual := IsPrivateData(tt.input)
		if actual != tt.expected {
			t.Errorf("IsPrivateData(%s) = %v, expected %v", tt.input, actual, tt.expected)
		}
	}
}

// TestTruncate tests string truncation
func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10", 10, "exactly10"},
		{"longer than max", 10, "longer tha...[truncated]"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		actual := Truncate(tt.input, tt.maxLen)
		if actual != tt.expected {
			t.Errorf("Truncate(%s, %d) = %s, expected %s", tt.input, tt.maxLen, actual, tt.expected)
		}
	}
}

// TestFilterJSON tests JSON string filtering
func TestFilterJSON(t *testing.T) {
	input := `{"api_key": "sk-123", "token": "Bearer xyz"}`
	output := FilterJSON(input)

	if strings.Contains(output, "sk-123") {
		t.Error("API key should be filtered")
	}

	if strings.Contains(output, "Bearer xyz") {
		t.Error("Bearer token should be filtered")
	}
}

// TestIsSensitiveField tests sensitive field detection
func TestIsSensitiveField(t *testing.T) {
	sensitive := []string{
		"password", "passwd", "pwd",
		"secret", "api_key", "apikey", "api-key",
		"token", "access_token", "auth_token",
		"credential", "private_key",
		"API_KEY", "Secret", "PASSWORD",
	}

	for _, field := range sensitive {
		if !IsSensitiveField(field) {
			t.Errorf("%s should be detected as sensitive", field)
		}
	}

	nonSensitive := []string{
		"name", "email", "content", "data",
		"username", "description", "title",
	}

	for _, field := range nonSensitive {
		if IsSensitiveField(field) {
			t.Errorf("%s should NOT be detected as sensitive", field)
		}
	}
}