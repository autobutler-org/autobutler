package favoritesutil

import "strings"

// isUniqueConstraintErr reports whether err is a SQLite unique-constraint
// violation (modernc.org/sqlite surfaces these as error strings).
func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
