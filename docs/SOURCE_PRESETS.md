# Source Presets

Trusted Evidence Engine supports reusable source presets so users do not have to hand-build every feed list from scratch.

## Why Presets Exist

Presets make it easier to bootstrap common workflows such as:

- sports monitoring
- crypto evidence gathering
- macro/policy tracking
- equity quote workflows

They do not replace custom configuration.  
They provide a safe, understandable starting point.

## Current Presets

Preset definitions live in:

- `configs/source-presets.json`

Current starter presets:

- `sports_basic`
- `crypto_basic`
- `macro_basic`
- `equity_basic`

## How Presets Are Used

Presets are currently used by the feed provider.

When `TEE_SOURCE_PRESETS` is set, the engine:

1. loads the matching presets from `configs/source-presets.json`
2. extracts their `feed_urls`
3. merges those feed URLs with any explicit `TEE_FEED_URLS`
4. extracts their `provider_hints`
5. uses those hints to constrain the default provider set
6. deduplicates the final feed list

## Environment Variables

```bash
TEE_SOURCE_PRESETS=sports_basic,crypto_basic
TEE_SOURCE_PRESETS_PATH=/abs/path/to/source-presets.json
```

If `TEE_SOURCE_PRESETS_PATH` is empty, the engine will look for:

- `configs/source-presets.json` in the current working directory

## What A Preset Contains

Each preset can define:

- `name`
- `description`
- `feed_urls`
- `provider_hints`
- `notes`

`provider_hints` are now executable defaults.

If at least one preset is active and provider hints are present, the default provider set is filtered by those hints.

Example:

- `sports_basic` can prefer `feed_aggregator` and `brave_search`
- `crypto_basic` can prefer `coingecko_search`, `coingecko_simple_price`, `feed_aggregator`, and `brave_search`

If no presets are active, the engine falls back to its normal environment-driven provider selection.

## Example

```bash
export TEE_SOURCE_PRESETS="sports_basic,macro_basic"
export TEE_FEED_MAX_ITEMS="20"

go run ./cmd/tee resolve --query "Lakers opener" --format json
```

## Important Boundary

Presets are convenience defaults, not final truth.

You should still:

- review the bundled feeds
- add or remove sources for your workflow
- tighten sources for high-stakes uses
