-- Complaint with a pending compensation stays visible as awaiting_finance and only
-- closes after finance decides (owner decision 2026-10-04): approve -> resolved with the
-- amount; reject -> back to support (in_progress), which may propose again.
ALTER TABLE tickets DROP CONSTRAINT IF EXISTS tickets_status_check;
ALTER TABLE tickets ADD CONSTRAINT tickets_status_check
    CHECK (status IN ('open', 'in_progress', 'awaiting_finance', 'resolved'));
