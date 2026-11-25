export interface ApiErrorEnvelope {
  code: string;
  message: string;
  details?: unknown;
  retryable: boolean;
  requestId?: string | undefined;
  traceId?: string | undefined;
}

export class AppError extends Error {
  public readonly code: string;
  public readonly details: unknown;
  public readonly retryable: boolean;
  public readonly requestId: string | undefined;
  public readonly traceId: string | undefined;

  public constructor(envelope: ApiErrorEnvelope, options?: ErrorOptions) {
    super(envelope.message, options);
    this.name = "AppError";
    this.code = envelope.code;
    this.details = envelope.details;
    this.retryable = envelope.retryable;
    this.requestId = envelope.requestId;
    this.traceId = envelope.traceId;
  }
}

export function toAppError(error: unknown): AppError {
  if (error instanceof AppError) {
    return error;
  }

  if (error instanceof Error) {
    return new AppError(
      {
        code: "CLIENT_ERROR",
        message: error.message,
        retryable: false,
      },
      { cause: error },
    );
  }

  return new AppError({
    code: "UNKNOWN_ERROR",
    message: "An unknown client error occurred.",
    retryable: false,
    details: error,
  });
}
