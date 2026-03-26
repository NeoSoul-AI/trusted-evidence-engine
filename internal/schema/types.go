package schema

import "time"

type ResolveInput struct {
	Query           string         `json:"query,omitempty"`
	Claim           string         `json:"claim,omitempty"`
	PredictionID    *int64         `json:"prediction_id,omitempty"`
	TopicID         *int64         `json:"topic_id,omitempty"`
	SourceURLs      []string       `json:"source_urls,omitempty"`
	ContextMetadata map[string]any `json:"context_metadata,omitempty"`
}

type SourceDocument struct {
	Title       string         `json:"title"`
	URL         string         `json:"url"`
	Domain      string         `json:"domain,omitempty"`
	SourceType  string         `json:"source_type,omitempty"`
	Snippet     string         `json:"snippet,omitempty"`
	PublishedAt *time.Time     `json:"published_at,omitempty"`
	Provider    string         `json:"provider"`
	FetchedAt   time.Time      `json:"fetched_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type RankingSignals struct {
	DomainTrust      float64 `json:"domain_trust"`
	Freshness        float64 `json:"freshness"`
	SourceTypeWeight float64 `json:"source_type_weight"`
	ProviderWeight   float64 `json:"provider_weight"`
	PolicyVersion    string  `json:"policy_version,omitempty"`
	Score            float64 `json:"score"`
}

type EvidenceItem struct {
	Title          string         `json:"title"`
	URL            string         `json:"url"`
	Domain         string         `json:"domain,omitempty"`
	SourceType     string         `json:"source_type,omitempty"`
	Snippet        string         `json:"snippet,omitempty"`
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
	Provider       string         `json:"provider"`
	FetchedAt      time.Time      `json:"fetched_at"`
	Reliability    float64        `json:"reliability"`
	RankingSignals RankingSignals `json:"ranking_signals"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type EvidencePack struct {
	SchemaVersion    string         `json:"schema_version"`
	Query           string         `json:"query,omitempty"`
	Claim           string         `json:"claim,omitempty"`
	PredictionID    *int64         `json:"prediction_id,omitempty"`
	TopicID         *int64         `json:"topic_id,omitempty"`
	SourceURLs      []string       `json:"source_urls,omitempty"`
	ContextMetadata map[string]any `json:"context_metadata,omitempty"`
	Policy          string         `json:"policy,omitempty"`
	PolicyVersion   string         `json:"policy_version,omitempty"`
	FetchedAt       time.Time      `json:"fetched_at"`
	Items           []EvidenceItem `json:"items"`
}
