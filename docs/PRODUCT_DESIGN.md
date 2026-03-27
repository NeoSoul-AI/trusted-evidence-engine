# Product Design

## Overview

Trusted Evidence Engine is an open-source evidence layer that sits between retrieval and final decision-making.

Its job is not to output truth, settlement, or an autonomous action.
Its job is to turn public information into an auditable `evidence pack` that humans and downstream systems can inspect.

This project exists for workflows where provenance matters more than one-shot answers:

- deep research
- agent grounding
- prediction-market review
- committee/oracle support
- high-stakes automation

## Problem

Today, most systems do one of three things:

- return raw search results
- hide evidence gathering inside private orchestration
- jump directly from query to model output

That leaves an important gap:

- users cannot see what evidence was gathered
- downstream runtimes cannot apply policy against a stable artifact
- humans cannot easily review the basis for a recommendation or action

Trusted Evidence Engine fills that gap by producing a structured, reviewable evidence artifact.

## Product Thesis

The core thesis is:

- search is not evidence
- model output is not evidence
- orchestration logs are not evidence
- a reusable, inspectable evidence pack is evidence infrastructure

The project should therefore be evaluated as a shared evidence layer, not as a chatbot or final actor.

## EvoEvo Origin And Extraction Logic

This project originates from a real product and infrastructure need inside EvoEvo.

It is being extracted because the underlying problem is broader than EvoEvo itself.

Inside EvoEvo, the same evidence-layer need already appears in multiple places:

- hosted-agent grounding
- committee and oracle review support
- runtime and skill integrations that need structured public evidence

The extraction rule is:

- keep the core engine generic
- keep business workflows outside the engine
- preserve the fields and boundaries that real downstream runtimes already need

That means the public project should remain close to real consumers without becoming a dump of EvoEvo-specific platform logic.

## Target Users

The primary users are system builders, not general end users.

### 1. Deep Research and Agent Teams

They need a source-aware layer before summarization, planning, or action.

They use Trusted Evidence Engine to:

- collect public evidence
- normalize heterogeneous sources
- preserve provenance
- pass a stable artifact into an agent or research workflow

### 2. Prediction-Market and Oracle Developers

They need prediction-native evidence and reviewable public-source context.

They use Trusted Evidence Engine to:

- ingest market URLs and related sources
- gather public market metadata
- package evidence for review, preview, or candidate resolution logic

### 3. High-Stakes Automation Teams

They need inspectable inputs for workflows that cannot rely on opaque generation alone.

They use Trusted Evidence Engine to:

- collect evidence for a claim or event
- keep source provenance explicit
- enforce downstream review rules before action

## Primary Use Cases

### Deep Research

Input:

- a query
- a claim
- a list of public source URLs

Output:

- ranked evidence items with provenance and policy signals

### Agent Grounding

Input:

- a query, claim, or workflow context from an agent runtime

Output:

- a structured evidence pack that the agent can summarize, reason over, or pass to a reviewer

### Prediction-Market Review

Input:

- a Polymarket or other prediction-market URL
- optional supporting source URLs
- optional prediction context metadata

Output:

- market-aware evidence items
- provider provenance
- a pack suitable for downstream review or candidate-resolution logic

### High-Stakes Automation

Input:

- an event claim
- relevant public links
- optional domain-specific metadata

Output:

- a reproducible evidence pack that a workflow engine can gate on before acting

## Product Boundaries

Trusted Evidence Engine is responsible for:

- provider fetch and normalization
- source-aware evidence packaging
- deduplication
- trust and freshness ranking
- provenance and policy metadata
- a stable API and schema for downstream use

Trusted Evidence Engine is not responsible for:

- final truth
- final settlement
- final legal or financial judgment
- private-key handling
- signing or transaction submission
- replacing an agent runtime or committee runtime

This boundary is especially important for committee workflows.

The engine may provide evidence for a candidate decision path, but it should not:

- become the signer
- become the settlement authority
- imply that evidence ranking is equivalent to final truth

