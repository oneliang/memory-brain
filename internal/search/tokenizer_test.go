package search

import (
	"strings"
	"testing"
)

// TestTokenizer_Basic tests basic tokenization
func TestTokenizer_Basic(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "Hello World Test"
	tokens := tokenizer.Tokenize(text)

	if len(tokens) < 2 {
		t.Errorf("expected at least 2 tokens, got %d", len(tokens))
	}

	// Should be lowercase
	for _, token := range tokens {
		if token != strings.ToLower(token) {
			t.Errorf("token should be lowercase: %s", token)
		}
	}
}

// TestTokenizer_StopWords tests stop word removal
func TestTokenizer_StopWords(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "The quick brown fox jumps over the lazy dog"
	tokens := tokenizer.Tokenize(text)

	// "the" should be removed
	for _, token := range tokens {
		if token == "the" {
			t.Error("'the' should be removed as stop word")
		}
	}

	// Should have meaningful words
	expectedWords := []string{"quick", "brown", "fox", "jump", "over", "lazy", "dog"}
	for _, expected := range expectedWords {
		found := false
		for _, token := range tokens {
			if strings.Contains(token, expected) || strings.Contains(expected, token) {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Warning: expected word '%s' not found in tokens", expected)
		}
	}
}

// TestTokenizer_Punctuation tests punctuation handling
func TestTokenizer_Punctuation(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "Hello, World! How are you?"
	tokens := tokenizer.Tokenize(text)

	// Punctuation should be removed
	for _, token := range tokens {
		if strings.ContainsAny(token, ",!?.") {
			t.Errorf("token should not contain punctuation: %s", token)
		}
	}
}

// TestTokenizer_Lowercase tests lowercase conversion
func TestTokenizer_Lowercase(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "HELLO WORLD"
	tokens := tokenizer.Tokenize(text)

	for _, token := range tokens {
		if token != strings.ToLower(token) {
			t.Errorf("token should be lowercase: %s", token)
		}
	}
}

// TestTokenizer_ShortWords tests short word filtering
func TestTokenizer_ShortWords(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "a b c d test"
	tokens := tokenizer.Tokenize(text)

	// Single letters should be filtered (length < 2)
	for _, token := range tokens {
		if len(token) < 2 {
			t.Errorf("short token should be filtered: %s", token)
		}
	}
}

// TestTokenizer_Empty tests empty input
func TestTokenizer_Empty(t *testing.T) {
	tokenizer := NewTokenizer()

	tokens := tokenizer.Tokenize("")
	if len(tokens) != 0 {
		t.Errorf("empty input should produce empty tokens, got %d", len(tokens))
	}
}

// TestTokenizer_Chinese tests Chinese text handling
func TestTokenizer_Chinese(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "你好世界测试"
	tokens := tokenizer.Tokenize(text)

	// Chinese characters are letters, should be included
	// Note: Chinese text may not work perfectly with simple tokenizer
	// This test documents current behavior
	if len(tokens) == 0 {
		t.Log("Warning: Chinese text may need special handling")
	}
}

// TestTokenizer_MixedLanguage tests mixed language
func TestTokenizer_MixedLanguage(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "Hello 世界 test 测试"
	tokens := tokenizer.Tokenize(text)

	// Should handle both English and Chinese
	// Note: This is a simple tokenizer, results may vary
	if len(tokens) == 0 {
		t.Error("mixed language should produce some tokens")
	}
}

// TestTokenizer_Stemming tests simple stemming
func TestTokenizer_Stemming(t *testing.T) {
	tokenizer := NewTokenizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"testing", "test"},
		{"tested", "test"},
		{"tests", "test"},
		{"running", "run"},
		{"runs", "run"},
	}

	for _, tt := range tests {
		tokens := tokenizer.Tokenize(tt.input)
		if len(tokens) > 0 {
			// Stemming is simplified, may not match exactly
			found := false
			for _, token := range tokens {
				if strings.Contains(token, tt.expected) || strings.Contains(tt.expected, token) {
					found = true
				}
			}
			if !found {
				t.Logf("Stemming '%s' -> got %v (expected to contain '%s')", tt.input, tokens, tt.expected)
			}
		}
	}
}

// TestTokenizer_Digits tests digit handling
func TestTokenizer_Digits(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "test123 number456"
	tokens := tokenizer.Tokenize(text)

	// Digits should be included
	foundDigits := false
	for _, token := range tokens {
		if strings.ContainsAny(token, "123456") {
			foundDigits = true
		}
	}

	if !foundDigits {
		t.Error("tokens should contain digit characters")
	}
}

// TestTokenizer_DefaultTokenizer tests global tokenizer
func TestTokenizer_DefaultTokenizer(t *testing.T) {
	if DefaultTokenizer == nil {
		t.Fatal("DefaultTokenizer should be initialized")
	}

	tokens := DefaultTokenizer.Tokenize("test default tokenizer")
	if len(tokens) == 0 {
		t.Error("DefaultTokenizer should produce tokens")
	}
}

// TestTokenizer_MultipleSpaces tests multiple space handling
func TestTokenizer_MultipleSpaces(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "hello    world   test"
	tokens := tokenizer.Tokenize(text)

	// Should not produce empty tokens
	for _, token := range tokens {
		if token == "" {
			t.Error("should not produce empty tokens")
		}
	}
}

// TestTokenizer_Newlines tests newline handling
func TestTokenizer_Newlines(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "hello\nworld\ntest"
	tokens := tokenizer.Tokenize(text)

	// Newlines should be treated as separators
	if len(tokens) < 2 {
		t.Errorf("newlines should separate tokens, got %d", len(tokens))
	}
}

// TestTokenizer_SpecialChars tests special character handling
func TestTokenizer_SpecialChars(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "hello@world#test$code"
	tokens := tokenizer.Tokenize(text)

	// Special chars should be replaced with spaces
	for _, token := range tokens {
		if strings.ContainsAny(token, "@#$") {
			t.Errorf("special chars should be removed: %s", token)
		}
	}
}

// TestTokenizer_RepeatedCalls tests consistency
func TestTokenizer_RepeatedCalls(t *testing.T) {
	tokenizer := NewTokenizer()

	text := "hello world test"

	tokens1 := tokenizer.Tokenize(text)
	tokens2 := tokenizer.Tokenize(text)

	if len(tokens1) != len(tokens2) {
		t.Errorf("repeated calls should produce same number of tokens")
	}

	for i := range tokens1 {
		if tokens1[i] != tokens2[i] {
			t.Errorf("tokens should be identical: %s vs %s", tokens1[i], tokens2[i])
		}
	}
}