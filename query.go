// Package query embraces the features of Go to query SQL databases.
package query

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"reflect"
)

// Transaction identifies a queryable database handle. This will most likely be a [sql.DB] or [sql.Tx].
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
	var results []Source
	query, bindings, complete := prepareSet(&results)

	rows, err := tx.QueryContext(ctx, query.SQL(), args...)
	if err != nil {
		return nil, fmt.Errorf("query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(bindings...); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		complete()
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close: %w", err)
	}

	transformed := make([]Destination, len(results))
	for i, result := range results {
		transformed[i] = transform(result)
	}
	return transformed, nil
}

// One is like [All] but returns only the first result of the query. An error will be returned if any of
// the [Transaction] operations fail.
func One[Source, Destination any](ctx context.Context, tx Transaction, transform Transform[Source, Destination], args ...any) (Destination, error) {
	var src Source

	// When the query contains a many relationship the call is passed to the [All] function to evaluate all of the
	// incoming rows to build up the necessary value hierarchy. The first result returned by [All] is passed
	// back to the caller.
	if hasMany(reflect.TypeOf(src)) {
		var dest Destination
		results, err := All(ctx, tx, transform, args...)
		if err != nil {
			return dest, err
		}
		if len(results) == 0 {
			return dest, sql.ErrNoRows
		}
		return results[0], nil
	}

	query, bindings, _ := prepare(reflect.ValueOf(&src))
	err := tx.QueryRowContext(ctx, query.SQL(), args...).Scan(bindings...)
	return transform(src), err
}

// prepareSet wraps prepareNestedSet to add the values to the top level slice rather than slices nested within
// stored values. This is intended to be the top level call when preparing a set of results.
func prepareSet(set any) (statement, []any, func()) {
	val := reflect.ValueOf(set)
	stmt, bindings, complete := prepareNestedSet(val)
	return stmt, bindings, func() {
		complete(val.Elem())
	}
}

// prepareNestedSet returns a prepared SQL statement, bindings suitable for use by [sql.Rows.Scan], and a completion
// function which is to be called after [sql.Rows.Scan] has been scanned into the bindings. The completion function
// adds the bound results to the passed in slice value.
func prepareNestedSet(set reflect.Value) (statement, []any, func(reflect.Value)) {
	val := set.Elem()
	row := reflect.New(val.Type().Elem())
	stmt, bindings, complete := prepare(row)

	var ident any
	stmt.columns = append([]column{{name: fmt.Sprintf("id AS ident%d", rand.Int31()), useTable: true}}, stmt.columns...)
	bindings = append([]any{&ident}, bindings...)
	visited := make(map[any]reflect.Value)
	return stmt, bindings, func(val reflect.Value) {
		if v, ok := visited[ident]; ok {
			complete(v)
			return
		}
		val.Set(reflect.Append(val, row.Elem()))
		added := val.Index(val.Len() - 1)
		visited[ident] = added
		complete(added)
	}
}

// prepare returns the prepared SQL query and destination bindings suitable for use by [sql.Rows.Scan].
func prepare(src reflect.Value) (statement, []any, func(reflect.Value)) {
	val := src.Elem()
	typ := val.Type()
	bindings := make([]any, 0, typ.NumField())
	stmt := statement{
		columns: make([]column, 0, cap(bindings)),
		table:   typ.Name(),
	}
	completion := func(reflect.Value) {}
	for i := 0; i < typ.NumField(); i++ {
		fld := typ.Field(i)
		tag := fld.Tag.Get("q")
		switch {
		case fld.Type == reflect.TypeOf(Table{}):
			stmt.table = tag
		case fld.Type == reflect.TypeOf(Conditions{}):
			stmt.conditions = append(stmt.conditions, tag)
		case fld.Type == reflect.TypeOf(OrderBy{}):
			stmt.order = append(stmt.order, tag)
		case fld.Type == reflect.TypeOf(GroupBy{}):
			stmt.group = append(stmt.group, tag)
		case fld.Type == reflect.TypeOf(LeftJoin{}):
			stmt.join = joinLeft
		case fld.Type == reflect.TypeOf(Limit{}):
			stmt.limit = tag
		case fld.Type == reflect.TypeOf(Offset{}):
			stmt.offset = tag
		case fld.Type.Kind() == reflect.Slice:
			s, b, f := prepareNestedSet(val.Field(i).Addr())
			if s.table == "" {
				s.table = fld.Name
			}
			if s.join == joinNone {
				s.join = joinInner
			}
			s.on = tag
			stmt.joins = append(stmt.joins, s)
			bindings = append(bindings, b...)
			idx := i
			completion = appendFn(completion, func(v reflect.Value) {
				f(v.Field(idx))
			})
		case fld.Type.Name() == "":
			if tag == "" {
				panic(fmt.Errorf("%T.%s requires a struct tag describing the join conditions", src, fld.Name))
			}
			s, b, f := prepare(val.Field(i).Addr())
			if s.table == "" {
				s.table = fld.Name
			}
			if s.join == joinNone {
				s.join = joinInner
			}
			s.on = tag
			stmt.joins = append(stmt.joins, s)
			bindings = append(bindings, b...)
			completion = appendFn(completion, f)
		default:
			if fld.Anonymous {
				s, b, f := prepare(val.Field(i).Addr())
				stmt.columns = append(s.columns, stmt.columns...)
				stmt.conditions = append(s.conditions, stmt.conditions...)
				stmt.group = append(s.group, stmt.group...)
				stmt.order = append(s.order, stmt.order...)
				stmt.joins = append(s.joins, stmt.joins...)
				bindings = append(bindings, b...)
				completion = appendFn(completion, f)
				continue
			}

			col := column{name: tag}
			if tag == "" {
				col = column{name: fld.Name, useTable: true}
			}
			stmt.columns = append(stmt.columns, col)
			bindings = append(bindings, val.Field(i).Addr().Interface())
		}
	}

	return stmt, bindings, completion
}

// appendFn returns a function that will call both the supplied head and tail functions. The supplied functions may
// be nil.
func appendFn(head, tail func(reflect.Value)) func(reflect.Value) {
	if head == nil {
		return tail
	}
	if tail == nil {
		return head
	}
	return func(v reflect.Value) {
		head(v)
		tail(v)
	}
}

// hasMany returns true if the input type produces a query that contains a many relationship.
func hasMany(src reflect.Type) bool {
	switch src.Kind() {
	case reflect.Ptr:
		return hasMany(src.Elem())
	case reflect.Slice:
		return true
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			if hasMany(src.Field(i).Type) {
				return true
			}
		}
	}
	return false
}
