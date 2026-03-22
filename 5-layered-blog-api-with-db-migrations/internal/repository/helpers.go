package repository

import "database/sql"

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
