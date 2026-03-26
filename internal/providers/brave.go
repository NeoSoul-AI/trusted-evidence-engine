package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

const braveWebSearchEndpoint = "https://api.search.brave.com/res/v1/web/search"

type BraveOption func(*BraveProvider)

type BraveProvider struct {
	apiKey     string
	endpoint   string
	country    string
	searchLang string
	count      int
	client     *http.Client
	now        func() time.Time
}

type braveWebSearchResponse struct {
	Web *struct {
		Results []braveWebResult `json:"results"`
	} `json:"web"`
}

type braveWebResult struct {
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Description   string   `json:"description"`
	PageAge       string   `json:"page_age"`
	Age           string   `json:"age"`
	Language      string   `json:"language"`
	ExtraSnippets []string `json:"extra_snippets"`
	Profile       *struct {
		Name     string `json:"name"`
		LongName string `json:"long_name"`
		URL      string `json:"url"`
	} `json:"profile"`
}

func WithBraveEndpoint(endpoint string) BraveOption {
	return func(provider *BraveProvider) {
		if strings.TrimSpace(endpoint) != "" {
			provider.endpoint = strings.TrimSpace(endpoint)
		}
	}
}

func WithBraveHTTPClient(client *http.Client) BraveOption {
	return func(provider *BraveProvider) {
		if client != nil {
			provider.client = client
		}
	}
}

func WithBraveCountry(country string) BraveOption {
	return func(provider *BraveProvider) {
		if strings.TrimSpace(country) != "" {
			provider.country = strings.TrimSpace(country)
		}
	}
}

func WithBraveSearchLang(searchLang string) BraveOption {
	return func(provider *BraveProvider) {
		if strings.TrimSpace(searchLang) != "" {
			provider.searchLang = strings.TrimSpace(searchLang)
		}
	}
}

func WithBraveCount(count int) BraveOption {
	return func(provider *BraveProvider) {
		if count > 0 {
			provider.count = count
		}
	}
}

func NewBraveProvider(apiKey string, opts ...BraveOption) (*BraveProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("brave api key is required")
	}

	provider := &BraveProvider{
		apiKey:     strings.TrimSpace(apiKey),
		endpoint:   braveWebSearchEndpoint,
		country:    "us",
		searchLang: "en",
		count:      10,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		now: func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(provider)
	}
	return provider, nil
}

func NewBraveProviderFromEnv() (*BraveProvider, bool) {
	apiKey := strings.TrimSpace(envOrDefault("BRAVE_SEARCH_API_KEY", ""))
	if apiKey == "" {
		return nil, false
	}

	provider, err := NewBraveProvider(
		apiKey,
		WithBraveEndpoint(envOrDefault("TEE_BRAVE_WEB_ENDPOINT", braveWebSearchEndpoint)),
		WithBraveCountry(envOrDefault("TEE_BRAVE_COUNTRY", "us")),
		WithBraveSearchLang(envOrDefault("TEE_BRAVE_SEARCH_LANG", "en")),
		WithBraveCount(envIntOrDefault("TEE_BRAVE_COUNT", 10)),
	)
	if err != nil {
		return nil, false
	}
	return provider, true
}

func (provider *BraveProvider) Name() string {
	return "brave_search"
}

func (provider *BraveProvider) Fetch(ctx context.Context, input schema.ResolveInput) ([]schema.SourceDocument, error) {
	subject := strings.TrimSpace(input.Query)
	if subject == "" {
		subject = strings.TrimSpace(input.Claim)
	}
	if subject == "" && len(input.SourceURLs) > 0 {
		subject = strings.TrimSpace(input.SourceURLs[0])
	}
	if subject == "" {
		return nil, nil
	}

	endpoint, err := url.Parse(provider.endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse brave endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("q", subject)
	query.Set("count", fmt.Sprintf("%d", provider.count))
	query.Set("country", provider.country)
	query.Set("search_lang", provider.searchLang)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create brave request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", provider.apiKey)

	resp, err := provider.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request brave search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, fmt.Errorf("brave search returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload braveWebSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode brave response: %w", err)
	}
	if payload.Web == nil || len(payload.Web.Results) == 0 {
		return nil, nil
	}

	fetchedAt := provider.now()
	documents := make([]schema.SourceDocument, 0, len(payload.Web.Results))
	for _, result := range payload.Web.Results {
		if strings.TrimSpace(result.URL) == "" {
			continue
		}

		documents = append(documents, schema.SourceDocument{
			Title:       firstNonEmpty(strings.TrimSpace(result.Title), strings.TrimSpace(result.URL)),
			URL:         strings.TrimSpace(result.URL),
			Domain:      hostOf(result.URL),
			SourceType:  inferSourceTypeFromDomain(hostOf(result.URL)),
			Snippet:     buildBraveSnippet(result),
			PublishedAt: parseBravePublishedAt(result.PageAge),
			Provider:    provider.Name(),
			FetchedAt:   fetchedAt,
			Metadata:    buildBraveMetadata(result),
		})
	}
	return documents, nil
}

func buildBraveSnippet(result braveWebResult) string {
	parts := make([]string, 0, 1+len(result.ExtraSnippets))
	if strings.TrimSpace(result.Description) != "" {
		parts = append(parts, strings.TrimSpace(result.Description))
	}
	for _, snippet := range result.ExtraSnippets {
		snippet = strings.TrimSpace(snippet)
		if snippet != "" {
			parts = append(parts, snippet)
		}
	}
	return strings.Join(parts, " ")
}

func buildBraveMetadata(result braveWebResult) map[string]any {
	metadata := map[string]any{}
	if strings.TrimSpace(result.Language) != "" {
		metadata["language"] = strings.TrimSpace(result.Language)
	}
	if strings.TrimSpace(result.Age) != "" {
		metadata["age"] = strings.TrimSpace(result.Age)
	}
	if result.Profile != nil {
		if strings.TrimSpace(result.Profile.Name) != "" {
			metadata["profile_name"] = strings.TrimSpace(result.Profile.Name)
		}
		if strings.TrimSpace(result.Profile.LongName) != "" {
			metadata["profile_long_name"] = strings.TrimSpace(result.Profile.LongName)
		}
		if strings.TrimSpace(result.Profile.URL) != "" {
			metadata["profile_url"] = strings.TrimSpace(result.Profile.URL)
		}
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func parseBravePublishedAt(raw string) *time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}

	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC1123,
		time.RFC1123Z,
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
}

func inferSourceTypeFromDomain(domain string) string {
	normalized := strings.TrimSpace(strings.ToLower(domain))
	switch {
	case normalized == "":
		return "reference"
	case strings.HasSuffix(normalized, ".gov"):
		return "official"
	case strings.HasSuffix(normalized, ".edu"):
		return "reference"
	case strings.Contains(normalized, "news"),
		strings.Contains(normalized, "reuters"),
		strings.Contains(normalized, "apnews"),
		strings.Contains(normalized, "bloomberg"),
		strings.Contains(normalized, "bbc"),
		strings.Contains(normalized, "cnn"):
		return "news"
	default:
		return "reference"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
