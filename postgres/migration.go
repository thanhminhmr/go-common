package postgres

import (
	"context"
	"time"

	"github.com/thanhminhmr/go-common/errors"
)

type MigrationPlan []MigrationRecord

type MigrationRecord struct {
	Id  string
	Sql []string
}

// language=PostgreSQL
const migrationCreateTable = `
CREATE TABLE IF NOT EXISTS _migrations_ (
	id CHARACTER VARYING(31) COLLATE "C.utf8" NOT NULL,
	applied_at TIMESTAMP WITH TIME ZONE NOT NULL,
	CONSTRAINT _migrations_pk PRIMARY KEY (id)
)`

// language=PostgreSQL
const migrationSelectIds = `SELECT id FROM _migrations_`

// language=PostgreSQL
const migrationCreateRecord = `INSERT INTO _migrations_ (id, applied_at) VALUES ($1, $2)`

func (migrationPlan MigrationPlan) migrate(ctx context.Context, database Database) error {
	// create migration table
	if _, err := database.Exec(ctx, migrationCreateTable); err != nil {
		return err
	}
	// get previous migration records
	appliedIds := map[string]struct{}{}
	collector := func(ctx context.Context, scanner RowScanner) error {
		var appliedId string
		if err := scanner(ctx, &appliedId); err != nil {
			return err
		}
		appliedIds[appliedId] = struct{}{}
		return nil
	}
	if _, err := database.Query(ctx, collector, migrationSelectIds); err != nil {
		return err
	}
	// run migration plans
	for _, record := range migrationPlan {
		// check if migration is already existed
		if _, exists := appliedIds[record.Id]; exists {
			continue
		}
		// apply migration
		if err := record.migrate(ctx, database); err != nil {
			return err
		}
	}
	return nil
}

func (migrationRecord MigrationRecord) migrate(ctx context.Context, database Database) (errorResult error) {
	// create new transaction
	transaction, err := database.Begin(ctx)
	if err != nil {
		return err
	}
	defer transaction.Finalize(ctx, &errorResult)
	// run each query
	for _, sql := range migrationRecord.Sql {
		if _, err := transaction.Exec(ctx, sql); err != nil {
			return err
		}
	}
	// create migration record
	tag, err := transaction.Exec(ctx, migrationCreateRecord, migrationRecord.Id, time.Now())
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.String("Failed to create migration record")
	}
	// success
	return nil
}
