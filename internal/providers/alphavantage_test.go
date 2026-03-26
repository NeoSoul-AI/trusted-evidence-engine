package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func TestAlphaVantageProviderFetchWithTicker(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("function"); got != "GLOBAL_QUOTE" {
			t.Fatalf("unexpected function: %q", got)
		}
		if got := r.URL.Query().Get("symbol"); got != "IBM" {
			t.Fatalf("unexpected symbol: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"Global Quote": {
				"01. symbol": "IBM",
				"02. open": "181.20",
				"03. high": "184.00",
				"04. low": "180.10",
				"05. price": "183.42",
				"06. volume": "3456789",
				"07. latest trading day": "2026-03-26",
				"08. previous close": "180.50",
				"09. change": "2.92",
				"10. change percent": "1.617%"
			}
		}`))
	}))
	defer server.Close()

	provider, err := NewAlphaVantageProvider(
		"demo",
		WithAlphaVantageBaseURL(server.URL),
		WithAlphaVantageHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("NewAlphaVantageProvider returned error: %v", err)
	}

	documents, err := provider.Fetch(context.Background(), schema.ResolveInput{
		Query: "IBM price",
		ContextMetadata: map[string]any{
			"ticker": "IBM",
		},
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}
	if documents[0].SourceType != "equity_quote" {
		t.Fatalf("unexpected source type: %q", documents[0].SourceType)
	}
}

func TestAlphaVantageProviderFetchWithSearch(t *testing.T) {
	t.Parallel()

	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Query().Get("function"))
		switch r.URL.Query().Get("function") {
		case "SYMBOL_SEARCH":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"bestMatches": [
					{
						"1. symbol": "MSFT",
						"2. name": "Microsoft Corporation",
						"9. matchScore": "0.9523"
					}
				]
			}`))
		case "GLOBAL_QUOTE":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"Global Quote": {
					"01. symbol": "MSFT",
					"05. price": "512.30",
					"07. latest trading day": "2026-03-26",
					"09. change": "1.11",
					"10. change percent": "0.22%"
				}
			}`))
		default:
			t.Fatalf("unexpected function: %q", r.URL.Query().Get("function"))
		}
	}))
	defer server.Close()

	provider, err := NewAlphaVantageProvider(
		"demo",
		WithAlphaVantageBaseURL(server.URL),
		WithAlphaVantageHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("NewAlphaVantageProvider returned error: %v", err)
	}

	documents, err := provider.Fetch(context.Background(), schema.ResolveInput{
		Query: "Microsoft price",
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 API calls, got %d", len(calls))
	}
}
