package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
)

const (
	CodeDependencyUnavailable = "DEPENDENCY_UNAVAILABLE"
	CodeDeadlineExceeded      = "DEADLINE_EXCEEDED"
	CodeRedisError            = "REDIS_ERROR"
)

type Error struct {
	Code      string
	Message   string
	Retryable bool
	Cause     error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return err
	}

	var redisErr *Error
	if errors.As(err, &redisErr) {
		return err
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{
			Code:      CodeDeadlineExceeded,
			Message:   "The Redis operation exceeded its deadline.",
			Retryable: true,
			Cause:     err,
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return &Error{
			Code:      CodeDependencyUnavailable,
			Message:   "Redis is temporarily unavailable.",
			Retryable: true,
			Cause:     err,
		}
	}

	return &Error{
		Code:    CodeRedisError,
		Message: "The Redis operation failed.",
		Cause:   err,
	}
}
