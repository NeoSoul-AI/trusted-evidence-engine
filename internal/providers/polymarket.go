package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

const polymarketGammaEndpoint = "https://gamma-api.polymarket.com"

type PolymarketProvider struct {
	endpoint string
	client   *http.Client
	now      func() time.Time
}

type PolymarketOption func(*PolymarketProvider)

type polymarketMarket struct {
	ID                  string  `json:"id"`
	Question            string  `json:"question"`
	Slug                string  `json:"slug"`
	ResolutionSource    string  `json:"resolutionSource"`
	Description         string  `json:"description"`
	Outcomes            string  `json:"outcomes"`
	OutcomePrices       string  `json:"outcomePrices"`
	Category            string  `json:"category"`
	MarketType          string  `json:"marketType"`
	Closed              *bool   `json:"closed"`
	Active              *bool   `json:"active"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
	ClosedTime          string  `json:"closedTime"`
	ResolvedBy          string  `json:"resolvedBy"`
	UmaResolutionStatus string  `json:"umaResolutionStatus"`
	EndDateIso          string  `json:"endDateIso"`
	VolumeNum           float64 `json:"volumeNum"`
	LiquidityNum        float64 `json:"liquidityNum"`
}

func NewPolymarketProvider(opts ...PolymarketOption) *PolymarketProvider {
	provider := &PolymarketProvider{
		endpoint: envOrDefault("TEE_POLYMARKET_GAMMA_ENDPOINT", polymarketGammaEndpoint),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		now: func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(provider)
	}
	return provider
}

func WithPolymarketEndpoint(endpoint string) PolymarketOption {
	return func(provider *PolymarketProvider) {
		if strings.TrimSpace(endpoint) != "" {
			provider.endpoint = strings.TrimSpace(endpoint)
		}
	}
}

func WithPolymarketHTTPClient(client *http.Client) PolymarketOption {
	return func(provider *PolymarketProvider) {
		if client != nil {
			provider.client = client
		}
	}
}

func (provider *PolymarketProvider) Name() string {
	return "polymarket_gamma"
}

func (provider *PolymarketProvider) Fetch(ctx context.Context, input schema.ResolveInput) ([]schema.SourceDocument, error) {
	slugs := collectPolymarketSlugs(input)
	if len(slugs) == 0 {
		return nil, nil
	}

	fetchedAt := provider.now()
	documents := make([]schema.SourceDocument, 0, len(slugs))
	for _, slug := range slugs {
		market, err := provider.fetchMarketBySlug(ctx, slug)
		if err != nil {
			return nil, err
		}
		if market == nil {
			continue
		}
		documents = append(documents, schema.SourceDocument{
			Title:       firstNonEmpty(strings.TrimSpace(market.Question), slug),
			URL:         "https://polymarket.com/event/" + slug,
			Domain:      "polymarket.com",
			SourceType:  "prediction_market",
			Snippet:     buildPolymarketSnippet(*market),
			PublishedAt: marketPublishedAt(*market),
			Provider:    provider.Name(),
			FetchedAt:   fetchedAt,
			Metadata:    buildPolymarketMetadata(*market),
		})
	}
	return documents, nil
}

func (provider *PolymarketProvider) fetchMarketBySlug(ctx context.Context, slug string) (*polymarketMarket, error) {
	base, err := url.Parse(provider.endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse polymarket endpoint: %w", err)
	}
	base.Path = path.Join(base.Path, "/markets/slug", slug)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create polymarket request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := provider.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request polymarket market: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, fmt.Errorf("polymarket market returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var market polymarketMarket
	if err := json.NewDecoder(resp.Body).Decode(&market); err != nil {
		return nil, fmt.Errorf("decode polymarket market: %w", err)
	}
	return &market, nil
}

func collectPolymarketSlugs(input schema.ResolveInput) []string {
	values := make([]string, 0, len(input.SourceURLs)+2)
	values = append(values, input.SourceURLs...)
	if strings.TrimSpace(input.Query) != "" {
		values = append(values, input.Query)
	}
	if strings.TrimSpace(input.Claim) != "" {
		values = append(values, input.Claim)
	}

	seen := map[string]struct{}{}
	slugs := make([]string, 0, len(values))
	for _, raw := range values {
		slug := extractPolymarketSlug(raw)
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	return slugs
}

func extractPolymarketSlug(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err == nil && parsed.Host != "" {
		host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
		segments := splitPathSegments(parsed.Path)
		if strings.Contains(host, "polymarket.com") {
			for i := 0; i < len(segments); i++ {
				if segments[i] == "event" && i+1 < len(segments) {
					return segments[i+1]
				}
			}
		}
		if strings.Contains(host, "gamma-api.polymarket.com") {
			for i := 0; i < len(segments); i++ {
				if segments[i] == "slug" && i+1 < len(segments) {
					return segments[i+1]
				}
			}
		}
	}
	return ""
}

func splitPathSegments(rawPath string) []string {
	trimmed := strings.Trim(strings.TrimSpace(rawPath), "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			segments = append(segments, part)
		}
	}
	return segments
}

func buildPolymarketSnippet(market polymarketMarket) string {
	parts := []string{}
	if question := strings.TrimSpace(market.Question); question != "" {
		parts = append(parts, question)
	}
	if category := strings.TrimSpace(market.Category); category != "" {
		parts = append(parts, "category="+category)
	}
	if market.Closed != nil {
		parts = append(parts, fmt.Sprintf("closed=%t", *market.Closed))
	}
	if resolvedBy := strings.TrimSpace(market.ResolvedBy); resolvedBy != "" {
		parts = append(parts, "resolved_by="+resolvedBy)
	}
	if resolutionSource := strings.TrimSpace(market.ResolutionSource); resolutionSource != "" {
		parts = append(parts, "resolution_source="+resolutionSource)
	}
	if prices := summarizeJSONList(market.OutcomePrices); prices != "" {
		parts = append(parts, "outcome_prices="+prices)
	}
	return strings.Join(parts, " | ")
}

func buildPolymarketMetadata(market polymarketMarket) map[string]any {
	metadata := map[string]any{
		"slug":        strings.TrimSpace(market.Slug),
		"market_id":   strings.TrimSpace(market.ID),
		"category":    strings.TrimSpace(market.Category),
		"market_type": strings.TrimSpace(market.MarketType),
		"provider":    "gamma",
	}
	if market.Closed != nil {
		metadata["closed"] = *market.Closed
	}
	if market.Active != nil {
		metadata["active"] = *market.Active
	}
	if value := strings.TrimSpace(market.ResolvedBy); value != "" {
		metadata["resolved_by"] = value
	}
	if value := strings.TrimSpace(market.ResolutionSource); value != "" {
		metadata["resolution_source"] = value
	}
	if value := strings.TrimSpace(market.UmaResolutionStatus); value != "" {
		metadata["uma_resolution_status"] = value
	}
	if outcomes := parseJSONList(market.Outcomes); len(outcomes) > 0 {
		metadata["outcomes"] = outcomes
	}
	if prices := parseJSONList(market.OutcomePrices); len(prices) > 0 {
		metadata["outcome_prices"] = prices
	}
	if market.VolumeNum > 0 {
		metadata["volume_num"] = market.VolumeNum
	}
	if market.LiquidityNum > 0 {
		metadata["liquidity_num"] = market.LiquidityNum
	}
	return metadata
}

func marketPublishedAt(market polymarketMarket) *time.Time {
	for _, candidate := range []string{
		strings.TrimSpace(market.ClosedTime),
		strings.TrimSpace(market.UpdatedAt),
		strings.TrimSpace(market.EndDateIso),
		strings.TrimSpace(market.CreatedAt),
	} {
		if parsed := parseTimestamp(candidate); parsed != nil {
			return parsed
		}
	}
	return nil
}

func parseTimestamp(raw string) *time.Time {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
}

func parseJSONList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		return values
	}
	return nil
}

func summarizeJSONList(raw string) string {
	values := parseJSONList(raw)
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, ",")
}
