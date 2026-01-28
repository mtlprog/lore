package database

import sq "github.com/Masterminds/squirrel"

// QB is the query builder with PostgreSQL placeholder format.
var QB = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
