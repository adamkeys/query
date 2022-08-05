// Package query embraces the features of Go to query SQL databases.
package query_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"adamkeys.ca/query"
	"github.com/google/go-cmp/cmp"
	_ "github.com/mattn/go-sqlite3"
)

func TestAll(t *testing.T) {
	runDB(t, "BasicSelect", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.OrderBy `name ASC`

			ID   string         `id`
			Name sql.NullString `name`
		}
		results, err := query.All(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		names := make([]string, len(results))
		for i, user := range results {
			names[i] = user.Name.String
		}
		exp := []string{"", "Bob", "Gary", "James", "Joe", "John"}
		if diff := cmp.Diff(exp, names); diff != "" {
			t.Error(diff)
		}
	})

	runDB(t, "GroupCount", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.GroupBy `SUBSTR(name, 1, 1)`
			query.OrderBy `c DESC`

			Count int `COUNT(*) AS c`
		}
		results, err := query.All(context.Background(), db, func(u users) int { return u.Count })
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected results")
		}

		exp := []int{3, 1, 1, 1}
		if diff := cmp.Diff(exp, results); diff != "" {
			t.Error(diff)
		}
	})

	runDB(t, "Conditions", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.Conditions `name = ?`

			Name sql.NullString `name`
		}
		results, err := query.All(context.Background(), db, query.Identity[users], "Bob")
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected results")
		}

		if len(results) != 1 {
			t.Errorf("unexpected number of results; got: %d", len(results))
		}
	})

	runDB(t, "TableName", func(t *testing.T, db *sql.DB) {
		type userTable struct {
			query.Table      `users`
			query.Conditions `name = ?`

			Name sql.NullString `name`
		}
		results, err := query.All(context.Background(), db, query.Identity[userTable], "Bob")
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected results")
		}

		if len(results) != 1 {
			t.Errorf("unexpected number of results; got: %d", len(results))
		}
	})

	runDB(t, "Join", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.OrderBy `name DESC`

			Name      string `name`
			Addresses struct {
				City string `city`
			} `users.address_id = addresses.id`
		}
		results, err := query.All(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected results")
		}

		exp := users{Name: "John", Addresses: struct {
			City string `city`
		}{"New York"}}
		if diff := cmp.Diff(exp, results[0]); diff != "" {
			t.Error(diff)
		}
	})

	runDB(t, "NestedJoin", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.OrderBy `users.name DESC`

			Name      string `users.name`
			Addresses struct {
				City    string `city`
				Country struct {
					query.Table `countries`

					Name string `countries.name`
				} `addresses.country_id = countries.id`
			} `users.address_id = addresses.id`
		}
		results, err := query.All(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected results")
		}

		exp := users{Name: "John"}
		exp.Addresses.City = "New York"
		exp.Addresses.Country.Name = "United States"
		if diff := cmp.Diff(exp, results[0]); diff != "" {
			t.Error(diff)
		}
	})

	runDB(t, "InvalidField", func(t *testing.T, db *sql.DB) {
		type users struct {
			Name sql.NullString `nam`
		}
		_, err := query.All(context.Background(), db, query.Identity[users])
		if err == nil || err.Error() != "query: no such column: nam" {
			t.Errorf("unexpected error; got: %v", err)
		}
	})

	runDB(t, "InvalidType", func(t *testing.T, db *sql.DB) {
		type foo struct{}
		type users struct {
			Name foo `name`
		}
		_, err := query.All(context.Background(), db, query.Identity[users])
		if err == nil || !strings.Contains(err.Error(), "unsupported Scan") {
			t.Errorf("unexpected error; got: %v", err)
		}
	})
}

func TestOne(t *testing.T) {
	runDB(t, "Conditions", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.Conditions `name = ?`

			ID   string         `id`
			Name sql.NullString `name`
		}
		user, err := query.One(context.Background(), db, query.Identity[users], "Bob")
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		if user.Name.String != "Bob" {
			t.Errorf("unexpected user; got: %v", user)
		}
	})

	runDB(t, "Count", func(t *testing.T, db *sql.DB) {
		type users struct {
			Count int `COUNT(*)`
		}
		count, err := query.One(context.Background(), db, func(u users) int { return u.Count })
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		if count != 6 {
			t.Errorf("unexpected count; got: %v", count)
		}
	})
}

func runDB(t *testing.T, name string, fn func(t *testing.T, db *sql.DB)) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		defer db.Close()

		exec := func(query string, args ...any) int64 {
			res, err := db.Exec(query, args...)
			if err != nil {
				t.Fatalf("%s: %v", query, err)
			}
			id, err := res.LastInsertId()
			if err != nil {
				t.Fatal(err)
			}
			return id
		}
		exec(`CREATE TABLE countries (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`)
		exec(`CREATE TABLE addresses (id INTEGER PRIMARY KEY AUTOINCREMENT, city TEXT, country_id INTEGER REFERENCES countries(id))`)
		exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, address_id INTEGER REFERENCES addresses(id))`)
		ctry := exec(`INSERT INTO countries (name) VALUES ('United States')`)
		addr := exec(`INSERT INTO addresses (city, country_id) VALUES ('San Francisco', $1), ('New York', $1)`, ctry)
		exec(`INSERT INTO users (name, address_id) VALUES ('John', $1), ('James', $1), ('Gary', $1), ('Joe', $1), ('Bob', $1), (NULL, NULL)`, addr)

		fn(t, db)
	})
}
