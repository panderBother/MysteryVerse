package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  string
		retryable bool
	}{
		{
			name:     "no rows",
			err:      sql.ErrNoRows,
			wantCode: CodeNotFound,
		},
		{
			name:     "duplicate",
			err:      &mysqlDriver.MySQLError{Number: 1062},
			wantCode: CodeConflict,
		},
		{
			name:      "deadlock",
			err:       &mysqlDriver.MySQLError{Number: 1213},
			wantCode:  CodeTransactionConflict,
			retryable: true,
		},
		{
			name:     "foreign key",
			err:      &mysqlDriver.MySQLError{Number: 1452},
			wantCode: CodeReferenceNotFound,
		},
		{
			name:      "unavailable",
			err:       &mysqlDriver.MySQLError{Number: 2003},
			wantCode:  CodeDependencyUnavailable,
			retryable: true,
		},
		{
			name:      "deadline",
			err:       context.DeadlineExceeded,
			wantCode:  CodeDeadlineExceeded,
			retryable: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapped := MapError(test.err)

			var storageErr *Error
			if !errors.As(mapped, &storageErr) {
				t.Fatalf("MapError() = %T, want *Error", mapped)
			}
			if storageErr.Code != test.wantCode {
				t.Fatalf("Code = %q, want %q", storageErr.Code, test.wantCode)
			}
			if storageErr.Retryable != test.retryable {
				t.Fatalf(
					"Retryable = %t, want %t",
					storageErr.Retryable,
					test.retryable,
				)
			}
		})
	}
}

func TestMapErrorPreservesMappedError(t *testing.T) {
	original := &Error{Code: CodeConflict, Message: "conflict"}
	if mapped := MapError(original); mapped != original {
		t.Fatal("MapError() changed an already mapped error")
	}
}
