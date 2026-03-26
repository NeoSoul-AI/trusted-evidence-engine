package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func TestCoinGeckoPriceProviderFetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/api/v3/simple/price" {
			t.Fatalf("unexpected path: %q", got)
		}
		if got := r.URL.Query().Get("symbols"); got != "eth" {
			t.Fatalf("unexpected symbol lookup: %q", got)
		}
		if got := r.URL.Query().Get("vs_currencies"); got != "usd" {
			t.Fatalf("unexpected vs currency: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"eth": {
				"usd": 3921.12,
				"usd_market_cap": 471000000000,
				"usd_24h_vol": 19000000000,
				"usd_24h_change": 4.2,
				"last_updated_at": 1774497600
			}
		}`))
	}))
	defer server.Close()

	provider, err := NewCoinGeckoPriceProvider(
		"demo-key",
		WithCoinGeckoPriceBaseURL(server.URL+"/api/v3"),
		WithCoinGeckoPriceHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("NewCoinGeckoPriceProvider returned error: %v", err)
	}

	documents, err := provider.Fetch(context.Background(), schema.ResolveInput{
		Query: "eth price",
		ContextMetadata: map[string]any{
			"crypto_symbol": "eth",
		},
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}
	if documents[0].SourceType != "crypto_quote" {
		t.Fatalf("unexpected source type: %q", documents[0].SourceType)
	}
	if got := documents[0].Metadata["price"]; got != 3921.12 {
		t.Fatalf("unexpected price metadata: %#v", got)
	}
}
