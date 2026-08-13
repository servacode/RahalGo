// Command moneycheck يفحص تماسكَ المال في القاعدة — لا يكتب شيئاً.
//
// (طلبُ المالك ٢٠٢٦-٠٨-٠٨: «الأهمّ أن تعطيني نتيجةً حقيقيّةً… والدفعاتُ
//
//	الماليّة تعمل كلُّها بشكلٍ صحيح».)
//
// # لماذا أداةٌ لا اختبار
//
// الاختبارُ يفحص مسلكاً كتبه أحد. **وهذا يفحص ما وقع فعلاً**: كلَّ رصيدٍ
// وكلَّ قيدٍ وكلَّ طلبٍ في القاعدة كما هي الآن.
//
// **ويُشغَّل على الإنتاج بلا خوف**: لا يكتب حرفاً.
//
// # ما يفحصه
//
//	١ · رصيدُ كلّ محفظةٍ = مجموعُ حركاتها
//	٢ · محتجَزُ كلّ صندوقٍ = مجموعُ قيوده
//	٣ · لا رصيدَ سالبٌ إلّا الخزينة
//	٤ · كلُّ طلبٍ مسلَّمٍ له مستحقُّ متجرٍ مقيَّد
//	٥ · كلُّ طلبٍ مدفوعٍ من المحفظة له خصمٌ مقابل
//	٦ · لا قيدَ بصفرٍ ولا نوعٍ مجهول
//	٧ · كلُّ سحبٍ مدفوعٍ له خصمٌ في المحفظة
//	٨ · لا تعويضَ مكرَّرٌ لطلبٍ واحد
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type check struct {
	name  string
	query string
	// hint ما يُقال حين يُوجد خللٌ — لا وصفُ الاستعلام.
	hint string
}

