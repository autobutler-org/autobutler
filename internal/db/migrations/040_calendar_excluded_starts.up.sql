-- A repeating event can skip occurrence starts (#3215). excluded_starts is a
-- JSON array of those starts, UTC instants in the same text form as
-- starts_at, or NULL when the series skips nothing. A one-off event never
-- carries any. The shape is checked in Go: a JSON array is a poor CHECK on
-- either engine.
ALTER TABLE calendar_events ADD COLUMN excluded_starts TEXT;
