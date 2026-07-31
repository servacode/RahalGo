-- المفضّلة والبحث.

-- **المفضّلة**: الزبون يطلب من ثلاثة متاجر ويتصفّح ثلاثين ليجدها.
CREATE TABLE user_favorites (
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    merchant_id uuid NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, merchant_id)
);
CREATE INDEX user_favorites_user_idx ON user_favorites (user_id, created_at DESC);

-- **البحث**: فهرسان ثلاثيّان (trigram) على اسم المتجر واسم الصنف.
--
-- ولماذا trigram لا full-text؟ لأن العربية في `to_tsvector` تحتاج قاموساً
-- لغوياً غير متوفّر افتراضياً، ولأن زبوننا يكتب «شاورمه» و«شورما» و«شاورما» —
-- والبحث المتسامح للخطأ الإملائي (`ILIKE '%…%'` المدعوم بـtrigram) يخدمه أكثر
-- من مطابقةٍ صرفية دقيقة لا تجد شيئاً.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX merchants_name_trgm_idx ON merchants USING gin (name gin_trgm_ops);
CREATE INDEX menu_items_name_trgm_idx ON menu_items USING gin (name gin_trgm_ops);
