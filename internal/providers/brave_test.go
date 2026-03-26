package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func TestBraveProviderFetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Subscription-Token"); got != "test-key" {
			t.Fatalf("unexpected subscription token: %q", got)
		}
		if got := r.URL.Query().Get("q"); got != "federal reserve march meeting" {
			t.Fatalf("unexpected query: %q", got)
		}
		if got := r.URL.Query().Get("count"); got != "3" {
			t.Fatalf("unexpected count: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"web": {
				"results": [
					{
						"title": "Federal Reserve statement",
						"url": "https://www.federalreserve.gov/newsevents/pressreleases/monetary20260318a.htm",
						"description": "Official statement from the Federal Reserve.",
						"page_age": "2026-03-18T18:00:00Z",
						"language": "en"
					},
					{
						"title": "Reuters coverage",
						"url": "https://www.reuters.com/world/us/fed-holds-rates-2026-03-18/",
						"description": "Reuters summarizes the decision.",
						"age": "1 day ago",
						"extra_snippets": ["Markets reacted immediately after the release."]
					}
				]
			}
		}`))
	}))
	defer server.Close()

	provider, err := NewBraveProvider(
		"test-key",
		WithBraveEndpoint(server.URL),
		WithBraveHTTPClient(server.Client()),
		WithBraveCount(3),
	)
	if err != nil {
		t.Fatalf("NewBraveProvider returned error: %v", err)
	}

	documents, err := provider.Fetch(context.Background(), schema.ResolveInput{
		Query: "federal reserve march meeting",
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(documents))
	}
	if documents[0].Provider != "brave_search" {
		t.Fatalf("unexpected provider: %q", documents[0].Provider)
	}
	if documents[0].SourceType != "official" {
		t.Fatalf("expected official source type for first result, got %q", documents[0].SourceType)
	}
	if documents[0].PublishedAt == nil {
		t.Fatal("expected published_at for first result")
	}
	if documents[1].SourceType != "news" {
		t.Fatalf("expected news source type for second result, got %q", documents[1].SourceType)
	}
	if documents[1].Snippet == "" {
		t.Fatal("expected combined snippet for second result")
	}
}

func TestBraveProviderRejectsEmptyAPIKey(t *testing.T) {
	t.Parallel()

	if _, err := NewBraveProvider(""); err == nil {
		t.Fatal("expected error for empty api key")
	}
}
