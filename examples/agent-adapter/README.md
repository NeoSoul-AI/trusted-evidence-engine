# Agent Adapter Example

This example describes how a hosted agent or generic agent runtime should consume Trusted Evidence Engine as a grounding layer.

The goal is simple:

- keep retrieval and evidence packaging outside the model prompt
- pass a structured `evidence pack` into the agent
- preserve source URLs and ranking signals for later review

## Recommended Flow

1. Agent receives a query, claim, or prediction context
2. Adapter calls Trusted Evidence Engine
3. Engine returns an `evidence pack`
4. Adapter filters or trims the pack for prompt size
5. Agent uses the pack as grounded input for report, opinion, or action generation
6. Agent output keeps source references instead of dropping provenance

## Minimal HTTP Example

Request:

```json
{
  "query": "Did the Federal Reserve cut rates?",
  "context_metadata": {
    "workflow": "hosted_agent_grounding"
  }
}
```

Typical adapter behavior after response:

- keep the top `N` evidence items
- require a minimum `reliability` threshold
- pass `title`, `url`, `snippet`, and selected metadata into the agent prompt
- preserve the full pack for logging or reviewer access

## Why This Matters

Without a dedicated evidence layer, many agents do one of two bad things:

- they see only raw search results
- or they jump straight to model output with weak provenance

Using the engine gives the agent a better contract:

- structured evidence in
- grounded generation out

## What The Adapter Should Not Do

The adapter should not:

- hide the evidence pack from reviewers
- treat `reliability` as final truth
- throw away URLs and provenance after generation

The point is not to make the agent magically correct.
The point is to make the agent's evidence basis inspectable.
