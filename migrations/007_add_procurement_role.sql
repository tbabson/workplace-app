-- Add procurement role to user_role enum
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'procurement';
