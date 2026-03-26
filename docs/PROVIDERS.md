# Providers

Trusted Evidence Engine is designed to consume multiple evidence providers.

## Current Providers

### Demo Provider

The demo provider exists so the repository can compile, test, and demonstrate the evidence-pack flow without any external dependency.

Use it when:

- you want to test the CLI or HTTP server locally
- you do not have any provider API keys yet
- you want deterministic skeleton output

The demo provider is the default only when no real provider is configured.

### Brave Search Web Provider

The Brave provider is the first real external provider in the skeleton.

It uses the Brave Search web endpoint and turns search results into normalized source documents.

Current behavior:

- reads `query`, `claim`, or the first `source_url` as the search subject
- requests the Brave web search API
- normalizes results into the shared source-document schema
- infers a rough `source_type` from domain heuristics
- returns structured documents for deduplication and ranking

Required environment variable:

```bash
BRAVE_SEARCH_API_KEY=...
```

Optional environment variables:

```bash
TEE_BRAVE_COUNTRY=us
TEE_BRAVE_SEARCH_LANG=en
TEE_BRAVE_COUNT=10
TEE_BRAVE_WEB_ENDPOINT=https://api.search.brave.com/res/v1/web/search
```

Important boundary:

- Brave Search is only one provider
- it is not the whole product
- the engine still has to normalize, deduplicate, rank, and emit a stable evidence pack

### Polymarket Market Provider

The Polymarket provider is a prediction-native provider that resolves public market metadata from Polymarket event URLs or Gamma market slug URLs.

Current behavior:

- reads Polymarket URLs from `source_urls`, `query`, or `claim`
- extracts the market slug
- fetches public market data from the Gamma API
- normalizes market metadata into the shared source-document schema
- emits a `prediction_market` evidence item with structured metadata

No API key is required for the basic public market lookup flow.

Optional environment variable:

```bash
TEE_POLYMARKET_GAMMA_ENDPOINT=https://gamma-api.polymarket.com
```

Current metadata coverage includes:

- `slug`
- `market_id`
- `category`
- `market_type`
- `closed`
- `active`
- `resolved_by`
- `resolution_source`
- `uma_resolution_status`
- `outcomes`
- `outcome_prices`
- `volume_num`
- `liquidity_num`

Important boundary:

- Polymarket metadata is useful evidence
- it is not automatically the final truth
- downstream runtimes still need their own review, planning, or settlement rules

### CoinGecko Search Provider

The CoinGecko provider is a crypto-native provider for asset discovery and market-reference metadata.

Current behavior:

- reads `query` or `claim` as the search subject
- calls the CoinGecko `/search` API
- normalizes matching coins into the shared source-document schema
- emits `crypto_market_data` evidence items

Required environment variable:

```bash
COINGECKO_API_KEY=...
```

Optional environment variables:

```bash
TEE_COINGECKO_PLAN=demo
TEE_COINGECKO_BASE_URL=https://api.coingecko.com/api/v3
TEE_COINGECKO_PER_PAGE=5
```

Current metadata coverage includes:

- `coin_id`
- `symbol`
- `api_symbol`
- `market_cap_rank`
- `thumb`
- `large`
- `plan`

Important boundary:

- CoinGecko is strong for crypto asset discovery and market-reference metadata
- it does not replace official issuer, exchange, or protocol sources

### CoinGecko Simple Price Provider

The CoinGecko simple price provider is a crypto quote provider for direct price lookups.

Current behavior:

- activates when the input clearly looks like a price query or when crypto price metadata is present
- uses `coin_id`, `crypto_symbol`, or `crypto_name` when available
- calls CoinGecko `simple/price`
- emits a `crypto_quote` evidence item

Required environment variable:

```bash
COINGECKO_API_KEY=...
```

Useful metadata:

```json
{
  "crypto_symbol": "eth",
  "vs_currency": "usd",
  "want_price": true
}
```

### Alpha Vantage Quote Provider

The Alpha Vantage provider is the first equity quote provider in the skeleton.

Current behavior:

- activates when the input clearly looks like a price query or when `ticker`/`equity_symbol` is present
- resolves a ticker from metadata, or falls back to `SYMBOL_SEARCH`
- fetches `GLOBAL_QUOTE`
- emits an `equity_quote` evidence item

Required environment variable:

```bash
ALPHAVANTAGE_API_KEY=...
```

Optional environment variables:

```bash
TEE_ALPHAVANTAGE_BASE_URL=https://www.alphavantage.co/query
TEE_ALPHAVANTAGE_ENTITLEMENT=
```

Useful metadata:

```json
{
  "ticker": "MSFT",
  "want_price": true
}
```

### Feed Provider

The feed provider is a configurable provider for vertical sources such as sports, crypto news, and niche industry feeds.

Current behavior:

- reads feed URLs from configuration
- fetches RSS or Atom feeds
- matches entries against query/claim terms
- emits normalized documents with inferred source types like `sports_news`, `crypto_news`, or `news`

Required environment variable:

```bash
TEE_FEED_URLS=https://example.com/feed.xml,https://example.org/atom.xml
```

Optional environment variables:

```bash
TEE_FEED_MAX_ITEMS=10
TEE_SOURCE_PRESETS=sports_basic,crypto_basic
TEE_SOURCE_PRESETS_PATH=/abs/path/to/source-presets.json
```

This is the easiest way to support vertical evidence sets such as:

- sports desks
- crypto newsrooms
- league or team announcement feeds
- company blogs
- protocol announcements

See also:

- [docs/SOURCE_PRESETS.md](./SOURCE_PRESETS.md)

## Provider Selection

Current default selection behavior:

1. Always include the Polymarket provider.
2. If `COINGECKO_API_KEY` is present, also include the CoinGecko provider.
3. If `COINGECKO_API_KEY` is present, also include the CoinGecko simple price provider.
4. If `ALPHAVANTAGE_API_KEY` is present, also include the Alpha Vantage quote provider.
5. If `TEE_FEED_URLS` is configured, also include the feed provider.
6. If `BRAVE_SEARCH_API_KEY` is present, also include the Brave provider.
7. If no extra real provider is configured, fall back to the demo provider for generic local testing.

This is intentionally simple for the skeleton stage. A later version can support:

- multiple real providers at once
- configurable provider order
- per-provider weighting
- provider-specific fetch policies

When `TEE_SOURCE_PRESETS` is active and the selected presets contain `provider_hints`, those hints constrain the default provider set.
