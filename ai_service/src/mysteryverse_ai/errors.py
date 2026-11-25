from typing import Any

from fastapi import Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException as StarletteHTTPException


class AppError(Exception):
    def __init__(
        self,
        *,
        code: str,
        message: str,
        status_code: int,
        retryable: bool = False,
        details: Any = None,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.status_code = status_code
        self.retryable = retryable
        self.details = details


def error_payload(
    *,
    code: str,
    message: str,
    request_id: str,
    trace_id: str,
    retryable: bool,
    details: Any = None,
) -> dict[str, Any]:
    return {
        "error": {
            "code": code,
            "message": message,
            "details": details,
            "request_id": request_id,
            "trace_id": trace_id,
            "retryable": retryable,
        }
    }


async def app_error_handler(request: Request, exc: Exception) -> JSONResponse:
    if not isinstance(exc, AppError):
        return await unhandled_error_handler(request, exc)

    request_id = getattr(request.state, "request_id", "")
    trace_id = getattr(request.state, "trace_id", request_id)
    return JSONResponse(
        status_code=exc.status_code,
        content=error_payload(
            code=exc.code,
            message=exc.message,
            details=exc.details,
            request_id=request_id,
            trace_id=trace_id,
            retryable=exc.retryable,
        ),
    )


async def unhandled_error_handler(request: Request, _exc: Exception) -> JSONResponse:
    request_id = getattr(request.state, "request_id", "")
    trace_id = getattr(request.state, "trace_id", request_id)
    return JSONResponse(
        status_code=500,
        content=error_payload(
            code="INTERNAL_ERROR",
            message="An internal error occurred.",
            request_id=request_id,
            trace_id=trace_id,
            retryable=True,
        ),
    )


async def http_exception_handler(
    request: Request,
    exc: Exception,
) -> JSONResponse:
    if not isinstance(exc, StarletteHTTPException):
        return await unhandled_error_handler(request, exc)

    request_id = getattr(request.state, "request_id", "")
    trace_id = getattr(request.state, "trace_id", request_id)
    message = exc.detail if isinstance(exc.detail, str) else "The request could not be completed."
    return JSONResponse(
        status_code=exc.status_code,
        content=error_payload(
            code=f"HTTP_{exc.status_code}",
            message=message,
            request_id=request_id,
            trace_id=trace_id,
            retryable=exc.status_code >= 500,
        ),
    )


async def request_validation_error_handler(
    request: Request,
    exc: Exception,
) -> JSONResponse:
    if not isinstance(exc, RequestValidationError):
        return await unhandled_error_handler(request, exc)
    request_id = getattr(request.state, "request_id", "")
    trace_id = getattr(request.state, "trace_id", request_id)
    details = [
        {
            "location": ".".join(str(item) for item in error.get("loc", [])),
            "message": str(error.get("msg", "Invalid value")),
            "type": str(error.get("type", "validation_error")),
        }
        for error in exc.errors()
    ]
    return JSONResponse(
        status_code=422,
        content=error_payload(
            code="VALIDATION_ERROR",
            message="The request does not match the required schema.",
            details=details,
            request_id=request_id,
            trace_id=trace_id,
            retryable=False,
        ),
    )
