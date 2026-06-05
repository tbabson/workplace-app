-- ═══════════════════════════════════════════════════════════════════
-- 005_new_features.sql
-- Tasks, Milestones, Comments, Status History, Budget, Files,
-- Leave, Reviews, Claims, Assets, Chat enhancements
-- ═══════════════════════════════════════════════════════════════════

-- ─── Project Tasks ──────────────────────────────────────────────────
DO $$ BEGIN
    CREATE TYPE task_status AS ENUM ('todo', 'in_progress', 'done');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE project_tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_id   UUID REFERENCES project_tasks(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      task_status NOT NULL DEFAULT 'todo',
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date    DATE,
    created_by  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Project Milestones ─────────────────────────────────────────────
CREATE TABLE project_milestones (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    due_date    DATE NOT NULL,
    achieved_at TIMESTAMPTZ,
    created_by  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Project Comments ───────────────────────────────────────────────
CREATE TABLE project_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_id  UUID REFERENCES project_comments(id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Project Status History ─────────────────────────────────────────
CREATE TABLE project_history (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    changed_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    field      VARCHAR(50) NOT NULL,
    from_value VARCHAR(100),
    to_value   VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Project Budget & Expenses ──────────────────────────────────────
ALTER TABLE projects ADD COLUMN IF NOT EXISTS budget DECIMAL(15,2);

CREATE TABLE project_expenses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    amount      DECIMAL(15,2) NOT NULL,
    date        DATE NOT NULL DEFAULT CURRENT_DATE,
    logged_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Project File Attachments ───────────────────────────────────────
CREATE TABLE project_files (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    url         TEXT NOT NULL,
    size        BIGINT NOT NULL DEFAULT 0,
    uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Leave Management ───────────────────────────────────────────────
DO $$ BEGIN
    CREATE TYPE leave_type AS ENUM ('annual','sick','maternity','paternity','unpaid','other');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE leave_status AS ENUM ('pending','approved','rejected','cancelled');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE leave_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        leave_type NOT NULL,
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    days        INT NOT NULL,
    reason      TEXT NOT NULL DEFAULT '',
    status      leave_status NOT NULL DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE leave_balances (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type    leave_type NOT NULL,
    year    INT NOT NULL,
    total   INT NOT NULL DEFAULT 0,
    used    INT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, type, year)
);

-- ─── Performance Reviews ────────────────────────────────────────────
CREATE TABLE performance_reviews (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period      VARCHAR(50) NOT NULL,
    rating      INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    goals       TEXT NOT NULL DEFAULT '',
    comments    TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Expense Claims ─────────────────────────────────────────────────
DO $$ BEGIN
    CREATE TYPE claim_status AS ENUM ('pending','approved','rejected','paid');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE expense_claims (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    amount      DECIMAL(15,2) NOT NULL,
    date        DATE NOT NULL DEFAULT CURRENT_DATE,
    description TEXT NOT NULL DEFAULT '',
    receipt_url TEXT,
    status      claim_status NOT NULL DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    paid_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Asset Management ───────────────────────────────────────────────
DO $$ BEGIN
    CREATE TYPE asset_status AS ENUM ('available','assigned','maintenance','retired');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE assets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    category      VARCHAR(100) NOT NULL,
    serial_number VARCHAR(255),
    status        asset_status NOT NULL DEFAULT 'available',
    assigned_to   UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at   TIMESTAMPTZ,
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Chat Enhancements ──────────────────────────────────────────────
ALTER TYPE room_type ADD VALUE IF NOT EXISTS 'direct';

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS is_deleted  BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE chat_room_members
    ADD COLUMN IF NOT EXISTS last_read_at TIMESTAMPTZ;

CREATE TABLE message_reactions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji      VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

-- ─── Indexes ────────────────────────────────────────────────────────
CREATE INDEX idx_tasks_project    ON project_tasks(project_id);
CREATE INDEX idx_tasks_parent     ON project_tasks(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_milestones_proj  ON project_milestones(project_id);
CREATE INDEX idx_comments_project ON project_comments(project_id);
CREATE INDEX idx_history_project  ON project_history(project_id);
CREATE INDEX idx_expenses_project ON project_expenses(project_id);
CREATE INDEX idx_files_project    ON project_files(project_id);
CREATE INDEX idx_leave_user       ON leave_requests(user_id);
CREATE INDEX idx_leave_status     ON leave_requests(status);
CREATE INDEX idx_reviews_user     ON performance_reviews(user_id);
CREATE INDEX idx_claims_user      ON expense_claims(user_id);
CREATE INDEX idx_assets_assigned  ON assets(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_reactions_msg    ON message_reactions(message_id);
