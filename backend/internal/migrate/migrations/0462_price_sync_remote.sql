-- ══════════════════════════════════════════════════════════════════════
-- **ساعي الأمين بلا إعدادٍ على كمبيوتر المول** (قرارُ المالك ٢٠٢٦-١٠-١٩)
-- ══════════════════════════════════════════════════════════════════════
--
-- «ملف واحد يشتغل بالخلفية تلقائي… ما بدنا المحاسب يتلبك». **فالإعدادُ كلُّه هنا**
-- ويصل الساعيَ في سؤاله كلَّ دقيقة، **وما يجده الساعي يعود إلينا**:
--
--   price_sync_sql_query        جملةُ القراءة — يضبطها فريقُ المنصّة (قراءةٌ فقط)
--   price_sync_sql_connection   عنوانُ القاعدة إن لم يجدها الساعي وحده
--   price_sync_probe_table      جدولٌ يُعرف به قاعدةُ الأمين بين قواعد الجهاز
--   price_sync_folder           مجلّدُ ملفّات Excel — إن لم تكن قراءةٌ مباشرة
--   price_sync_currency         عملةُ أسعار الأمين
--   price_sync_stop             «إيقاف البرنامج» من اللوحة — يزيل نفسَه من الإقلاع
--   price_sync_agent_error      آخرُ خطأٍ عند الساعي — يُقرأ في اللوحة لا على كمبيوتر المحاسب
--   price_sync_agent_status     ما وجده: الطريقةُ والقاعدةُ والنسخة
--   price_sync_want_schema      «ابعث جداول الأمين» — ليُضبط سطرُ القراءة بلا زيارة
--   price_sync_schema           الجداولُ والأعمدةُ وعددُ الصفوف كما بعثها
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_sql_query text;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_sql_connection text;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_probe_table text NOT NULL DEFAULT 'mt000';
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_folder text;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_currency text NOT NULL DEFAULT 'SYP'
    CHECK (price_sync_currency IN ('SYP', 'USD'));
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_stop boolean NOT NULL DEFAULT false;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_agent_error text;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_agent_status jsonb;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_want_schema boolean NOT NULL DEFAULT true;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_schema jsonb;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_schema_at timestamptz;
