package qa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حسمُ خطرَين ماليَّين بحدود المعاملة** — `R4` · `R11`
// ══════════════════════════════════════════════════════════════════════
//
// **البوّابةُ تطلب لهما دليلاً واحداً**:
//
//	اختبارٌ يثبته فيصير عيباً، أو ينفيه فيُغلَق
//
// **وكلاهما بقي `NOT_RUN` منذ فُتح** — **وخطرٌ لا يُحسَم يبقى مانعاً
// إلى الأبد**، لا لأنّه وقع بل لأنّ أحداً لم يسأل.

// ══════════════════════════════════════════════════════════════════════
// **`R4` — «`treasury.go` يعتمد معاملةَ المنادي»**
// ══════════════════════════════════════════════════════════════════════
//
// # ما هو الخطر
//
// **دوالُّ الخزينة كلُّها تأخذ `q wallet.Querier`** — **ولا تفتح
// معاملةً بنفسها.** **و`*pgxpool.Pool` يحقّق الواجهةَ كما تحقّقها
// `pgx.Tx`** — **فمن مرّرها المَسبَحَ كتب في الخزينة خارجَ المعاملة**،
// **فتُقيَّد ربحيّةٌ لعمليّةٍ ارتدّت.**
//
// # ولماذا حارسٌ ساكنٌ لا نداءٌ حيّ
//
// **النداءُ الحيُّ يُثبت موضعاً واحداً** — **والخطرُ في كلّ موضعٍ
// يُكتب غداً.** **فيُقرأ الشجرُ النحويُّ ويُسأل كلُّ مُنادٍ**: بمَ
// ناديتَ؟
//
// **والوسيطُ المقبول**: `tx` أو `q` مُمرَّرٌ من فوق. **والمرفوض**:
// `s.db` و`s.pg` — **وهما المَسبَح.**
func TestFIN_R4_TreasuryNeverWritesOutsideTransaction(t *testing.T) {
	root := backendRoot(t)

	// **أسماءُ ما يكتب في الخزينة** — من `internal/orders/treasury.go`.
	writers := map[string]bool{
		"creditTreasury":       true,
		"CreditTreasuryTx":     true,
		"CreditTreasuryDirect": true,
		"DebitTreasury":        true,
	}
	// **وما لا يجوز أن يُمرَّر** — المَسبَح بأسمائه في هذا المستودع.
	pools := map[string]bool{"db": true, "pg": true, "Pool": true}

	var checked, bad int
	var offenders []string

	err := filepath.Walk(filepath.Join(root, "internal"), func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".go") ||
			strings.HasSuffix(p, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, p, nil, 0)
		if perr != nil {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !writers[sel.Sel.Name] || len(call.Args) < 2 {
				return true
			}
			checked++
			// **الوسيطُ الثاني هو المنفّذ** — بعد `ctx`.
			arg, ok := call.Args[1].(*ast.SelectorExpr)
			if !ok {
				return true // `tx` أو `q` — معرّفٌ بسيطٌ مقبول
			}
			if pools[arg.Sel.Name] {
				bad++
				offenders = append(offenders,
					relPath(root, p)+": "+sel.Sel.Name+"(… s."+arg.Sel.Name+" …)")
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("مسحُ الشجر: %v", err)
	}

	if checked == 0 {
		t.Fatal("**لم يُفحَص أيُّ مُنادٍ للخزينة** — **والحارسُ الأعمى " +
			"أسوأُ من لا حارس**: يمرّ أبداً ولا يرى شيئاً")
	}
	sort.Strings(offenders)
	t.Logf("مُنادو الخزينة المفحوصون: %d · خارجَ المعاملة: %d", checked, bad)
	for _, o := range offenders {
		t.Errorf("**كتابةٌ في الخزينة بالمَسبَح** — %s\n"+
			"**فتُقيَّد ربحيّةٌ لعمليّةٍ قد ترتدّ.**", o)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وقيدُ الخزينة يشارك معاملةَ عمليّته حقّاً**
// ══════════════════════════════════════════════════════════════════════
//
// **والحارسُ الساكنُ يقول «مُرِّرت معاملة»** — **ولا يقول إنّها تعمل.**
//
// **و`creditTreasury` آخرُ ما يقع في تسوية التسليم** — فلا كتابةَ
// بعدها تُحقَن. **فيُقلَب الإثبات**: **يُسقَط قيدُ الخزينة، ويُنظَر
// هل ارتدّ ما قبلَه** — حالُ الطلب ومستحقُّ المتجر وأجرُ السائق.
//
// **فإن بقي شيءٌ منها فالخزينةُ خارجَ المعاملة** — **وهو `R4` بعينه.**
func TestFIN_R4_TreasurySharesTheOperationTransaction(t *testing.T) {
	h := New(t)
	treasury(t, h)
	_, item := repFixture(t, h)

	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)

	// **يُمشى إلى عتبة التسليم بلا حقن** — **والقيدُ في الخزينة يقع
	// في خطواتٍ سابقةٍ أيضاً**، فحقنٌ من البداية يُسقط ما لا يُقصَد.
	walkToDropoff(t, h, oid, drv)

	before := orderSnapshot(t, h, oid)
	t.Logf("قبل: حالٌ=%q · مستحقُّ متجرٍ=%d · أجرُ سائقٍ=%d · ربحُ منصّةٍ=%d",
		before.Status, before.MerchantEarning, before.DriverFee, before.Platform)

	// **ثمّ يُسقَط قيدُ ربح المنصّة في خطوة التسليم وحدَها.**
	fp := h.Arm("R4/platform-profit", "wallet_transactions", "INSERT",
		1, "kind", "platform_profit")
	code := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}).Code
	fp.MustFire(t)

	after := orderSnapshot(t, h, oid)
	t.Logf("بعد (%d): حالٌ=%q · مستحقُّ متجرٍ=%d · أجرُ سائقٍ=%d · ربحُ منصّةٍ=%d",
		code, after.Status, after.MerchantEarning, after.DriverFee, after.Platform)

	if after.Status != before.Status {
		t.Errorf("**حالُ الطلب تبدّلت وقيدُ الخزينة سقط**: %q ← %q — "+
			"**فالخزينةُ خارجَ معاملة عمليّتها.**", before.Status, after.Status)
	}
	if after.MerchantEarning != before.MerchantEarning {
		t.Errorf("**مستحقُّ متجرٍ قُيّد وقيدُ الخزينة سقط**: %d ← %d",
			before.MerchantEarning, after.MerchantEarning)
	}
	if after.DriverFee != before.DriverFee {
		t.Errorf("**أجرُ سائقٍ قُيّد وقيدُ الخزينة سقط**: %d ← %d",
			before.DriverFee, after.DriverFee)
	}
	if after.Platform != before.Platform {
		t.Errorf("**ربحُ منصّةٍ قُيّد والحقنُ أسقطه**: %d ← %d",
			before.Platform, after.Platform)
	}
}

