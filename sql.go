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
}

// SQL returns the query specified by the statement structure.
func (s *statement) SQL() string {
	var query strings.Builder
	query.WriteString("SELECT ")
	for i, col := range s.columns {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(col)
	}
	query.WriteString(" FROM ")
	query.WriteString(s.table)

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
