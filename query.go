// Package query embraces the features of Go to query SQL databases.
package query

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

// Transaction identifies a queriable database handle. This will most likley be a [sql.DB] or [sql.Tx].
type Transaction interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Transform identifies a transformation function used to transform results from a row to an application
// model structure. Source is the structure prepared from the database row and Destination is the
// transformed output.
type Transform[Source, Destination any] func(Source) Destination

// The Identity function is a Transform function that returns the original value. This function can be used as the
// transform when the caller wishes to receive the source value.
func Identity[Source any](src Source) Source { return src }

// All returns a collection of results from the database which are queried according to the Source type. Each row of
// the database is transformed to the desired Destination type using the supplied transform function. The caller may
// use the [Identity] function if the caller wishes for Source and Destination to be equal.
//
// The Source type uses struct tags to specify the properties of the database query. If a struct tag is not provided,
// the name will be inferred from the field name. Example:
//
//	type users struct {
//		ID   int            `q:"id"`
//		Name sql.NullString `q:"name"`
//	}
//	// Query: SELECT id, name FROM users
//	results, _ := query.All(context.Background(), db, query.Identity[users])
//
// Additional properties may be added to the query by composing query operator structs provided by this package. These
// include [Conditions], [GroupBy], and [OrderBy]. Example:
//
//	type users struct {
//		query.Conditions `q:"id = ?"`
//		query.GroupBy    `q:"name"`
//		query.OrderBy    `q:"id"`
//
//		ID   int            `q:"id"`
//		Name sql.NullString `q:"name"`
//	}
//
// The table name is inferred from the struct name but may be overwritten by composing the [Table] struct. Example:
//
//	type users struct {
//		query.Table `q:"user"`
//
//		Name sql.NullString `q:"name"`
//	}
//
// Joins may be specified by including a child structure. A struct tag defining the join relationship is required.
// Example:
//
//	type users struct {
//		Name      string
//		Addresses struct {
//			City string
//		} `q:"users.address_id = addresses.id"`
//	}
//
// The caller should note that the Source value is reused on each row iteration and should take care to ensure that
// values are copied in the transform function.
//
// An error will be returned if any of the [Transaction] operations fail.
func All[Source, Destination any](ctx context.Context, tx Transaction, transform Transform[Source, Destination], args ...any) ([]Destination, error) {
	var src Source
	query, bindings := prepare(&src)

	rows, err := tx.QueryContext(ctx, query.SQL(), args...)
	if err != nil {
		return nil, fmt.Errorf("query: %v", err)
	}
	defer rows.Close()

	var results []Destination
	for rows.Next() {
		if err := rows.Scan(bindings...); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		results = append(results, transform(src))
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close: %w", err)
	}

	return results, nil
}

// One is like [All] but returns only the first result of the query. An error will be returned if any of the [Transaction] operations fail.
func One[Source, Destination any](ctx context.Context, tx Transaction, transform Transform[Source, Destination], args ...any) (Destination, error) {
	var src Source
	query, bindings := prepare(&src)

	err := tx.QueryRowContext(ctx, query.SQL(), args...).Scan(bindings...)
	return transform(src), err
}

// prepare returns the prepared SQL query and destination bindings suitable for use by [sql.Rows.Scan].
func prepare(src any) (statement, []any) {
	typ := reflect.TypeOf(src).Elem()
	val := reflect.ValueOf(src).Elem()
	bindings := make([]any, 0, typ.NumField())
	stmt := statement{
		columns: make([]column, 0, cap(bindings)),
		table:   typ.Name(),
	}
	for i := 0; i < cap(stmt.columns); i++ {
		fld := typ.Field(i)
		tag := fld.Tag.Get("q")
		switch {
		case fld.Type == reflect.TypeOf(Table{}):
			stmt.table = tag
		case fld.Type == reflect.TypeOf(Conditions{}):
			stmt.conditions = tag
		case fld.Type == reflect.TypeOf(OrderBy{}):
			stmt.order = tag
		case fld.Type == reflect.TypeOf(GroupBy{}):
			stmt.group = tag
		case fld.Type == reflect.TypeOf(LeftJoin{}):
			stmt.join = joinLeft
		case fld.Type == reflect.TypeOf(Limit{}):
			stmt.limit = tag
		case fld.Type == reflect.TypeOf(Offset{}):
			stmt.offset = tag
		case fld.Type.Name() == "":
			if tag == "" {
				panic(fmt.Errorf("%T.%s requires a struct tag describing the join conditions", src, fld.Name))
			}
			s, b := prepare(val.Field(i).Addr().Interface())
			if s.table == "" {
				s.table = fld.Name
			}
			if s.join == joinNone {
				s.join = joinInner
			}
			s.on = tag
			stmt.joins = append(stmt.joins, s)
			bindings = append(bindings, b...)
		default:
			col := column{name: tag}
			if tag == "" {
				col = column{name: fld.Name, useTable: true}
			}
			stmt.columns = append(stmt.columns, col)
			bindings = append(bindings, val.Field(i).Addr().Interface())
		}
	}

	return stmt, bindings
}
