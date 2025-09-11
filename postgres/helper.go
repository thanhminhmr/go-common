package postgres

import (
	"context"
)

type CommandTag interface {
	String() string
	RowsAffected() int64
	Insert() bool
	Update() bool
	Delete() bool
	Select() bool
}

type CommandTagHandler func(ctx context.Context, tag CommandTag) error

type RowScanner func(destination ...any) error

type RowCollector func(ctx context.Context, row RowScanner) error
