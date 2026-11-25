# Context

`builder.py` turns Go-authorized `AgentRequest` context and a short-term memory
view into budgeted provider messages. It records only section hashes and token
estimates in trace summaries.

Python never fetches extra game data by entity ID. ADR-008 remains Proposed, so
PlayerContext persistence is not implemented.
