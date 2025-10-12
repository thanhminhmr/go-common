package postgres

import (
	"context"
	"sync/atomic"

	"github.com/thanhminhmr/go-exception"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Batch interface {
	Exec(handler CommandTagHandler, sql string, args ...any)
	Query(collector RowCollector, handler CommandTagHandler, sql string, args ...any)
	QueryRow(collector RowCollector, sql string, args ...any)
	Send() error
}

type _batch struct {
	ctx        context.Context
	batch      pgx.Batch
	connection atomic.Value
}

func (b *_batch) Exec(handler CommandTagHandler, sql string, args ...any) {
	query := b.batch.Queue(sql, args...)
	if handler != nil {
		query.Exec(func(tag pgconn.CommandTag) error {
			if err := handler(b.ctx, tag); err != nil {
				return exception.String("Exec in batch failed").AddCause(err)
			}
			return nil
		})
	}
}

func (b *_batch) Query(collector RowCollector, handler CommandTagHandler, sql string, args ...any) {
	if collector == nil {
		panic("BUG: collector is nil")
	}
	b.batch.Queue(sql, args...).Query(func(rows pgx.Rows) (errorResult error) {
		var ex exception.Exception
		defer func() {
			rows.Close()
			if err := rows.Err(); err != nil {
				if ex != nil {
					ex = ex.AddSuppressed(err)
				} else {
					ex = exception.String("Query in batch failed").AddCause(err)
				}
			} else if ex == nil && handler != nil {
				if err := handler(b.ctx, rows.CommandTag()); err != nil {
					ex = exception.String("Query in batch failed").AddCause(err)
				}
			}
			errorResult = ex
		}()
		for rows.Next() {
			if err := collector(b.ctx, rows.Scan); err != nil {
				ex = exception.String("Query in batch failed").AddCause(err)
				return
			}
		}
		return
	})
}

func (b *_batch) QueryRow(collector RowCollector, sql string, args ...any) {
	if collector == nil {
		panic("BUG: collector is nil")
	}
	b.batch.Queue(sql, args...).Query(func(rows pgx.Rows) (errorResult error) {
		var ex exception.Exception
		defer func() {
			rows.Close()
			if err := rows.Err(); err != nil {
				if ex != nil {
					ex = ex.AddSuppressed(err)
				} else {
					ex = exception.String("QueryRow in batch failed").AddCause(err)
				}
			}
			errorResult = ex
		}()
		if !rows.Next() {
			ex = exception.String("QueryRow in batch failed: no rows returned")
			return
		}
		if err := collector(b.ctx, rows.Scan); err != nil {
			ex = exception.String("QueryRow in batch failed").AddCause(err)
			return
		}
		if rows.Next() {
			ex = exception.String("QueryRow in batch failed: more than one row returned")
			return
		}
		return
	})
}

func (b *_batch) Send() error {
	if connection, _ := b.connection.Swap(nil).(Database); connection == nil {
		panic("BUG: batch already sent")
	} else if err := connection.internalSendBatch(b.ctx, &b.batch).Close(); err != nil {
		return exception.String("Send batch failed").AddCause(err)
	}
	return nil
}
