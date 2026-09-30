package qa

// ══════════════════════════════════════════════════════════════════════
//  **إجهادُ صفِّ المتجر — `REL-CONC-01`**
// ══════════════════════════════════════════════════════════════════════
//
// **وقع ٢٠٢٦-٠٩-٣٠**: زبونان يُنشئان طلبَين على متجرٍ واحدٍ معاً، **فيُردّ
// أحدهما بخمسمئة ولا طلبَ له** — `deadlock detected (SQLSTATE 40P01)`.
//
// # والسببُ كان ترقيةَ قفل
//
// `INSERT INTO orders` يأخذ `FOR KEY SHARE` على صفِّ المتجر ضمنيّاً
// (يفرضه `orders_merchant_id_fkey`)، **ثمّ كانت قراءةُ `settlement_method`
// تطلب `FOR UPDATE` على الصفِّ نفسِه** — **ترقيةٌ من مشتركٍ إلى حصريّ.**
// **فكلُّ معاملةٍ تحمل ما تنتظره الأخرى.**
//
// # ولماذا لا يكفي `D4-T10`
//
// **`D4-T10` زبونان ودورةٌ واحدة** — **والعطبُ احتماليٌّ** (قِيس: واحدٌ من
// عشرين بالحصريّ). **فبندٌ يقع مرّةً كلَّ عشرين يُقرأ تقلّباً ويُتجاوَز.**
//
// **وهذا يُجهد الصفَّ نفسَه بثمانيةِ زبائنَ في وقتٍ واحد** — فالاحتمالُ
// يصير يقيناً إن عاد العطب.
//
// # وما يُقاس ليس «لا خطأ» فقط
//
//	صفرُ خمسمئة       · ولا نداءَ يسقط
//	صفرُ تعارضٍ        · في سجلّ المحرّك
//	عددُ الطلبات = المنتظَر · لا زائدٌ ولا ناقص
//	مفتاحٌ واحدٌ ⇒ طلبٌ واحد · ولا تكرارَ ماليّ
//	أسلوبُ التسوية كما كان · ولا كتابةٌ ضائعة

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// ── أ · ثمانيةُ زبائنَ على متجرٍ واحدٍ معاً ─────────────────────────────
func TestConc_ManyCustomersOneMerchant_NoDeadlock(t *testing.T) {
	const customers = 8

	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	// **والسقفُ يُرفع** — فالمقيسُ هنا القفلُ لا حدُّ المفتوح.
	h.Setting("orders.max_open_per_customer", "5")
	treasury(t, h)
	f := h.Factory()
	dlBefore := deadlocks(t, h)

	item := h.NewItem(1000)
	users := make([]*User, customers)
	for i := range users {
		users[i] = f.NewUserWith("customer")
	}

	// **والانطلاقُ معاً** — بوّابةٌ واحدةٌ تُفتح، فلا يتقدّم أحدٌ بثوانٍ.
	var gate sync.WaitGroup
	gate.Add(1)
	var wg sync.WaitGroup
	res := make([]Res, customers)
	keys := make([]string, customers)
	for i := range users {
		keys[i] = uniq(fmt.Sprintf("conc-a-%d", i))
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			gate.Wait()
			res[i] = h.POSTKey("/api/v1/orders", users[i].Token, keys[i], orderBody(item, 1))
		}(i)
	}
	gate.Done()
	wg.Wait()

	// ── صفرُ خمسمئة ──
	created := 0
	for i, r := range res {
		switch {
		case r.Code == 201:
			created++
		case r.Code >= 500:
			t.Errorf("زبون %d: **ردٌّ %d** — %s", i, r.Code, r.String())
		default:
			// **ورفضٌ بقاعدةِ عملٍ مقبولٌ ويُسمّى** — لا يُخفى في «غير 201».
			t.Errorf("زبون %d: رُدّ %d بلا سببٍ متوقَّع — %s", i, r.Code, r.String())
		}
	}
	if created != customers {
		t.Fatalf("أُنشئ %d من %d — **وكلُّ زبونٍ مؤهَّلٌ يجب أن يُنشئ**", created, customers)
	}

	// ── صفرُ تعارضٍ في سجلّ المحرّك ──
	assertNoNewDeadlocks(t, h, dlBefore)

	// ── والعددُ في القاعدة هو المنتظَر: لا زائدٌ ولا ناقص ──
	var n int
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM orders WHERE merchant_id = $1::uuid`, item.MerchantID).Scan(&n); err != nil {
		t.Fatalf("عدُّ الطلبات: %v", err)
	}
	if n != customers {
		t.Fatalf("في القاعدة %d طلباً والمنتظَر %d — **زائدٌ أو ناقص**", n, customers)
	}

	// ── ولا كتابةٌ ضائعةٌ على صفّ المتجر ──
	var method string
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT settlement_method FROM merchants WHERE id = $1::uuid`, item.MerchantID).
		Scan(&method); err != nil {
		t.Fatalf("قراءةُ الأسلوب: %v", err)
	}
	if method == "" {
		t.Fatal("أسلوبُ التسوية فارغٌ بعد الإجهاد — **كتابةٌ ضائعة**")
	}
}

