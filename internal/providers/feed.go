package providers

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

type FeedProvider struct {
	feedURLs []string
	maxItems int
	client   *http.Client
	now      func() time.Time
}

type FeedOption func(*FeedProvider)

type rssFeed struct {
	Channel struct {
		Title string    `xml:"title"`
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type atomFeed struct {
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	Summary string     `xml:"summary"`
	Updated string     `xml:"updated"`
	Link    []atomLink `xml:"link"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

func NewFeedProvider(feedURLs []string, opts ...FeedOption) *FeedProvider {
	provider := &FeedProvider{
		feedURLs: feedURLs,
		maxItems: 10,
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

func NewFeedProviderFromEnv() (*FeedProvider, bool) {
	feedURLs := envList("TEE_FEED_URLS")
	presetFeedURLs, err := feedURLsFromPresets()
	if err == nil && len(presetFeedURLs) > 0 {
		feedURLs = append(feedURLs, presetFeedURLs...)
	}
	feedURLs = dedupeStrings(feedURLs)
	if len(feedURLs) == 0 {
		return nil, false
	}
	return NewFeedProvider(
		feedURLs,
		WithFeedMaxItems(envIntOrDefault("TEE_FEED_MAX_ITEMS", 10)),
	), true
}

func WithFeedHTTPClient(client *http.Client) FeedOption {
	return func(provider *FeedProvider) {
		if client != nil {
			provider.client = client
		}
	}
}

func WithFeedMaxItems(maxItems int) FeedOption {
	return func(provider *FeedProvider) {
		if maxItems > 0 {
			provider.maxItems = maxItems
		}
	}
}

func (provider *FeedProvider) Name() string {
	return "feed_aggregator"
}

func (provider *FeedProvider) Fetch(ctx context.Context, input schema.ResolveInput) ([]schema.SourceDocument, error) {
	terms := matchTerms(input)
	if len(terms) == 0 {
		return nil, nil
	}

	documents := make([]schema.SourceDocument, 0, provider.maxItems)
	for _, rawURL := range provider.feedURLs {
		items, err := provider.fetchFeed(ctx, rawURL, terms)
		if err != nil {
			return nil, err
		}
		documents = append(documents, items...)
		if len(documents) >= provider.maxItems {
			return documents[:provider.maxItems], nil
		}
	}
	return documents, nil
}

func (provider *FeedProvider) fetchFeed(ctx context.Context, rawURL string, terms []string) ([]schema.SourceDocument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create feed request: %w", err)
	}
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")

	resp, err := provider.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, fmt.Errorf("feed returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return nil, fmt.Errorf("read feed body: %w", err)
	}
	if items := parseRSSFeed(provider.now(), rawURL, body, terms); len(items) > 0 {
		return items, nil
	}
	return parseAtomFeed(provider.now(), rawURL, body, terms), nil
}

func parseRSSFeed(fetchedAt time.Time, rawURL string, body []byte, terms []string) []schema.SourceDocument {
	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil
	}
	items := make([]schema.SourceDocument, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		link := strings.TrimSpace(item.Link)
		if link == "" {
			continue
		}
		text := strings.ToLower(item.Title + " " + item.Description)
		if !matchesAnyTerm(text, terms) {
			continue
		}
		items = append(items, schema.SourceDocument{
			Title:       firstNonEmpty(strings.TrimSpace(item.Title), link),
			URL:         link,
			Domain:      hostOf(link),
			SourceType:  inferFeedSourceType(hostOf(link)),
			Snippet:     strings.TrimSpace(item.Description),
			PublishedAt: parseFeedTime(item.PubDate),
			Provider:    "feed_aggregator",
			FetchedAt:   fetchedAt,
			Metadata: map[string]any{
				"feed_url":    rawURL,
				"feed_format": "rss",
				"feed_title":  strings.TrimSpace(feed.Channel.Title),
			},
		})
	}
	return items
}

func parseAtomFeed(fetchedAt time.Time, rawURL string, body []byte, terms []string) []schema.SourceDocument {
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil
	}
	items := make([]schema.SourceDocument, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		link := firstAtomLink(entry.Link)
		if link == "" {
			continue
		}
		text := strings.ToLower(entry.Title + " " + entry.Summary)
		if !matchesAnyTerm(text, terms) {
			continue
		}
		items = append(items, schema.SourceDocument{
			Title:       firstNonEmpty(strings.TrimSpace(entry.Title), link),
			URL:         link,
			Domain:      hostOf(link),
			SourceType:  inferFeedSourceType(hostOf(link)),
			Snippet:     strings.TrimSpace(entry.Summary),
			PublishedAt: parseFeedTime(entry.Updated),
			Provider:    "feed_aggregator",
			FetchedAt:   fetchedAt,
			Metadata: map[string]any{
				"feed_url":    rawURL,
				"feed_format": "atom",
				"feed_title":  strings.TrimSpace(feed.Title),
			},
		})
	}
	return items
}

func firstAtomLink(links []atomLink) string {
	for _, link := range links {
		if href := strings.TrimSpace(link.Href); href != "" && (link.Rel == "" || link.Rel == "alternate") {
			return href
		}
	}
	return ""
}

func parseFeedTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
}

func matchTerms(input schema.ResolveInput) []string {
	text := strings.ToLower(strings.TrimSpace(input.Query + " " + input.Claim))
	if text == "" {
		return nil
	}
	replacer := strings.NewReplacer(",", " ", ".", " ", "?", " ", "!", " ", ":", " ", ";", " ", "/", " ", "-", " ", "_", " ", "(", " ", ")", " ", "\"", " ", "'", " ")
	parts := strings.Fields(replacer.Replace(text))
	terms := make([]string, 0, len(parts))
	stopwords := map[string]struct{}{
		"will": {}, "this": {}, "that": {}, "with": {}, "from": {}, "have": {}, "what": {}, "when": {}, "where": {},
		"which": {}, "already": {}, "happen": {}, "outcome": {}, "settled": {}, "does": {}, "into": {}, "after": {},
	}
	seen := map[string]struct{}{}
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		if _, ok := stopwords[part]; ok {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		terms = append(terms, part)
	}
	return terms
}

func matchesAnyTerm(text string, terms []string) bool {
	if len(terms) == 0 {
		return false
	}
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func inferFeedSourceType(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	switch {
	case strings.Contains(domain, "espn"),
		strings.Contains(domain, "nba"),
		strings.Contains(domain, "nfl"),
		strings.Contains(domain, "mlb"),
		strings.Contains(domain, "nhl"),
		strings.Contains(domain, "fifa"),
		strings.Contains(domain, "uefa"):
		return "sports_news"
	case strings.Contains(domain, "coindesk"),
		strings.Contains(domain, "cointelegraph"),
		strings.Contains(domain, "binance"),
		strings.Contains(domain, "theblock"):
		return "crypto_news"
	default:
		return "news"
	}
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	output := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	return output
}
