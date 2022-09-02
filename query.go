// Package query embraces the features of Go to query SQL databases.
package query

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
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
// Joins may also be defined in a "has many" fashion. These are defined like standard joins but as a slice type.
// Note that the query is modified to include primary key columns for relating records together. This may impact
// some queries (e.g. GROUP BY) in potentially unexpected ways. The primary key must be named "id". Example:
//
//	type users struct {
//		Name      string
//		Addresses []struct {
//			City string
//		} `q:"users.address_id = addresses.id"`
//	}
//
// The caller should note that the Source value is reused on each row iteration and should take care to ensure that
// values are copied in the transform function. Slices excepted.
//
// An error will be returned if any of the [Transaction] operations fail.
func All[Source, Destination any](ctx context.Context, tx Transaction, transform Transform[Source, Destination], args ...any) ([]Destination, error) {
	var results []Source
	query, bindings, complete := prepareSet(nameWith(tx), &results)

	log(tx, query, args)
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
	if err := rows.Err(); err != nil {
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

	query, bindings, _ := prepare(nameWith(tx), reflect.ValueOf(&src), 0)
	log(tx, query, args)
	err := tx.QueryRowContext(ctx, query.SQL(), args...).Scan(bindings...)
	return transform(src), err
}

// log calls the Log method on the [Transaction], if implemented, with the query and arguments used in the
// calling query operation. This method is provided when a database is opened using [Open].
func log(tx Transaction, query statement, args []any) {
	txl, ok := tx.(interface{ Log(string, []any) })
	if !ok {
		return
	}
	txl.Log(query.SQL(), args)
}

// nameWith returns the namer associated with the [Transaction], if implemented, and otherwise returns the default
// namer.
func nameWith(tx Transaction) Namer {
	txn, ok := tx.(interface{ NameWith() Namer })
	if !ok {
		return defaultNamer
	}

	namer := txn.NameWith()
	if namer == nil {
		return defaultNamer
	}
	return namer
}

// prepareSet wraps prepareNestedSet to add the values to the top level slice rather than slices nested within
// stored values. This is intended to be the top level call when preparing a set of results.
func prepareSet(namer Namer, set any) (statement, []any, func()) {
	val := reflect.ValueOf(set)
	elem := val.Elem()
	if !hasMany(elem.Type().Elem()) {
		row := reflect.New(elem.Type().Elem())
		stmt, bindings, _ := prepare(namer, row, 0)
		return stmt, bindings, func() {
			elem.Set(reflect.Append(elem, row.Elem()))
		}
	}

	stmt, bindings, complete := prepareNestedSet(namer, val)
	return stmt, bindings, func() {
		complete(nil, elem)
	}
}

// prepareNestedSet returns a prepared SQL statement, bindings suitable for use by [sql.Rows.Scan], and a completion
// function which is to be called after [sql.Rows.Scan] has been scanned into the bindings. The completion function
// adds the bound results to the passed in slice value.
func prepareNestedSet(namer Namer, set reflect.Value) (statement, []any, func(*rowRef, reflect.Value)) {
	val := set.Elem()
	row := reflect.New(val.Type().Elem())
	stmt, bindings, complete := prepare(namer, row, 0)

	var ident any
	stmt.columns = append([]column{{
		name:     namer.Ident(val.Type()),
		as:       fmt.Sprintf("ident%d", rand.Int31()),
		useTable: true,
	}}, stmt.columns...)
	bindings = append([]any{&ident}, bindings...)

	visited := make(map[string]reflect.Value)
	return stmt, bindings, func(parent *rowRef, val reflect.Value) {
		_ = set
		if ident == nil {
			return
		}

		ref := rowRef{parent: parent, ident: asString(ident)}
		hash := ref.Hash()
		if v, ok := visited[hash]; ok {
			complete(&ref, v)
			return
		}
		val.Set(reflect.Append(val, row.Elem()))
		added := val.Index(val.Len() - 1)
		visited[hash] = added
		complete(&ref, added)
	}
}

// prepare returns the prepared SQL query and destination bindings suitable for use by [sql.Rows.Scan].
func prepare(namer Namer, src reflect.Value, depth int) (statement, []any, func(*rowRef, reflect.Value)) {
	val := src.Elem()
	typ := val.Type()
	bindings := make([]any, 0, typ.NumField())
	stmt := statement{
		columns: make([]column, 0, cap(bindings)),
	}
	completion := func(*rowRef, reflect.Value) {}
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
			if tag == "" {
				panic(fmt.Errorf("%T.%s requires a struct tag describing the join conditions", src, fld.Name))
			}
			s, b, f := prepareNestedSet(namer, val.Field(i).Addr())
			if s.table == "" {
				s.table = namer.Table(fieldInfo{fld})
			}
			if s.join == joinNone {
				s.join = joinInner
			}
			s.on = tag
			stmt.joins = append(stmt.joins, s)
			bindings = append(bindings, b...)
			idx := i
			completion = appendFn(completion, func(r *rowRef, v reflect.Value) {
				f(r, v.Field(idx))
			})
		case fld.Type.Name() == "":
			if tag == "" {
				panic(fmt.Errorf("%T.%s requires a struct tag describing the join conditions", src, fld.Name))
			}
			s, b, f := prepare(namer, val.Field(i).Addr(), depth+1)
			if s.table == "" {
				s.table = namer.Table(fieldInfo{fld})
			}
			if s.join == joinNone {
				s.join = joinInner
			}
			s.on = tag
			stmt.joins = append(stmt.joins, s)
			bindings = append(bindings, b...)
			idx := i
			completion = appendFn(completion, func(r *rowRef, v reflect.Value) {
				f(r, v.Field(idx))
			})
		default:
			if fld.Anonymous {
				s, b, f := prepare(namer, val.Field(i).Addr(), depth+1)
				if stmt.table == "" {
					stmt.table = s.table
				}
				stmt.columns = append(s.columns, stmt.columns...)
				stmt.conditions = append(s.conditions, stmt.conditions...)
				stmt.group = append(s.group, stmt.group...)
				stmt.order = append(s.order, stmt.order...)
				stmt.joins = append(s.joins, stmt.joins...)
				bindings = append(bindings, b...)
				idx := i
				completion = appendFn(completion, func(r *rowRef, v reflect.Value) {
					f(r, v.Field(idx))
				})
				continue
			}

			col := column{name: tag}
			if tag == "" {
				col = column{
					name:     namer.Column(fieldInfo{fld}),
					useTable: true,
				}
			}
			stmt.columns = append(stmt.columns, col)
			bindings = append(bindings, val.Field(i).Addr().Interface())
		}
	}

	if depth == 0 && stmt.table == "" {
		stmt.table = namer.Table(typ)
	}

	return stmt, bindings, completion
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

// appendFn returns a function that will call both the supplied head and tail functions. The supplied functions may
// be nil.
func appendFn(head, tail func(*rowRef, reflect.Value)) func(*rowRef, reflect.Value) {
	if head == nil {
		return tail
	}
	if tail == nil {
		return head
	}
	return func(r *rowRef, v reflect.Value) {
		head(r, v)
		tail(r, v)
	}
}

// rowRef identifies a table identity column value along with the identities of the related tables used to build
// a many relationship hierarchy.
type rowRef struct {
	parent *rowRef
	ident  string
}

// Hash returns a hash key to be used in a map.
func (r *rowRef) Hash() string {
	var builder strings.Builder
	for p := r; p != nil; p = p.parent {
		builder.WriteByte('.')
		builder.WriteString(p.ident)
	}
	return builder.String()
}

// asString attempts to convert the supplied value to a string.
func asString(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case *string:
		return *v
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.Itoa(int(v))
	case int64:
		return strconv.Itoa(int(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}
