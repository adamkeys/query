package query

import (
	"strings"
)

// statement represents the properties of a query. It is used to facilitate the generation of a SQL query.
type statement struct {
	columns    []string
	table      string
	conditions string
	order      string
	group      string
	limit      string
	offset     string
	joins      []joinStatement
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
	if s.conditions != "" {
		query.WriteString(" WHERE ")
		query.WriteString(s.conditions)
	}
	if s.group != "" {
		query.WriteString(" GROUP BY ")
		query.WriteString(s.group)
	}
	if s.order != "" {
		query.WriteString(" ORDER BY ")
		query.WriteString(s.order)
	}
	if s.limit != "" {
		query.WriteString(s.limit)
	}
	if s.offset != "" {
		query.WriteString(s.offset)
	}

	return query.String()
}

func (s *statement) writeColumns(w *strings.Builder, i int) {
	for _, col := range s.columns {
		if i > 0 {
			w.WriteString(", ")
		}
		w.WriteString(col)
		i++
	}
	for _, join := range s.joins {
		join.writeColumns(w, i)
	}
}

// a joinStatement represents a statement that is included as a join table. It consists of a statement along with
// join conditions.
type joinStatement struct {
	statement
	on string
}

func (j *joinStatement) writeJoin(w *strings.Builder) {
	w.WriteString(" INNER JOIN ")
	w.WriteString(j.table)
	w.WriteString(" ON ")
	w.WriteString(j.on)

	for _, join := range j.joins {
		join.writeJoin(w)
	}
}
