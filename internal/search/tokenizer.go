package search

import (
	"strings"
	"unicode"
)

// Tokenizer handles text tokenization
type Tokenizer struct {
	stopWords map[string]bool
}

// NewTokenizer creates a new tokenizer
func NewTokenizer() *Tokenizer {
	// Common English stop words
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"shall": true, "can": true, "need": true, "dare": true, "ought": true,
		"used": true, "to": true, "of": true, "in": true, "for": true,
		"on": true, "with": true, "at": true, "by": true, "from": true,
		"as": true, "into": true, "through": true, "during": true,
		"before": true, "after": true, "above": true, "below": true,
		"between": true, "under": true, "again": true, "further": true,
		"then": true, "once": true, "here": true, "there": true, "when": true,
		"where": true, "why": true, "how": true, "all": true, "each": true,
		"few": true, "more": true, "most": true, "other": true, "some": true,
		"such": true, "no": true, "nor": true, "not": true, "only": true,
		"own": true, "same": true, "so": true, "than": true, "too": true,
		"very": true, "s": true, "t": true, "just": true, "don": true,
		"now": true, "i": true, "me": true, "my": true, "myself": true,
		"we": true, "our": true, "ours": true, "ourselves": true,
		"you": true, "your": true, "yours": true, "yourself": true,
		"he": true, "him": true, "his": true, "himself": true,
		"she": true, "her": true, "hers": true, "herself": true,
		"it": true, "its": true, "itself": true, "they": true,
		"them": true, "their": true, "theirs": true, "themselves": true,
		"what": true, "which": true, "who": true, "whom": true,
		"this": true, "that": true, "these": true, "those": true,
		"am": true, "if": true, "but": true, "because": true,
		"until": true, "while": true, "although": true, "though": true,
	}
	return &Tokenizer{stopWords: stopWords}
}

// Tokenize splits text into tokens
func (t *Tokenizer) Tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Replace punctuation with spaces
	var result strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else {
			result.WriteRune(' ')
		}
	}

	// Split by spaces
	words := strings.Fields(result.String())

	// Filter stop words and short words
	var tokens []string
	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		if t.stopWords[word] {
			continue
		}
		// Simple stemming: remove common suffixes
		word = simpleStem(word)
		tokens = append(tokens, word)
	}

	return tokens
}

// simpleStem applies simple stemming rules
func simpleStem(word string) string {
	// Remove common suffixes (simplified)
	if len(word) > 4 {
		if strings.HasSuffix(word, "ing") {
			return word[:len(word)-3]
		}
		if strings.HasSuffix(word, "tion") {
			return word[:len(word)-4]
		}
		if strings.HasSuffix(word, "ed") {
			return word[:len(word)-2]
		}
	}
	if len(word) > 3 {
		if strings.HasSuffix(word, "s") {
			return word[:len(word)-1]
		}
		if strings.HasSuffix(word, "es") {
			return word[:len(word)-2]
		}
	}
	return word
}

// DefaultTokenizer is the global tokenizer
var DefaultTokenizer = NewTokenizer()