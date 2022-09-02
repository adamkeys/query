package query

import (
	"context"
	"database/sql"
)

// Options identifies optional parameters that may be used when performing queries. Default struct values
// signify default behaviour.
type Options struct {
	Namer  Namer
	Logger func(query string, args []any)
}

// Name with returns the defined Namer option or nil.
func (o *Options) NameWith() Namer {
	if o == nil {
		return nil
	}
	return o.Namer
}

// Log calls the [Options.Logger] function if defined in the [Options].
// query functions.
func (o *Options) Log(query string, args []any) {
	if o == nil || o.Logger == nil {
		return
	}
	o.Logger(query, args)
}

// DB identifies a query database handle that wraps [sql.DB] for use as a transaction in query operations. Its use
// is not required to use query, but adds additional features not available when using [sql.DB] or [sql.Tx] directly.
type DB struct {
	*sql.DB
	*Options
}

// Open opens a new database connection using the supplied options.
func Open(driverName, dataSource string, options *Options) (*DB, error) {
	db, err := sql.Open(driverName, dataSource)
	return &DB{DB: db, Options: options}, err
}

// Begin returns a new transaction with options using [sql.Begin].
func (d *DB) Begin() (*Tx, error) {
	return d.BeginTx(context.Background(), nil)
}

// BeginTx returns a new transaction with options using [sql.BeginTx].
func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := d.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, Options: d.Options}, nil
}

// Tx identifies a transaction that also provides query options.
type Tx struct {
	*sql.Tx
	*Options
}
