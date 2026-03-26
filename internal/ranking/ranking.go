package ranking

import (
	"sort"
	"strings"
	"time"

	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

func ScoreDocuments(now time.Time, documents []schema.SourceDocument) []schema.EvidenceItem {
	if len(documents) == 0 {
		return nil
	}

	items := make([]schema.EvidenceItem, 0, len(documents))
	for _, document := range documents {
		signals := schema.RankingSignals{
			DomainTrust:      domainTrustScore(document),
			Freshness:        freshnessScore(now, document.PublishedAt),
			SourceTypeWeight: sourceTypeWeight(document.SourceType),
			ProviderWeight:   providerWeight(document.Provider),
			PolicyVersion:    "2026-03-26",
		}
		signals.Score = weightedScore(signals)
		items = append(items, schema.EvidenceItem{
			Title:          document.Title,
			URL:            document.URL,
			Domain:         document.Domain,
			SourceType:     document.SourceType,
			Snippet:        document.Snippet,
			PublishedAt:    document.PublishedAt,
			Provider:       document.Provider,
			FetchedAt:      document.FetchedAt,
			Reliability:    signals.Score,
			RankingSignals: signals,
			Metadata:       document.Metadata,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Reliability != items[j].Reliability {
			return items[i].Reliability > items[j].Reliability
		}
		return items[i].URL < items[j].URL
	})
	return items
}

func domainTrustScore(document schema.SourceDocument) float64 {
	sourceType := strings.TrimSpace(strings.ToLower(document.SourceType))
	domain := strings.TrimSpace(strings.ToLower(document.Domain))
	switch {
	case sourceType == "official":
		return 0.98
	case sourceType == "prediction_market":
		return 0.90
	case sourceType == "crypto_market_data":
		return 0.89
	case sourceType == "crypto_quote":
		return 0.91
	case sourceType == "equity_quote":
		return 0.90
	case sourceType == "sports_news":
		return 0.84
	case sourceType == "crypto_news":
		return 0.83
	case strings.HasSuffix(domain, ".gov"):
		return 0.97
	case strings.HasSuffix(domain, ".edu"):
		return 0.94
	case sourceType == "reference":
		return 0.90
	case sourceType == "news":
		return 0.82
	default:
		return 0.70
	}
}

func freshnessScore(now time.Time, publishedAt *time.Time) float64 {
	if publishedAt == nil {
		return 0.55
	}
	age := now.Sub(publishedAt.UTC())
	switch {
	case age <= 6*time.Hour:
		return 1.00
	case age <= 24*time.Hour:
		return 0.92
	case age <= 7*24*time.Hour:
		return 0.80
	case age <= 30*24*time.Hour:
		return 0.68
	default:
		return 0.50
	}
}

func sourceTypeWeight(sourceType string) float64 {
	switch strings.TrimSpace(strings.ToLower(sourceType)) {
	case "official":
		return 1.00
	case "prediction_market":
		return 0.90
	case "crypto_market_data":
		return 0.88
	case "crypto_quote":
		return 0.92
	case "equity_quote":
		return 0.90
	case "sports_news":
		return 0.84
	case "crypto_news":
		return 0.83
	case "reference":
		return 0.90
	case "news":
		return 0.82
	default:
		return 0.70
	}
}

func providerWeight(provider string) float64 {
	switch strings.TrimSpace(strings.ToLower(provider)) {
	case "demo":
		return 0.65
	case "polymarket_gamma":
		return 0.90
	case "coingecko_search":
		return 0.88
	case "coingecko_simple_price":
		return 0.90
	case "alphavantage_quote":
		return 0.89
	case "feed_aggregator":
		return 0.78
	default:
		return 0.80
	}
}

func weightedScore(signals schema.RankingSignals) float64 {
	return clamp(
		signals.DomainTrust*0.40+
			signals.Freshness*0.25+
			signals.SourceTypeWeight*0.20+
			signals.ProviderWeight*0.15,
		0,
		1,
	)
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
