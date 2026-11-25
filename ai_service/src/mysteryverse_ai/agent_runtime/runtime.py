from __future__ import annotations

import asyncio
import json
import logging
import time
import uuid
from contextlib import suppress
from dataclasses import asdict
from typing import Any

from langgraph.graph import END, START, StateGraph
from pydantic import ValidationError

from mysteryverse_ai.agent_runtime.outputs import (
    ModelAction,
    OutputSchemaRegistration,
    OutputSchemaRegistry,
    OutputSchemaSelectionError,
    validation_issues,
)
from mysteryverse_ai.agent_runtime.run_store import AgentRunStore, InMemoryAgentRunStore
from mysteryverse_ai.agent_runtime.state import AgentGraphState
from mysteryverse_ai.config import Settings
from mysteryverse_ai.context.builder import BuiltContext, ContextBuilder
from mysteryverse_ai.memory.base import MemoryStore
from mysteryverse_ai.memory.models import MemoryMessage, MemoryScope
from mysteryverse_ai.observability.redaction import log_structured
from mysteryverse_ai.observability.trace import hash_payload, trace_step
from mysteryverse_ai.providers.models import (
    ModelRequest,
    ProviderError,
    ProviderMessage,
    ProviderRouter,
)
from mysteryverse_ai.schemas.generated.models import (
    AgentExecutionLimits,
    AgentExecutionStatus,
    AgentOutput,
    AgentOutputValidationIssue,
    AgentRequest,
    AgentResponse,
    AgentToolCall,
    AgentTraceStep,
    AgentUsage,
    AIError,
    BusinessErrorCode,
    Metadata,
    Sensitivity,
    Status,
    StepType,
    VisibilityScope,
)
from mysteryverse_ai.tools.models import ToolInvocationContext
from mysteryverse_ai.tools.registry import ToolRegistry