## UX Contract

The product should hide source-specific complexity from the caller.

Callers should be able to provide:

- a query
- a claim
- a raw public URL
- a prediction-market page URL

Callers should not need to know:

- provider-specific API endpoints
- provider-specific slug formats
- internal fetch routing details

Example:

- A user should be able to paste a raw Polymarket page URL.
- The engine should determine whether that URL is a market page, event page, or unsupported pattern.
- The engine should return structured evidence and explicit warnings when a page cannot be fully resolved.

## Input Model

Current canonical input:

- `query`
- `claim`
- `source_urls`
- `prediction_id`
- `topic_id`
- `context_metadata`

Future-friendly input expectations:

- raw public URLs should be first-class inputs
- prediction context should remain optional and additive
- URL ingestion should classify the page type before provider-specific resolution

## Output Model

The primary output is an `evidence pack`.

It must be:

- structured
- reviewable
- source-aware
- policy-aware
- reproducible enough for inspection

Minimum output fields:

- `schema_version`
- `query`
- `claim`
- `source_urls`
- `policy`
- `policy_version`
- `fetched_at`
- `items`

Each evidence item should expose:

- `title`
- `url`
- `domain`
- `source_type`
- `snippet`
- `published_at`
- `provider`
- `fetched_at`
- `reliability`
- `ranking_signals`
- `metadata`

The output contract should also support explicit operational feedback.

Recommended near-term additions:

- `warnings`
- `provider_status`

These fields are important when:

- a provider failed
- a URL resolved to no known entity
- a prediction-market event contains multiple child markets
- a provider returned partial results

Recommended next-stage additions for stricter committee and audit workflows:

- `content_hash`
- `snapshot_uri`
- `reasoning_hash`
- `evidence_bundle_hash`

These are not required for the base public evidence-pack schema, but they matter for high-assurance workflows that need stronger replay and audit guarantees.

See also:

- `docs/ATTESTED_EVIDENCE_PROFILE.md`

## Architecture Responsibilities

The intended engine pipeline is:

1. accept query, claim, and raw URLs
2. route inputs to relevant providers
3. normalize source documents into a shared schema
4. deduplicate overlapping documents
5. score trust and freshness with explicit policy signals
6. emit a structured evidence pack

Downstream systems may then:

- summarize the pack
- build a review UI
- generate a candidate resolution preview
- require human approval
- continue into their own runtime-specific logic

## LLM Boundary

LLMs are optional enrichment, not the default parsing layer.

The design principle is:

- deterministic parsing first
- optional LLM enrichment second

Trusted Evidence Engine should not begin by asking an LLM to read arbitrary webpages.
It should first:

- classify the input
- fetch the source
- parse known metadata
- normalize evidence

Optional LLM enrichment can then help with:

- claim extraction
- semantic relevance
- cross-source summarization
- conflict explanation

This preserves reviewability and reduces black-box behavior in the core evidence path.

## Policy Model

Trusted does not mean "official only".

Trusted means:

- source-aware
- policy-driven
- reviewable
- reproducible

Policy should remain:

- explicit
- versioned
- inspectable

Near-term policy signals include:

- domain trust
- freshness
- source-type weight
- provider weight

Longer-term policy evolution should support:

- configurable policy bundles
- allowlists and blocklists
- cross-source corroboration rules
- stricter modes for high-stakes workflows

## Open-Source Shape

This project is a good fit for open-source collaboration, but not as an ungoverned source aggregator.

## Current Maturity And Honest Positioning

The current repository should be positioned honestly as:

- a working `v0` engine
- already useful for experimental integrations
- already useful for prediction-native and agent-grounding scenarios
- not yet a full committee-grade attestation system by itself

What is already real today:

- real provider integrations
- a runnable CLI
- a runnable HTTP API
- a stable initial schema
- a concrete committee-runtime integration pattern

What is still early:

- generic hosted-agent adapter examples
- OpenClaw adapter packaging
- richer policy controls
- audit-grade bundle fields on top of the base evidence pack

