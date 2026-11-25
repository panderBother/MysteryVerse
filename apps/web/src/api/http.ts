import { AppError } from "../lib/errors";

type HttpMethod = "DELETE" | "GET" | "PATCH" | "POST" | "PUT";

interface RequestOptions {
  body?: unknown;
  headers?: Record<string, string>;
  method?: HttpMethod;
  signal?: AbortSignal;
  timeoutMs?: number;
}

interface SuccessEnvelope<T> {
  data: T;
  meta?: {
    request_id?: string;
    server_time?: string;
    api_version?: string;
  };
}

interface ServerErrorEnvelope {
  error: {
    code: string;
    message: string;
    details?: unknown;
    retryable?: boolean;
    request_id?: string;
    trace_id?: string;
  };
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

function isServerErrorEnvelope(value: unknown): value is ServerErrorEnvelope {
  if (typeof value !== "object" || value === null || !("error" in value)) {
    return false;
  }

  const errorValue = value.error;
  return (
    typeof errorValue === "object" &&
    errorValue !== null &&
    "code" in errorValue &&
    "message" in errorValue
  );
}

async function parseJson(response: Response): Promise<unknown> {
  const text = await response.text();
  return text.length > 0 ? JSON.parse(text) : null;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const timeoutController = new AbortController();
  const timeoutId = window.setTimeout(() => timeoutController.abort(), options.timeoutMs ?? 10_000);
  const signal = options.signal ?? timeoutController.signal;

  try {
    const requestInit: RequestInit = {
      method: options.method ?? "GET",
      headers: {
        Accept: "application/json",
        ...(options.body === undefined ? {} : { "Content-Type": "application/json" }),
        ...options.headers,
      },
      credentials: "include",
      signal,
    };

    if (options.body !== undefined) {
      requestInit.body = JSON.stringify(options.body);
    }

    const response = await fetch(`${API_BASE_URL}${path}`, requestInit);

    const payload = await parseJson(response);

    if (!response.ok) {
      if (isServerErrorEnvelope(payload)) {
        throw new AppError({
          code: payload.error.code,
          message: payload.error.message,
          details: payload.error.details,
          retryable: payload.error.retryable ?? false,
          requestId: payload.error.request_id,
          traceId: payload.error.trace_id,
        });
      }

      throw new AppError({
        code: `HTTP_${response.status}`,
        message: response.statusText || "Request failed.",
        retryable: response.status >= 500,
      });
    }

    return (payload as SuccessEnvelope<T>).data;
  } catch (error) {
    if (error instanceof AppError) {
      throw error;
    }

    throw new AppError(
      {
        code:
          error instanceof DOMException && error.name === "AbortError"
            ? "REQUEST_TIMEOUT"
            : "NETWORK_ERROR",
        message: "The service could not be reached.",
        retryable: true,
      },
      { cause: error },
    );
  } finally {
    window.clearTimeout(timeoutId);
  }
}

export const http = {
  get: <T>(path: string, options?: Omit<RequestOptions, "body" | "method">) =>
    request<T>(path, { ...options, method: "GET" }),
  post: <T>(path: string, body: unknown, options?: Omit<RequestOptions, "body" | "method">) =>
    request<T>(path, { ...options, method: "POST", body }),
};
