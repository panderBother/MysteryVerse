# Agents

The generic runtime lives in `agent_runtime/`.

It executes a finite LangGraph model/tool loop and returns a contract-validated
`AgentResponse`. It does not contain AI DM, NPC or script-generation business
logic, and it never mutates Go game state.
