-- Add 'user' role and make it the default
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'user';
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'user';