// ── ب · ومفتاحٌ واحدٌ متزامنٌ ⇒ طلبٌ واحدٌ وأثرٌ ماليٌّ واحد ─────────────
//
// **والقفلُ لو تغيّر فأفرج عن معاملتَين معاً** لَأنشأ المفتاحُ الواحدُ
// طلبَين — **وهذا يُمسكه.**
func TestConc_SameKeyParallel_CreatesOneOrder(t *testing.T) {
	const tries = 6

	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "5")
	treasury(t, h)
	f := h.Factory()
	dlBefore := deadlocks(t, h)

	item := h.NewItem(1000)
	u := f.NewUserWith("customer")
	key := uniq("conc-same-key")

	var gate sync.WaitGroup
	gate.Add(1)
	var wg sync.WaitGroup
	res := make([]Res, tries)
	for i := 0; i < tries; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			gate.Wait()
			res[i] = h.POSTKey("/api/v1/orders", u.Token, key, orderBody(item, 1))
		}(i)
	}
	gate.Done()
	wg.Wait()

	for i, r := range res {
		if r.Code >= 500 {
			t.Errorf("محاولة %d: **ردٌّ %d** — %s", i, r.Code, r.String())
		}
	}
	assertNoNewDeadlocks(t, h, dlBefore)

	var orders int
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, u.ID).Scan(&orders); err != nil {
		t.Fatalf("عدُّ الطلبات: %v", err)
	}
	if orders != 1 {
		t.Fatalf("مفتاحٌ واحدٌ أنشأ %d طلباً — **والعقدُ طلبٌ واحد**", orders)
	}

	// ══════════════════════════════════════════════════════════════
	// **والأثرُ المكرَّرُ يُقاس حيث يقع** — لا حيث يحسن الظنّ
	// ══════════════════════════════════════════════════════════════
	//
	// **وكان هنا شرطٌ على `wallet_transactions`** — **وهو ميتٌ**: قِيس
	// ٢٠٢٦-٠٩-٣٠ أنّ إنشاءَ الطلب العاديِّ **لا يكتب قيداً ماليّاً
	// إطلاقاً** (الحجزُ في مسار الطلب الخاصّ وحدَه،
	// `custom_quote.go:98`)، **فالعددُ صفرٌ دائماً والشرطُ لا يسقط أبداً.**
	//
	// **وحارسٌ لا يستطيع أن يسقط أسوأُ من لا حارس** — يُقرأ تغطيةً وهو
	// فراغ. (أمسكه تدقيقٌ خصميٌّ في اليوم نفسِه.)
	//
	// **فالمقيسُ هنا أثرُ الإنشاءِ المكرَّر**: صفُّ مولدٍ واحد، وأصنافٌ
	// لا تُضاعَف. **وتكرارُ المالِ يُقاس في اختبارات التسوية** — هناك
	// يتحرّك المال.
	var events int
	if err := h.Pool.QueryRow(context.Background(), `
		SELECT count(*) FROM order_events oe
		  JOIN orders o ON o.id = oe.order_id
		 WHERE o.customer_id = $1::uuid`, u.ID).Scan(&events); err != nil {
		t.Fatalf("عدُّ أحداث الطلب: %v", err)
	}
	if events != 1 {
		t.Fatalf("أحداثُ ميلادٍ %d لطلبٍ واحد — **أثرٌ مكرَّر**", events)
	}

	var items int
	if err := h.Pool.QueryRow(context.Background(), `
		SELECT count(*) FROM order_items oi
		  JOIN orders o ON o.id = oi.order_id
		 WHERE o.customer_id = $1::uuid`, u.ID).Scan(&items); err != nil {
		t.Fatalf("عدُّ أصناف الطلب: %v", err)
	}
	if items != 1 {
		t.Fatalf("أصنافٌ %d لطلبٍ بصنفٍ واحد — **أثرٌ مكرَّر**", items)
	}
}

