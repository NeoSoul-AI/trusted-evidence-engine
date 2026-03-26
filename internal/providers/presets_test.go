package providers

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadSourcePresets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "source-presets.json")
	if err := os.WriteFile(path, []byte(`{
		"presets": {
			"sports_basic": {
				"name": "sports_basic",
				"description": "Sports feeds",
				"feed_urls": ["https://example.com/sports.xml"]
			},
			"crypto_basic": {
				"name": "crypto_basic",
				"description": "Crypto feeds",
				"feed_urls": ["https://example.com/crypto.xml"]
			}
		}
	}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	presets, err := loadSourcePresets(path, []string{"sports_basic", "crypto_basic"})
	if err != nil {
		t.Fatalf("loadSourcePresets returned error: %v", err)
	}
	if len(presets) != 2 {
		t.Fatalf("expected 2 presets, got %d", len(presets))
	}
	if presets[0].Name != "sports_basic" {
		t.Fatalf("unexpected first preset: %q", presets[0].Name)
	}
}

func TestFeedURLsFromPresets(t *testing.T) {
	t.Setenv("TEE_SOURCE_PRESETS", "sports_basic,crypto_basic")

	dir := t.TempDir()
	path := filepath.Join(dir, "source-presets.json")
	t.Setenv("TEE_SOURCE_PRESETS_PATH", path)
	if err := os.WriteFile(path, []byte(`{
		"presets": {
			"sports_basic": {
				"feed_urls": ["https://example.com/sports.xml", "https://example.com/shared.xml"]
			},
			"crypto_basic": {
				"feed_urls": ["https://example.com/crypto.xml", "https://example.com/shared.xml"]
			}
		}
	}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	got, err := feedURLsFromPresets()
	if err != nil {
		t.Fatalf("feedURLsFromPresets returned error: %v", err)
	}
	want := []string{
		"https://example.com/sports.xml",
		"https://example.com/shared.xml",
		"https://example.com/crypto.xml",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected urls: got=%v want=%v", got, want)
	}
}

func TestProviderHintsFromPresets(t *testing.T) {
	t.Setenv("TEE_SOURCE_PRESETS", "sports_basic,crypto_basic")

	dir := t.TempDir()
	path := filepath.Join(dir, "source-presets.json")
	t.Setenv("TEE_SOURCE_PRESETS_PATH", path)
	if err := os.WriteFile(path, []byte(`{
		"presets": {
			"sports_basic": {
				"provider_hints": ["feed_aggregator", "brave_search"]
			},
			"crypto_basic": {
				"provider_hints": ["coingecko_search", "feed_aggregator"]
			}
		}
	}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	got, err := providerHintsFromPresets()
	if err != nil {
		t.Fatalf("providerHintsFromPresets returned error: %v", err)
	}
	want := map[string]struct{}{
		"feed_aggregator":  {},
		"brave_search":     {},
		"coingecko_search": {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected hints: got=%v want=%v", got, want)
	}
}

func TestProviderEnabledByHints(t *testing.T) {
	hints := map[string]struct{}{
		"feed_aggregator": {},
	}
	if !providerEnabledByHints("feed_aggregator", hints) {
		t.Fatal("expected feed_aggregator to be enabled")
	}
	if providerEnabledByHints("brave_search", hints) {
		t.Fatal("expected brave_search to be disabled")
	}
	if !providerEnabledByHints("anything", nil) {
		t.Fatal("expected provider to be enabled when no hints exist")
	}
}
