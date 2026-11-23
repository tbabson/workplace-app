-- Intentionally empty.
--
-- This file used to insert a super_admin with a hard-coded password. The first
-- super_admin is now created by the app on startup from the
-- BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD environment variables
-- (only when no super_admin exists yet). The file is kept so migration
-- numbering stays the same.
SELECT 1;
