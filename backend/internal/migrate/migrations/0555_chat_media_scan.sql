-- Chat media scan (owner decision 2026-11-20): photos that contain a phone number are held
-- (not delivered) until the office releases them; voice notes are transcribed after delivery and
-- flagged with a warning when a number is heard.
ALTER TABLE order_messages ADD COLUMN IF NOT EXISTS held_at     timestamptz;
ALTER TABLE order_messages ADD COLUMN IF NOT EXISTS released_at timestamptz;
ALTER TABLE order_messages ADD COLUMN IF NOT EXISTS released_by uuid REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE order_messages ADD COLUMN IF NOT EXISTS scan_text   text;
ALTER TABLE order_messages ADD COLUMN IF NOT EXISTS scanned_at  timestamptz;

CREATE INDEX IF NOT EXISTS order_messages_held_idx ON order_messages (held_at)
    WHERE held_at IS NOT NULL AND released_at IS NULL;