// orderState ما قُيّد على طلبٍ وحالُه.
type orderState struct {
	Status                               string
	MerchantEarning, DriverFee, Platform int64
}

func orderSnapshot(t *testing.T, h *Harness, orderID string) orderState {
	t.Helper()
	var s orderState
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT status FROM orders WHERE id::text = $1),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $1::text AND kind = 'merchant_earning'), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $1::text AND kind = 'delivery_fee'), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $1::text AND kind = 'platform_profit'), 0)`,
		orderID).Scan(&s.Status, &s.MerchantEarning, &s.DriverFee, &s.Platform); err != nil {
		t.Fatalf("لقطةُ الطلب: %v", err)
	}
	return s
}

// walkToDropoff يمشي بالطلب إلى عتبة التسليم بلا أن يُسلّمه.
func walkToDropoff(t *testing.T, h *Harness, oid string, drv *User) {
	t.Helper()
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); got.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, got)
		}
	}
	if skip := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": "R4 — حدُّ معاملةِ الخزينة"}); skip.Code >= 400 {
		t.Fatalf("تخطّي الإثبات رُدّ: %s", skip)
	}
}

func treasuryBalance(t *testing.T, h *Harness, tid string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`,
		tid).Scan(&v); err != nil {
		t.Fatalf("رصيدُ الخزينة: %v", err)
	}
	return v
}

func backendRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("المجلّدُ الحاليّ: %v", err)
	}
	return filepath.Dir(filepath.Dir(wd)) // internal/qa ⇒ backend
}

func relPath(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return filepath.ToSlash(r)
	}
	return p
}

// ══════════════════════════════════════════════════════════════════════
// **`R11` — «الطلبُ الخاصُّ يُنشأ بلا معاملة»**
// ══════════════════════════════════════════════════════════════════════
//
// # ما هو الخطر
//
// **إنشاءُ الطلب الخاصّ لا يفتح معاملة** — **فإن كان يكتب أكثرَ من
// صفٍّ أو يحرّك مالاً، سقوطُ إحداها يترك نصفَ عمليّة** (وهو مرضُ
// `PF-01` و`PF-02` نفسُه).
//
// # وما قِيس
//
// **`CreateCustom` إدخالٌ واحدٌ في `orders`** (`custom.go:98`)،
// **و`AgreeCustom` قراءةٌ ثمّ تحديثٌ واحد** (`custom.go:202`).
// **والعبارةُ الواحدةُ ذرّيّةٌ في بوستغرس بلا معاملةٍ صريحة** —
// **فالخطرُ لا يقع لأنّه لا شيءَ ثانٍ ليُفقَد.**
//
// **ولا مالَ في الطريق**: **الطلبُ الخاصُّ بلا سعرٍ حتّى يتّفقا**،
// فلا مستحقَّ ولا أجرةَ ولا خزينة.
//
// **وهذا الحارسُ يُثبت الأمرين**: **أثرٌ واحدٌ أو لا أثر**، **ولا قيدٌ
// في الدفتر.** **فإن أُضيفت كتابةٌ ثانيةٌ غداً سقط** — **وعندها يصير
// `R11` عيباً بحقّ.**
func TestFIN_R11_CustomOrderCreationIsSingleWriteAndMoneyless(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()

	before := ledgerCount(t, h)

	// ── ١ ── الإنشاءُ يقع أو لا يقع ────────────────────────────────
	fp := h.ArmAny("R11/create-custom", "orders", "INSERT")
	failed := h.POST("/api/v1/orders/custom", cust.Token, customBody("دواءٌ من صيدليّةٍ في السوق"))
	fp.MustFire(t)

	orphans := countRows(t, h, `
		SELECT count(*) FROM orders
		 WHERE customer_id = $1::uuid AND kind = 'custom'`, cust.ID)
	t.Logf("مع الحقن: الردُّ %d · طلباتٌ خاصّةٌ باقيةٌ %d", failed.Code, orphans)
	if orphans != 0 {
		t.Errorf("**طلبٌ خاصٌّ بقي والإنشاءُ سقط** (%d)", orphans)
	}

	// ── ٢ ── وبلا حقنٍ: صفٌّ واحدٌ ولا مال ─────────────────────────
	made := h.POST("/api/v1/orders/custom", cust.Token, customBody("دواءٌ من صيدليّةٍ في السوق"))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب الخاصّ: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	rows := countRows(t, h, `
		SELECT count(*) FROM orders
		 WHERE customer_id = $1::uuid AND kind = 'custom'`, cust.ID)
	moved := countRows(t, h,
		`SELECT count(*) FROM wallet_transactions WHERE ref = $1::text`, oid)
	after := ledgerCount(t, h)
	t.Logf("بلا حقن: طلباتٌ %d · قيودُ هذا الطلب %d · الدفترُ %d ← %d",
		rows, moved, before, after)

	if rows != 1 {
		t.Errorf("**صفوفٌ %d والمتوقَّع واحد**", rows)
	}
	if moved != 0 {
		t.Errorf("**الطلبُ الخاصُّ حرّك مالاً عند الإنشاء** — %d قيداً", moved)
	}
	if after != before {
		t.Errorf("**الدفترُ تحرّك بإنشاء طلبٍ خاصّ**: %d ← %d", before, after)
	}
}

func ledgerCount(t *testing.T, h *Harness) int {
	t.Helper()
	return countRows(t, h, `SELECT count(*) FROM wallet_transactions`)
}
