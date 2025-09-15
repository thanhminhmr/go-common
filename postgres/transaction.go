package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thanhminhmr/go-common/exception"
)

type Transaction interface {
	Connection

	// Finalize safely concludes a database transaction by either committing or
	// rolling back. It rolls back the transaction if any of the following conditions
	// are met: the input error is not null, a panic occurred earlier in execution,
	// the provided context has an error (canceled or expired), or the commit
	// operation itself fails. This ensures reliable and consistent transaction
	// handling.
	Finalize(ctx context.Context, errorResult *error)
}

type _transaction struct {
	_connection[pgx.Tx]
}

func (t _transaction) Finalize(ctx context.Context, errorResult *error) {
	if errorResult == nil {
		panic("BUG: errorResult is nil")
	}
	var recovered any
	var errorChain exception.Exception
	// check for commit condition and try to commit
	if *errorResult != nil {
		// transaction rollback on error
	} else if recovered = recover(); recovered != nil {
		// transaction rollback on panic without changing anything
	} else if err := ctx.Err(); err != nil {
		errorChain = exception.String("transaction rollback on context error").AddCause(err)
	} else if err := t.pgx.Commit(ctx); err != nil {
		errorChain = exception.String("transaction rollback on commit error").AddCause(err)
	} else {
		return
	}
	// either commit condition failed or commit failed, try rolling back
	if err := t.pgx.Rollback(ctx); err != nil && recovered == nil {
		if errorChain == nil {
			// only wrap the error if needed
			var ok bool
			if errorChain, ok = (*errorResult).(exception.Exception); !ok {
				errorChain = exception.String("transaction rollback on error").AddCause(*errorResult)
			}
		}
		errorChain = errorChain.AddSuppressed(exception.String("transaction rollback failed").AddCause(err))
	}
	// if recovered from panic, re-panic as it
	if recovered != nil {
		panic(recovered)
	}
	// if error got wrapped, return the wrapped error
	if errorChain != nil {
		*errorResult = errorChain
	}
}
