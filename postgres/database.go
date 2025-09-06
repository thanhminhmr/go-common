package postgres

type Database interface {
	Connection

	Close()
}

type _pgxDatabase interface {
	_pgxConnection

	Close()
}

type _database[pgxDatabase _pgxDatabase] struct {
	_connection[pgxDatabase]
}

func (d _database[pgxDatabase]) Close() {
	d.pgx.Close()
}
