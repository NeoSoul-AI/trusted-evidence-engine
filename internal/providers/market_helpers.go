package providers

import (
	"fmt"
	"strings"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func stringMetadata(input schema.ResolveInput, key string) string {
	if input.ContextMetadata == nil {
		return ""
	}
	value, ok := input.ContextMetadata[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func boolMetadata(input schema.ResolveInput, key string) bool {
	value := strings.ToLower(stringMetadata(input, key))
	return value == "true" || value == "1" || value == "yes"
}

func firstNonEmptyMetadata(input schema.ResolveInput, keys ...string) string {
	for _, key := range keys {
		if value := stringMetadata(input, key); value != "" {
			return value
		}
	}
	return ""
}

func seemsPriceQuery(input schema.ResolveInput) bool {
	if boolMetadata(input, "want_price") {
		return true
	}

	text := strings.ToLower(strings.TrimSpace(input.Query + " " + input.Claim))
	if text == "" {
		return false
	}
	priceWords := []string{
		"price",
		"quote",
		"quoted",
		"market cap",
		"last price",
		"24h",
		"24hr",
		"24-hour",
		"ticker",
	}
	for _, word := range priceWords {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}
