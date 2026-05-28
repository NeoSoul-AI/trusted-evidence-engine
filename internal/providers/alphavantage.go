package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

const alphaVantageBaseURL = "https://www.alphavantage.co/query"

var tickerPattern = regexp.MustCompile(`^[A-Za-z0-9.\-]{1,12}$`)

type AlphaVantageOption func(*AlphaVantageProvider)

type AlphaVantageProvider struct {
	apiKey      string
	baseURL     string
	client      *http.Client
	now         func() time.Time
	entitlement string
}

type alphaVantageSymbolSearchResponse struct {
	Note        string              `json:"Note"`
	Information string              `json:"Information"`
	BestMatches []map[string]string `json:"bestMatches"`
}

func NewAlphaVantageProvider(apiKey string, opts ...AlphaVantageOption) (*AlphaVantageProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("alphavantage api key is required")
	}

	provider := &AlphaVantageProvider{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: alphaVantageBaseURL,
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

func NewAlphaVantageProviderFromEnv() (*AlphaVantageProvider, bool) {
	apiKey := strings.TrimSpace(envOrDefault("ALPHAVANTAGE_API_KEY", ""))
	if apiKey == "" {
		return nil, false
	}

	provider, err := NewAlphaVantageProvider(
		apiKey,
		WithAlphaVantageBaseURL(envOrDefault("TEE_ALPHAVANTAGE_BASE_URL", alphaVantageBaseURL)),
		WithAlphaVantageEntitlement(envOrDefault("TEE_ALPHAVANTAGE_ENTITLEMENT", "")),
	)
	if err != nil {
		return nil, false
	}
	return provider, true
}

func WithAlphaVantageBaseURL(baseURL string) AlphaVantageOption {
	return func(provider *AlphaVantageProvider) {
		if strings.TrimSpace(baseURL) != "" {
			provider.baseURL = strings.TrimSpace(baseURL)
		}
	}
}

func WithAlphaVantageHTTPClient(client *http.Client) AlphaVantageOption {
	return func(provider *AlphaVantageProvider) {
		if client != nil {
			provider.client = client
		}
	}
}

func WithAlphaVantageEntitlement(entitlement string) AlphaVantageOption {
	return func(provider *AlphaVantageProvider) {
		provider.entitlement = strings.TrimSpace(entitlement)
	}
}

func (provider *AlphaVantageProvider) Name() string {
	return "alphavantage_quote"
}

func (provider *AlphaVantageProvider) Fetch(ctx context.Context, input schema.ResolveInput) ([]schema.SourceDocument, error) {
	symbol, metadata, err := provider.resolveSymbol(ctx, input)
	if err != nil || symbol == "" {
		return nil, err
	}

	values, err := provider.globalQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}

	fetchedAt := provider.now()
	doc := schema.SourceDocument{
		Title:      fmt.Sprintf("Alpha Vantage quote for %s", symbol),
		URL:        "https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=" + url.QueryEscape(symbol),
		Domain:     "www.alphavantage.co",
		SourceType: "equity_quote",
		Snippet:    buildAlphaVantageSnippet(symbol, values),
		Provider:   provider.Name(),
		FetchedAt:  fetchedAt,
		Metadata: map[string]any{
			"symbol":         symbol,
			"open":           stringToFloat(values["02. open"]),
			"high":           stringToFloat(values["03. high"]),
			"low":            stringToFloat(values["04. low"]),
			"price":          stringToFloat(values["05. price"]),
			"volume":         stringToFloat(values["06. volume"]),
			"latest_day":     values["07. latest trading day"],
			"previous_close": stringToFloat(values["08. previous close"]),
			"change":         stringToFloat(values["09. change"]),
			"change_percent": stringToFloat(values["10. change percent"]),
		},
	}
	for key, value := range metadata {
		doc.Metadata[key] = value
	}
	if latestDay := strings.TrimSpace(values["07. latest trading day"]); latestDay != "" {
		doc.PublishedAt = parseTimestamp(latestDay)
	}
	return []schema.SourceDocument{doc}, nil
}

