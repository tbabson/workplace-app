-- Sub-departments: a department may sit under one parent department
-- (e.g. HR under Admin). Only one level of nesting is allowed; the app enforces it.
ALTER TABLE departments
    ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES departments(id) ON DELETE SET NULL;

-- Department names are unique, ignoring case.
CREATE UNIQUE INDEX IF NOT EXISTS departments_name_unique ON departments (lower(name));

-- The company's departments.
INSERT INTO departments (name) VALUES
    ('Account'),
    ('Admin'),
    ('Procurement'),
    ('Marketing'),
    ('Press'),
    ('Pre-Press'),
    ('Finishing')
ON CONFLICT ((lower(name))) DO NOTHING;

-- Sub-departments.
INSERT INTO departments (name, parent_id)
SELECT 'HR', id FROM departments WHERE lower(name) = 'admin'
ON CONFLICT ((lower(name))) DO NOTHING;

INSERT INTO departments (name, parent_id)
SELECT 'Digital Marketing', id FROM departments WHERE lower(name) = 'marketing'
ON CONFLICT ((lower(name))) DO NOTHING;
