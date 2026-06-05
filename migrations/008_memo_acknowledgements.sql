-- ═══════════════════════════════════════════════════════════════════
-- 008_memo_acknowledgements.sql
-- Per-user memo acknowledgement tracking
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS memo_acknowledgements (
    memo_id         UUID NOT NULL REFERENCES memos(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (memo_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_memo_ack_memo ON memo_acknowledgements(memo_id);
CREATE INDEX IF NOT EXISTS idx_memo_ack_user ON memo_acknowledgements(user_id);
