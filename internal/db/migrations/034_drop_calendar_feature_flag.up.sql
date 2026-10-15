-- The calendar left beta (#2610). An install that had the flag off stored it
-- as featureFlags.calendar = false. Deleting the row leaves the flag unset,
-- and with the flag gone from the registry the calendar is on for everyone.
DELETE FROM settings
WHERE
    key = 'featureFlags.calendar';
