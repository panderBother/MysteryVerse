"""API route modules."""

from mysteryverse_ai.api.routes.agent import router as agent_router
from mysteryverse_ai.api.routes.health import router as health_router

__all__ = ["agent_router", "health_router"]
