-- دوام السائق.
--
-- الإسناد كان يدوياً بالكامل: موظّف العمليات يتّصل بالسائق ليسأله إن كان يعمل.
-- والسائق نفسه لا يملك أن يقول «أنا متاح» أو «انتهى دوامي» — فيُتّصل به وهو نائم،
-- ولا يُتّصل به وهو ينتظر.
--
-- **علَمٌ يرفعه هو لا يُفترض عنه**: التوفّر قرارُ صاحبه، واستنتاجُه من آخر ظهور
-- يخطئ في الاتجاهين — من نسي تطبيقه مفتوحاً يبدو متاحاً، ومن يقود ولا ينظر
-- يبدو غائباً.
ALTER TABLE users ADD COLUMN on_shift boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN shift_started_at timestamptz;

-- شاشة العمليات تسأل «من يعمل الآن؟» في كل ثانية
CREATE INDEX users_on_shift_idx ON users (on_shift) WHERE on_shift;
