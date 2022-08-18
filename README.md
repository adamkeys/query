![query](logo.svg)

# **query** – A Go SQL query package

Abusing language features for fun and profit, **query** enables writing SQL queries using `struct` types with liberal use of struct tags. No longer does the query need to be disjoined from the structure. The row results are populated right into the same query.

## Introduction

**query** provides a set of helper functions to assist with querying a SQL database. The caller is able to prepare SQL queries through defining a Go `struct` which doubles as the scan target when retrieving a SQL result set. The results may then be transformed into the application domain model in a type-safe manner, or the original query `struct` can be used by the application using the `Ident` function. This reduces the load on the developer to have to define a query, define SQL-safe variables to scan into, and convert the results into a usable model by combining the first and second operation.

## Example Queries

    type users struct {
        ID   int            `q:"id"`
        Name sql.NullString `q:"name"`
    }
    // SELECT id, name FROM users

&nbsp;

    type users struct {
        query.Table      `q:"user"`
        query.Conditions `q:"id = ?"`
        query.GroupBy    `q:"id, name"`
        query.OrderBy    `q:"name DESC"`
        query.Limit      `q:"10"`
        query.Offset     `q:"?"`

        ID   int            `q:"id"`
        Name sql.NullString `q:"name"`
    }
    // SELECT id, name FROM user WHERE id = ?
    //   GROUP BY id, name ORDER BY name DESC
    //   LIMIT 10 OFFSET ?

&nbsp;

    type users struct {
        Count int `q:"COUNT(*)"`
    }
    // SELECT COUNT(*) FROM users
