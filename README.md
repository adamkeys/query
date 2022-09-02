![query](logo.svg)

# **query** – A Go SQL query package

Abusing language features for fun and profit, **query** provides a different take for querying SQL databases in Go keeping SQL front and centre, but using language constructs to generate the SQL.

## Introduction

**query** provides a set of helper functions to assist with querying a SQL database. The caller is able to prepare SQL queries through defining a Go `struct` which doubles as the scan target when retrieving a SQL result set. The results may then be transformed into the application domain model in a type-safe manner, or the original query `struct` can be used by the application using the `Ident` function. This reduces the load on the developer to have to define a query, define SQL-safe variables to scan into, and convert the results into a usable model by combining the first and second operation.

## Examples

### Find One

    type User struct {
        ID   int
        Name string
    }

    func FindUser(ctx context.Context, db *sql.DB, userID int) (User, error) {
        type users struct {
            query.Conditions `q:"id = $1"`
            ID               int
            Name             string
        }
        return query.One(ctx, db, func(row users) User {
            return User{ID: row.ID, Name: row.Name}
        }, userID)
    }

### Find Many

    type User struct {
        ID   int
        Name string
    }

    func FindUsers(ctx context.Context, db *sql.DB, limit, offset int) ([]User, error) {
        type users struct {
            query.Limit  `q:"$1"`
            query.Offset `q:"$2"`
            ID           int
            Name         string
        }
        return query.All(ctx, db, func(row users) User {
            return User{ID: row.ID, Name: row.Name}
        }, limit, offset)
    }

### Has Many

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

### Basic Select

    type users struct {
        ID   int            `q:"id"`
        Name sql.NullString `q:"name"`
    }
    // SELECT id, name FROM users

### Select With Attributes

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

### Count Query

    type users struct {
        Count int `q:"COUNT(*)"`
    }
    // SELECT COUNT(*) FROM users

### Composition

    type usersQuery struct {
        query.Table `q:"users"`

        ID   int
        Name string
    }

    type usersByIDQuery struct {
        query.Conditions `q:"id = ?"`
        usersQuery
    }
    // SELECT id, name FROM users WHERE id = ?

### Inner Join

    type usersQuery struct {
        query.Table `q:"users"`

        ID        int
        Name      string
        Addresses struct {
            City string
        } `q:"users.address_id = addresses.id`
    }
    // SELECT users.id, users.name, addresses.city FROM users INNER JOIN addresses ON users.address_id = addresses.id

### Left Join

    type usersQuery struct {
        query.Table `q:"users"`

        ID        int
        Name      string
        Addresses struct {
            query.LeftJoin

            City sql.NullString
        } `q:"users.address_id = addresses.id`
    }
    // SELECT users.id, users.name, addresses.city FROM users LEFT JOIN addresses ON users.address_id = addresses.id

### Has Many Join

    type usersQuery struct {
        query.Table `q:"users"`

        ID        int
        Name      string
        Addresses []struct {
            City string
        } `q:"users.address_id = addresses.id`
    }
    // SELECT users.id, users.name, addresses.city FROM users INNER JOIN addresses ON users.address_id = addresses.id

## Options

While query works with standard `database/sql` database/transaction handles, additional features can be unlocked by opening the database using **query**'s `Open` function. The following options are available:

### Namer

The `Namer` configuration open allows the caller to specify an object that implements the `Namer` interface to provide custom query naming rules. This allows customization for databases that do not match the default standard namer conventions used by **query**.

### Logger

The `Logger` holds a function that accepts a query and arguments which is called when a query is executed. This can be used to help with debugging or to keep tabs on what queries are being executed.

### Example

    db, err := query.Open("sqlite3", "myfile.db", &query.Options{
        Namer: myNamer,
        Logger: func(query string, args []any) {
            fmt.Println(query, args)
        },
    })
