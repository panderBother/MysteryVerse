# Observability

`trace.py` builds safe step records with timings, usage, status and content
hashes. `redaction.py` removes credentials, prompts, content and private fields
before structured logging.

The runtime propagates request_id, trace_id and run_id through HTTP execution,
model steps, tool calls and the final response. Full private payload logging is
disabled by default.
