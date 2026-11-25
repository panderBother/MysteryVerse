from mysteryverse_ai.workflows.generation.graph import build_foundation_graph


def test_foundation_graph_executes() -> None:
    graph = build_foundation_graph()
    result = graph.invoke({"request_id": "test-request"})

    assert result["request_id"] == "test-request"
