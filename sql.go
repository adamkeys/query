package query

import (
	"strings"
)

// column identifies a select column. It contains the column name and a useTable flag. If useTable is set, the
// query builder will specify that the column name is associated with the current table. This prevents overlapping
// column names in joins.
type column struct {
	name     string
	useTable bool
}

// join identifies a join type that specifies how tables should be joined.
type join int

// The available join models.
const (
	joinNone join = iota
	joinInner
	joinLeft
)

// statement represents the properties of a query. It is used to facilitate the generation of a SQL query.
type statement struct {
	columns    []column
	table      string
	conditions []string
	order      []string
	group      []string
	limit      string
	offset     string

	join join
	on   string

	joins []statement
}

// SQL returns the query specified by the statement structure.
func (s *statement) SQL() string {
	var query strings.Builder
	query.WriteString("SELECT ")
	s.writeColumns(&query, 0)
	query.WriteString(" FROM ")
	query.WriteString(s.table)

	for _, join := range s.joins {
		join.writeJoin(&query)
	}
	if s.hasConditions() {
		query.WriteString(" WHERE ")
		s.writeConditions(&query, 0)
	}
	if s.hasGroup() {
		query.WriteString(" GROUP BY ")
		s.writeGroup(&query, 0)
	}
	if s.hasOrder() {
		query.WriteString(" ORDER BY ")
		s.writeOrder(&query, 0)
	}
	if s.limit != "" {
		query.WriteString(" LIMIT ")
		query.WriteString(s.limit)
	}
	if s.offset != "" {
		query.WriteString(" OFFSET ")
		query.WriteString(s.offset)
	}

	return query.String()
}

func (s *statement) writeColumns(w *strings.Builder, i int) {
	for _, col := range s.columns {
		if i > 0 {
			w.WriteString(", ")
		}
		if col.useTable {
			w.WriteString(s.table)
			w.WriteByte('.')
		}
		w.WriteString(col.name)
		i++
	}
	for _, join := range s.joins {
		join.writeColumns(w, i)
	}
}

func (s *statement) writeJoin(w *strings.Builder) {
	switch s.join {
	case joinNone:
		return
	case joinInner:
		w.WriteString(" INNER JOIN ")
	case joinLeft:
		w.WriteString(" LEFT JOIN ")
	}
	w.WriteString(s.table)
	w.WriteString(" ON ")
	w.WriteString(s.on)

	for _, join := range s.joins {
		join.writeJoin(w)
	}
}

func (s *statement) hasConditions() bool {
	for _, s := range s.joins {
		if s.hasConditions() {
			return true
		}
	}
	return len(s.conditions) > 0
}

func (s *statement) writeConditions(query *strings.Builder, depth int) {
	for i, condition := range s.conditions {
		if i > 0 {
			query.WriteString(" AND ")
		}
		query.WriteByte('(')
		query.WriteString(condition)
		query.WriteByte(')')
	}
	for _, join := range s.joins {
		join.writeConditions(query, depth+1)
	}
}

func (s *statement) hasGroup() bool {
	for _, s := range s.joins {
		if s.hasGroup() {
			return true
		}
	}
	return len(s.group) > 0
}

func (s *statement) writeGroup(query *strings.Builder, depth int) {
	for i, group := range s.group {
		if i+depth > 0 {
			query.WriteString(", ")
		}
		query.WriteString(group)
	}
	for _, join := range s.joins {
		join.writeGroup(query, depth+1)
	}
}

func (s *statement) hasOrder() bool {
	for _, s := range s.joins {
		if s.hasOrder() {
			return true
		}
	}
	return len(s.order) > 0
}

func (s *statement) writeOrder(query *strings.Builder, depth int) {
	for i, order := range s.order {
		if i+depth > 0 {
			query.WriteString(", ")
		}
		query.WriteString(order)
	}
	for _, join := range s.joins {
		join.writeOrder(query, depth+1)
	}
}
