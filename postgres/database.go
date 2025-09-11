package postgres

import "github.com/jackc/pgx/v5/pgxpool"

type Database interface {
	Connection

	close()
}

type _database struct {
	_connection[*pgxpool.Pool]
}

func (d _database) close() {
	d.pgx.Close()
}
