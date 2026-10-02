-- ══════════════════════════════════════════════════════════════════════
-- **أمرُ الإدارة عند باب الزبون** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢)
-- ══════════════════════════════════════════════════════════════════════
--
-- «يبقى الطلبُ مع السائق إلى أن تُحلّ القصّة… وقتها الإدارةُ هي تُنهي الطلبَ من
-- عندها، يصل أمرٌ للسائق — مثلاً العودة إلى المكتب.»
--
-- **السائقُ لا يُنهي الطلبَ عند الباب** — يُبلّغ، **والإدارةُ تقرّر**: سلّم الآن
-- (`deliver_now`) أو عُد إلى المكتب (`return_to_office`). **والأمرُ يُكتب على الطلب**
-- فيقرؤه تطبيقُ السائق ولو ضاع الإشعار.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS door_instruction text NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS door_instruction_note text NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS door_instruction_at timestamptz;
