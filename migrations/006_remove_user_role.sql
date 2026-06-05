-- Update any existing users with role 'user' to 'staff'
UPDATE users SET role = 'staff' WHERE role = 'user';

-- Change the column default to 'staff'
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'staff';

-- Remove the 'user' value from the enum
-- PostgreSQL does not support DROP VALUE directly, so we rename and recreate
ALTER TYPE user_role RENAME TO user_role_old;

CREATE TYPE user_role AS ENUM ('super_admin', 'dept_head', 'staff');

ALTER TABLE users
  ALTER COLUMN role DROP DEFAULT,
  ALTER COLUMN role TYPE user_role USING role::text::user_role,
  ALTER COLUMN role SET DEFAULT 'staff';

DROP TYPE user_role_old;
