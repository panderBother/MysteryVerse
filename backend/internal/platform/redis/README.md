# Redis Platform

Redis is a non-authoritative acceleration and coordination layer.

Allowed uses:

- Presence and socket routing hints.
- Temporary room projections and caches.
- Owner leases and rate limits.
- Pub/Sub, Streams and short-lived working memory.

Never use Redis as the only copy of game state, events, votes, settlements or
audit records. Losing Redis must not lose committed business facts.