That positioning is important because it keeps the project credible to both EvoEvo stakeholders and outside open-source users.

## Public Value Beyond EvoEvo

The public value of the project comes from the fact that the underlying problem is widely shared.

Many teams need a layer that is:

- more structured than search results
- more reviewable than model output
- more reusable than private orchestration logs

That makes the project relevant to:

- research-agent teams
- prediction-market developers
- review-heavy workflow automation
- any system that wants a policy-scored evidence artifact before downstream action

The project succeeds publicly when it stays focused on that common layer.
It loses value if it drifts into being only an EvoEvo-specific adapter or only a thin search wrapper.

The intended contribution model is:

- core engine stays opinionated and tightly owned
- extensions become easier to contribute over time

Core areas that should stay tightly controlled:

- schema
- evidence-pack contract
- provider interfaces
- trust-policy framework
- dedup and ranking behavior

Extension areas that can become community-friendly:

- provider adapters
- source presets
- vertical taxonomies
- benchmarks and evaluation cases
- runtime adapters and demos

## Why Now

The current AI ecosystem is seeing strong demand around:

- deep research
- agent workflows
- inspectable automation

Many projects optimize for action, orchestration, or generation.
Trusted Evidence Engine addresses the missing layer underneath them:

- auditable evidence
- explicit provenance
- reusable review artifacts

The project should therefore present itself as an evidence layer for modern agent and research workflows, without collapsing into "just another agent framework".

## Success Criteria

The project is succeeding if:

- a caller can pass a raw public URL or claim and get a useful evidence pack
- the pack preserves provider provenance and ranking signals
- downstream systems can inspect or consume the pack without hidden private logic
- the engine remains generic across research, prediction, and automation workflows

The project is failing if:

- it becomes a thin search wrapper
- evidence quality cannot be reviewed
- provider-specific complexity leaks into user-facing inputs
- policy decisions are hidden or inconsistent

## Current Alignment Review

This section compares the current implementation against the intended design.

### Areas That Already Match

- The repository already treats the engine as an evidence layer rather than a final actor.
- The codebase is cleanly split into `providers`, `dedup`, `ranking`, `schema`, `engine`, and `httpapi`.
- The output schema already includes policy and provenance-oriented fields such as `provider`, `ranking_signals`, `policy`, `policy_version`, and `fetched_at`.
- The committee adapter documentation preserves the boundary between evidence collection and settlement execution.

### Areas That Partially Match

- The current input model supports `query`, `claim`, `source_urls`, and prediction context, but URL ingestion is still provider-specific rather than truly page-type-aware.
- The trust-policy shape exists, but it is hard-coded rather than configurable or centrally versioned.
- The project already supports multiple providers, but the default experience is still partly skeleton/demo-driven.

### Areas That Do Not Yet Match

- The engine does not currently return `warnings` or `provider_status`, so partial failures and unresolved URLs are invisible to callers.
- Empty results can serialize as `items: null`, which weakens the output contract and looks like an engine error rather than an empty pack.
- The current Polymarket flow assumes event-style URLs can always be resolved through a market-slug lookup, which does not satisfy the UX contract for raw page URLs.
- Provider errors are currently swallowed during fan-out, which conflicts with the design goal of explicit operational feedback.
- Deduplication is still shallow and URL-based, not event-aware or canonicalization-aware.
- Ranking policy is implemented as static heuristics with duplicated policy-version constants.
- The demo provider can still shape the default experience, which is useful for bootstrapping but misaligned with a long-term trusted-evidence product.

## Priority Gaps To Close

The highest-priority design gaps are:

1. Add explicit `warnings` and `provider_status` to evidence output.
2. Make empty packs serialize as `items: []`.
3. Upgrade raw URL handling so callers can paste a Polymarket page without knowing slugs or provider internals.
4. Move trust-policy configuration and versioning into a single explicit policy layer.
5. Improve deduplication and cross-source corroboration so the engine is more than a retrieval wrapper.
