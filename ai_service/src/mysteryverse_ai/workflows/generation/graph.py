from typing import TypedDict

from langgraph.graph import END, START, StateGraph


class FoundationState(TypedDict, total=False):
    request_id: str


def _identity(state: FoundationState) -> FoundationState:
    return state


def build_foundation_graph():
    builder = StateGraph(FoundationState)
    builder.add_node("foundation", _identity)
    builder.add_edge(START, "foundation")
    builder.add_edge("foundation", END)
    return builder.compile()
