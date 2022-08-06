package query

// Table can be composed in a query struct to assign the name of the table.
//
// Example:
//
//	type users struct {
//	  query.Table `q:"user"`
//	}
type Table struct{}

// Conditions can be composed in a query struct to assign query conditions. This is the WHERE section of the SQL
// statment. Example:
//
//	type users struct {
//	  query.Conditions `q:"name = ?"`
//	}
type Conditions struct{}

// OrderBy can be composed in a query struct to define column ordering. This is the ORDER BY section of the SQL
// statement. Example:
//
//	type users struct {
//	  query.OrderBy `q:"name"`
//	}
type OrderBy struct{}

// GroupBy can be composed in a query struct to group columns by. This is the GROUP BY section of the SQL statement.
// Example:
//
//	type users struct {
//	  query.GroupBy `q:"name"`
//	}
type GroupBy struct{}

// Limit can be composed in a query struct to limit the number of results. This is the LIMIT section of the SQL statement.
// Example:
//
//	type users struct {
//	  query.Limit `q:"10"`
//	}
type Limit struct{}

// Offset can be composed in a query struct to offset the results. This is the OFFSET section of the SQL statement.
// Example:
//
//	type users struct {
//	  query.Offset `q:"10"`
//	}
type Offset struct{}

// LeftJoin can be composed in a join query struct to signify that a left join should be used. This is the LEFT JOIN
// section of the SQL statement. Example:
//
//	type users struct {
//		Addresses struct {
//			query.LeftJoin
//		} `users.address_id = addresses.id`
//	}
type LeftJoin struct{}
