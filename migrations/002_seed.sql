-- Seed: initial super_admin account
-- email: tbabson20@gmail.com  |  password: admin1234
INSERT INTO users (name, email, password_hash, role, is_active)
VALUES (
    'Admin',
    'tbabson20@gmail.com',
    '$2a$10$g7rIgQRvRt110gL5dsvJ5eFmJcFYpyaOmkc6am88msYw5WkzGwCbm',
    'super_admin',
    true
)
ON CONFLICT (email) DO NOTHING;
