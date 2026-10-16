-- Slides left beta (#2869). An install that had the flag off stored it
-- as featureFlags.slides = false. Deleting the row leaves the flag unset,
-- and with the flag gone from the registry slides is on for everyone.
DELETE FROM settings
WHERE
    key = 'featureFlags.slides';
