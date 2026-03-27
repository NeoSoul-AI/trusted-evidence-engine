# Attested Evidence Profile

This document describes a stricter profile that can sit on top of the base Trusted Evidence Engine `evidence pack`.

The goal is to support higher-assurance workflows such as committee review, audit trails, and replayable resolution support without forcing every generic evidence consumer to adopt committee-specific fields.

## Why This Exists

The base `evidence pack` is designed to be:

- generic
- reviewable
- easy to consume from CLI or HTTP
- useful for agents, research, and workflow automation

That is the right default for open-source reuse.

However, some workflows need more than a ranked list of source-aware evidence items.

Committee and oracle workflows often also need:

- stronger replay guarantees
- stable artifact hashing
- explicit source snapshots
- a link between evidence and a candidate resolution path

Those needs are better represented as a profile layered on top of the base pack, not as a requirement for every caller.

## Base Pack Versus Attested Profile

The distinction is:

- base `evidence pack`: generic evidence retrieval and ranking artifact
- `attested evidence profile`: stricter, audit-oriented envelope for high-assurance workflows

The base pack answers:

- what evidence was collected
- where it came from
- how it was ranked

The attested profile additionally answers:

- what exact content was inspected
- what snapshot or immutable reference was preserved
- what candidate outcome the evidence supported
- what bundle hash identifies the review artifact

## Scope

This profile is intended for workflows such as:

- committee resolution review
- oracle candidate-resolution preparation
- high-stakes workflow approval
- evidence replay and audit

It is not required for:

- generic agent grounding
- basic research workflows
- simple CLI usage

## Design Principle

The profile should preserve a clean boundary:

- Trusted Evidence Engine still resolves and ranks evidence
- downstream systems add stricter attestation and action-specific fields
- signing, settlement, and private-key handling still stay outside the engine

## Profile Shape

Recommended profile name:

- `committee_attested.v0alpha1`

Recommended envelope shape:

```json
{
  "profile": "committee_attested.v0alpha1",
  "base_pack": {
    "schema_version": "tee.v0alpha1",
    "policy": "default_trust_policy",
    "policy_version": "2026-03-26",
    "fetched_at": "2026-03-27T12:00:00Z",
    "items": []
  },
  "attestation": {
    "prediction_id": 101,
    "topic_id": 55,
    "member_token_id": 12,
    "resolution_kind": "resolved",
    "winning_option_index": 1,
    "option_votes": [12, 0],
    "reasoning_hash": "sha256:...",
    "evidence_bundle_hash": "sha256:..."
  },
  "source_items": [
    {
      "source_url": "https://polymarket.com/event/example",
      "source_type": "prediction_market",
      "source_publisher": "Polymarket",
      "source_published_at": "2026-03-26T14:00:00Z",
      "fetched_at": "2026-03-27T12:00:00Z",
      "content_hash": "sha256:...",
      "snapshot_uri": "ipfs://... or s3://...",
      "supports_outcome": true
    }
  ]
}
```

## Recommended Fields

### Top-Level Profile Fields

- `profile`
- `base_pack`
- `attestation`
- `source_items`

### Attestation Fields

- `prediction_id`
- `topic_id`
- `member_token_id`
- `resolution_kind`
- `winning_option_index`
- `option_votes`
- `reasoning_hash`
- `evidence_bundle_hash`

These fields help downstream systems connect evidence review to a candidate committee decision path.

### Source Item Fields

- `source_url`
- `source_type`
- `source_publisher`
- `source_published_at`
- `fetched_at`
- `content_hash`
- `snapshot_uri`
- `supports_outcome`

Optional additions:

- `source_title`
- `source_domain`
- `extractor_version`
- `normalization_version`
- `review_notes`
- `reviewer_labels`

## Why The Hash Fields Matter

Recommended hash-oriented fields:

- `content_hash`
- `reasoning_hash`
- `evidence_bundle_hash`

Their roles are different:

- `content_hash`: identifies the exact normalized source content or snapshot payload that was inspected
- `reasoning_hash`: identifies the reasoning payload or structured decision basis used downstream
- `evidence_bundle_hash`: identifies the attested review bundle as a whole

This separation matters because a workflow may reuse the same source content with different review logic or different downstream candidate outcomes.

## Snapshot Strategy

For higher-assurance workflows, a source URL alone is not enough.

URLs can change, disappear, or present updated content later.

That is why the profile should support:

- `snapshot_uri`
- `content_hash`

Recommended pattern:

1. fetch and normalize the source
2. store an immutable snapshot or normalized record
3. hash the stored content
4. include both the pointer and the hash in the attested profile

The engine does not need to own the storage backend for this to work.
It only needs to remain compatible with downstream systems that do.

## Relationship To The Base Engine

The current Trusted Evidence Engine repository already provides most of the base layer needed for this profile:

- request and response schema
- provider provenance
- policy metadata
- trust/freshness signals
- prediction-native inputs

What it does not yet do by default is produce a full attested profile automatically.

That is intentional.

The profile belongs to a stricter workflow tier than the base pack.

## Recommended Implementation Path

The practical rollout path is:

1. keep the base `evidence pack` as the main public API output
2. let committee or audit-oriented adapters build this stricter profile on top
3. later add optional engine support for profile-aware output modes if needed

That sequencing keeps the public engine reusable while still giving high-assurance consumers a clear upgrade path.

## What This Means For EvoEvo

For EvoEvo specifically, this profile is the missing layer between:

- generic public evidence resolution
- fully auditable committee evidence bundles

That means the public project can stay generic without losing alignment with EvoEvo's stricter committee needs.

## Status

Current status:

- base `evidence pack`: implemented
- committee-runtime consumption pattern: implemented at integration level
- attested evidence profile: documented design target

So this document should be read as:

- a profile design for the next stage
- not a claim that the current engine already emits all of these fields by default
