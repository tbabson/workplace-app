-- Set the database's default timezone to the application timezone, so direct
-- connections (psql, the Neon SQL editor, reporting tools) show timestamps and
-- CURRENT_DATE in local time. Keep in sync with APP_TIMEZONE.
--
-- The app itself does not depend on this: connection poolers (e.g. Neon's
-- PgBouncer endpoint) override the session timezone, so the app computes
-- "today" itself (package clock) and uses AT TIME ZONE when formatting.
DO $$
BEGIN
    EXECUTE format('ALTER DATABASE %I SET timezone TO %L', current_database(), 'Africa/Lagos');
END $$;
