package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func TestPolymarketProviderFetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/markets/slug/fed-decision-in-october" {
			t.Fatalf("unexpected path: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "123",
			"question": "Will the Fed cut rates in October?",
			"slug": "fed-decision-in-october",
			"resolutionSource": "https://www.federalreserve.gov/",
			"description": "Polymarket market for a Fed decision.",
			"outcomes": "[\"Yes\",\"No\"]",
			"outcomePrices": "[\"0.73\",\"0.27\"]",
			"category": "Politics",
			"marketType": "binary",
			"closed": true,
			"active": false,
			"updatedAt": "2026-03-26T03:00:00Z",
			"closedTime": "2026-03-26T03:05:00Z",
			"resolvedBy": "UMA",
			"umaResolutionStatus": "resolved"
		}`))
	}))
	defer server.Close()

	provider := NewPolymarketProvider(
		WithPolymarketEndpoint(server.URL),
		WithPolymarketHTTPClient(server.Client()),
	)
	documents, err := provider.Fetch(context.Background(), schema.ResolveInput{
		SourceURLs: []string{"https://polymarket.com/event/fed-decision-in-october"},
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}
	if documents[0].Provider != "polymarket_gamma" {
		t.Fatalf("unexpected provider: %q", documents[0].Provider)
	}
	if documents[0].SourceType != "prediction_market" {
		t.Fatalf("unexpected source type: %q", documents[0].SourceType)
	}
	if documents[0].PublishedAt == nil {
		t.Fatal("expected published_at")
	}
	if documents[0].Metadata["resolved_by"] != "UMA" {
		t.Fatalf("unexpected resolved_by metadata: %#v", documents[0].Metadata["resolved_by"])
	}
}

func TestPolymarketProviderNoSlugNoop(t *testing.T) {
	t.Parallel()

	provider := NewPolymarketProvider()
	documents, err := provider.Fetch(context.Background(), schema.ResolveInput{
		Query: "plain text query",
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(documents) != 0 {
		t.Fatalf("expected no documents, got %d", len(documents))
	}
}
