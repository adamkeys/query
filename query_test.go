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
			query.OrderBy `q:"name ASC"`

			ID   string         `q:"id"`
			Name sql.NullString `q:"name"`
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

	runDB(t, "InferredSelect", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.OrderBy `q:"name ASC"`

			ID   string
			Name sql.NullString
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

	runDB(t, "InferredJoin", func(t *testing.T, db *sql.DB) {
		type users struct {
			ID        string
			Addresses struct {
				ID string
			} `q:"users.address_id = addresses.id"`
		}
		results, err := query.All(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		for _, r := range results {
			if r.ID == "" || r.Addresses.ID == "" {
				t.Errorf("expected value to have IDs; got: %v", r)
			}
		}
	})

	runDB(t, "GroupCount", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.GroupBy `q:"SUBSTR(name, 1, 1)"`
			query.OrderBy `q:"c DESC"`

			Count int `q:"COUNT(*) AS c"`
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
			query.Conditions `q:"name = ?"`

			Name sql.NullString `q:"name"`
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
			query.Table      `q:"users"`
			query.Conditions `q:"name = ?"`

			Name sql.NullString `q:"name"`
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
			query.OrderBy `q:"name DESC"`

			Name      string `q:"name"`
			Addresses struct {
				City string `q:"city"`
			} `q:"users.address_id = addresses.id"`
		}
		results, err := query.All(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected results")
		}

		exp := users{Name: "John", Addresses: struct {
			City string `q:"city"`
		}{"New York"}}
		if diff := cmp.Diff(exp, results[0]); diff != "" {
			t.Error(diff)
		}
	})

	runDB(t, "NestedJoin", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.OrderBy `q:"users.name DESC"`

			Name      string `q:"users.name"`
			Addresses struct {
				City    string `q:"city"`
				Country struct {
					query.Table `q:"countries"`

					Name string `q:"countries.name"`
				} `q:"addresses.country_id = countries.id"`
			} `q:"users.address_id = addresses.id"`
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

	runDB(t, "ParallelJoin", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.OrderBy `q:"users.name DESC"`

			Name     string `q:"users.name"`
			Address1 struct {
				query.Table `q:"addresses a1"`
				City        string `q:"a1.city"`
			} `q:"users.address_id = a1.id"`
			Address2 struct {
				query.Table `q:"addresses a2"`
				City        string `q:"a2.city"`
			} `q:"users.address_id = a2.id"`
		}
		results, err := query.All(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected results")
		}

		exp := users{Name: "John"}
		exp.Address1.City = "New York"
		exp.Address2.City = "New York"
		if diff := cmp.Diff(exp, results[0]); diff != "" {
			t.Error(diff)
		}
	})

	runDB(t, "LeftJoin", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.Conditions `q:"users.name IS NOT NULL"`

			Name      string
			Addresses struct {
				query.LeftJoin

				City sql.NullString
			} `q:"users.name = addresses.id"`
		}
		results, err := query.All(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected results")
		}

		if results[0].Addresses.City.Valid {
			t.Error("expected city to be invalid")
		}
	})

	runDB(t, "InvalidField", func(t *testing.T, db *sql.DB) {
		type users struct {
			Name sql.NullString `q:"nam"`
		}
		_, err := query.All(context.Background(), db, query.Identity[users])
		if err == nil || err.Error() != "query: no such column: nam" {
			t.Errorf("unexpected error; got: %v", err)
		}
	})

	runDB(t, "InvalidType", func(t *testing.T, db *sql.DB) {
		type foo struct{}
		type users struct {
			Name foo `q:"name"`
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
			query.Conditions `q:"name = ?"`

			ID   string         `q:"id"`
			Name sql.NullString `q:"name"`
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
			Count int `q:"COUNT(*)"`
		}
		count, err := query.One(context.Background(), db, func(u users) int { return u.Count })
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		if count != 6 {
			t.Errorf("unexpected count; got: %v", count)
		}
	})

	runDB(t, "Conditions", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.Conditions `q:"name = ?"`

			Name sql.NullString `q:"name"`
		}
		result, err := query.One(context.Background(), db, query.Identity[users], "Bob")
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		if result.Name.String != "Bob" {
			t.Errorf("unexpected result; got: %v", result)
		}
	})

	runDB(t, "TableName", func(t *testing.T, db *sql.DB) {
		type userTable struct {
			query.Table      `q:"users"`
			query.Conditions `q:"name = ?"`

			Name sql.NullString `q:"name"`
		}
		result, err := query.One(context.Background(), db, query.Identity[userTable], "Bob")
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		if result.Name.String != "Bob" {
			t.Errorf("unexpected result; got: %v", result)
		}
	})

	runDB(t, "Join", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.OrderBy `q:"name DESC"`

			Name      string `q:"name"`
			Addresses struct {
				City string `q:"city"`
			} `q:"users.address_id = addresses.id"`
		}
		result, err := query.One(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		exp := users{Name: "John", Addresses: struct {
			City string `q:"city"`
		}{"New York"}}
		if diff := cmp.Diff(exp, result); diff != "" {
			t.Error(diff)
		}
	})

	runDB(t, "NestedJoin", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.OrderBy `q:"users.name DESC"`

			Name      string `q:"users.name"`
			Addresses struct {
				City    string `q:"city"`
				Country struct {
					query.Table `q:"countries"`

					Name string `q:"countries.name"`
				} `q:"addresses.country_id = countries.id"`
			} `q:"users.address_id = addresses.id"`
		}
		result, err := query.One(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		exp := users{Name: "John"}
		exp.Addresses.City = "New York"
		exp.Addresses.Country.Name = "United States"
		if diff := cmp.Diff(exp, result); diff != "" {
			t.Error(diff)
		}
	})

	runDB(t, "ParallelJoin", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.OrderBy `q:"users.name DESC"`

			Name     string `q:"users.name"`
			Address1 struct {
				query.Table `q:"addresses a1"`
				City        string `q:"a1.city"`
			} `q:"users.address_id = a1.id"`
			Address2 struct {
				query.Table `q:"addresses a2"`
				City        string `q:"a2.city"`
			} `q:"users.address_id = a2.id"`
		}
		result, err := query.One(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		exp := users{Name: "John"}
		exp.Address1.City = "New York"
		exp.Address2.City = "New York"
		if diff := cmp.Diff(exp, result); diff != "" {
			t.Error(diff)
		}
	})

	runDB(t, "LeftJoin", func(t *testing.T, db *sql.DB) {
		type users struct {
			query.Conditions `q:"users.name IS NOT NULL"`

			Name      string
			Addresses struct {
				query.LeftJoin

				City sql.NullString
			} `q:"users.name = addresses.id"`
		}
		result, err := query.One(context.Background(), db, query.Identity[users])
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}

		if result.Addresses.City.Valid {
			t.Error("expected city to be invalid")
		}
	})

	runDB(t, "InvalidField", func(t *testing.T, db *sql.DB) {
		type users struct {
			Name sql.NullString `q:"nam"`
		}
		_, err := query.One(context.Background(), db, query.Identity[users])
		if err == nil || err.Error() != "no such column: nam" {
			t.Errorf("unexpected error; got: %v", err)
		}
	})

	runDB(t, "InvalidType", func(t *testing.T, db *sql.DB) {
		type foo struct{}
		type users struct {
			Name foo `q:"name"`
		}
		_, err := query.One(context.Background(), db, query.Identity[users])
		if err == nil || !strings.Contains(err.Error(), "unsupported Scan") {
			t.Errorf("unexpected error; got: %v", err)
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
