-- Chat left beta (#2577). An install that had the flag off stored it as
-- featureFlags.chat = false. Deleting the row leaves the flag unset, and
-- with the flag gone from the registry chat is on for everyone.
DELETE FROM settings
WHERE
    key = 'featureFlags.chat';
