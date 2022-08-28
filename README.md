![query](logo.svg)

# **query** – A Go SQL query package

Abusing language features for fun and profit, **query** provides a different take for querying SQL databases in Go keeping SQL front and centre, but using language constructs to generate the SQL.

## Introduction

**query** provides a set of helper functions to assist with querying a SQL database. The caller is able to prepare SQL queries through defining a Go `struct` which doubles as the scan target when retrieving a SQL result set. The results may then be transformed into the application domain model in a type-safe manner, or the original query `struct` can be used by the application using the `Ident` function. This reduces the load on the developer to have to define a query, define SQL-safe variables to scan into, and convert the results into a usable model by combining the first and second operation.

## Example

    type User struct {
        ID        int
        Name      string
        Addresses []Address
    }

    type Address struct {
        ID    int
        City  string
        State string
    }

    func FindUser(ctx context.Context, db *sql.DB, userID int) (User, error) {
        return query.One(ctx, db, func(u userQuery) User {
            user := User{
                ID:        u.ID,
                Name:      u.Name,
                Addresses: make([]Address, len(u.Addresses)),
            }
            for i, addr := range u.Addresses {
                user.Addresses[i] = Address{
                    ID:    addr.ID,
                    City:  addr.City,
                    State: addr.State,
                }
            }
            return user
        }, userID)
    }

    type usersQuery struct {
        query.Table `q:"users"`

        ID        int
        Name      string

        Addresses []struct {
            ID    int
            City  string
            State string
        } `q:"users.address_id = addresses.id"`
    }

    type userQuery struct {
        query.Conditions `q:"users.id = $1"`
        usersQuery
    }

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
