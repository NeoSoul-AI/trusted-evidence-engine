package dedup

import (
	"strings"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func Merge(documents []schema.SourceDocument) []schema.SourceDocument {
	if len(documents) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(documents))
	out := make([]schema.SourceDocument, 0, len(documents))
	for _, document := range documents {
		key := identity(document)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, document)
	}
	return out
}

func identity(document schema.SourceDocument) string {
	urlKey := strings.TrimSpace(strings.ToLower(document.URL))
	if urlKey != "" {
		return urlKey
	}
	return strings.TrimSpace(strings.ToLower(document.Provider + "::" + document.Title))
}
