-- ══════════════════════════════════════════════════════════════════════
-- **زوّارُ الموقع وتحميلاتُ التطبيقات** (طلبُ المالك ٢٠٢٦-١٠-٠٧)
-- ══════════════════════════════════════════════════════════════════════
--
-- صفٌّ واحدٌ لكلّ شخصٍ في كلّ يومٍ لكلّ نوع — **فمن زار عشرَ مرّاتٍ في يومٍ
-- يُعدّ مرّة، وتنزيلٌ مقطّعٌ بطلباتِ مدىً يُعدّ مرّة.**
--
--   kind     'visit' أو 'download:customer|driver|merchant|rep'
--   visitor  بصمةٌ مقطوعةٌ (٣٢ خانة) من sha256(العنوان|المتصفّح) — **ولا
--            يُحفظ عنوانٌ خامّ.**
--   day      يومُ دمشق لا يومُ غرينتش.
CREATE TABLE IF NOT EXISTS site_hits (
    day     date NOT NULL,
    kind    text NOT NULL,
    visitor text NOT NULL,
    PRIMARY KEY (day, kind, visitor)
);

-- **والعدُّ «منذ البداية» بالنوع والزائر** — بلا مسحِ الجدول كلِّه.
CREATE INDEX IF NOT EXISTS site_hits_kind_visitor ON site_hits (kind, visitor);
