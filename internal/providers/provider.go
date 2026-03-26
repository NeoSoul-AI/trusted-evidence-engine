package providers

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

type Provider interface {
	Name() string
	Fetch(ctx context.Context, input schema.ResolveInput) ([]schema.SourceDocument, error)
}

type DemoProvider struct{}

func DefaultProviders() []Provider {
	providerHints, err := providerHintsFromPresets()
	if err != nil {
		providerHints = nil
	}

	defaults := []Provider{}
	if providerEnabledByHints("polymarket_gamma", providerHints) {
		defaults = append(defaults, NewPolymarketProvider())
	}
	hasConfiguredRealProvider := false
	if coinGecko, ok := NewCoinGeckoProviderFromEnv(); ok {
		if providerEnabledByHints(coinGecko.Name(), providerHints) {
			defaults = append(defaults, coinGecko)
			hasConfiguredRealProvider = true
		}
		if coinGeckoPrice, ok := NewCoinGeckoPriceProviderFromEnv(); ok {
			if providerEnabledByHints(coinGeckoPrice.Name(), providerHints) {
				defaults = append(defaults, coinGeckoPrice)
				hasConfiguredRealProvider = true
			}
		}
	}
	if alphaVantage, ok := NewAlphaVantageProviderFromEnv(); ok {
		if providerEnabledByHints(alphaVantage.Name(), providerHints) {
			defaults = append(defaults, alphaVantage)
			hasConfiguredRealProvider = true
		}
	}
	if feeds, ok := NewFeedProviderFromEnv(); ok {
		if providerEnabledByHints(feeds.Name(), providerHints) {
			defaults = append(defaults, feeds)
			hasConfiguredRealProvider = true
		}
	}
	if brave, ok := NewBraveProviderFromEnv(); ok {
		if providerEnabledByHints(brave.Name(), providerHints) {
			defaults = append(defaults, brave)
			hasConfiguredRealProvider = true
		}
	}
	if !hasConfiguredRealProvider {
		defaults = append(defaults, DemoProvider{})
	}
	return defaults
}

func (DemoProvider) Name() string {
	return "demo"
}

func (DemoProvider) Fetch(_ context.Context, input schema.ResolveInput) ([]schema.SourceDocument, error) {
	for _, raw := range input.SourceURLs {
		if strings.TrimSpace(raw) != "" {
			return nil, nil
		}
	}
	if looksLikeURL(input.Query) || looksLikeURL(input.Claim) {
		return nil, nil
	}

	subject := canonicalSubject(input)

	now := time.Now().UTC()
	documents := []schema.SourceDocument{
		{
			Title:      fmt.Sprintf("Official reference for %s", subject),
			URL:        "https://example.com/official-result",
			Domain:     hostOf("https://example.com/official-result"),
			SourceType: "official",
			Snippet:    "This is a demo official source document returned by the skeleton provider.",
			Provider:   "demo",
			FetchedAt:  now,
			PublishedAt: func() *time.Time {
				t := now.Add(-2 * time.Hour)
				return &t
			}(),
		},
		{
			Title:      fmt.Sprintf("Reference coverage for %s", subject),
			URL:        "https://news.example.org/article",
			Domain:     hostOf("https://news.example.org/article"),
			SourceType: "news",
			Snippet:    "This is a demo news source document returned by the skeleton provider.",
			Provider:   "demo",
			FetchedAt:  now,
			PublishedAt: func() *time.Time {
				t := now.Add(-6 * time.Hour)
				return &t
			}(),
		},
	}
	return documents, nil
}

func canonicalSubject(input schema.ResolveInput) string {
	subject := strings.TrimSpace(input.Query)
	if subject == "" {
		subject = strings.TrimSpace(input.Claim)
	}
	if subject == "" && len(input.SourceURLs) > 0 {
		subject = strings.TrimSpace(input.SourceURLs[0])
	}
	if subject == "" {
		subject = "trusted evidence engine demo query"
	}
	return subject
}

func hostOf(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Hostname())
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envIntOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	replacer := strings.NewReplacer("\n", ",", "\r", ",", "\t", ",", ";", ",")
	raw = replacer.Replace(raw)
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func looksLikeURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return parsed.Scheme != "" && parsed.Host != ""
}
