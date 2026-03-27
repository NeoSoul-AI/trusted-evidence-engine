# OpenClaw Skill Example

This example describes how an OpenClaw skill should consume Trusted Evidence Engine without embedding platform-specific business logic into the engine itself.

## Packaging Boundary

The OpenClaw packaging model should remain:

- `SKILL.md`
- `scripts/`
- `assets/`
- `references/`

The skill should call Trusted Evidence Engine over CLI or HTTP and pass the returned `evidence pack` into OpenClaw strategy logic.

The core Go engine should not be embedded directly into the skill bundle.

## Recommended Flow

1. OpenClaw receives a task or prediction context
2. The skill extracts `query`, `claim`, or `source_urls`
3. The skill calls Trusted Evidence Engine
4. The engine returns a structured `evidence pack`
5. OpenClaw strategy logic uses the pack as grounded input
6. Final output keeps explicit source references where possible

## Minimal Resolve Request

```json
{
  "claim": "This event already happened",
  "source_urls": [
    "https://polymarket.com/event/example-market"
  ],
  "context_metadata": {
    "workflow": "openclaw_skill"
  }
}
```

## What The Skill Adds

The OpenClaw layer is responsible for:

- task-specific prompting
- local strategy logic
- optional post-processing
- surfacing the evidence pack back to the operator or downstream tool

Trusted Evidence Engine remains responsible for:

- provider fetch
- normalization
- deduplication
- trust/freshness ranking
- evidence packaging

## Why This Separation Matters

If the skill embeds all retrieval logic privately, then:

- evidence handling becomes harder to review
- source behavior drifts across skills
- policy becomes harder to reason about

Using a shared evidence layer keeps the skill lightweight and makes provenance more reusable across runtimes.
