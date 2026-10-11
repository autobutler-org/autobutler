package sqlutil

import (
	"errors"
	"strings"
)

// postgresUniqueViolation is the SQLSTATE PostgreSQL gives a unique violation.
const postgresUniqueViolation = "23505"

// sqlStateError is an error that carries an SQLSTATE, as pgconn.PgError does.
type sqlStateError interface {
	error
	SQLState() string
}

// IsUniqueConstraintErr reports whether err is a unique-constraint violation
// on either engine. modernc.org/sqlite surfaces one as an error string.
// PostgreSQL's driver surfaces a typed error carrying an SQLSTATE, read here
// through the method pgconn.PgError has for it so that this package imports
// no driver.
//
// On PostgreSQL the statement that failed has also aborted its transaction:
// a caller inside db.InTx must return, not carry on with the next statement.
func IsUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	if stateErr, ok := errors.AsType[sqlStateError](err); ok {
		return stateErr.SQLState() == postgresUniqueViolation
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
