package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func TestFeedProviderFetchRSS(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
			<rss version="2.0">
			  <channel>
			    <title>Sports Feed</title>
			    <item>
			      <title>Lakers win season opener</title>
			      <link>https://www.espn.com/nba/story/_/id/123/lakers-win-season-opener</link>
			      <description>Los Angeles wins behind a big fourth quarter.</description>
			      <pubDate>Wed, 26 Mar 2026 10:00:00 +0000</pubDate>
			    </item>
			    <item>
			      <title>Baseball roundup</title>
			      <link>https://example.com/baseball</link>
			      <description>Other league recap.</description>
			      <pubDate>Wed, 26 Mar 2026 10:00:00 +0000</pubDate>
			    </item>
			  </channel>
			</rss>`))
	}))
	defer server.Close()

	provider := NewFeedProvider([]string{server.URL}, WithFeedHTTPClient(server.Client()))
	documents, err := provider.Fetch(context.Background(), schema.ResolveInput{Query: "Lakers opener"})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}
	if documents[0].SourceType != "sports_news" {
		t.Fatalf("unexpected source type: %q", documents[0].SourceType)
	}
}

func TestFeedProviderFetchAtom(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
			<feed xmlns="http://www.w3.org/2005/Atom">
			  <title>Crypto Feed</title>
			  <entry>
			    <title>Ethereum ETF demand grows</title>
			    <summary>Institutional demand continues to rise.</summary>
			    <updated>2026-03-26T10:00:00Z</updated>
			    <link href="https://www.coindesk.com/markets/2026/03/26/ethereum-etf-demand-grows/"></link>
			  </entry>
			</feed>`))
	}))
	defer server.Close()

	provider := NewFeedProvider([]string{server.URL}, WithFeedHTTPClient(server.Client()))
	documents, err := provider.Fetch(context.Background(), schema.ResolveInput{Query: "Ethereum demand"})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}
	if documents[0].SourceType != "crypto_news" {
		t.Fatalf("unexpected source type: %q", documents[0].SourceType)
	}
}
