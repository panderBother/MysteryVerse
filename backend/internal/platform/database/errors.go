package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

const (
	CodeConflict              = "RESOURCE_CONFLICT"
	CodeNotFound              = "NOT_FOUND"
	CodeReferenceNotFound     = "REFERENCE_NOT_FOUND"
	CodeResourceInUse         = "RESOURCE_IN_USE"
	CodeTransactionConflict   = "TRANSACTION_CONFLICT"
	CodeDeadlineExceeded      = "DEADLINE_EXCEEDED"
	CodeDependencyUnavailable = "DEPENDENCY_UNAVAILABLE"
	CodeDatabaseError         = "DATABASE_ERROR"
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

func MapError(err error) error {
	if err == nil {
		return nil
	}

	var storageErr *Error
	if errors.As(err, &storageErr) {
		return err
	}

	if errors.Is(err, sql.ErrNoRows) {
		return &Error{
			Code:    CodeNotFound,
			Message: "The requested record does not exist.",
			Cause:   err,
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{
			Code:      CodeDeadlineExceeded,
			Message:   "The database operation exceeded its deadline.",
			Retryable: true,
			Cause:     err,
		}
	}

	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mapMySQLError(mysqlErr)
	}

	var netErr net.Error
	if errors.As(err, &netErr) || errors.Is(err, mysqlDriver.ErrInvalidConn) {
		return &Error{
			Code:      CodeDependencyUnavailable,
			Message:   "The database is temporarily unavailable.",
			Retryable: true,
			Cause:     err,
		}
	}

	return &Error{
		Code:    CodeDatabaseError,
		Message: "The database operation failed.",
		Cause:   err,
	}
}

func mapMySQLError(err *mysqlDriver.MySQLError) error {
	switch err.Number {
	case 1062:
		return &Error{
			Code:    CodeConflict,
			Message: "The operation conflicts with an existing record.",
			Cause:   err,
		}
	case 1213:
		return &Error{
			Code:      CodeTransactionConflict,
			Message:   "The transaction was rolled back due to a deadlock.",
			Retryable: true,
			Cause:     err,
		}
	case 1205:
		return &Error{
			Code:      CodeTransactionConflict,
			Message:   "The database lock wait timed out.",
			Retryable: true,
			Cause:     err,
		}
	case 1451:
		return &Error{
			Code:    CodeResourceInUse,
			Message: "The record is still referenced by another resource.",
			Cause:   err,
		}
	case 1452:
		return &Error{
			Code:    CodeReferenceNotFound,
			Message: "A referenced record does not exist.",
			Cause:   err,
		}
	case 1040, 1045, 1129, 2002, 2003, 2006, 2013:
		return &Error{
			Code:      CodeDependencyUnavailable,
			Message:   "The database is temporarily unavailable.",
			Retryable: true,
			Cause:     err,
		}
	default:
		return &Error{
			Code:    CodeDatabaseError,
			Message: "The database operation failed.",
			Cause:   err,
		}
	}
}
