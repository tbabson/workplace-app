-- Add optional position and department_name fields to users
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS position        VARCHAR(255),
    ADD COLUMN IF NOT EXISTS department_name VARCHAR(255);
