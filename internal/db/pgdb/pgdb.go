// Package pgdb is the PostgreSQL half of Quark's query layer: the methods sqlc generates from sql/queries/ and
// sql/queries/postgres/ against the schema in internal/db/migrations_postgres/. Everything in it but this file
// is sqlc's output. Nothing outside internal/db calls it: db.NewQueries and db.InTx wrap it in an adapter that
// satisfies db.Querier whenever the connection they are given is a PostgreSQL one.
package pgdb