func (provider *AlphaVantageProvider) resolveSymbol(ctx context.Context, input schema.ResolveInput) (string, map[string]any, error) {
	if !seemsPriceQuery(input) &&
		firstNonEmptyMetadata(input, "ticker", "equity_symbol") == "" {
		return "", nil, nil
	}

	if symbol := firstNonEmptyMetadata(input, "ticker", "equity_symbol"); symbol != "" {
		return strings.ToUpper(symbol), nil, nil
	}

	candidate := strings.TrimSpace(input.Query)
	if candidate == "" {
		candidate = strings.TrimSpace(input.Claim)
	}
	if candidate == "" {
		return "", nil, nil
	}

	candidate = strings.Fields(candidate)[0]
	if tickerPattern.MatchString(candidate) && strings.ToUpper(candidate) == candidate {
		return candidate, nil, nil
	}

	base, err := url.Parse(provider.baseURL)
	if err != nil {
		return "", nil, fmt.Errorf("parse alphavantage base url: %w", err)
	}
	query := base.Query()
	query.Set("function", "SYMBOL_SEARCH")
	query.Set("keywords", strings.TrimSpace(input.Query))
	query.Set("apikey", provider.apiKey)
	base.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return "", nil, fmt.Errorf("create alphavantage symbol search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := provider.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("request alphavantage symbol search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return "", nil, fmt.Errorf("alphavantage symbol search returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload alphaVantageSymbolSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", nil, fmt.Errorf("decode alphavantage symbol search: %w", err)
	}
	if msg := alphaVantageAPIMessage(payload.Note, payload.Information); msg != "" {
		return "", nil, alphaVantageMessageError(msg)
	}
	if len(payload.BestMatches) == 0 {
		return "", nil, nil
	}

	match := payload.BestMatches[0]
	symbol := strings.TrimSpace(match["1. symbol"])
	if symbol == "" {
		return "", nil, nil
	}
	metadata := map[string]any{}
	for key, value := range match {
		metadata[normalizeAlphaKey(key)] = strings.TrimSpace(value)
	}
	return symbol, metadata, nil
}

func (provider *AlphaVantageProvider) globalQuote(ctx context.Context, symbol string) (map[string]string, error) {
	base, err := url.Parse(provider.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse alphavantage base url: %w", err)
	}
	query := base.Query()
	query.Set("function", "GLOBAL_QUOTE")
	query.Set("symbol", symbol)
	query.Set("apikey", provider.apiKey)
	if provider.entitlement != "" {
		query.Set("entitlement", provider.entitlement)
	}
	base.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create alphavantage quote request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := provider.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request alphavantage quote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, fmt.Errorf("alphavantage quote returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode alphavantage quote: %w", err)
	}
	var note, info string
	if v, ok := raw["Note"]; ok {
		_ = json.Unmarshal(v, &note)
	}
	if v, ok := raw["Information"]; ok {
		_ = json.Unmarshal(v, &info)
	}
	if msg := alphaVantageAPIMessage(note, info); msg != "" {
		return nil, alphaVantageMessageError(msg)
	}
	var quotes map[string]string
	if v, ok := raw["Global Quote"]; ok {
		if err := json.Unmarshal(v, &quotes); err != nil {
			return nil, fmt.Errorf("decode alphavantage quote fields: %w", err)
		}
	}
	return quotes, nil
}

func buildAlphaVantageSnippet(symbol string, values map[string]string) string {
	parts := []string{fmt.Sprintf("symbol=%s", symbol)}
	for _, key := range []string{"05. price", "09. change", "10. change percent", "07. latest trading day"} {
		if value := strings.TrimSpace(values[key]); value != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", normalizeAlphaKey(key), value))
		}
	}
	return strings.Join(parts, " | ")
}

func normalizeAlphaKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.TrimLeft(key, "0123456789. ")
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "%", "percent")
	return key
}

func stringToFloat(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if parsed, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64); err == nil {
		return parsed
	}
	return value
}

// alphaVantageAPIMessage returns the first non-empty "Note" or "Information"
// message the API embeds in a 200 response when the caller is rate-limited or
// has exhausted its daily quota. Returns "" when neither field is present.
func alphaVantageAPIMessage(note, information string) string {
	if v := strings.TrimSpace(note); v != "" {
		return v
	}
	return strings.TrimSpace(information)
}

// alphaVantageMessageError converts a rate-limit / quota message from
// Alpha Vantage into an error whose text classifies correctly in classifyError:
//   - contains "rate limit" → kind="rate_limit"
//   - contains "quota"      → kind="quota_exceeded"
func alphaVantageMessageError(msg string) error {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "minute") || strings.Contains(lower, "per minute") || strings.Contains(lower, "frequency") {
		return fmt.Errorf("alphavantage rate limit: %s", msg)
	}
	return fmt.Errorf("alphavantage quota exceeded: %s", msg)
}