// ── ج · وتبديلُ أسلوب التسوية أثناء الإجهاد — `SET-06` باقٍ ───────────
//
// **والقفلُ الجديدُ `FOR SHARE`** — **ويجب أن يبقى متعارضاً مع مُبدِّل
// الأسلوب** (`FOR UPDATE`)، **فلا يرى طلبٌ نصفَ تبديل.**
//
// ══════════════════════════════════════════════════════════════════════
// **وكان هذا الاختبارُ يُبدّل بيده فلا يختبر ما يزعم**
// ══════════════════════════════════════════════════════════════════════
//
// **كان يُنفّذ `UPDATE merchants` خاماً بالمَسبح** — **وذاك يأخذ
// `FOR NO KEY UPDATE` لا `FOR UPDATE`.** **والادّعاءُ المُراد إثباتُه هو
// أنّ `SetSettlementMethod` يأخذ `FOR UPDATE` فيبقى التسلسل** — **فكان
// الاختبارُ يتخطّى الجملةَ الوحيدةَ التي تهمّ.**
//
// **ويُضاف**: شرطٌ يمنع النجاحَ الفارغ — **لقطةٌ واحدةٌ على الأقلّ**،
// **وطلبٌ بعد التبديل يحمل الجديدةَ قطعاً** (حتميٌّ لا سباق).
//
// (أمسكهما تدقيقٌ خصميٌّ ٢٠٢٦-٠٩-٣٠.)
func TestConc_SettlementChangeDuringLoad_SnapshotIsWhole(t *testing.T) {
	const customers = 6

	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "5")
	treasury(t, h)
	f := h.Factory()
	dlBefore := deadlocks(t, h)

	item := h.NewItem(1000)
	// **ويُبدَأ من `cash`** — **ومِسنَدُ المِسْكَب يصنع المتجر بـ`wallet`**،
	// فتبديلٌ إلى `wallet` لا يبدّل شيئاً **واللقطاتُ كلُّها `wallet` سلفاً
	// فيمرّ الشرطُ فارغاً.** (أمسكه القياسُ: `map[wallet:6]` قبل هذا السطر.)
	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE merchants SET settlement_method = 'cash' WHERE id = $1::uuid`,
		item.MerchantID); err != nil {
		t.Fatalf("تهيئةُ الأسلوب الابتدائيّ: %v", err)
	}
	users := make([]*User, customers)
	for i := range users {
		users[i] = f.NewUserWith("customer")
	}

	var gate sync.WaitGroup
	gate.Add(1)
	var wg sync.WaitGroup
	res := make([]Res, customers)
	for i := range users {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			gate.Wait()
			res[i] = h.POSTKey("/api/v1/orders", users[i].Token,
				uniq(fmt.Sprintf("conc-set-%d", i)), orderBody(item, 1))
		}(i)
	}
	// ══════════════════════════════════════════════════════════════
	// **والمُبدِّلُ يحمل قفلَ المسار الحقيقيِّ نفسَه**
	// ══════════════════════════════════════════════════════════════
	//
	// **ولا يُنادى الباب**: `PATCH /admin/merchants/{id}/settlement-method`
	// **فعلٌ حسّاسٌ يلزمه تصعيدُ رمز** (`authz/sensitive.go:126`)، **وللتصعيد
	// عقدُه واختباراتُه** (`step_up_contract_test.go`) — **وطبقةُ التخويل
	// ليست ما يُقاس هنا.** (وقِيس: النداءُ بلا تصعيدٍ يُردّ ٤٠٣، **فكان
	// المُبدِّلُ لا يعمل والاختبارُ يمرّ.**)
	//
	// **وكانت هنا جملةُ `UPDATE` خامّة** — **وتلك تأخذ `FOR NO KEY UPDATE`
	// لا `FOR UPDATE`**، **فكانت تتخطّى الجملةَ الوحيدةَ التي تهمّ.**
	//
	// **فتُحمَل المعاملةُ قفلَ `SetSettlementMethod` حرفاً**: `FOR UPDATE`
	// ثمّ `UPDATE`. **وأنّ المسارَ الحقيقيَّ يأخذه يحرسه
	// `TestConc_SettlementChangerKeepsExclusiveLock`.**
	var changeErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		gate.Wait()
		ctx := context.Background()
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			changeErr = err
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		var cur string
		if err := tx.QueryRow(ctx,
			`SELECT settlement_method FROM merchants WHERE id = $1::uuid FOR UPDATE`,
			item.MerchantID).Scan(&cur); err != nil {
			changeErr = err
			return
		}
		if _, err := tx.Exec(ctx,
			`UPDATE merchants SET settlement_method = 'wallet', updated_at = now()
			  WHERE id = $1::uuid`, item.MerchantID); err != nil {
			changeErr = err
			return
		}
		changeErr = tx.Commit(ctx)
	}()
	gate.Done()
	wg.Wait()

	for i, r := range res {
		if r.Code >= 500 {
			t.Errorf("زبون %d: **ردٌّ %d** — %s", i, r.Code, r.String())
		}
	}
	if changeErr != nil {
		t.Errorf("مُبدِّلُ الأسلوب سقط: %v", changeErr)
	}
	assertNoNewDeadlocks(t, h, dlBefore)

	// **ولا لقطةَ فارغةٌ ولا ثالثة** — العقدُ يقول: القديمةَ أو الجديدة.
	//
	// **ويُعَدُّ ما وُجد** — **فنجاحٌ على صفرِ لقطاتٍ نجاحٌ فارغ.**
	rows, err := h.Pool.Query(context.Background(), `
		SELECT COALESCE(oi.merchant_settlement_method, ''), count(*)
		  FROM order_items oi JOIN orders o ON o.id = oi.order_id
		 WHERE o.merchant_id = $1::uuid
		 GROUP BY 1`, item.MerchantID)
	if err != nil {
		t.Fatalf("قراءةُ لقطات الأصناف: %v", err)
	}
	seen := map[string]int{}
	for rows.Next() {
		var m string
		var c int
		if err := rows.Scan(&m, &c); err != nil {
			rows.Close()
			t.Fatalf("scan: %v", err)
		}
		seen[m] = c
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("صفوفُ اللقطات: %v", err)
	}
	total := 0
	for m, c := range seen {
		if m != "cash" && m != "wallet" {
			t.Fatalf("لقطةُ أسلوبٍ %q — **والعقدُ cash أو wallet، لا فراغَ ولا ثالثة**", m)
		}
		total += c
	}
	if total == 0 {
		t.Fatal("صفرُ لقطاتٍ — **والاختبارُ لا يقيس شيئاً**")
	}
	t.Logf("SET-06: لقطاتٌ = %v (المجموع %d) · المُبدِّل التزم=%v", seen, total, changeErr == nil)

	// ══════════════════════════════════════════════════════════════
	// **وحتميٌّ لا سباق**: طلبٌ **بعد** أن التزم التبديلُ يحمل الجديدةَ
	// ══════════════════════════════════════════════════════════════
	//
	// **فالسباقُ وحدَه قد ينتهي كلُّه قبل التبديل** — فيمرّ الاختبارُ
	// بلا أن يُثبت أنّ التبديلَ يصل اللقطةَ أصلاً.
	if changeErr == nil {
		late := f.NewUserWith("customer")
		lateRes := h.POSTKey("/api/v1/orders", late.Token, uniq("conc-set-late"),
			orderBody(item, 1))
		if lateRes.Code != 201 {
			t.Fatalf("طلبٌ بعد التبديل رُدّ %d — %s", lateRes.Code, lateRes.String())
		}
		var m string
		if err := h.Pool.QueryRow(context.Background(), `
			SELECT COALESCE(oi.merchant_settlement_method, '')
			  FROM order_items oi JOIN orders o ON o.id = oi.order_id
			 WHERE o.customer_id = $1::uuid LIMIT 1`, late.ID).Scan(&m); err != nil {
			t.Fatalf("لقطةُ الطلب المتأخّر: %v", err)
		}
		if m != "wallet" {
			t.Fatalf("طلبٌ بعد التبديل حمل %q — **والتبديلُ لا يصل اللقطة**", m)
		}
	}
}

// ── د · و`CreateTx` لا يكتب صفَّ المتجر — حارسٌ على المصدر ─────────────
//
// ══════════════════════════════════════════════════════════════════════
//
// **قبل الإصلاح كان `FOR UPDATE` يجعل أيَّ `UPDATE merchants` لاحقاً في
// المعاملة نفسِها آمناً.** **وبعده صار أيُّ كتابةٍ على ذلك الصفِّ ترقيةً
// من `SHARE` إلى `NO KEY UPDATE`** — **وهي `D4-T10` من جديد بحرفها.**
//
// **والمكوّنُ موجودٌ في الشجرة**: `obligations.go` فيه
// `UPDATE merchants SET debt = debt + $2`، **ويُنادى من أربعة مواضعَ في
// مسار الطلب** — لكنْ في معاملاتٍ أخرى لا في هذه.
//
// **فمن أضاف غداً احتباسَ نقدٍ لحظةَ الإنشاء، أو عدّادَ طلباتٍ على
// المتجر، أعاد العطبَ بسطرٍ لا يبدو خطراً** — **والحاميُ الوحيدُ قبل هذا
// الحارس تعليقٌ يُقرأ ولا يُلزم.**
//
// (أوصى به ناقدُ اكتمالٍ في تدقيقٍ خصميّ ٢٠٢٦-٠٩-٣٠.)
func TestConc_CreateTxNeverWritesMerchantRow(t *testing.T) {
	src := geoSource(t, "backend/internal/orders/service.go")

	i := strings.Index(src, "func (s *Service) CreateTx(")
	if i < 0 {
		t.Fatal("**ذهبت `CreateTx`** — يُعاد بناءُ الحارس")
	}
	// **وحدُّ الجسم أوّلُ تصريحٍ بعده** — لا عددُ أسطرٍ مكتوبٌ بيد.
	j := strings.Index(src[i+10:], "\nfunc ")
	if j < 0 {
		t.Fatal("**لم أجد نهايةَ `CreateTx`**")
	}
	body := src[i : i+10+j]

	// **والقفلُ المشتركُ شرطُ الإصلاح** — فإن عاد حصريّاً عاد العطب.
	if !strings.Contains(body, "FROM merchants WHERE id = $1 FOR SHARE") {
		t.Fatal("**قفلُ صفِّ المتجر في `CreateTx` ليس `FOR SHARE`** — " +
			"وإن صار `FOR UPDATE` عاد `REL-CONC-01`: إدراجُ الطلب يأخذ " +
			"`FOR KEY SHARE` ضمنيّاً، فالحصريُّ بعده ترقيةٌ تتعارض معه")
	}

	// **ولا كتابةَ على الصفّ بعده** — الترقيةُ الثانية.
	for _, bad := range []string{"UPDATE merchants", "FOR UPDATE` FROM merchants", "DELETE FROM merchants"} {
		if strings.Contains(body, bad) {
			t.Fatalf("**`CreateTx` يكتب صفَّ المتجر (%q)** — **وذاك ترقيةٌ من "+
				"`SHARE` إلى `NO KEY UPDATE` تُعيد `REL-CONC-01` حرفاً.** "+
				"إن لزمت الكتابةُ فلتكن في معاملةٍ أخرى، أو يُنقَل القفلُ "+
				"قبل إدراج الطلب — **وحينها يتسلسل الإنشاءُ ويسقط `D4-T10`.**", bad)
		}
	}
}