class AgentRuntime:
    def __init__(
        self,
        *,
        provider: ProviderRouter,
        tools: ToolRegistry,
        memory: MemoryStore,
        context_builder: ContextBuilder,
        settings: Settings,
        output_schemas: OutputSchemaRegistry | None = None,
        run_store: AgentRunStore | None = None,
    ) -> None:
        self._provider = provider
        self._tools = tools
        self._memory = memory
        self._context_builder = context_builder
        self._settings = settings
        self._output_schemas = output_schemas or OutputSchemaRegistry()
        self._run_store = run_store or InMemoryAgentRunStore()
        self._logger = logging.getLogger("mysteryverse_ai.agent_runtime")

    async def aclose(self) -> None:
        close = getattr(self._provider, "aclose", None)
        if close is not None:
            await close()

    async def run(
        self,
        request: AgentRequest,
        *,
        cancellation_event: asyncio.Event | None = None,
    ) -> AgentResponse:
        started = time.monotonic()
        trace_id = request.trace_id or request.request_id
        run_started = False
        try:
            task_type = request.task_type
            if task_type is None or not request.task_goal:
                return self._failure_response(
                    request,
                    trace_id=trace_id,
                    error_code="VALIDATION_ERROR",
                    error_message="task_type and task_goal are required.",
                    retryable=False,
                    steps=[],
                    started=started,
                )
            try:
                output_schema = self._output_schemas.resolve(
                    task_type,
                    requested_key=request.output_schema_key,
                    requested_version=request.output_schema_version,
                )
            except OutputSchemaSelectionError as exc:
                return self._failure_response(
                    request,
                    trace_id=trace_id,
                    error_code=exc.code,
                    error_message=exc.message,
                    retryable=False,
                    steps=[],
                    started=started,
                )
            limits = self._resolve_limits(request)
            try:
                await self._run_store.start(
                    run_id=str(request.run_id),
                    request_id=request.request_id,
                    trace_id=trace_id,
                    input_hash=hash_payload(request.model_dump(mode="json")),
                    output_schema_key=output_schema.key,
                    output_schema_version=output_schema.version,
                    ttl_seconds=self._settings.agent_run_ttl_seconds,
                )
            except RuntimeError:
                return self._failure_response(
                    request,
                    trace_id=trace_id,
                    error_code="WORKFLOW_FAILED",
                    error_message="The agent run already exists.",
                    retryable=False,
                    steps=[],
                    started=started,
                )
            run_started = True
            scope = MemoryScope(
                actor_type=request.actor_type.value,
                actor_id=request.actor_id,
                game_id=str(request.game_id) if request.game_id else None,
                run_id=str(request.run_id),
            )
            memory_view = await self._memory.load(scope, max_messages=12)
            allowed_tools = {item.root for item in (request.allowed_tools or [])}
            allowed_permissions = {item.root for item in (request.allowed_permissions or [])}
            tool_specs = self._tools.describe(allowed_tools)
            context_started = time.monotonic()
            context = self._context_builder.build(
                request=request,
                memory=memory_view,
                max_tokens=limits.max_context_tokens,
                tools=tool_specs,
                output_schema=output_schema.schema,
                allow_restricted_sensitivity=self._settings.allow_restricted_model_context,
            )
            context_duration_ms = _elapsed_ms(context_started)
            response = await self._execute_graph(
                request=request,
                trace_id=trace_id,
                limits=limits,
                context=context,
                output_schema=output_schema,
                allowed_tools=allowed_tools,
                allowed_permissions=allowed_permissions,
                scope=scope,
                cancellation_event=cancellation_event,
                started=started,
                context_duration_ms=context_duration_ms,
            )
            await self._finish_run(str(request.run_id), response)
            return response
        except Exception:
            if run_started:
                with suppress(Exception):
                    await self._run_store.finish(
                        str(request.run_id),
                        status="failed",
                        step_count=0,
                        input_tokens=0,
                        output_tokens=0,
                        error_code="INTERNAL_ERROR",
                    )
            self._logger.exception(
                "agent runtime failed",
                extra={
                    "structured": {
                        "run_id": str(request.run_id),
                        "request_id": request.request_id,
                        "trace_id": trace_id,
                    }
                },
            )
            return self._failure_response(
                request,
                trace_id=trace_id,
                error_code="INTERNAL_ERROR",
                error_message="The agent runtime failed.",
                retryable=True,
                steps=[],
                started=started,
            )

    async def _execute_graph(
        self,
        *,
        request: AgentRequest,
        trace_id: str,
        limits: AgentExecutionLimits,
        context: BuiltContext,
        output_schema: OutputSchemaRegistration,
        allowed_tools: set[str],
        allowed_permissions: set[str],
        scope: MemoryScope,
        cancellation_event: asyncio.Event | None,
        started: float,
        context_duration_ms: int,
    ) -> AgentResponse:
        graph = self._build_graph(
            request=request,
            trace_id=trace_id,
            output_schema=output_schema,
            allowed_tools=allowed_tools,
            allowed_permissions=allowed_permissions,
        )
        initial_state: AgentGraphState = {
            "messages": [
                {"role": message.role, "content": message.content} for message in context.messages
            ],
            "steps": [
                trace_step(
                    step_index=0,
                    step_type=StepType.context,
                    name="context_builder",
                    status=Status.succeeded,
                    duration_ms=context_duration_ms,
                    summary={
                        "sections": [asdict(section) for section in context.sections],
                        "restricted_messages_omitted": context.restricted_messages_omitted,
                    },
                    error_code=(
                        "SENSITIVE_CONTEXT_BLOCKED" if context.restricted_messages_omitted else None
                    ),
                ).model_dump(mode="json")
            ],
            "tool_calls": [],
            "usage": {"input_tokens": 0, "output_tokens": 0, "cost_usd": None},
            "step_count": 0,
            "next_action": None,
            "status": None,
            "error_code": None,
            "error_message": None,
            "error_retryable": False,
            "structured_output": None,
            "validation_errors": [],
            "output_schema_key": output_schema.key,
            "output_schema_version": output_schema.version,
        }
        execution_task = asyncio.create_task(
            graph.ainvoke(
                initial_state,
                config={"recursion_limit": limits.max_steps * 2 + 6},
            )
        )
        cancel_task: asyncio.Task[bool] | None = None
        if cancellation_event is not None:
            cancel_task = asyncio.create_task(cancellation_event.wait())

        try:
            if cancel_task is None:
                state = await asyncio.wait_for(
                    execution_task,
                    timeout=min(limits.timeout_ms, request.budget_ms) / 1000,
                )
            else:
                done, _ = await asyncio.wait(
                    {execution_task, cancel_task},
                    timeout=min(limits.timeout_ms, request.budget_ms) / 1000,
                    return_when=asyncio.FIRST_COMPLETED,
                )
                if cancel_task in done and cancellation_event is not None:
                    execution_task.cancel()
                    with suppress(asyncio.CancelledError):
                        await execution_task
                    return self._controlled_response(
                        request,
                        trace_id=trace_id,
                        status=AgentExecutionStatus.cancelled,
                        error_code=None,
                        message="The agent execution was cancelled.",
                        steps=initial_state["steps"],
                        started=started,
                    )
                if execution_task not in done:
                    execution_task.cancel()
                    with suppress(asyncio.CancelledError):
                        await execution_task
                    return self._controlled_response(
                        request,
                        trace_id=trace_id,
                        status=AgentExecutionStatus.timeout,
                        error_code="AI_TIMEOUT",
                        message="The agent execution timed out.",
                        steps=initial_state["steps"],
                        started=started,
                    )
                state = execution_task.result()
        except TimeoutError:
            execution_task.cancel()
            with suppress(asyncio.CancelledError):
                await execution_task
            return self._controlled_response(
                request,
                trace_id=trace_id,
                status=AgentExecutionStatus.timeout,
                error_code="AI_TIMEOUT",
                message="The agent execution timed out.",
                steps=initial_state["steps"],
                started=started,
            )
        finally:
            if cancel_task is not None:
                cancel_task.cancel()
                with suppress(asyncio.CancelledError):
                    await cancel_task

        response = self._success_response(
            request,
            trace_id=trace_id,
            state=state,
        )
        if response.status == AgentExecutionStatus.succeeded and request.task_goal:
            answer = ""
            if response.structured_output is not None:
                answer = str(response.structured_output.model_dump().get("answer", ""))
            await self._memory.append_short_term(
                scope,
                [
                    MemoryMessage(
                        role="user",
                        content=request.task_goal,
                        sensitivity=Sensitivity.d0,
                    ),
                    MemoryMessage(
                        role="assistant",
                        content=answer,
                        sensitivity=Sensitivity.d0,
                    ),
                ],
                max_messages=24,
            )
        log_structured(
            self._logger,
            "agent run completed",
            run_id=str(request.run_id),
            request_id=request.request_id,
            trace_id=trace_id,
            status=response.status.value if response.status else None,
            step_count=len(response.steps or []),
            tool_call_count=len(response.tool_calls or []),
            input_tokens=response.usage.input_tokens,
            output_tokens=response.usage.output_tokens,
        )
        return response

    async def _finish_run(self, run_id: str, response: AgentResponse) -> None:
        status = response.status.value if response.status else "failed"
        if status not in {"succeeded", "failed", "timeout", "cancelled"}:
            status = "failed"
        await self._run_store.finish(
            run_id,
            status=status,  # type: ignore[arg-type]
            step_count=len(response.steps or []),
            input_tokens=response.usage.input_tokens,
            output_tokens=response.usage.output_tokens,
            error_code=response.error.code.value if response.error else None,
        )

    def _build_graph(
        self,
        *,
        request: AgentRequest,
        trace_id: str,
        output_schema: OutputSchemaRegistration,
        allowed_tools: set[str],
        allowed_permissions: set[str],
    ) -> Any:
        limits = self._resolve_limits(request)
        graph = StateGraph(AgentGraphState)

        async def model_node(state: AgentGraphState) -> AgentGraphState:
            return await self._model_node(
                state,
                request=request,
                limits=limits,
            )

        async def tool_node(state: AgentGraphState) -> AgentGraphState:
            return await self._tool_node(
                state,
                request=request,
                trace_id=trace_id,
                allowed_tools=allowed_tools,
                allowed_permissions=allowed_permissions,
            )

        async def validate_node(state: AgentGraphState) -> AgentGraphState:
            return self._validate_node(state, output_schema=output_schema)

        graph.add_node("model", model_node)
        graph.add_node("tool", tool_node)
        graph.add_node("validate", validate_node)
        graph.add_edge(START, "model")
        graph.add_conditional_edges(
            "model",
            self._route_after_model,
            {"tool": "tool", "validate": "validate", "end": END},
        )
        graph.add_conditional_edges(
            "tool",
            self._route_after_tool,
            {"model": "model", "end": END},
        )
        graph.add_edge("validate", END)
        return graph.compile()

    async def _model_node(
        self,
        state: AgentGraphState,
        *,
        request: AgentRequest,
        limits: AgentExecutionLimits,
    ) -> AgentGraphState:
        steps = list(state.get("steps", []))
        if state.get("step_count", 0) >= limits.max_steps:
            return self._failed_state(
                state,
                code="WORKFLOW_FAILED",
                message="The maximum agent step count was reached.",
                step_name="max_steps",
            )
        started = time.monotonic()
        messages = [ProviderMessage.model_validate(item) for item in state.get("messages", [])]
        provider_request = ModelRequest(
            messages=tuple(messages),
            max_output_tokens=limits.max_output_tokens or 512,
            metadata={
                "task_type": request.task_type.value if request.task_type else "",
                "run_id": str(request.run_id),
            },
        )
        try:
            result = await self._provider.generate(
                provider_request,
                timeout_ms=min(limits.timeout_ms, request.budget_ms),
            )
        except ProviderError as exc:
            steps.append(
                trace_step(
                    step_index=len(steps),
                    step_type=StepType.model,
                    name=self._settings.model_provider,
                    status=Status.timeout if exc.code == "AI_TIMEOUT" else Status.failed,
                    duration_ms=_elapsed_ms(started),
                    error_code=exc.code,
                ).model_dump(mode="json")
            )
            return {
                **state,
                "steps": steps,
                "step_count": state.get("step_count", 0) + 1,
                "status": "failed",
                "error_code": exc.code,
                "error_message": exc.message,
                "error_retryable": exc.retryable,
            }
        try:
            action = ModelAction.model_validate(json.loads(result.content))
        except json.JSONDecodeError:
            steps.append(
                trace_step(
                    step_index=len(steps),
                    step_type=StepType.model,
                    name=result.model_name,
                    status=Status.failed,
                    duration_ms=_elapsed_ms(started),
                    model_name=result.model_name,
                    error_code="VALIDATION_ERROR",
                ).model_dump(mode="json")
            )
            return {
                **state,
                "steps": steps,
                "step_count": state.get("step_count", 0) + 1,
                "status": "failed",
                "error_code": "VALIDATION_ERROR",
                "error_message": "The model returned an invalid action payload.",
                "error_retryable": False,
                "validation_errors": [
                    {
                        "path": "action",
                        "message": "The model response is not valid JSON.",
                        "type": "json_invalid",
                    }
                ],
            }
        except ValidationError as exc:
            steps.append(
                trace_step(
                    step_index=len(steps),
                    step_type=StepType.model,
                    name=result.model_name,
                    status=Status.failed,
                    duration_ms=_elapsed_ms(started),
                    model_name=result.model_name,
                    error_code="VALIDATION_ERROR",
                ).model_dump(mode="json")
            )
            return {
                **state,
                "steps": steps,
                "step_count": state.get("step_count", 0) + 1,
                "status": "failed",
                "error_code": "VALIDATION_ERROR",
                "error_message": "The model returned an invalid action payload.",
                "error_retryable": False,
                "validation_errors": [
                    issue.model_dump(mode="json") for issue in validation_issues(exc)
                ],
            }
        usage = result.usage
        cumulative = _merge_usage(state.get("usage", {}), usage.input_tokens, usage.output_tokens)
        steps.append(
            trace_step(
                step_index=len(steps),
                step_type=StepType.model,
                name=result.model_name,
                status=Status.succeeded,
                duration_ms=_elapsed_ms(started),
                usage=AgentUsage(
                    input_tokens=usage.input_tokens,
                    output_tokens=usage.output_tokens,
                    cost_usd=usage.cost_usd,
                ),
                model_name=result.model_name,
                summary={"content": result.content},
            ).model_dump(mode="json")
        )
        updated_messages = [*state.get("messages", [])]
        updated_messages.append({"role": "assistant", "content": result.content})
        return {
            **state,
            "messages": updated_messages,
            "steps": steps,
            "usage": cumulative,
            "step_count": state.get("step_count", 0) + 1,
            "next_action": action.action,
            "pending_tool_name": action.tool_name,
            "pending_arguments": action.arguments,
            "pending_output": action.output,
        }

    async def _tool_node(
        self,
        state: AgentGraphState,
        *,
        request: AgentRequest,
        trace_id: str,
        allowed_tools: set[str],
        allowed_permissions: set[str],
    ) -> AgentGraphState:
        limits = self._resolve_limits(request)
        if state.get("step_count", 0) >= limits.max_steps:
            return self._failed_state(
                state,
                code="WORKFLOW_FAILED",
                message="The maximum agent step count was reached.",
                step_name="max_steps",
            )
        tool_name = state.get("pending_tool_name")
        if not tool_name:
            return self._failed_state(
                state,
                code="VALIDATION_ERROR",
                message="The model requested a tool without a name.",
                step_name="tool",
            )
        execution = await self._tools.invoke(
            call_id=uuid.uuid4().hex,
            tool_name=tool_name,
            arguments=state.get("pending_arguments", {}),
            allowed_tools=allowed_tools,
            allowed_permissions=allowed_permissions,
            context=ToolInvocationContext(
                run_id=str(request.run_id),
                request_id=request.request_id,
                trace_id=trace_id,
                actor_type=request.actor_type.value,
                actor_id=request.actor_id,
            ),
        )
        steps = list(state.get("steps", []))
        steps.append(
            trace_step(
                step_index=len(steps),
                step_type=StepType.tool,
                name=tool_name,
                status=Status(execution.status),
                duration_ms=execution.duration_ms,
                error_code=execution.error_code,
                summary={
                    "arguments_hash": execution.arguments_hash,
                    "summary": execution.summary,
                },
            ).model_dump(mode="json")
        )
        tool_calls = list(state.get("tool_calls", []))
        tool_calls.append(
            AgentToolCall(
                call_id=execution.call_id,
                tool_name=execution.tool_name,
                arguments_hash=execution.arguments_hash,
                status=Status(execution.status),
                duration_ms=execution.duration_ms,
                error_code=execution.error_code,
            ).model_dump(mode="json")
        )
        updated: AgentGraphState = {
            **state,
            "steps": steps,
            "tool_calls": tool_calls,
            "step_count": state.get("step_count", 0) + 1,
        }
        if execution.status != "succeeded":
            updated.update(
                {
                    "status": "failed",
                    "next_action": None,
                    "error_code": execution.error_code or "WORKFLOW_FAILED",
                    "error_message": "The tool execution failed.",
                    "error_retryable": execution.status == "timeout",
                }
            )
            return updated
        messages = list(state.get("messages", []))
        messages.append(
            {
                "role": "tool",
                "content": json.dumps(
                    execution.content or {"summary": execution.summary or ""},
                    ensure_ascii=False,
                    separators=(",", ":"),
                ),
            }
        )
        updated.update(
            {
                "messages": messages,
                "next_action": "model",
                "pending_tool_name": None,
                "pending_arguments": {},
                "pending_output": None,
            }
        )
        return updated

    def _validate_node(
        self,
        state: AgentGraphState,
        *,
        output_schema: OutputSchemaRegistration,
    ) -> AgentGraphState:
        started = time.monotonic()
        steps = list(state.get("steps", []))
        try:
            parsed = output_schema.schema.model_validate(state.get("pending_output"))
        except ValidationError as exc:
            steps.append(
                trace_step(
                    step_index=len(steps),
                    step_type=StepType.validate,
                    name=output_schema.display_name,
                    status=Status.failed,
                    duration_ms=_elapsed_ms(started),
                    error_code="VALIDATION_ERROR",
                ).model_dump(mode="json")
            )
            return {
                **state,
                "steps": steps,
                "step_count": state.get("step_count", 0) + 1,
                "status": "failed",
                "error_code": "VALIDATION_ERROR",
                "error_message": "The structured agent output is invalid.",
                "error_retryable": False,
                "validation_errors": [
                    issue.model_dump(mode="json") for issue in validation_issues(exc)
                ],
                "next_action": None,
            }
        steps.append(
            trace_step(
                step_index=len(steps),
                step_type=StepType.validate,
                name=output_schema.display_name,
                status=Status.succeeded,
                duration_ms=_elapsed_ms(started),
                summary=parsed.model_dump(mode="json"),
            ).model_dump(mode="json")
        )
        return {
            **state,
            "steps": steps,
            "step_count": state.get("step_count", 0) + 1,
            "status": "succeeded",
            "structured_output": parsed.model_dump(mode="json"),
            "validation_errors": [],
            "next_action": None,
        }

    @staticmethod
    def _route_after_model(state: AgentGraphState) -> str:
        if state.get("status") == "failed":
            return "end"
        if state.get("next_action") == "tool":
            return "tool"
        return "validate"

    @staticmethod
    def _route_after_tool(state: AgentGraphState) -> str:
        if state.get("status") == "failed":
            return "end"
        return "model"

    def _failed_state(
        self,
        state: AgentGraphState,
        *,
        code: str,
        message: str,
        step_name: str,
    ) -> AgentGraphState:
        steps = list(state.get("steps", []))
        steps.append(
            trace_step(
                step_index=len(steps),
                step_type=StepType.model,
                name=step_name,
                status=Status.failed,
                duration_ms=0,
                error_code=code,
            ).model_dump(mode="json")
        )
        return {
            **state,
            "steps": steps,
            "status": "failed",
            "error_code": code,
            "error_message": message,
            "error_retryable": False,
            "next_action": None,
        }

    def _resolve_limits(self, request: AgentRequest) -> AgentExecutionLimits:
        return request.execution_limits or AgentExecutionLimits(
            timeout_ms=min(request.budget_ms, self._settings.agent_default_timeout_ms),
            max_steps=self._settings.agent_default_max_steps,
            max_context_tokens=self._settings.agent_default_context_tokens,
        )

    def _success_response(
        self,
        request: AgentRequest,
        *,
        trace_id: str,
        state: AgentGraphState,
    ) -> AgentResponse:
        status = (
            AgentExecutionStatus.succeeded
            if state.get("status") == "succeeded"
            else AgentExecutionStatus.failed
        )
        structured = state.get("structured_output")
        output: AgentOutput | None = None
        if status == AgentExecutionStatus.succeeded and structured is not None:
            output = AgentOutput(
                speech=str(structured.get("answer", "")),
                intent=Metadata.model_validate(
                    {"task_type": request.task_type.value if request.task_type else "demo"}
                ),
                referenced_event_ids=[],
                visibility=VisibilityScope.runtime,
            )
        error = None
        if status == AgentExecutionStatus.failed:
            error = AIError(
                code=_error_code(state.get("error_code")),
                message=state.get("error_message") or "The agent execution failed.",
                retryable=state.get("error_retryable", False),
                run_id=request.run_id,
            )
        return AgentResponse(
            run_id=request.run_id,
            schema_version=request.schema_version,
            output=output,
            safety_flags=[],
            usage=self._usage(state),
            request_id=request.request_id,
            trace_id=trace_id,
            status=status,
            structured_output=Metadata.model_validate(structured) if structured else None,
            output_schema_key=state.get("output_schema_key"),
            output_schema_version=state.get("output_schema_version"),
            validation_errors=[
                AgentOutputValidationIssue.model_validate(item)
                for item in state.get("validation_errors", [])
            ],
            tool_calls=[AgentToolCall.model_validate(item) for item in state.get("tool_calls", [])],
            steps=[AgentTraceStep.model_validate(item) for item in state.get("steps", [])],
            error=error,
        )

    def _controlled_response(
        self,
        request: AgentRequest,
        *,
        trace_id: str,
        status: AgentExecutionStatus,
        error_code: str | None,
        message: str,
        steps: list[dict[str, Any]],
        started: float,
    ) -> AgentResponse:
        updated_steps = list(steps)
        updated_steps.append(
            trace_step(
                step_index=len(updated_steps),
                step_type=StepType.model,
                name="agent_runtime",
                status=(
                    Status.timeout if status == AgentExecutionStatus.timeout else Status.cancelled
                ),
                duration_ms=_elapsed_ms(started),
                error_code=error_code,
            ).model_dump(mode="json")
        )
        error = (
            AIError(
                code=_error_code(error_code),
                message=message,
                retryable=status == AgentExecutionStatus.timeout,
                run_id=request.run_id,
            )
            if error_code
            else None
        )
        return AgentResponse(
            run_id=request.run_id,
            schema_version=request.schema_version,
            output=None,
            safety_flags=[],
            usage=self._usage({}),
            request_id=request.request_id,
            trace_id=trace_id,
            status=status,
            structured_output=None,
            output_schema_key=request.output_schema_key,
            output_schema_version=request.output_schema_version,
            validation_errors=[],
            tool_calls=[],
            steps=[AgentTraceStep.model_validate(item) for item in updated_steps],
            error=error,
        )

    def _failure_response(
        self,
        request: AgentRequest,
        *,
        trace_id: str,
        error_code: str,
        error_message: str,
        retryable: bool,
        steps: list[dict[str, Any]],
        started: float,
    ) -> AgentResponse:
        step = trace_step(
            step_index=len(steps),
            step_type=StepType.validate,
            name="agent_runtime",
            status=Status.failed,
            duration_ms=_elapsed_ms(started),
            error_code=error_code,
        )
        return AgentResponse(
            run_id=request.run_id,
            schema_version=request.schema_version,
            output=None,
            safety_flags=[],
            usage=self._usage({}),
            request_id=request.request_id,
            trace_id=trace_id,
            status=AgentExecutionStatus.failed,
            structured_output=None,
            output_schema_key=request.output_schema_key,
            output_schema_version=request.output_schema_version,
            validation_errors=[],
            tool_calls=[],
            steps=[*[AgentTraceStep.model_validate(item) for item in steps], step],
            error=AIError(
                code=_error_code(error_code),
                message=error_message,
                retryable=retryable,
                run_id=request.run_id,
            ),
        )

    @staticmethod
    def _usage(state: AgentGraphState) -> AgentUsage:
        usage = state.get("usage", {})
        return AgentUsage(
            input_tokens=int(usage.get("input_tokens", 0)),
            output_tokens=int(usage.get("output_tokens", 0)),
            cost_usd=usage.get("cost_usd"),
        )


def _elapsed_ms(started: float) -> int:
    return max(round((time.monotonic() - started) * 1000), 0)


def _merge_usage(existing: dict[str, Any], input_tokens: int, output_tokens: int) -> dict[str, Any]:
    return {
        "input_tokens": int(existing.get("input_tokens", 0)) + input_tokens,
        "output_tokens": int(existing.get("output_tokens", 0)) + output_tokens,
        "cost_usd": existing.get("cost_usd"),
    }


def _error_code(value: str | None) -> BusinessErrorCode:
    if value is None:
        return BusinessErrorCode.workflow_failed
    try:
        return BusinessErrorCode(value)
    except ValueError:
        return BusinessErrorCode.workflow_failed
