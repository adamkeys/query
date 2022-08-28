package query_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/adamkeys/query"
	_ "github.com/mattn/go-sqlite3"
)

func openDB() *sql.DB {
	db, _ := sql.Open("sqlite3", ":memory:")
	db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`)
	db.Exec(`CREATE TABLE addresses (id INTEGER PRIMARY KEY AUTOINCREMENT, city TEXT)`)
	db.Exec(`CREATE TABLE locations (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, address_id INTEGER)`)
	db.Exec(`INSERT INTO users (name) VALUES ('John'), ('Jane')`)
	db.Exec(`INSERT INTO addresses (city) VALUES ('Paris'), ('San Francisco')`)
	db.Exec(`INSERT INTO locations (user_id, address_id) VALUES (1, 1), (1, 2), (2, 2)`)
	return db
}

// usersQuery represents the SQL query:
//
//	SELECT users.name, addresses.city FROM users
//	  INNER JOIN locations ON locations.user_id = users.id
//	  INNER JOIN addresses ON locations.address_id = addresses.id
type usersQuery struct {
	query.Table `q:"users"`
	Name        sql.NullString
	Locations   []struct {
		Addresses struct {
			City sql.NullString
		} `q:"locations.address_id = addresses.id"`
	} `q:"locations.user_id = users.id"`
}

func ExampleAll() {
	db := openDB()
	users, _ := query.All(context.Background(), db, func(row usersQuery) string {
		cities := make([]string, len(row.Locations))
		for i, location := range row.Locations {
			cities[i] = location.Addresses.City.String
		}
		return fmt.Sprintf("%s (%s)", row.Name.String, strings.Join(cities, ", "))
	})
	fmt.Println(strings.Join(users, "; "))
	// Output: John (Paris, San Francisco); Jane (San Francisco)
}

// userLocationCountQuery represents the SQL query:
//
//	SELECT COUNT(locations.id) FROM users
//	  INNER JOIN locations ON users.id = locations.user_id
//	  WHERE (users.id = $1)
type userLocationCountQuery struct {
	query.Table      `q:"users"`
	query.Conditions `q:"users.id = $1"`
	Locations        struct {
		Count int `q:"COUNT(locations.id)"`
	} `q:"users.id = locations.user_id"`
}

func ExampleOne() {
	db := openDB()
	count, _ := query.One(context.Background(), db, func(row userLocationCountQuery) int {
		return row.Locations.Count
	}, 1)
	fmt.Println(count)
	// Output: 2
}
