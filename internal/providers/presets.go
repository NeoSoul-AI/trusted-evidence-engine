package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultSourcePresetPath = "configs/source-presets.json"

type SourcePreset struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	FeedURLs      []string `json:"feed_urls"`
	ProviderHints []string `json:"provider_hints,omitempty"`
	Notes         []string `json:"notes,omitempty"`
}

type SourcePresetCatalog struct {
	Presets map[string]SourcePreset `json:"presets"`
}

func loadSourcePresets(path string, names []string) ([]SourcePreset, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = defaultSourcePresetPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var catalog SourcePresetCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("decode source presets: %w", err)
	}

	presets := make([]SourcePreset, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		preset, ok := catalog.Presets[name]
		if !ok {
			return nil, fmt.Errorf("unknown source preset: %s", name)
		}
		if strings.TrimSpace(preset.Name) == "" {
			preset.Name = name
		}
		presets = append(presets, preset)
	}
	return presets, nil
}

func sourcePresetPathFromEnv() string {
	configured := strings.TrimSpace(envOrDefault("TEE_SOURCE_PRESETS_PATH", ""))
	if configured != "" {
		return configured
	}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		return filepath.Join(cwd, defaultSourcePresetPath)
	}
	return defaultSourcePresetPath
}

func sourcePresetNamesFromEnv() []string {
	return envList("TEE_SOURCE_PRESETS")
}

func feedURLsFromPresets() ([]string, error) {
	presets, err := activeSourcePresetsFromEnv()
	if err != nil {
		return nil, err
	}
	if len(presets) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	urls := make([]string, 0)
	for _, preset := range presets {
		for _, rawURL := range preset.FeedURLs {
			rawURL = strings.TrimSpace(rawURL)
			if rawURL == "" {
				continue
			}
			if _, ok := seen[rawURL]; ok {
				continue
			}
			seen[rawURL] = struct{}{}
			urls = append(urls, rawURL)
		}
	}
	return urls, nil
}

func activeSourcePresetsFromEnv() ([]SourcePreset, error) {
	names := sourcePresetNamesFromEnv()
	if len(names) == 0 {
		return nil, nil
	}
	return loadSourcePresets(sourcePresetPathFromEnv(), names)
}

func providerHintsFromPresets() (map[string]struct{}, error) {
	presets, err := activeSourcePresetsFromEnv()
	if err != nil {
		return nil, err
	}
	if len(presets) == 0 {
		return nil, nil
	}

	hints := map[string]struct{}{}
	for _, preset := range presets {
		for _, hint := range preset.ProviderHints {
			hint = strings.TrimSpace(hint)
			if hint == "" {
				continue
			}
			hints[hint] = struct{}{}
		}
	}
	if len(hints) == 0 {
		return nil, nil
	}
	return hints, nil
}

func providerEnabledByHints(providerName string, hints map[string]struct{}) bool {
	if len(hints) == 0 {
		return true
	}
	_, ok := hints[strings.TrimSpace(providerName)]
	return ok
}
