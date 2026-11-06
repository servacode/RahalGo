-- **إنذارٌ واحدٌ لكلّ (طلب، متجر)** — طلبٌ بعدّة محطّات (قرارُ المالك ٢٠٢٦-١١-٠٦، ٦أ).
--
-- **رفضُ متجرٍ قسمَه ذنبُه** — يُكتب عليه إنذارٌ ويُعدّ مخالفةً كرفض طلبٍ عاديّ، **وشيلُ المكتب
-- يختار ذنبَه.** **وكان الفهرسُ إنذاراً واحداً لكلّ طلب** (`0090`) — فإنذارُ المتجر الثاني على
-- الطلب نفسِه يُبتلع، أو يقع على الأوّل.
--
-- **فيصير الفهرسُ لكلّ (طلب، متجر)** لإنذارات المتاجر، **ويبقى واحداً لكلّ طلب** لما لا متجرَ
-- له (إنذارُ سائقٍ أو زبونٍ يدويّ على طلب).
--
-- **وموضعا `ON CONFLICT` في `orders/warnings.go` تبدّلا في الالتزام نفسِه** — `ON CONFLICT`
-- بلا فهرسٍ فريدٍ يطابقه خطأٌ وقتَ التنفيذ (انظر `0090`).
--
-- **والبياناتُ القائمةُ تطابق الجديدَ من نفسها**: كانت واحداً لكلّ طلب، فهي واحدٌ لكلّ (طلب، متجر).
DROP INDEX IF EXISTS warnings_one_per_order;

CREATE UNIQUE INDEX IF NOT EXISTS warnings_one_per_order_store
    ON warnings (order_id, merchant_id) WHERE order_id IS NOT NULL AND merchant_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS warnings_one_per_order_other
    ON warnings (order_id) WHERE order_id IS NOT NULL AND merchant_id IS NULL;
