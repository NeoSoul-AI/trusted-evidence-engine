package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func TestCoinGeckoProviderFetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/api/v3/search" {
			t.Fatalf("unexpected path: %q", got)
		}
		if got := r.URL.Query().Get("query"); got != "ethereum" {
			t.Fatalf("unexpected query: %q", got)
		}
		if got := r.Header.Get("x-cg-demo-api-key"); got != "demo-key" {
			t.Fatalf("unexpected demo api key: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"coins": [
				{
					"id": "ethereum",
					"name": "Ethereum",
					"api_symbol": "ethereum",
					"symbol": "eth",
					"market_cap_rank": 2,
					"thumb": "https://assets.coingecko.com/coins/images/279/thumb/ethereum.png",
					"large": "https://assets.coingecko.com/coins/images/279/large/ethereum.png"
				}
			]
		}`))
	}))
	defer server.Close()

	provider, err := NewCoinGeckoProvider(
		"demo-key",
		WithCoinGeckoBaseURL(server.URL+"/api/v3"),
		WithCoinGeckoHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("NewCoinGeckoProvider returned error: %v", err)
	}

	documents, err := provider.Fetch(context.Background(), schema.ResolveInput{Query: "ethereum"})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}
	if documents[0].Provider != "coingecko_search" {
		t.Fatalf("unexpected provider: %q", documents[0].Provider)
	}
	if documents[0].SourceType != "crypto_market_data" {
		t.Fatalf("unexpected source type: %q", documents[0].SourceType)
	}
	if got := documents[0].Metadata["coin_id"]; got != "ethereum" {
		t.Fatalf("unexpected coin_id metadata: %#v", got)
	}
}

func TestCoinGeckoProviderRejectsEmptyKey(t *testing.T) {
	t.Parallel()

	if _, err := NewCoinGeckoProvider(""); err == nil {
		t.Fatal("expected error for empty api key")
	}
}
