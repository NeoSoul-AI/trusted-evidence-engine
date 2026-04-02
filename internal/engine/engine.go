package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/evoevo/trusted-evidence-engine/internal/dedup"
	"github.com/evoevo/trusted-evidence-engine/internal/metrics"
	"github.com/evoevo/trusted-evidence-engine/internal/providers"
	"github.com/evoevo/trusted-evidence-engine/internal/ranking"
	"github.com/evoevo/trusted-evidence-engine/internal/schema"
)

var ErrEmptyInput = errors.New("query, claim, or context is required")

const (
	DefaultSchemaVersion = "tee.v0alpha1"
	DefaultPolicyName    = "default_trust_policy"
	DefaultPolicyVersion = "2026-03-26"
)

type Engine struct {
	providers []providers.Provider
	now       func() time.Time
}

func New(providerSet []providers.Provider) *Engine {
	if len(providerSet) == 0 {
		providerSet = providers.DefaultProviders()
	}
	return &Engine{
		providers: providerSet,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (e *Engine) Resolve(ctx context.Context, input schema.ResolveInput) (schema.EvidencePack, error) {
	if strings.TrimSpace(input.Query) == "" &&
		strings.TrimSpace(input.Claim) == "" &&
		len(input.SourceURLs) == 0 &&
		len(input.ContextMetadata) == 0 &&
		input.PredictionID == nil &&
		input.TopicID == nil {
		return schema.EvidencePack{}, ErrEmptyInput
	}

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		documents []schema.SourceDocument
	)

	for _, provider := range e.providers {
		provider := provider
		wg.Add(1)
		go func() {
			defer wg.Done()
			startedAt := time.Now()
			items, err := provider.Fetch(ctx, input)
			metrics.ObserveProvider(provider.Name(), startedAt, err, len(items))
			if err != nil || len(items) == 0 {
				return
			}
			mu.Lock()
			documents = append(documents, items...)
			mu.Unlock()
		}()
	}
	wg.Wait()

	documents = dedup.Merge(documents)
	items := ranking.ScoreDocuments(e.now(), documents)
	return schema.EvidencePack{
		SchemaVersion:   DefaultSchemaVersion,
		Query:           strings.TrimSpace(input.Query),
		Claim:           strings.TrimSpace(input.Claim),
		PredictionID:    input.PredictionID,
		TopicID:         input.TopicID,
		SourceURLs:      input.SourceURLs,
		ContextMetadata: input.ContextMetadata,
		Policy:          DefaultPolicyName,
		PolicyVersion:   DefaultPolicyVersion,
		FetchedAt:       e.now(),
		Items:           items,
	}, nil
}
