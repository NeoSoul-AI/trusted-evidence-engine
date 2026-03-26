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

type CoinGeckoPriceOption func(*CoinGeckoPriceProvider)

type CoinGeckoPriceProvider struct {
	apiKey  string
	plan    string
	baseURL string
	client  *http.Client
	now     func() time.Time
}

func NewCoinGeckoPriceProvider(apiKey string, opts ...CoinGeckoPriceOption) (*CoinGeckoPriceProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("coingecko api key is required")
	}

	provider := &CoinGeckoPriceProvider{
		apiKey:  strings.TrimSpace(apiKey),
		plan:    "demo",
		baseURL: coinGeckoDemoBaseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		now: func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(provider)
	}
	if provider.plan == "pro" && provider.baseURL == coinGeckoDemoBaseURL {
		provider.baseURL = coinGeckoProBaseURL
	}
	return provider, nil
}

func NewCoinGeckoPriceProviderFromEnv() (*CoinGeckoPriceProvider, bool) {
	apiKey := strings.TrimSpace(envOrDefault("COINGECKO_API_KEY", ""))
	if apiKey == "" {
		return nil, false
	}

	plan := strings.ToLower(envOrDefault("TEE_COINGECKO_PLAN", "demo"))
	baseURL := envOrDefault("TEE_COINGECKO_BASE_URL", "")
	if baseURL == "" {
		if plan == "pro" {
			baseURL = coinGeckoProBaseURL
		} else {
			baseURL = coinGeckoDemoBaseURL
		}
	}
	provider, err := NewCoinGeckoPriceProvider(
		apiKey,
		WithCoinGeckoPricePlan(plan),
		WithCoinGeckoPriceBaseURL(baseURL),
	)
	if err != nil {
		return nil, false
	}
	return provider, true
}

func WithCoinGeckoPricePlan(plan string) CoinGeckoPriceOption {
	return func(provider *CoinGeckoPriceProvider) {
		plan = strings.ToLower(strings.TrimSpace(plan))
		if plan == "pro" || plan == "demo" {
			provider.plan = plan
		}
	}
}

func WithCoinGeckoPriceBaseURL(baseURL string) CoinGeckoPriceOption {
	return func(provider *CoinGeckoPriceProvider) {
		if strings.TrimSpace(baseURL) != "" {
			provider.baseURL = strings.TrimSpace(baseURL)
		}
	}
}

func WithCoinGeckoPriceHTTPClient(client *http.Client) CoinGeckoPriceOption {
	return func(provider *CoinGeckoPriceProvider) {
		if client != nil {
			provider.client = client
		}
	}
}

func (provider *CoinGeckoPriceProvider) Name() string {
	return "coingecko_simple_price"
}

func (provider *CoinGeckoPriceProvider) Fetch(ctx context.Context, input schema.ResolveInput) ([]schema.SourceDocument, error) {
	identifierType, identifierValue, vsCurrency := coinGeckoPriceRequest(input)
	if identifierType == "" || identifierValue == "" {
		return nil, nil
	}

	base, err := url.Parse(provider.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse coingecko base url: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/simple/price"
	query := base.Query()
	query.Set("vs_currencies", vsCurrency)
	query.Set(identifierType, identifierValue)
	query.Set("include_market_cap", "true")
	query.Set("include_24hr_vol", "true")
	query.Set("include_24hr_change", "true")
	query.Set("include_last_updated_at", "true")
	base.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create coingecko price request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if provider.plan == "pro" {
		req.Header.Set("x-cg-pro-api-key", provider.apiKey)
	} else {
		req.Header.Set("x-cg-demo-api-key", provider.apiKey)
	}

	resp, err := provider.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request coingecko price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, fmt.Errorf("coingecko price returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode coingecko price: %w", err)
	}

	var key string
	for candidate := range payload {
		key = candidate
		break
	}
	if key == "" {
		return nil, nil
	}
	values := payload[key]
	fetchedAt := provider.now()
	title := fmt.Sprintf("CoinGecko price for %s", key)
	priceKey := strings.ToLower(vsCurrency)
	urlValue := "https://www.coingecko.com/"
	if identifierType == "ids" {
		urlValue = "https://www.coingecko.com/en/coins/" + key
	}

	return []schema.SourceDocument{
		{
			Title:       title,
			URL:         urlValue,
			Domain:      "www.coingecko.com",
			SourceType:  "crypto_quote",
			Snippet:     buildCoinGeckoPriceSnippet(key, priceKey, values),
			Provider:    provider.Name(),
			FetchedAt:   fetchedAt,
			PublishedAt: timestampFromUnix(values["last_updated_at"]),
			Metadata: map[string]any{
				"lookup_type":     identifierType,
				"lookup_value":    identifierValue,
				"asset_key":       key,
				"vs_currency":     priceKey,
				"price":           values[priceKey],
				"market_cap":      values[priceKey+"_market_cap"],
				"volume_24h":      values[priceKey+"_24h_vol"],
				"change_24h":      values[priceKey+"_24h_change"],
				"last_updated_at": values["last_updated_at"],
				"provider_plan":   provider.plan,
			},
		},
	}, nil
}

func coinGeckoPriceRequest(input schema.ResolveInput) (identifierType, identifierValue, vsCurrency string) {
	if !seemsPriceQuery(input) &&
		firstNonEmptyMetadata(input, "coin_id", "crypto_symbol", "crypto_name") == "" {
		return "", "", ""
	}

	vsCurrency = strings.ToLower(firstNonEmptyMetadata(input, "vs_currency", "price_vs_currency"))
	if vsCurrency == "" {
		vsCurrency = "usd"
	}

	if coinID := firstNonEmptyMetadata(input, "coin_id"); coinID != "" {
		return "ids", strings.ToLower(coinID), vsCurrency
	}
	if symbol := firstNonEmptyMetadata(input, "crypto_symbol"); symbol != "" {
		return "symbols", strings.ToLower(symbol), vsCurrency
	}
	if name := firstNonEmptyMetadata(input, "crypto_name"); name != "" {
		return "names", name, vsCurrency
	}

	compactQuery := strings.TrimSpace(input.Query)
	if compactQuery != "" && !looksLikeURL(compactQuery) && len(strings.Fields(compactQuery)) == 1 {
		return "symbols", strings.ToLower(compactQuery), vsCurrency
	}
	return "", "", ""
}

func buildCoinGeckoPriceSnippet(key, vsCurrency string, values map[string]any) string {
	parts := []string{fmt.Sprintf("%s/%s", strings.ToUpper(key), strings.ToUpper(vsCurrency))}
	if price, ok := values[vsCurrency]; ok {
		parts = append(parts, fmt.Sprintf("price=%v", price))
	}
	if change, ok := values[vsCurrency+"_24h_change"]; ok {
		parts = append(parts, fmt.Sprintf("24h_change=%v", change))
	}
	if marketCap, ok := values[vsCurrency+"_market_cap"]; ok {
		parts = append(parts, fmt.Sprintf("market_cap=%v", marketCap))
	}
	return strings.Join(parts, " | ")
}

func timestampFromUnix(value any) *time.Time {
	switch typed := value.(type) {
	case float64:
		timestamp := time.Unix(int64(typed), 0).UTC()
		return &timestamp
	case int64:
		timestamp := time.Unix(typed, 0).UTC()
		return &timestamp
	case int:
		timestamp := time.Unix(int64(typed), 0).UTC()
		return &timestamp
	default:
		return nil
	}
}
