# Committee Runtime Adapter Example

This folder documents how a committee runtime should consume Trusted Evidence Engine without collapsing the boundary between evidence collection and settlement execution.

Recommended flow:

1. Runtime builds `prediction context`
2. Runtime calls Trusted Evidence Engine over HTTP
3. Engine returns an `evidence pack`
4. Runtime planner turns the pack into a preview or candidate resolution
5. Signer and chain client remain responsible for submission

This adapter should stay outside the core engine so the engine remains generic and reusable.

## Recommended Runtime Mode

For the current committee runtime integration, the recommended planner is:

- `external_evidence_pack`

That planner should:

1. Fetch the prediction snapshot from the public read side
2. Gather `topic_source_url` and public evidence URLs
3. POST them to `/v1/evidence/resolve`
4. Inspect the returned `evidence pack`
5. Turn only actionable items into a preview or candidate submission

The runtime should not:

- move private keys into the engine
- let the engine submit onchain transactions
- treat every evidence item as automatically actionable

## Minimal Environment Example

```env
EVO_ORACLE_COMMITTEE_RUNTIME_PLANNER_STRATEGY=external_evidence_pack
EVO_ORACLE_COMMITTEE_RUNTIME_TRUSTED_EVIDENCE_ENGINE_URL=http://trusted-evidence-engine:8080
EVO_ORACLE_COMMITTEE_RUNTIME_TRUSTED_EVIDENCE_ENGINE_API_KEY=
```

## Current Actionable Path

The first actionable integration path is:

- `prediction_market` evidence items
- especially binary `Yes/No` markets with a clear resolved state

Other evidence types such as news, feeds, and quotes remain valuable for discovery and review, but they should not automatically become settlement payloads until the runtime has explicit mapping logic for them.
