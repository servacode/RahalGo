package support_test

// **حلّان متزامنان لشكوى واحدة — والتعويضُ يُدفع مرّتين.**
//
// (فحصُ المشروع ٢٠٢٦-٠٨-٠٧، بقرار المالك: «نبدأ إذاً».)
//
// # المرض
//
// حلُّ الشكوى كان أربعَ خطواتٍ بلا معاملةٍ ولا قفل:
//
//	SELECT status FROM tickets WHERE id = $1     ← بلا FOR UPDATE
//	if status == "resolved" { رفض }
//	UPDATE tickets SET status='resolved', compensation=$3
//	wallet.Apply(compensation)                    ← معاملةٌ منفصلة
//
// **والترتيبُ مقلوب**: الحالةُ تُكتب قبل المال. فإن سقط القيدُ **بقيت
// التذكرةُ تقول «عُوِّض خمسةَ آلاف» ولا خمسةَ آلافٍ في محفظته** — خسارةٌ
// صامتةٌ للزبون لا يكشفها سجلٌّ.
//
// **والمتزامنان يُعوّضان مرّتين** — يقرآن «مفتوحة» كلاهما فيمرّان.
//
// # ولماذا اختبارُ تزامنٍ لا مراجعة
//
// **العينُ لا ترى سباقاً في شيفرةٍ كلُّ سطرٍ فيها صحيح.** والعطبُ في
// المسافة بين السطرين، لا في سطر.

import (
	"context"
	"sync"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestResolveCompensatesOnlyOnce(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	walletSvc := wallet.NewService(pool)
	svc := support.NewService(pool, nil, walletSvc)

	agent := testdb.NewUser(t, pool, "admin")
	// **وزبونٌ واحدٌ لكلّ الجولات** — والقيدُ مربوطٌ برقم التذكرة فلا تختلط
	// جولةٌ بأخرى. **وقاعدةُ الاختبار لا تُنظَّف**: كلُّ مستخدمٍ يُنشأ يبقى،
	// وعشرون في كلّ تشغيلٍ تدفع تسلسلَ الأرقام إلى مدىً مأهولٍ فيتصادم.
	customer := testdb.NewUser(t, pool, "customer")

	// **والنافذةُ بين القراءة والكتابة ضيّقة** — فجولةٌ واحدةٌ قد تفوتها.
	// **واختبارٌ لا يسقط قبل الإصلاح لا يُثبت شيئاً**، فتُعاد الحالةُ على
	// تذاكرَ جديدةٍ حتّى تُلتقط أو يثبت أنّها لا تقع.
	const rounds = 20
	const n = 8
	const compensation int64 = 5_000

	worst := 0
	var seen int64
	for round := 0; round < rounds; round++ {
		var ticketID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO tickets (customer_id, subject, status)
			VALUES ($1, 'other', 'open') RETURNING id`, customer).Scan(&ticketID); err != nil {
			t.Fatalf("تعذّر إنشاء تذكرة: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM tickets WHERE id = $1`, ticketID)
		})

		var start, done sync.WaitGroup
		start.Add(1)
		for i := 0; i < n; i++ {
			done.Add(1)
			go func() {
				defer done.Done()
				start.Wait()
				_, _ = svc.Resolve(context.Background(), agent, ticketID, "عولجت", compensation, "127.0.0.1")
			}()
		}
		start.Done()
		done.Wait()

		var paid int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM wallet_transactions
			WHERE user_id = $1 AND kind = 'compensation' AND ref = $2`, customer, ticketID).Scan(&paid); err != nil {
			t.Fatalf("تعذّر عدُّ القيود: %v", err)
		}
		// **والفرقُ لا الرصيدُ المطلق** — الزبونُ نفسُه يعبر الجولاتِ كلَّها.
		var balance int64
		_ = pool.QueryRow(ctx, `SELECT COALESCE(balance,0) FROM wallets WHERE user_id = $1`, customer).Scan(&balance)
		gained := balance - seen
		seen = balance

		if paid > worst {
			worst = paid
		}
		if paid != 1 || gained != compensation {
			t.Fatalf("الجولةُ %d: **التعويضُ دُفع %d مرّةً والرصيدُ %d** — والمنتظَر مرّةً واحدةً و%d. "+
				"**والعلاجُ معاملةٌ واحدةٌ تلفّ القراءةَ والحالةَ والقيد، وقفلٌ "+
				"`FOR UPDATE` على صفّ التذكرة، والمالُ قبل الحالة لا بعدها.**",
				round+1, paid, gained, compensation)
		}
	}
	t.Logf("%d جولةً × %d نداءً متزامناً — وأكثرُ ما دُفع في جولةٍ: %d", rounds, n, worst)
}
