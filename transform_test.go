package query_test

import (
	"testing"
	"testing/quick"

	"adamkeys.ca/query"
)

func TestIdentity(t *testing.T) {
	cases := []struct {
		name string
		fn   any
	}{
		{"string", func(v string) bool { return query.Identity(v) == v }},
		{"int", func(v int) bool { return query.Identity(v) == v }},
		{"struct", func(v struct{ Name string }) bool { return query.Identity(v) == v }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := quick.Check(tc.fn, nil); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestAutoBasicTransform(t *testing.T) {
	type users struct {
		ID   string `q:"id"`
		Name string `q:"name"`
	}
	type User struct {
		ID   string
		Name string
	}
	exp := User{ID: "1", Name: "George"}
	if v := query.Auto[users, User](users{ID: "1", Name: "George"}); v != exp {
		t.Errorf("expected value to be: %v; got: %v", exp, v)
	}
}

func TestAutoNestedTransform(t *testing.T) {
	type users struct {
		ID      string `q:"id"`
		Name    string `q:"name"`
		Address struct {
			query.Table `q:"addresses"`
			City        string
		} `q:"users.address_id = addresses.id"`
	}
	type Address struct {
		City string
	}
	type User struct {
		ID      string
		Name    string
		Address Address
	}

	in := users{ID: "1", Name: "George"}
	in.Address.City = "New York"
	exp := User{
		ID:      "1",
		Name:    "George",
		Address: Address{City: "New York"},
	}
	if v := query.Auto[users, User](in); v != exp {
		t.Errorf("expected value to be: %v; got: %v", exp, v)
	}
}
