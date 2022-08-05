// Package query embraces the features of Go to query SQL databases.
package query_test

import (
	"context"
	"database/sql"
	"log"
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
			t.Fatalf("all: %v", err)
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
		type users struct {
			Name struct{} `name`
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

		exec := func(query string) {
			_, err := db.Exec(query)
			if err != nil {
				log.Fatal(err)
			}
		}
		exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`)
		exec(`INSERT INTO users (name) VALUES ('John'), ('James'), ('Gary'), ('Joe'), ('Bob'), (NULL)`)

		fn(t, db)
	})
}
