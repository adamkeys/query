# **query** – A Go database query package

Abusing language features for fun and profit, **query** enables writing SQL queries using `q:"struct"` types with liberal use of struct tags.

## Introduction

**query** provides a set of helper functions to assist with querying a SQL database. The caller is able to prepare SQL queries through defining a Go `q:"struct"` which doublts as the scan target when retriving a SQL resultset.

## Example Queries

    type users struct {
        ID   int            `q:"id"`
        Name sql.NullString `q:"name"`
    }
    // SELECT id, name FROM users

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


    type users struct {
        Count int `q:"COUNT(*)"`
    }
    // SELECT COUNT(*) FROM users
