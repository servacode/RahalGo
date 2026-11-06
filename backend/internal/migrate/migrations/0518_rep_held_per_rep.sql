-- **عمولةٌ محجوزةٌ لكلّ (طلب، مندوب)** — طلبٌ بعدّة محطّات (قرارُ المالك ٢٠٢٦-١١-٠٦، ٢أ).
--
-- **كلُّ مندوبٍ يأخذ عمولتَه من ربح متجره هو** — فطلبٌ من متجرين لمندوبين اثنين يكتب عمولتين،
-- **وإن كان أحدُهما موقوفاً حُجزت عمولتُه وحدَه.** **وكان الفريدُ `order_id` وحدَه** (`0190`) —
-- فحجزُ المندوب الثاني على الطلب نفسِه يُبتلع.
--
-- **وموضعُ `ON CONFLICT` في `orders/accounts_rep_held.go` تبدّل في الالتزام نفسِه** — `ON CONFLICT`
-- بلا فريدٍ يطابقه خطأٌ وقتَ التنفيذ (انظر `0090`).
--
-- **والبياناتُ القائمةُ تطابق الجديدَ من نفسها**: كانت واحدةً لكلّ طلب، فهي واحدةٌ لكلّ (طلب، مندوب).
ALTER TABLE rep_held_commissions DROP CONSTRAINT IF EXISTS rep_held_commissions_order_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS rep_held_one_per_order_rep
    ON rep_held_commissions (order_id, rep_id);
