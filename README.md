# **query** – A Go database query package

Abusing language features for fun and profit, **query** enables writing SQL queries using `struct` types with liberal use of struct tags.

## Introduction

**query** provides a set of helper functions to assist with querying a SQL database. The caller is able to prepare SQL queries through defining a Go `struct` which doublts as the scan target when retriving a SQL resultset.

## Example Queries

    type users struct {
        ID   int            `id`
        Name sql.NullString `name`
    }
    // SELECT id, name FROM users

    type users struct {
        query.Table      `user`
        query.Conditions `id = ?`
        query.GroupBy    `id, name`
        query.OrderBy    `name DESC`
        query.Limit      `10`
        query.Offset     `?`

        ID   int            `id`
        Name sql.NullString `name`
    }
    // SELECT id, name FROM user WHERE id = ?
    //   GROUP BY id, name ORDER BY name DESC
    //   LIMIT 10 OFFSET ?


    type users struct {
        Count int `COUNT(*)`
    }
    // SELECT COUNT(*) FROM users

### Conventional Struct Tags

The caller may choose to use the conventional struct tag format. To use the query tag may be assigned the `q` tag name.

    type users struct {
        Count int `q:"COUNT(*)"`
    }
    // SELECT COUNT(*) FROM users
