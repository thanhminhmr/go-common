package postgres

import (
	"context"
	"sync/atomic"

	"github.com/thanhminhmr/go-exception"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Connection interface {
	// Begin starts a transaction.
	Begin(ctx context.Context) (Transaction, error)

	// Batch creates a batch of commands.
	Batch(ctx context.Context) Batch

	// Exec execute the command.
	Exec(ctx context.Context, sql string, args ...any) (CommandTag, error)

	// Query scan the result rows by calling the collector repeatedly.
	Query(ctx context.Context, collector RowCollector, sql string, args ...any) (CommandTag, error)

	// QueryRow expects the result is exactly one row.
	QueryRow(ctx context.Context, sql string, args ...any) (RowScanner, error)

	// internalSendBatch internal function to send a batch
	internalSendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults

	// internalCopyFrom internal function to copy any data from source to database
	internalCopyFrom(
		ctx context.Context,
		tableName string,
		columnNames []string,
		source pgx.CopyFromSource,
	) (int64, error)
}

type _pgxConnection interface {
	Begin(context.Context) (pgx.Tx, error)
	CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error)
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type _connection[pgxConnection _pgxConnection] struct {
	pgx pgxConnection
}

func (c _connection[pgxConnection]) Begin(ctx context.Context) (Transaction, error) {
	if tx, err := c.pgx.Begin(ctx); err == nil {
		return &_transaction{
			_connection: _connection[pgx.Tx]{
				pgx: tx,
			},
		}, nil
	} else {
		return nil, exception.String("Begin transaction failed").AddCause(err)
	}
}

func (c _connection[pgxConnection]) Batch(ctx context.Context) Batch {
	return &_batch{
		ctx:        ctx,
		batch:      pgx.Batch{},
		connection: atomic.Value{},
	}
}

func (c _connection[pgxConnection]) Exec(ctx context.Context, sql string, args ...any) (CommandTag, error) {
	if tag, err := c.pgx.Exec(ctx, sql, args...); err != nil {
		return nil, exception.String("Exec failed").AddCause(err)
	} else {
		return &tag, nil
	}
}

func (c _connection[pgxConnection]) Query(
	ctx context.Context,
	collector RowCollector,
	sql string,
	args ...any,
) (tag CommandTag, errorResult error) {
	if collector == nil {
		panic("BUG: collector is nil")
	}
	if rows, err := c.pgx.Query(ctx, sql, args...); err != nil {
		return nil, exception.String("Query failed").AddCause(err)
	} else {
		var ex exception.Exception
		defer func() {
			rows.Close()
			if err := rows.Err(); err != nil {
				if ex != nil {
					ex = ex.AddSuppressed(err)
				} else {
					ex = exception.String("Query failed").AddCause(err)
				}
			} else if ex == nil {
				tag = rows.CommandTag()
			}
			errorResult = ex
		}()
		for rows.Next() {
			if err := collector(ctx, rows.Scan); err != nil {
				ex = exception.String("Query failed").AddCause(err)
				return
			}
		}
		return
	}
}

func (c _connection[pgxConnection]) QueryRow(ctx context.Context, sql string, args ...any) (RowScanner, error) {
	if rows, err := c.pgx.Query(ctx, sql, args...); err != nil {
		return nil, exception.String("QueryRow failed").AddCause(err)
	} else {
		if !rows.Next() {
			return nil, exception.String("QueryRow failed: no rows returned")
		}
		return func(destination ...any) (errorResult error) {
			var ex exception.Exception
			defer func() {
				rows.Close()
				if err := rows.Err(); err != nil {
					if ex != nil {
						ex = ex.AddSuppressed(err)
					} else {
						ex = exception.String("QueryRow failed").AddCause(err)
					}
				}
				errorResult = ex
			}()
			if err := rows.Scan(destination...); err != nil {
				ex = exception.String("QueryRow failed").AddCause(err)
				return
			}
			if rows.Next() {
				ex = exception.String("QueryRow failed: more than one row returned")
				return
			}
			return
		}, nil
	}
}

func (c _connection[pgxConnection]) internalSendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return c.pgx.SendBatch(ctx, batch)
}

func (c _connection[pgxConnection]) internalCopyFrom(
	ctx context.Context,
	tableName string,
	columnNames []string,
	source pgx.CopyFromSource,
) (int64, error) {
	return c.pgx.CopyFrom(ctx, pgx.Identifier{tableName}, columnNames, source)
}

// ========================================

func CopyAll[T any](
	connection Connection,
	ctx context.Context,
	tableName string,
	columnNames []string,
	input []T,
	outputMapper SliceMapper[T],
) (errorResult error) {
	// create transaction
	transaction, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer transaction.Finalize(ctx, &errorResult)
	// create source
	source := &fromSlice[T]{
		mapper: outputMapper,
		input:  input,
		output: make([]any, len(columnNames)),
		index:  -1,
	}
	// call raw copy and check the result
	if count, err := transaction.internalCopyFrom(ctx, tableName, columnNames, source); err != nil {
		return exception.String("CopyAll failed").AddCause(err)
	} else if count != int64(len(input)) {
		return exception.String("CopyAll failed: cannot copy all from source")
	}
	return nil
}

func CopyAny[T any](
	connection Connection,
	ctx context.Context,
	tableName string,
	columnNames []string,
	input []T,
	outputMapper SliceMapper[T],
) (int64, error) {
	source := &fromSlice[T]{
		mapper: outputMapper,
		input:  input,
		output: make([]any, len(columnNames)),
		index:  -1,
	}
	count, err := connection.internalCopyFrom(ctx, tableName, columnNames, source)
	if err != nil {
		return count, exception.String("CopyAny failed").AddCause(err)
	}
	return count, nil
}

type SliceMapper[T any] func(output []any, input T)

type fromSlice[T any] struct {
	mapper SliceMapper[T]
	input  []T
	output []any
	index  int
}

func (copy *fromSlice[T]) Next() bool {
	copy.index++
	return copy.index < len(copy.input)
}

func (copy *fromSlice[T]) Values() ([]any, error) {
	clear(copy.output)
	copy.mapper(copy.output, copy.input[copy.index])
	return copy.output, nil
}

func (copy *fromSlice[T]) Err() error {
	return nil
}
