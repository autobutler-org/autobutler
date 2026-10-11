# `sql/queries/AGENTS.md`

How to write a query that both of Quark's database engines accept. The root [`AGENTS.md`](../../AGENTS.md)
says where SQL lives and how it reaches Go; [`internal/db/AGENTS.md`](../../internal/db/AGENTS.md) covers the
schema side.

Quark runs on SQLite and is gaining PostgreSQL as a second backend (#2955). sqlc compiles one query set for
both engines, so every file in this directory is written in the subset of SQL the two share. SQLite is the
reference: a rule here never changes what a query does on SQLite.

> **Status.** Every query here follows the rules, and `internal/db/prepare_test.go` prepares each generated
> statement on SQLite. `sqlc.yaml` still generates the SQLite package only, so nothing yet rejects a query
> PostgreSQL would refuse: #2957 adds the PostgreSQL generation and the check that prepares every statement
> there too.

## Shared by default

Every query lives in `sql/queries/<topic>.sql` and is compiled for both engines. A query is written once per
engine only when it needs a construct that has **no form both engines accept** — see
[Per-engine queries](#per-engine-queries). Awkward, verbose, or slower-to-write is not a reason: every rule
below is a rewrite that keeps a query shared.

## The rules

Each rule shows the SQLite-only spelling and the shared one.

### 1. Named parameters only

A positional `?` is SQLite-only and `$1` is PostgreSQL-only; `sqlc.arg(name)` compiles to each engine's own
placeholder. Name the argument after the column it is compared with or assigned to, which is the name sqlc
infers for a `?`, so the generated Go field keeps its name.

```sql
-- before
DELETE FROM sessions WHERE user_id = ? AND token != ?;
-- after
DELETE FROM sessions WHERE user_id = sqlc.arg(user_id) AND token != sqlc.arg(token);
```

### 2. Quote a parameter named after a reserved word

```sql
-- before
LIMIT sqlc.arg(limit)
-- after
LIMIT CAST(sqlc.arg('limit') AS BIGINT)
```

### 3. `CURRENT_TIMESTAMP`, not `datetime('now')`

```sql
-- before
UPDATE vault_location SET updated_at = datetime('now') WHERE id = 1;
-- after
UPDATE vault_location SET updated_at = CURRENT_TIMESTAMP WHERE id = 1;
```

On SQLite the two store the same text, `YYYY-MM-DD HH:MM:SS` in UTC, which is also what the DSN's
`_time_format=datetime&_timezone=UTC` makes the driver write for a `time.Time` (`internal/db/dsn.go`). That
matters because every SQLite `DATETIME` comparison is a text comparison: one value in another format would
sort wrongly against the rest.

### 4. `CAST(... AS BIGINT)`, never `AS INTEGER`

`INTEGER` is 64-bit on SQLite and 32-bit on PostgreSQL, so the generated Go type would be `int64` in one
package and `int32` in the other.

```sql
-- before
CAST(sqlc.arg(user_id) AS INTEGER)
-- after
CAST(sqlc.arg(user_id) AS BIGINT)
```

### 5. Cast `LIMIT` and `OFFSET` parameters to `BIGINT`

Without the cast sqlc types the parameter `int64` on SQLite and `int32` on PostgreSQL.

```sql
-- before
ORDER BY id DESC LIMIT ?;
-- after
ORDER BY id DESC LIMIT CAST(sqlc.arg('limit') AS BIGINT);
```

### 6. `LOWER(expr)`, not `COLLATE NOCASE`

`NOCASE` is a SQLite collation. In an `ORDER BY`, wrap the full expression: PostgreSQL cannot apply a function
to an output alias.

```sql
-- before
SELECT CAST(COALESCE(users.username, groups.name) AS TEXT) AS name ... ORDER BY name COLLATE NOCASE;
-- after
SELECT CAST(COALESCE(users.username, groups.name) AS TEXT) AS name ... ORDER BY LOWER(COALESCE(users.username, groups.name));
```

Case-insensitive uniqueness belongs to the schema, not a query: see `internal/db/AGENTS.md`.

### 7. A boolean never passes through a numeric cast

SQLite's booleans are integers; PostgreSQL's are not and will not cast to `BIGINT`.

```sql
-- before
CAST(NOT EXISTS (SELECT 1 FROM ...) AS INTEGER) AS is_private
-- after
CAST(CASE WHEN NOT EXISTS (SELECT 1 FROM ...) THEN 1 ELSE 0 END AS BIGINT) AS is_private
```

### 8. Cast a bare `IS [NOT] NULL` in a select list to `BOOLEAN`

```sql
-- before
SELECT content_hash IS NOT NULL AS has_content_hash FROM photo_hashes;
-- after
SELECT CAST(content_hash IS NOT NULL AS BOOLEAN) AS has_content_hash FROM photo_hashes;
```

### 9. A `BOOLEAN` column takes `TRUE` / `FALSE`

This is about columns declared `BOOLEAN`. A 0/1 flag declared `INTEGER` (`users.is_admin`,
`groups.builtin`) is a number on both engines and keeps `0` / `1`.

```sql
-- before
UPDATE photo_hashes SET taken_checked = 1 WHERE ...;
-- after
UPDATE photo_hashes SET taken_checked = TRUE WHERE ...;
```

### 10. In `ON CONFLICT ... DO UPDATE`, qualify the target row

PostgreSQL calls a bare column ambiguous between the existing row and `excluded`.

```sql
-- before
ON CONFLICT (ip_address, user_agent) DO UPDATE SET request_count = request_count + 1
-- after
ON CONFLICT (ip_address, user_agent) DO UPDATE SET request_count = connected_devices.request_count + 1
```

The same goes for a `WHERE` on the `DO UPDATE`: `WHERE vfs_db_entries.is_dir = FALSE`.

### 11. No `INSERT OR IGNORE`, no `UPDATE OR IGNORE`

```sql
-- before
INSERT OR IGNORE INTO vfs_db_entries (namespace, path, ...) VALUES (...);
-- after
INSERT INTO vfs_db_entries (namespace, path, ...) VALUES (...) ON CONFLICT (namespace, path) DO NOTHING;
```

`UPDATE OR IGNORE` becomes a guard that skips the rows the update would collide with, written as a count:

```sql
-- before
UPDATE OR IGNORE photo_favorites SET rel_path = sqlc.arg(new_rel_path) WHERE rel_path = sqlc.arg(old_rel_path);
-- after
UPDATE photo_favorites SET rel_path = sqlc.arg(new_rel_path)
WHERE photo_favorites.rel_path = sqlc.arg(old_rel_path)
  AND (SELECT COUNT(*) FROM photo_favorites AS held
       WHERE held.user_id = photo_favorites.user_id AND held.rel_path = sqlc.arg(new_rel_path)) = 0;
```

Never `NOT EXISTS` with a parameter inside it: sqlc v1.30.0's SQLite engine leaves a parameter under a `NOT`
unexpanded ([sqlc-dev/sqlc#4110](https://github.com/sqlc-dev/sqlc/issues/4110)), and the statement fails to
prepare.

### 12. Spell out null-safe equality

`IS` between two columns is SQLite-only and `IS NOT DISTINCT FROM` is PostgreSQL-only.

```sql
-- before
newer.user_id IS jobs.user_id
-- after
(newer.user_id = jobs.user_id OR (newer.user_id IS NULL AND jobs.user_id IS NULL))
```

### 13. `ORDER BY` on a nullable column says `NULLS FIRST`

SQLite sorts NULL first by default, PostgreSQL last.

```sql
-- before
ORDER BY parent_id, name;
-- after
ORDER BY parent_id NULLS FIRST, name;
```

### 14. A constant date is a literal

```sql
-- before
VALUES (?, '', datetime('now', '-100 years'))
-- after
VALUES (sqlc.arg(name), '', '1970-01-01 00:00:00')
```

### 15. An IN-list over a parameter cannot be shared

`sqlc.slice` exists for SQLite and MySQL only and generates wrong code on PostgreSQL by design
([sqlc-dev/sqlc#2565](https://github.com/sqlc-dev/sqlc/issues/2565)); PostgreSQL's form,
`= ANY(sqlc.arg(x)::TEXT[])`, needs arrays, which SQLite does not have. This is the one construct with no
shared form, so such a query is a [per-engine query](#per-engine-queries).

### 16. A SQLite function needs a PostgreSQL function of the same name

A query may call a function PostgreSQL lacks only when the PostgreSQL schema defines a function of that name
for the form the query uses. Two are in use, both only where the value depends on the database's own clock or
on JSON stored in a column:

- `datetime('now', '-' || CAST(sqlc.arg(seconds) AS BIGINT) || ' seconds')` — the job lease and maintenance
  interval comparisons.
- `json_extract(column, '$.key')` — `GetActiveBackupJob`.

Adding a use, or a new form of an existing one, means adding or extending the function in the PostgreSQL
schema and its test. Prefer computing the value in Go and passing it in as a parameter, so that neither
engine needs a function the other lacks.

### 17. A `SELECT` whose callers depend on row order has an `ORDER BY`

Without one SQLite happens to return rows in rowid order. PostgreSQL makes no such promise.

### 18. Portable as written

`RETURNING`, `ON CONFLICT ... DO UPDATE`, `ON CONFLICT ... DO NOTHING`, `excluded.`, `SUBSTR`, `LENGTH`, `||`,
`COALESCE`, `CAST(... AS TEXT)`, and `sqlc.embed` need no change.

## Per-engine queries

A query that cannot be shared is written twice, once in `sql/queries/sqlite/` and once in
`sql/queries/postgres/`, with **the same query name and the same generated Go signature**. Everything else
stays in the shared directory.

- **What qualifies:** a construct with no form both engines accept. Today that is rule 15, the IN-list over a
  parameter, and the per-engine set is exactly `ListJobs` and `ListUserJobs`.
- **File names carry the engine:** `sql/queries/sqlite/<topic>_sqlite.sql` and
  `sql/queries/postgres/<topic>_postgres.sql`. sqlc names a generated file after the source file's base name,
  so a `jobs.sql` in an engine directory would merge into the shared `jobs.sql.go`.
- **Both files change together.** Adding, renaming or removing a query in one engine's file without the other
  leaves the two generated packages with different methods.

> **Status.** The engine directories appear with the PostgreSQL generation (#2957). Until then `ListJobs` and
> `ListUserJobs` sit in `jobs.sql` in their SQLite form.

## sqlc traps

- **sqlc never deletes a generated file.** Rename or remove a `.sql` file and its old `*.sql.go` stays in
  `internal/db`, still compiled, until you delete it by hand.
- **The same query name in two files is a `duplicate query name` error**, including a query left behind in the
  shared file after it was copied to an engine directory.
- **Regenerate and commit.** After any change here run `make generate/backend/sqlc` and commit the result; CI
  regenerates and fails on a diff.