// كلُّ فحصٍ يردّ الصفوفَ المختلّة. صفرُ صفوفٍ = سليم.
var checks = []check{
	{
		name: "للمنصّة خزينةٌ واحدة",
		hint: "بلا خزينةٍ يُتجاهَل مصروفُ المنصة بصمت — فتقريرُ الخسائر يبقى فارغاً أبداً",
		query: `
			SELECT 'عددُ الخزائن' AS الحالة, count(*) AS العدد
			FROM wallets WHERE is_treasury
			HAVING count(*) <> 1`,
	},
	{
		name: "رصيدُ المحفظة = مجموعُ حركاتها",
		hint: "رصيدٌ لا يطابق قيودَه — فكشفُ الحساب يقول غيرَ ما في الجيب",
		query: `
			SELECT u.full_name, w.balance, COALESCE(SUM(t.amount), 0) AS مجموع_القيود
			FROM wallets w
			JOIN users u ON u.id = w.user_id
			LEFT JOIN wallet_transactions t ON t.user_id = w.user_id
			GROUP BY u.full_name, w.balance
			HAVING w.balance <> COALESCE(SUM(t.amount), 0)`,
	},
	{
		name: "محتجَزُ الصندوق = مجموعُ قيوده",
		hint: "صندوقٌ لا يطابق قيودَه — فتسويةُ السائق تُبنى على رقمٍ خاطئ",
		query: `
			SELECT u.full_name, b.held, COALESCE(SUM(e.amount), 0) AS مجموع_القيود
			FROM driver_cash_boxes b
			JOIN users u ON u.id = b.driver_id
			LEFT JOIN driver_cash_entries e ON e.driver_id = b.driver_id
			GROUP BY u.full_name, b.held
			HAVING b.held <> COALESCE(SUM(e.amount), 0)`,
	},
	{
		name: "لا رصيدَ سالبٌ إلّا الخزينة",
		hint: "محفظةٌ بالسالب — ومالٌ صُرف ولم يكن موجوداً",
		query: `
			SELECT u.full_name, w.balance
			FROM wallets w JOIN users u ON u.id = w.user_id
			WHERE w.balance < 0 AND NOT w.is_treasury`,
	},
	{
		name: "كلُّ طلبٍ مسلَّمٍ له مستحقُّ متجر",
		hint: "طلبٌ سُلّم ولم يُقيَّد مستحقُّ متجره — فالمتجرُ لم يُدفع له",
		// **والطلبُ الخاصُّ مستثنًى** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩): لا متجرَ له
		// ولا مالَ للمنصّة فيه. **السائقُ يدفع من جيبه ويستردّ عند التسليم**،
		// والمنصّةُ توثّق ولا تحاسب.
		//
		// **ولولا الاستثناء لأنذر الدفترُ على كلّ طلبٍ خاصٍّ يُسلَّم** — فيُقرأ
		// خللاً وهو الصواب، **وأداةٌ تُنذر كذباً تُهمَل ثمّ لا تُقرأ يوم تصدق.**
		query: `
			SELECT o.number, o.total, o.status
			FROM orders o
			WHERE o.status = 'delivered' AND o.kind <> 'custom'
			  AND NOT EXISTS (
				SELECT 1 FROM wallet_transactions t
				WHERE t.ref = o.id::text AND t.kind = 'merchant_earning')`,
	},
	{
		name: "كلُّ طلبٍ من المحفظة له خصمٌ مقابل",
		hint: "طلبٌ دُفع من المحفظة ولم يُخصم — فالزبونُ أخذ بلا مقابل",
		query: `
			SELECT o.number, o.wallet_paid
			FROM orders o
			WHERE o.wallet_paid > 0 AND o.status = 'delivered'
			  AND NOT EXISTS (
				SELECT 1 FROM wallet_transactions t
				WHERE t.ref = o.id::text AND t.kind = 'order_payment')`,
	},
	{
		name:  "لا قيدَ بصفر",
		hint:  "قيدٌ بمبلغِ صفرٍ — ضجيجٌ في الكشف بلا معنى",
		query: `SELECT id, user_id, kind FROM wallet_transactions WHERE amount = 0`,
	},
	{
		name: "كلُّ سحبٍ مدفوعٍ له خصم",
		hint: "سحبٌ حالتُه «مدفوع» ولا خصمَ له — فالمالُ خرج من الورق لا من المحفظة",
		query: `
			SELECT p.amount, u.full_name
			FROM payout_requests p JOIN users u ON u.id = p.user_id
			WHERE p.status = 'paid'
			  AND NOT EXISTS (
				SELECT 1 FROM wallet_transactions t
				WHERE t.ref = p.id::text AND t.kind = 'payout')`,
	},
	{
		name: "لا تعويضَ مكرَّرٌ لطلب",
		hint: "طلبٌ عُوِّض أكثرَ من مرّة — وهو ما أُصلح ٢٠٢٦-٠٨-٠٨",
		query: `
			SELECT t.ref, count(*) AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'compensation' AND t.ref <> ''
			GROUP BY t.ref HAVING count(*) > 1`,
	},
	{
		name: "لا تعويضَ شكوى مكرَّر",
		hint: "شكوى عُوِّضت أكثرَ من مرّة — وهو ما أُصلح ٢٠٢٦-٠٨-٠٧",
		query: `
			SELECT tk.number, count(*) AS مرّات
			FROM tickets tk
			JOIN wallet_transactions t ON t.ref = tk.id::text AND t.kind = 'compensation'
			GROUP BY tk.number HAVING count(*) > 1`,
	},
	{
		name: "لا مستحقَّ متجرٍ مكرَّر",
		hint: "طلبٌ قُيّد مستحقُّه مرّتين — فالمتجرُ قُبض له ضِعف",
		query: `
			SELECT t.ref, count(*) AS مرّات
			FROM wallet_transactions t
			WHERE t.kind = 'merchant_earning'
			GROUP BY t.ref HAVING count(*) > 1`,
	},
	{
		name: "مجاميعُ الطلب متّسقة",
		hint: "طلبٌ مجموعُه لا يساوي أجزاءَه — فالفاتورةُ تكذب",
		query: `
			SELECT number, subtotal, delivery_fee, discount, total
			FROM orders
			WHERE total <> subtotal + delivery_fee - discount`,
	},
	{
		name: "الدفعُ يغطّي المجموع",
		hint: "طلبٌ مدفوعُه لا يساوي مجموعَه — فرقٌ ضائعٌ لا يعرف أحدٌ أين ذهب",
		query: `
			SELECT number, total, wallet_paid, cash_due
			FROM orders
			-- **والخاصُّ خارجَ السؤال** — (قرارُ المالك ٢٠٢٦-٠٨-١٣:
			-- يُكتب إجماليُّه ليُقرأ).
			--
			-- **والمنصّةُ لا تحاسب فيه**: يدفع السائقُ من جيبه ويقبض
			-- بيده، فلا مدفوعُ محفظةٍ ولا نقدٌ مستحقّ — وسؤالُ
			-- «أيغطّي المدفوعُ المجموع؟» **يُسأل عن مالٍ مرَّ بالمنصّة**،
			-- وهذا لم يمرّ بها.
			WHERE status = 'delivered' AND kind <> 'custom'
			  AND wallet_paid + cash_due <> total`,
	},
}

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL غير مضبوط")
		os.Exit(2)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "اتّصال:", err)
		os.Exit(2)
	}
	defer pool.Close()

	bad := 0
	for _, c := range checks {
		rows, err := pool.Query(ctx, c.query)
		if err != nil {
			fmt.Printf("✗ %-42s تعذّر الفحص: %v\n", c.name, err)
			bad++
			continue
		}
		var found [][]any
		for rows.Next() {
			v, _ := rows.Values()
			found = append(found, v)
		}
		rows.Close()
		if len(found) == 0 {
			fmt.Printf("✓ %s\n", c.name)
			continue
		}
		bad++
		fmt.Printf("✗ %s — %d حالة\n   %s\n", c.name, len(found), c.hint)
		for i, f := range found {
			if i >= 5 {
				fmt.Printf("   … و%d غيرها\n", len(found)-5)
				break
			}
			fmt.Printf("   · %v\n", f)
		}
	}

	fmt.Println()
	if bad == 0 {
		fmt.Printf("دفترُ المال متماسك — %d فحصاً بلا خلل\n", len(checks))
		return
	}
	fmt.Printf("**خللٌ في %d فحصاً من %d**\n", bad, len(checks))
	os.Exit(1)
}
