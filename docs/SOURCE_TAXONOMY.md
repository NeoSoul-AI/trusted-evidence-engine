# Source Taxonomy

Trusted Evidence Engine does not treat all sources as equal.

This project aims to be trusted not because it collects "official links only", but because it is explicit about source categories, their tradeoffs, and how they should be interpreted downstream.

## Principle

Trusted does not mean "official only".

Trusted means:

- source-aware
- policy-driven
- reviewable
- reproducible

## Source Categories

### 1. Official

Examples:

- government announcements
- regulator pages
- league announcements
- company investor relations pages
- protocol foundation statements

Strengths:

- strong authority
- usually the most defensible citation

Weaknesses:

- can be late
- can omit important context
- may not exist for some event types

Typical `source_type` values:

- `official`

### 2. Prediction Market

Examples:

- Polymarket market pages
- public market metadata
- resolution-source and resolved-by metadata

Strengths:

- prediction-native structure
- useful for market context and publicly visible resolution metadata

Weaknesses:

- not always the final legal or official authority
- can reflect intermediary resolution systems

Typical `source_type` values:

- `prediction_market`

### 3. Market Data

Examples:

- CoinGecko asset search
- exchange reference metadata
- public pricing/reference endpoints

Strengths:

- strong for entity resolution and market-reference context
- useful in crypto-native workflows

Weaknesses:

- usually not a final settlement authority
- often describes an asset, not an event outcome

Typical `source_type` values:

- `crypto_market_data`

### 4. Mainstream News

Examples:

- Reuters
- AP
- Bloomberg
- BBC
- ESPN

Strengths:

- broad coverage
- fast updates
- often better context than official statements

Weaknesses:

- still secondary reporting
- may summarize before the underlying authority is fully updated

Typical `source_type` values:

- `news`
- `sports_news`
- `crypto_news`

### 5. Vertical / Specialist Sources

Examples:

- sports desks
- crypto-native newsrooms
- protocol blogs
- niche industry publications

Strengths:

- domain expertise
- often faster than broad mainstream outlets

Weaknesses:

- quality varies sharply by outlet
- not always broadly trusted outside the niche

Typical `source_type` values:

- `sports_news`
- `crypto_news`
- `reference`

### 6. Community / Low-Confidence Sources

Examples:

- personal blogs
- community posts
- unattributed social chatter

Strengths:

- sometimes earliest signal

Weaknesses:

- weak authority
- often hard to verify
- high rumor risk

These should generally rank low or be excluded by stricter downstream policies.

## How To Use The Taxonomy

The taxonomy is not a final verdict system.

It exists so that:

- humans can understand what kind of evidence they are looking at
- runtimes can apply policy by source category
- committee systems can decide how much corroboration they require

Example:

- an `official` source may be enough for some workflows
- a `prediction_market` source may require one or more corroborating public sources
- a `crypto_news` source may be useful but not sufficient by itself

## Future Direction

Later versions can formalize this into:

- configurable source policies
- required cross-category corroboration
- source allowlists and blocklists
- stricter review modes for high-stakes workflows
