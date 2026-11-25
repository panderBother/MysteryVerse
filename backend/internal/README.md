# Backend Internal

Domain modules will follow the `domain`, `app`, `ports`, `adapters` and
`transport` boundaries defined in `TECH_DESIGN.md`.

Phase 0 currently contains reusable platform packages, typed `sqlc` queries in
`storage/queries`, generated SQL code and the base storage Repository adapter.
HTTP handlers must not issue ad hoc SQL.
