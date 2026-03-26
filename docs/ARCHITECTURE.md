# Architecture Notes

Trusted Evidence Engine is intentionally split into these layers:

- `providers`: fetch external material
- `dedup`: remove repeated documents
- `ranking`: score trust and freshness
- `schema`: define stable request and response shapes
- `engine`: orchestrate fetch, dedup, and ranking
- `httpapi`: expose the engine as an HTTP service

The current implementation is a minimal skeleton and uses a demo provider so the project compiles and runs before real provider adapters are extracted.
