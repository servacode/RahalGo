-- Per-store flag: does the store own the Al-Ameen program (owner decision 2026-11-06).
-- Off: nothing about Al-Ameen shows for the store. Stores that already have Ameen codes
-- on items or used price sync start ON so nothing disappears for them.
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS uses_ameen boolean NOT NULL DEFAULT false;

UPDATE merchants m SET uses_ameen = true
 WHERE m.uses_ameen = false
   AND (m.price_sync_token_hash IS NOT NULL
        OR m.price_sync_mapping IS NOT NULL
        OR EXISTS (SELECT 1 FROM price_sync_runs r WHERE r.merchant_id = m.id)
        OR EXISTS (SELECT 1 FROM menu_items i
                    WHERE i.merchant_id = m.id
                      AND (COALESCE(i.external_code, '') <> '' OR COALESCE(i.barcode, '') <> '')));
