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
// The Source type uses struct tags to specify the properties of the database query. Each field in the struct should
// have a corresponding field tag which provides the database field name in the query. Example:
//
//	type users struct {
//		ID   int            `id`
//		Name sql.NullString `name`
//	}
//	// Query: SELECT id, name FROM users
//	results, _ := query.All(context.Background(), db, query.Identity[users])
//
// Additional properties may be added to the query by composing query operator structs provided by this package. These
// include [Conditions], [GroupBy], and [OrderBy]. Example:
//
//	type users struct {
//		query.Conditions `id = ?`
//		query.GroupBy    `name`
//		query.OrderBy    `id`
//
//		ID               int            `id`
//		Name             sql.NullString `name`
//	}
//
// The table name is inferred from the struct name but may be overwritten by composing the [Table] struct. Example:
//
//	type users struct {
//		query.Table      `user`
//
//		Name             sql.NullString `name`
//	}
//
// The caller should note that the Source value is reused on each row iteration and should take care to ensure that
// values are copied in the transform function.
//
// An error will be returned if any of the [Transaction] operations fail.
func All[Source, Destination any](ctx context.Context, tx Transaction, transform Transform[Source, Destination], args ...any) ([]Destination, error) {
	var src Source
	query, bindings := prepare(&src)

	rows, err := tx.QueryContext(ctx, query, args...)
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

	err := tx.QueryRowContext(ctx, query, args...).Scan(bindings...)
	return transform(src), err
}

// prepare returns the prepared SQL query and destination bindings suitable for use by [sql.Rows.Scan].
func prepare[Source any](src *Source) (string, []any) {
	t := reflect.TypeOf(src).Elem()
	v := reflect.ValueOf(src).Elem()

	bindings := make([]any, 0, t.NumField())
	stmt := statement{
		columns: make([]string, 0, cap(bindings)),
		table:   t.Name(),
	}
	for i := 0; i < cap(stmt.columns); i++ {
		switch t.Field(i).Type {
		case reflect.TypeOf(Table{}):
			stmt.table = string(t.Field(i).Tag)
		case reflect.TypeOf(Conditions{}):
			stmt.conditions = string(t.Field(i).Tag)
		case reflect.TypeOf(OrderBy{}):
			stmt.order = string(t.Field(i).Tag)
		case reflect.TypeOf(GroupBy{}):
			stmt.group = string(t.Field(i).Tag)
		default:
			stmt.columns = append(stmt.columns, string(t.Field(i).Tag))
			bindings = append(bindings, v.Field(i).Addr().Interface())
		}
	}

	return stmt.SQL(), bindings
}
