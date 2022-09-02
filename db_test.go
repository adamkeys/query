package query_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/adamkeys/query"
)

func TestOpen(t *testing.T) {
	db, err := query.Open("sqlite3", ":memory:", nil)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	type users struct{ Name string }
	_, err = query.One(context.Background(), db, query.Identity[users])
	if err.Error() != "no such table: users" {
		t.Errorf("unexpected error; got: %v", err)
	}
}

func TestOptionsLog(t *testing.T) {
	var loggedQuery string
	dbh := query.DB{
		DB:      db,
		Options: &query.Options{Logger: func(query string, args []any) { loggedQuery = query }},
	}

	type users struct{ Name string }
	query.One(context.Background(), dbh, query.Identity[users])

	const exp = "SELECT users.name FROM users"
	if loggedQuery != exp {
		t.Errorf("expected query to be: %q; got: %q", exp, loggedQuery)
	}
}

func TestOptionsLogBeginTransaction(t *testing.T) {
	var loggedQuery string
	dbh := query.DB{
		DB:      db,
		Options: &query.Options{Logger: func(query string, args []any) { loggedQuery = query }},
	}

	tx, err := dbh.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	type users struct{ Name string }
	query.One(context.Background(), tx, query.Identity[users])

	const exp = "SELECT users.name FROM users"
	if loggedQuery != exp {
		t.Errorf("expected query to be: %q; got: %q", exp, loggedQuery)
	}
}

func TestOptionsLogBeginTxTransaction(t *testing.T) {
	var loggedQuery string
	dbh := query.DB{
		DB:      db,
		Options: &query.Options{Logger: func(query string, args []any) { loggedQuery = query }},
	}

	tx, err := dbh.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	type users struct{ Name string }
	query.One(context.Background(), tx, query.Identity[users])

	const exp = "SELECT users.name FROM users"
	if loggedQuery != exp {
		t.Errorf("expected query to be: %q; got: %q", exp, loggedQuery)
	}
}

func TestOptionsNamer(t *testing.T) {
	dbh := query.DB{
		DB:      db,
		Options: &query.Options{Namer: testNamer{}},
	}
	type people struct{ UserName string }
	user, err := query.One(context.Background(), dbh, query.Identity[people])
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	const exp = "John"
	if user.UserName != exp {
		t.Errorf("expected name to be: %q; got: %q", exp, user.UserName)
	}
}

type testNamer struct{}

func (t testNamer) Ident(info query.ElementInfo) string  { return "name" }
func (t testNamer) Table(info query.ElementInfo) string  { return "users" }
func (t testNamer) Column(info query.ElementInfo) string { return "name" }
