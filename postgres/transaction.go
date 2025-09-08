package postgres

import (
	"context"

	"github.com/thanhminhmr/go-common/errors"
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

type _pgxTransaction interface {
	_pgxConnection

	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type _transaction[pgxTransaction _pgxTransaction] struct {
	_connection[pgxTransaction]
}

func (t _transaction[pgxTransaction]) Finalize(ctx context.Context, errorResult *error) {
	if errorResult == nil {
		panic("BUG: errorResult is nil")
	}
	var recovered any
	var errorChain errors.Error
	// check for commit condition and try to commit
	if *errorResult != nil {
		// transaction rollback on error
	} else if recovered = recover(); recovered != nil {
		// transaction rollback on panic without changing anything
	} else if err := ctx.Err(); err != nil {
		errorChain = errors.String("transaction rollback on context error").AddCause(err)
	} else if err := t.pgx.Commit(ctx); err != nil {
		errorChain = errors.String("transaction rollback on commit error").AddCause(err)
	} else {
		return
	}
	// either commit condition failed or commit failed, try rolling back
	if err := t.pgx.Rollback(ctx); err != nil && recovered == nil {
		if errorChain == nil {
			// only wrap the error if needed
			var ok bool
			if errorChain, ok = (*errorResult).(errors.Error); !ok {
				errorChain = errors.String("transaction rollback on error").AddCause(*errorResult)
			}
		}
		errorChain = errorChain.AddSuppressed(errors.String("transaction rollback failed").AddCause(err))
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
