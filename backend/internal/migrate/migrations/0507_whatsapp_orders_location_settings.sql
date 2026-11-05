-- ══════════════════════════════════════════════════════════════════════
-- **طلباتُ الواتساب — ملحقُ المالك** (٢٠٢٦-١١-٠٥، بعد ٠٥٠٠)
-- ══════════════════════════════════════════════════════════════════════
--
-- «من لوحة التحكم مثل توصيلة متجر وامانة نحط الرقم صاحب الطلب ويشارك موقعه ويجي لعندنا»
-- و«اجرة التوصيل للواتس اب ايضا تكون ليها خيارات مثل الاجرة وحصة المنصة».
--
--	أ · «اطلب الموقع» من نموذج الواتساب — الصفُّ نفسُه بنوع `whatsapp` صاحبُه المكتب
--	ب · الطلبُ لا ينتظر الموقع — يُنشأ بالعنوان، والموقعُ يُكتب فيه حين يصل (لا عمودَ جديد)
--	د · «اكتبلنا عنوانك بالتفصيل» مرّةً واحدةً بعد دقائق بلا موقع (`followup_sent_at`)
--	هـ · مفاتيحُ أجرة «طلب واتساب» وحصّتِه — تُنسخ من «لدي توصيلة» فلا يتبدّل رقمٌ يومَ النشر

-- ── هـ · مفاتيحُ أجرة «طلب واتساب» وحصّتِه ──────────────────────────────
--
-- **افتراضُها قيمُ «لدي توصيلة» المحفوظةُ الآن** — فطلبُ الواتساب يوم النشر يُسعَّر كما كان يُسعَّر
-- قبله، **حتّى يعدّلها المالك.** و`ON CONFLICT DO NOTHING`: قيمةٌ ضُبطت قبلها لا تُمسّ.
--
-- **والأجرةُ الثابتةُ لـ«لدي توصيلة»** (`delivery.merchant_fee` فوق الصفر) **كانت تسبق القاعدة** —
-- ولا ثابتةَ لطلب الواتساب، **فتُنسخ أساساً بمسافةٍ مطفأة** (وهو الرقمُ نفسُه)، وتسبق النسخَ العامّ.
INSERT INTO app_settings (key, value)
SELECT m.new_key, m.val
  FROM app_settings s
 CROSS JOIN LATERAL (VALUES ('delivery.whatsapp_base_fee', s.value),
                            ('delivery.whatsapp_by_distance', 'false'::jsonb)) AS m(new_key, val)
 WHERE s.key = 'delivery.merchant_fee'
   AND jsonb_typeof(s.value) = 'number' AND (s.value #>> '{}')::numeric > 0
ON CONFLICT (key) DO NOTHING;

INSERT INTO app_settings (key, value)
SELECT m.new_key, s.value
  FROM (VALUES ('delivery.merchant_base_fee',   'delivery.whatsapp_base_fee'),
               ('delivery.merchant_by_distance', 'delivery.whatsapp_by_distance'),
               ('delivery.merchant_per_km',      'delivery.whatsapp_per_km'),
               ('delivery.merchant_max_fee',     'delivery.whatsapp_max_fee'),
               ('delivery.share_merchant_mode',  'delivery.share_whatsapp_mode'),
               ('delivery.share_merchant',       'delivery.share_whatsapp')) AS m(old_key, new_key)
  JOIN app_settings s ON s.key = m.old_key
ON CONFLICT (key) DO NOTHING;
