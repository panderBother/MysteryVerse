# Memory

`MemoryStore` is a non-authoritative short-term context port.

`InMemoryMemoryStore` is bounded and process-local. It must not be treated as
game truth. Long-term NPC memory, MemoryRecord writes and MemoryCandidate
acceptance remain blocked by ADR-001 and still require Go authority.
