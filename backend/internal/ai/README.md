# Go AI Integration

`client/` is the minimal Go-to-Python transport for the internal Agent Runtime.

It propagates service identity, `request_id`, `trace_id` and
`Idempotency-Key`, supports context deadlines and bounded retry, performs strict
JSON decoding and validates the Agent response envelope.

It is not yet wired into AI DM, NPC or script-generation game flows. Game state
changes remain in Go domain/application code; an Agent response is only a
candidate or narrative output.
