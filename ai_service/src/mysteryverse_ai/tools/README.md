# Tools

Tools are registered explicitly with a Pydantic argument schema, declared
permissions, allowed actor types and a bounded timeout.

The model can invoke only tools present in `AgentRequest.allowed_tools`.
`AgentRequest.allowed_permissions` must cover all permissions declared by the
tool. Arguments are validated before execution. Tool arguments and results are
not copied into ordinary logs; traces keep hashes and bounded summaries.

`ToolSpec.mcp_server` reserves a source identifier for future MCP adapters.
This task does not implement an MCP client or arbitrary remote tool discovery.
