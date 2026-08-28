package db

import "strings"

// timeParams makes the driver store every time.Time as SQLite's own canonical
// UTC format ("YYYY-MM-DD HH:MM:SS", format 3), which is exactly what
// datetime('now') produces.
//
// Both halves are required, and neither is sufficient alone (#1650):
//
//   - _timezone=UTC converts the value to UTC before it is formatted. Without
//     it, format 3 writes local wall-clock time and silently discards the
//     offset, which is worse than the bug it replaces — the value reads back
//     as a UTC instant hours away from the one that was written.
//   - _time_format=datetime picks format 3. Without it the driver writes Go's
//     t.String() ("2026-08-28 16:04:23.848055 -0700 PDT m=+3600.0"), which no
//     SQLite date function can parse and which sorts wrongly against
//     datetime('now').
//
// This is set at the connection rather than by calling .UTC() at each write
// site on purpose: queries compare these columns as text, so a single write
// site that forgot would silently reintroduce the bug.
const timeParams = "_time_format=datetime&_timezone=UTC"

// DSN returns the connection string for a SQLite database at path, carrying
// the timestamp settings every Quark database is expected to be opened with.
// Any database holding timestamps must be opened through this, including from
// tests — a connection that skips it writes a format the queries misread.
func DSN(path string) string {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + timeParams
}
