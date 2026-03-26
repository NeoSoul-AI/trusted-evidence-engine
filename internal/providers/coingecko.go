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

const (
	coinGeckoDemoBaseURL = "https://api.coingecko.com/api/v3"
	coinGeckoProBaseURL  = "https://pro-api.coingecko.com/api/v3"
)

type CoinGeckoOption func(*CoinGeckoProvider)

type CoinGeckoProvider struct {
	apiKey  string
	plan    string
	baseURL string
	perPage int
	client  *http.Client
	now     func() time.Time
}

type coinGeckoSearchResponse struct {
	Coins []coinGeckoCoin `json:"coins"`
}

type coinGeckoCoin struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	APISymbol     string `json:"api_symbol"`
	Symbol        string `json:"symbol"`
	MarketCapRank int    `json:"market_cap_rank"`
	Thumb         string `json:"thumb"`
	Large         string `json:"large"`
}

func NewCoinGeckoProvider(apiKey string, opts ...CoinGeckoOption) (*CoinGeckoProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("coingecko api key is required")
	}

	provider := &CoinGeckoProvider{
		apiKey:  strings.TrimSpace(apiKey),
		plan:    "demo",
		baseURL: coinGeckoDemoBaseURL,
		perPage: 5,
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

func NewCoinGeckoProviderFromEnv() (*CoinGeckoProvider, bool) {
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
	provider, err := NewCoinGeckoProvider(
		apiKey,
		WithCoinGeckoPlan(plan),
		WithCoinGeckoBaseURL(baseURL),
		WithCoinGeckoPerPage(envIntOrDefault("TEE_COINGECKO_PER_PAGE", 5)),
	)
	if err != nil {
		return nil, false
	}
	return provider, true
}

func WithCoinGeckoPlan(plan string) CoinGeckoOption {
	return func(provider *CoinGeckoProvider) {
		plan = strings.ToLower(strings.TrimSpace(plan))
		if plan == "pro" || plan == "demo" {
			provider.plan = plan
		}
	}
}

func WithCoinGeckoBaseURL(baseURL string) CoinGeckoOption {
	return func(provider *CoinGeckoProvider) {
		if strings.TrimSpace(baseURL) != "" {
			provider.baseURL = strings.TrimSpace(baseURL)
		}
	}
}

func WithCoinGeckoPerPage(perPage int) CoinGeckoOption {
	return func(provider *CoinGeckoProvider) {
		if perPage > 0 {
			provider.perPage = perPage
		}
	}
}

func WithCoinGeckoHTTPClient(client *http.Client) CoinGeckoOption {
	return func(provider *CoinGeckoProvider) {
		if client != nil {
			provider.client = client
		}
	}
}

func (provider *CoinGeckoProvider) Name() string {
	return "coingecko_search"
}

func (provider *CoinGeckoProvider) Fetch(ctx context.Context, input schema.ResolveInput) ([]schema.SourceDocument, error) {
	subject := cryptoSubject(input)
	if subject == "" {
		return nil, nil
	}

	base, err := url.Parse(provider.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse coingecko base url: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/search"
	query := base.Query()
	query.Set("query", subject)
	base.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create coingecko request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if provider.plan == "pro" {
		req.Header.Set("x-cg-pro-api-key", provider.apiKey)
	} else {
		req.Header.Set("x-cg-demo-api-key", provider.apiKey)
	}

	resp, err := provider.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request coingecko search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, fmt.Errorf("coingecko search returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload coinGeckoSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode coingecko search: %w", err)
	}

	limit := provider.perPage
	if len(payload.Coins) < limit {
		limit = len(payload.Coins)
	}
	fetchedAt := provider.now()
	documents := make([]schema.SourceDocument, 0, limit)
	for _, coin := range payload.Coins[:limit] {
		coinID := strings.TrimSpace(coin.ID)
		if coinID == "" {
			continue
		}
		documents = append(documents, schema.SourceDocument{
			Title:      firstNonEmpty(strings.TrimSpace(coin.Name), coinID),
			URL:        "https://www.coingecko.com/en/coins/" + coinID,
			Domain:     "www.coingecko.com",
			SourceType: "crypto_market_data",
			Snippet:    buildCoinGeckoSnippet(coin),
			Provider:   provider.Name(),
			FetchedAt:  fetchedAt,
			Metadata: map[string]any{
				"coin_id":         coinID,
				"symbol":          strings.TrimSpace(coin.Symbol),
				"api_symbol":      strings.TrimSpace(coin.APISymbol),
				"market_cap_rank": coin.MarketCapRank,
				"thumb":           strings.TrimSpace(coin.Thumb),
				"large":           strings.TrimSpace(coin.Large),
				"plan":            provider.plan,
			},
		})
	}
	return documents, nil
}

func buildCoinGeckoSnippet(coin coinGeckoCoin) string {
	parts := []string{}
	if name := strings.TrimSpace(coin.Name); name != "" {
		parts = append(parts, name)
	}
	if symbol := strings.TrimSpace(coin.Symbol); symbol != "" {
		parts = append(parts, "symbol="+strings.ToUpper(symbol))
	}
	if coin.MarketCapRank > 0 {
		parts = append(parts, fmt.Sprintf("market_cap_rank=%d", coin.MarketCapRank))
	}
	return strings.Join(parts, " | ")
}

func cryptoSubject(input schema.ResolveInput) string {
	candidates := []string{
		strings.TrimSpace(input.Query),
		strings.TrimSpace(input.Claim),
	}
	for _, candidate := range candidates {
		if candidate != "" && !looksLikeURL(candidate) {
			return candidate
		}
	}
	return ""
}
