# Trust Policy

This document explains why Trusted Evidence Engine should be considered a trust-scored evidence layer instead of a generic search wrapper.

## What Makes This Project "Trusted"

This project is not trusted because it claims authority.

It is trusted to the extent that it is:

- explicit about source categories
- explicit about ranking policy
- transparent about provider origin
- reproducible enough for review
- limited in scope

The engine does not output final truth.  
It outputs a structured evidence pack with policy signals attached.

## Current Policy Shape

The current skeleton uses a simple weighted policy:

- `domain_trust`
- `freshness`
- `source_type_weight`
- `provider_weight`

These signals are emitted with each evidence item.

Current defaults are implemented in:

- `internal/ranking/ranking.go`

The current policy name and version are emitted in evidence output:

- `policy=default_trust_policy`
- `policy_version=2026-03-26`

## Why "Official Only" Is Not Enough

Official sources matter, but they are not sufficient as the only idea of trust.

Reasons:

- some events have no clean official page
- official pages can lag behind reality
- official pages can omit the context a runtime needs
- non-official but high-quality sources may still be valuable corroboration

Trust therefore has to mean more than "official link present".

## Reproducibility

Trust also depends on reproducibility.

At minimum, a reviewer should be able to inspect:

- which providers were used
- which sources were returned
- what the ranking signals were
- which policy version was applied

This is why the engine emits:

- `provider`
- `source_type`
- `ranking_signals`
- `policy`
- `policy_version`
- `schema_version`

## Reviewability

The evidence pack is designed so that:

- committee runtimes can inspect evidence before acting
- humans can challenge or confirm the evidence basis
- other systems can consume the same pack without private hidden logic

The project becomes untrustworthy if it drifts into:

- hidden private ranking rules
- unexplained source promotion
- opaque one-shot answers without evidence packaging

## Future Policy Work

Later versions should support:

- configurable trust policies
- policy bundles for different workflows
- source allowlists and blocklists
- cross-source corroboration rules
- stricter modes for high-stakes workflows
