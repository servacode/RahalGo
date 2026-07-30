-- مناطق التغطية: مضلعات جغرافية برسوم توصيل وحد أدنى للطلب لكل منطقة
CREATE TABLE delivery_zones (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name         text NOT NULL,
    polygon      geography(Polygon, 4326) NOT NULL,
    delivery_fee bigint NOT NULL DEFAULT 0 CHECK (delivery_fee >= 0),  -- ل.س
    min_order    bigint NOT NULL DEFAULT 0 CHECK (min_order >= 0),     -- ل.س
    active       boolean NOT NULL DEFAULT true,
    sort_order   int NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX delivery_zones_polygon_idx ON delivery_zones USING GIST (polygon);