// deadlocks **عدّادُ بوستغرس نفسُه** — لا قراءةُ سجلٍّ ولا تخمين.
//
// **و`pg_stat_database.deadlocks` تراكميٌّ للقاعدة كلِّها** — فيُقرأ قبل
// الإجهاد وبعده، **والفارقُ هو ما وقع في هذا الاختبار وحدَه.**
func deadlocks(t *testing.T, h *Harness) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT deadlocks FROM pg_stat_database WHERE datname = current_database()`).
		Scan(&n); err != nil {
		t.Fatalf("قراءةُ عدّاد التعارضات: %v", err)
	}
	return n
}

// assertNoNewDeadlocks **صفرُ تعارضٍ جديدٍ** — بالعدّاد لا بالظنّ.
func assertNoNewDeadlocks(t *testing.T, h *Harness, before int64) {
	t.Helper()
	after := deadlocks(t, h)
	if after != before {
		t.Fatalf("**وقع %d تعارضاً في القاعدة** (%d ⇒ %d) — REL-CONC-01 عاد",
			after-before, before, after)
	}
}

// ── هـ · والمُبدِّلُ يبقى حصريّاً — وإلّا سقط التسلسل ──────────────────
//
// ══════════════════════════════════════════════════════════════════════
//
// **الإصلاحُ يقوم على ساقَين**: `CreateTx` مشتركٌ (`FOR SHARE`)،
// **و`SetSettlementMethod` حصريٌّ (`FOR UPDATE`)** — **ومشتركٌ يتعارض مع
// حصريّ، فالتسلسلُ باقٍ كما ينصّ البندُ ٥.**
//
// **فلو خُفِّف المُبدِّلُ إلى `FOR SHARE` يوماً** — التماساً لإنتاجيّةٍ —
// **لصار المشتركانِ متوافقَين**، **فيقرأ طلبٌ الأسلوبَ القديمَ ويلتزم
// المُبدِّلُ في أثنائه**: لقطةٌ لا تصف شيئاً، ولا حارسَ يُنبّه.
//
// **وهذا الحارسُ هو الساقُ الثانية** — والأولى
// `TestConc_CreateTxNeverWritesMerchantRow`.
func TestConc_SettlementChangerKeepsExclusiveLock(t *testing.T) {
	src := geoSource(t, "backend/internal/catalog/catalog.go")

	i := strings.Index(src, "func (s *Service) SetSettlementMethod(")
	if i < 0 {
		t.Fatal("**ذهبت `SetSettlementMethod`** — يُعاد بناءُ الحارس")
	}
	j := strings.Index(src[i+10:], "\nfunc ")
	if j < 0 {
		t.Fatal("**لم أجد نهايةَ `SetSettlementMethod`**")
	}
	body := src[i : i+10+j]

	if !strings.Contains(body, "FROM merchants WHERE id = $1 FOR UPDATE") {
		t.Fatal("**مُبدِّلُ أسلوب التسوية لا يأخذ `FOR UPDATE` على صفِّ المتجر** — " +
			"**فالتسلسلُ مع إنشاء الطلب يسقط**: `CreateTx` يأخذ `FOR SHARE`، " +
			"ومشتركٌ مع مشتركٍ متوافقان، **فيلتزم التبديلُ في أثناء قراءةِ " +
			"طلبٍ فتصير اللقطةُ لا تصف شيئاً** (البند ٥ / SET-06)")
	}
	// **والقفلُ قبل الكتابة لا بعدها** — وإلّا كانت الكتابةُ بلا حماية.
	lk := strings.Index(body, "FOR UPDATE")
	wr := strings.Index(body, "UPDATE merchants SET settlement_method")
	if wr >= 0 && lk > wr {
		t.Fatal("**الكتابةُ قبل القفل في `SetSettlementMethod`** — والقفلُ بعدها لا يحمي شيئاً")
	}
}
