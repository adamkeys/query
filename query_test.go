// Package query embraces the features of Go to query SQL databases.
package query_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"adamkeys.ca/query"
	"github.com/google/go-cmp/cmp"
	_ "github.com/mattn/go-sqlite3"
)

func TestAllBasicSelect(t *testing.T) {
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
}

func TestAllInferredSelect(t *testing.T) {
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
}

func TestAllInferredJoin(t *testing.T) {
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
}

func TestAllGroupCount(t *testing.T) {
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
}

func TestAllConditions(t *testing.T) {
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
}

func TestAllTableName(t *testing.T) {
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
}

func TestAllJoin(t *testing.T) {
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
}

func TestAllNestedJoin(t *testing.T) {
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
}

func TestAllParallelJoin(t *testing.T) {
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
}

func TestAllLeftJoin(t *testing.T) {
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
}

func TestAllJoinConditions(t *testing.T) {
	type users struct {
		query.OrderBy `q:"name DESC"`

		Name      string `q:"name"`
		Addresses struct {
			query.Conditions `q:"city != 'New York'"`

			City string `q:"city"`
		} `q:"users.address_id = addresses.id"`
	}
	results, err := query.All(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if len(results) != 0 {
		t.Fatal("expected number of results")
	}
}

func TestAllJoinProperties(t *testing.T) {
	type users struct {
		Name      string `q:"name"`
		Addresses struct {
			query.OrderBy    `q:"name DESC"`
			query.GroupBy    `q:"name"`
			query.Conditions `q:"city != 'New York'"`

			City string `q:"city"`
		} `q:"users.address_id = addresses.id"`
	}
	results, err := query.All(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if len(results) != 0 {
		t.Fatal("expected number of results")
	}
}

func TestAllJoinMany(t *testing.T) {
	type addresses struct {
		City  string
		Users []struct {
			query.Conditions `q:"name = 'John' OR name = 'Bob'"`
			query.OrderBy    `q:"name"`

			Name string
		} `q:"users.address_id = addresses.id"`
	}
	results, err := query.All(context.Background(), db, query.Identity[addresses])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	addr := addresses{}
	addr.City = "New York"
	for _, name := range []string{"Bob", "John"} {
		addr.Users = append(addr.Users, struct {
			query.Conditions `q:"name = 'John' OR name = 'Bob'"`
			query.OrderBy    `q:"name"`
			Name             string
		}{Name: name})
	}
	if diff := cmp.Diff([]addresses{addr}, results); diff != "" {
		t.Error(diff)
	}
}

func TestAllLeftJoinMany(t *testing.T) {
	type addresses struct {
		City  string
		Users []struct {
			query.LeftJoin
			query.Conditions `q:"name = 'John' OR name = 'Bob' OR name IS NULL"`
			query.OrderBy    `q:"name"`

			Name sql.NullString
		} `q:"users.address_id = addresses.id"`
	}
	results, err := query.All(context.Background(), db, query.Identity[addresses])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	addr := addresses{}
	addr.City = "New York"
	for _, name := range []string{"Bob", "John"} {
		addr.Users = append(addr.Users, struct {
			query.LeftJoin
			query.Conditions `q:"name = 'John' OR name = 'Bob' OR name IS NULL"`
			query.OrderBy    `q:"name"`
			Name             sql.NullString
		}{Name: sql.NullString{String: name, Valid: true}})
	}
	if diff := cmp.Diff([]addresses{{City: "San Francisco"}, addr}, results); diff != "" {
		t.Error(diff)
	}
}

func TestAllJoinManySharedRow(t *testing.T) {
	type users struct {
		Name      string
		Addresses []struct {
			City string
		} `q:"users.address_id = addresses.id"`
	}
	results, err := query.All(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	for _, user := range results {
		if len(user.Addresses) == 0 {
			t.Errorf("expected user %q to have an address", user.Name)
		}
	}
}

func TestAllLimit(t *testing.T) {
	type users struct {
		query.Limit `q:"1"`

		Name string
	}
	results, err := query.All(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("unexpected number of results; got: %v", len(results))
	}
}

func TestAllOffset(t *testing.T) {
	type users struct {
		query.Limit  `q:"10"`
		query.Offset `q:"100"`

		Name string
	}
	results, err := query.All(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("unexpected number of results; got: %v", len(results))
	}
}

func TestAllComposition(t *testing.T) {
	type BaseUser struct {
		ID   string
		Name sql.NullString
	}
	type users struct {
		query.OrderBy `q:"name ASC"`

		BaseUser
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
}

func TestAllCompositionDefinedTable(t *testing.T) {
	type BaseUser struct {
		query.Table `q:"users"`
		ID          string
		Name        sql.NullString
	}
	type usersQuery struct {
		query.OrderBy `q:"name ASC"`

		BaseUser
	}
	results, err := query.All(context.Background(), db, query.Identity[usersQuery])
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
}

func TestAllCompositionJoinMany(t *testing.T) {
	type BaseAddresses struct {
		query.Table `q:"addresses"`
		City        string
		Users       []struct {
			query.Conditions `q:"name = 'John' OR name = 'Bob'"`
			query.OrderBy    `q:"name"`

			Name string
		} `q:"users.address_id = addresses.id"`
	}
	type addresses struct {
		BaseAddresses
	}
	results, err := query.All(context.Background(), db, query.Identity[addresses])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	addr := addresses{}
	addr.City = "New York"
	for _, name := range []string{"Bob", "John"} {
		addr.Users = append(addr.Users, struct {
			query.Conditions `q:"name = 'John' OR name = 'Bob'"`
			query.OrderBy    `q:"name"`
			Name             string
		}{Name: name})
	}
	if diff := cmp.Diff([]addresses{addr}, results); diff != "" {
		t.Error(diff)
	}
}

func TestAllCompositionProperties(t *testing.T) {
	type BaseUser struct {
		query.Conditions `q:"name = 'Bob' OR name = 'James'"`
		query.OrderBy    `q:"name ASC"`

		ID   string
		Name sql.NullString
	}
	type users struct {
		query.Conditions `q:"name = 'Bob'"`

		BaseUser
	}
	results, err := query.All(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	names := make([]string, len(results))
	for i, user := range results {
		names[i] = user.Name.String
	}
	exp := []string{"Bob"}
	if diff := cmp.Diff(exp, names); diff != "" {
		t.Error(diff)
	}
}

func TestAllCompositionGroup(t *testing.T) {
	type BaseUser struct {
		query.GroupBy `q:"SUBSTR(name, 1, 1)"`
		query.OrderBy `q:"c DESC"`

		Count int `q:"COUNT(*) AS c"`
	}
	type users struct {
		BaseUser
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
}

func TestAllCompositionJoin(t *testing.T) {
	type BaseUser struct {
		Addresses struct {
			City string `q:"city"`
		} `q:"users.address_id = addresses.id"`
	}
	type users struct {
		query.OrderBy `q:"name"`

		BaseUser
	}
	results, err := query.All(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}

	exp := users{
		BaseUser: BaseUser{
			Addresses: struct {
				City string `q:"city"`
			}{"New York"},
		},
	}
	if diff := cmp.Diff(exp, results[0]); diff != "" {
		t.Error(diff)
	}
}

func TestAllInvalidField(t *testing.T) {
	type users struct {
		Name sql.NullString `q:"nam"`
	}
	_, err := query.All(context.Background(), db, query.Identity[users])
	if err == nil || err.Error() != "query: no such column: nam" {
		t.Errorf("unexpected error; got: %v", err)
	}
}

func TestAllInvalidType(t *testing.T) {
	type foo struct{}
	type users struct {
		Name foo `q:"name"`
	}
	_, err := query.All(context.Background(), db, query.Identity[users])
	if err == nil || !strings.Contains(err.Error(), "unsupported Scan") {
		t.Errorf("unexpected error; got: %v", err)
	}
}

func TestOneConditions(t *testing.T) {
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
}

func TestOneCount(t *testing.T) {
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
}

func TestOneTableName(t *testing.T) {
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
}

func TestOneJoin(t *testing.T) {
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
}

func TestOneNestedJoin(t *testing.T) {
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
}

func TestOneParallelJoin(t *testing.T) {
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
}

func TestOneLeftJoin(t *testing.T) {
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
}

func TestOneJoinConditions(t *testing.T) {
	type users struct {
		query.OrderBy `q:"name DESC"`

		Name      string `q:"name"`
		Addresses struct {
			query.Conditions `q:"city != 'New York'"`

			City string `q:"city"`
		} `q:"users.address_id = addresses.id"`
	}
	_, err := query.One(context.Background(), db, query.Identity[users])
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("failed to filter join results: %v", err)
	}
}

func TestOneJoinMany(t *testing.T) {
	type addresses struct {
		City  string
		Users []struct {
			query.OrderBy    `q:"name"`
			query.Conditions `q:"name = 'John' OR name = 'Bob'"`

			Name string
		} `q:"users.address_id = addresses.id"`
	}
	result, err := query.One(context.Background(), db, query.Identity[addresses])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	addr := addresses{}
	addr.City = "New York"
	for _, name := range []string{"Bob", "John"} {
		addr.Users = append(addr.Users, struct {
			query.OrderBy    `q:"name"`
			query.Conditions `q:"name = 'John' OR name = 'Bob'"`
			Name             string
		}{Name: name})
	}
	if diff := cmp.Diff(addr, result); diff != "" {
		t.Error(diff)
	}
}

func TestOneJoinManySharedRow(t *testing.T) {
	type users struct {
		Name      string
		Addresses []struct {
			City string
		} `q:"users.address_id = addresses.id"`
	}
	result, err := query.One(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if len(result.Addresses) == 0 {
		t.Errorf("expected user %q to have an address", result.Name)
	}
}

func TestOneComposition(t *testing.T) {
	type BaseUser struct {
		ID   string
		Name string
	}
	type users struct {
		query.OrderBy `q:"name DESC"`

		BaseUser
	}
	result, err := query.One(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	const exp = "John"
	if result.Name != exp {
		t.Errorf("expected name to be: %q; got: %q", exp, result.Name)
	}
}

func TestOneCompositionDefinedTable(t *testing.T) {
	type BaseUser struct {
		query.Table `q:"users"`
		ID          string
		Name        sql.NullString
	}
	type usersQuery struct {
		query.OrderBy `q:"name DESC"`

		BaseUser
	}
	result, err := query.One(context.Background(), db, query.Identity[usersQuery])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	const exp = "John"
	if result.Name.String != exp {
		t.Errorf("expected name to be: %q; got: %q", exp, result.Name.String)
	}
}

func TestOneCompositionProperties(t *testing.T) {
	type BaseUser struct {
		query.Conditions `q:"name = 'Bob' OR name = 'James'"`
		query.OrderBy    `q:"name ASC"`

		ID   string
		Name string
	}
	type users struct {
		query.Conditions `q:"name = 'Bob'"`

		BaseUser
	}
	result, err := query.One(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	const exp = "Bob"
	if result.Name != exp {
		t.Errorf("expected name to be: %q; got: %q", exp, result.Name)
	}
}

func TestOneCompositionGroup(t *testing.T) {
	type BaseUser struct {
		Count int `q:"COUNT(*) AS c"`
	}
	type users struct {
		BaseUser
	}
	result, err := query.One(context.Background(), db, func(u users) int { return u.Count })
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	const exp = 6
	if result != exp {
		t.Errorf("expected count to be: %d; got: %d", exp, result)
	}
}

func TestOneCompositionJoin(t *testing.T) {
	type BaseUser struct {
		Addresses struct {
			City string `q:"city"`
		} `q:"users.address_id = addresses.id"`
	}
	type users struct {
		query.OrderBy `q:"name"`

		BaseUser
	}
	result, err := query.One(context.Background(), db, query.Identity[users])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	exp := users{
		BaseUser: BaseUser{
			Addresses: struct {
				City string `q:"city"`
			}{"New York"},
		},
	}
	if diff := cmp.Diff(exp, result); diff != "" {
		t.Error(diff)
	}
}

func TestOneCompositionJoinMany(t *testing.T) {
	type BaseAddresses struct {
		query.Table `q:"addresses"`
		City        string
		Users       []struct {
			query.Conditions `q:"name = 'John' OR name = 'Bob'"`
			query.OrderBy    `q:"name"`

			Name string
		} `q:"users.address_id = addresses.id"`
	}
	type addresses struct {
		BaseAddresses
	}
	result, err := query.One(context.Background(), db, query.Identity[addresses])
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	addr := addresses{}
	addr.City = "New York"
	for _, name := range []string{"Bob", "John"} {
		addr.Users = append(addr.Users, struct {
			query.Conditions `q:"name = 'John' OR name = 'Bob'"`
			query.OrderBy    `q:"name"`
			Name             string
		}{Name: name})
	}
	if diff := cmp.Diff(addr, result); diff != "" {
		t.Error(diff)
	}
}

func TestOneInvalidField(t *testing.T) {
	type users struct {
		Name sql.NullString `q:"nam"`
	}
	_, err := query.One(context.Background(), db, query.Identity[users])
	if err == nil || err.Error() != "no such column: nam" {
		t.Errorf("unexpected error; got: %v", err)
	}
}

func TestOneInvalidType(t *testing.T) {
	type foo struct{}
	type users struct {
		Name foo `q:"name"`
	}
	_, err := query.One(context.Background(), db, query.Identity[users])
	if err == nil || !strings.Contains(err.Error(), "unsupported Scan") {
		t.Errorf("unexpected error; got: %v", err)
	}
}

var setupQueries = []string{
	`CREATE TABLE countries (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`,
	`CREATE TABLE addresses (id INTEGER PRIMARY KEY AUTOINCREMENT, city TEXT, country_id INTEGER REFERENCES countries(id))`,
	`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, address_id INTEGER REFERENCES addresses(id))`,
	`INSERT INTO countries (name) VALUES ('United States')`,
	`INSERT INTO addresses (city, country_id) VALUES ('San Francisco', 1), ('New York', 1)`,
	`INSERT INTO users (name, address_id) VALUES ('John', 2), ('James', 2), ('Gary', 2), ('Joe', 2), ('Bob', 2), (NULL, NULL)`,
}

var db *sql.DB

func TestMain(m *testing.M) {
	var err error
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v", err)
		os.Exit(1)
	}

	for _, q := range setupQueries {
		_, err := db.Exec(q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "exec: %v", err)
			os.Exit(1)
		}
	}

	os.Exit(m.Run())
}
