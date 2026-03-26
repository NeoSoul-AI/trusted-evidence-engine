package engine

import (
	"context"
	"testing"

	"github.com/evoevo/trusted-evidence-engine/internal/providers"
	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func TestResolveReturnsEvidencePack(t *testing.T) {
	e := New([]providers.Provider{providers.DemoProvider{}})
	pack, err := e.Resolve(context.Background(), schema.ResolveInput{Query: "demo event"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(pack.Items) == 0 {
		t.Fatalf("Resolve() returned no items")
	}
	if pack.Items[0].Reliability <= 0 {
		t.Fatalf("expected positive reliability score, got %.2f", pack.Items[0].Reliability)
	}
}

func TestResolveRejectsEmptyInput(t *testing.T) {
	e := New([]providers.Provider{providers.DemoProvider{}})
	if _, err := e.Resolve(context.Background(), schema.ResolveInput{}); err == nil {
		t.Fatalf("Resolve() error = nil, want non-nil")
	}
}

func TestResolveAllowsSourceURLsOnly(t *testing.T) {
	e := New([]providers.Provider{providers.DemoProvider{}})
	_, err := e.Resolve(context.Background(), schema.ResolveInput{
		SourceURLs: []string{"https://example.com/result"},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
}
