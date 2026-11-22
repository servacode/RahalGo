-- Deleting a coverage zone (owner report 2026-11-22: "coverage won't delete, it gives an error —
-- so I can add one zone that covers everything"). orders.zone_id referenced delivery_zones with
-- no ON DELETE rule, so any zone that ever had an order could not be removed. Past orders keep
-- everything they need (fees are snapshotted on the order); only the zone link is cleared.
-- Zones with open orders are still refused in catalog.DeleteZone with a clear message.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_zone_id_fkey;
ALTER TABLE orders
    ADD CONSTRAINT orders_zone_id_fkey
    FOREIGN KEY (zone_id) REFERENCES delivery_zones (id) ON DELETE SET NULL;
