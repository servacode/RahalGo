package qa

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/server"
)

// ══════════════════════════════════════════════════════════════════════
// **فعلٌ حسّاسٌ لا ينجح بلا أثر** — `PF-06` · `AQ-4`
// ══════════════════════════════════════════════════════════════════════
//
//	A SUCCESSFUL SENSITIVE BUSINESS MUTATION
//	MUST HAVE A DURABLE AUDIT RECORD
//
// **وحالان لا ثالث**: **يقعان معاً أو لا يقع أحدُهما.**
//
// **وليس كلُّ سطرٍ تدقيقاً حسّاساً** — **من جعل كلَّ سطرٍ معامليّاً
// أسقط بيعاً لأنّ سطرَ سجلٍّ تعذّر.** **والصنفُ `A` ما يحرّك مالاً.**

func auditRows(t *testing.T, h *Harness, action, entityID string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM audit_log WHERE action = $1 AND entity_id = $2`,
		action, entityID).Scan(&n); err != nil {
		t.Fatalf("عدُّ التدقيق: %v", err)
	}
	return n
}

func walletBalance(t *testing.T, h *Harness, userID string) int64 {
	t.Helper()
	var v int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`,
		userID).Scan(&v)
	return v
}

func applyWallet(t *testing.T, h *Harness, admin *User, target string, amount int64) int {
	t.Helper()
	return h.POSTKey("/api/v1/admin/users/"+target+"/wallet", admin.Token,
		uniq("aq4"), map[string]any{
			"amount": amount, "kind": "topup", "note": "PF-06",
		}).Code
}

// ══════════════════════════════════════════════════════════════════════
// **A1+A7 · نجاحٌ ⇒ المالُ والأثرُ معاً، والأثرُ يقول من وعلى من**
// ══════════════════════════════════════════════════════════════════════
func TestAQ4_A1_SuccessCommitsBoth(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	target := h.Factory().NewUserWith("customer")

	code := applyWallet(t, h, admin, target.ID, 12_000)
	bal := walletBalance(t, h, target.ID)
	rows := auditRows(t, h, "finance.wallet_apply", target.ID)
	t.Logf("نجاحٌ: الردُّ %d · رصيدٌ %d · قيودُ تدقيقٍ %d", code, bal, rows)

	if code >= 400 || bal != 12_000 {
		t.Fatalf("الفعلُ لم يقع: %d · رصيدٌ %d", code, bal)
	}
	if rows != 1 {
		t.Errorf("**قيودُ التدقيق %d والمتوقَّع واحد**", rows)
	}

	// **والأثرُ يقول من فعلها وعلى من وبكم.**
	var actor, entity string
	var details string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(actor_user_id::text, ''), entity, details::text
		  FROM audit_log WHERE action = 'finance.wallet_apply' AND entity_id = $1`,
		target.ID).Scan(&actor, &entity, &details); err != nil {
		t.Fatalf("قراءةُ الأثر: %v", err)
	}
	t.Logf("الأثرُ: فاعلٌ=%s · كيانٌ=%s · تفصيلٌ=%s", first8(actor), entity, details)
	if actor != admin.ID {
		t.Errorf("**الفاعلُ %q والمتوقَّع %q**", actor, admin.ID)
	}
	if entity != "user" || !strings.Contains(details, "12000") {
		t.Errorf("**الأثرُ لا يصف الفعل**: كيانٌ=%q · %s", entity, details)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A2+A9 · سقوطُ الأثر يُسقط المال — ولا أثرَ كاذبٌ يبقى**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو `PF-06` بعينه**: كان الردُّ `200` والرصيدُ `12000` وقيودُ
// التدقيق صفر.
func TestAQ4_A2_AuditFailureRollsBackMoney(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	target := h.Factory().NewUserWith("customer")

	fp := h.ArmAny("AQ-4/audit-write", "audit_log", "INSERT")
	code := applyWallet(t, h, admin, target.ID, 12_000)
	fp.MustFire(t)

	bal := walletBalance(t, h, target.ID)
	rows := auditRows(t, h, "finance.wallet_apply", target.ID)
	t.Logf("سقوطُ الأثر: الردُّ %d · رصيدٌ %d · قيودُ تدقيقٍ %d", code, bal, rows)

	if bal != 0 {
		t.Errorf("**المالُ تحرّك والأثرُ سقط** (رصيدٌ %d) — "+
			"**فعلٌ حسّاسٌ بلا أثرٍ لا يُراجَع.** (`PF-06` · `AQ-4`)", bal)
	}
	if rows != 0 {
		t.Errorf("**أثرٌ بقي وقد سقط إدخالُه** (%d)", rows)
	}
	if code < 400 {
		t.Errorf("**رُدَّ نجاحٌ ولم يقع شيء** (%d) — **ولا يُقال «تمّ» لما لم يتمّ**",
			code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A3 · سقوطُ الفعل ⇒ لا أثرَ يزعم أنّه وقع**
// ══════════════════════════════════════════════════════════════════════
func TestAQ4_A3_BusinessFailureLeavesNoAudit(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	target := h.Factory().NewUserWith("customer")

	fp := h.Arm("AQ-4/money-write", "wallet_transactions", "INSERT", 1, "kind", "topup")
	code := applyWallet(t, h, admin, target.ID, 12_000)
	fp.MustFire(t)

	bal := walletBalance(t, h, target.ID)
	rows := auditRows(t, h, "finance.wallet_apply", target.ID)
	t.Logf("سقوطُ الفعل: الردُّ %d · رصيدٌ %d · قيودُ تدقيقٍ %d", code, bal, rows)
	if bal != 0 || rows != 0 {
		t.Errorf("**أثرٌ أو مالٌ بقي**: رصيدٌ %d · قيودٌ %d", bal, rows)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A5 · سقوطٌ ثمّ إعادة ⇒ فعلٌ واحدٌ وأثرٌ واحد**
// ══════════════════════════════════════════════════════════════════════
func TestAQ4_A5_RetryGivesOneOfEach(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	target := h.Factory().NewUserWith("customer")

	fp := h.ArmAny("AQ-4/retry-audit", "audit_log", "INSERT")
	_ = applyWallet(t, h, admin, target.ID, 12_000)
	fp.MustFire(t)

	code := applyWallet(t, h, admin, target.ID, 12_000)
	bal := walletBalance(t, h, target.ID)
	rows := auditRows(t, h, "finance.wallet_apply", target.ID)
	t.Logf("بعد الإعادة: الردُّ %d · رصيدٌ %d · قيودُ تدقيقٍ %d", code, bal, rows)

	if bal != 12_000 {
		t.Errorf("**الرصيدُ %d والمتوقَّع 12000** — أثرُ إعادةٍ واحدة", bal)
	}
	if rows != 1 {
		t.Errorf("**قيودُ التدقيق %d والمتوقَّع واحد**", rows)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A6 · أفعالٌ متزامنةٌ ⇒ لكلٍّ أثرُه ولا تشابك**
// ══════════════════════════════════════════════════════════════════════
func TestAQ4_A6_ConcurrentActionsKeepTheirOwnAudit(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	admin := h.NewUser("admin")
	a := f.NewUserWith("customer")
	b := f.NewUserWith("customer")

	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "قيدٌ-أ", Do: func(ctx context.Context) any {
			return applyWallet(t, h, admin, a.ID, 5_000)
		}},
		Actor{Name: "قيدٌ-ب", Do: func(ctx context.Context) any {
			return applyWallet(t, h, admin, b.ID, 7_000)
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	ra := auditRows(t, h, "finance.wallet_apply", a.ID)
	rb := auditRows(t, h, "finance.wallet_apply", b.ID)
	t.Logf("متزامنان: رصيدُ أ=%d أثرُه=%d · رصيدُ ب=%d أثرُه=%d — %s",
		walletBalance(t, h, a.ID), ra, walletBalance(t, h, b.ID), rb, r)

	if ra != 1 || rb != 1 {
		t.Errorf("**تشابكُ آثار**: أ=%d · ب=%d", ra, rb)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A8 · وما لم يُغطَّ بعدُ يبقى أفضلَ جهد** — حالٌ قائمةٌ لا عقد
// ══════════════════════════════════════════════════════════════════════
//
// ⚠️ **وهذا الفحصُ يوثّق ما هو قائمٌ لا ما يوجبه العقد.**
//
// **وكنتُ كتبتُه في دورةِ ١٣ بوصفه عقداً** — **«الإعداداتُ غيرُ
// حسّاسة»** — **والعقدُ يقول عكسَه بنصّه**: `AQ-4` يشمل «الإعداداتِ
// الحسّاسة». **فاخترتُ السياسةَ من الشيفرة لا من الحقيقة.**
//
// **فسُجّلت `XG-35`** بما بقي مكشوفاً، **ولم تُنفَّذ بأمر المالك**
// (مصالحةُ دورةِ ١٤). **ويومَ تُغطّى ينقلب هذا الفحص.**
//
// **وما يُثبته اليومَ صحيحٌ ومحدود**: **الفعلُ غيرُ المُغطّى لا يسقط
// بسقوط أثرِه** — **ولا يُسقَط بيعٌ لأنّ سطرَ سجلٍّ تعذّر.**
func TestAQ4_A8_UncoveredActionsRemainBestEffort(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")

	fp := h.ArmAny("AQ-4/non-critical", "audit_log", "INSERT")
	got := h.Call("PUT", "/api/v1/admin/settings/orders.auto_transfer",
		admin.Token, map[string]any{"value": true}, nil)
	fired := fp.Fired()
	var value string
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT value::text FROM app_settings WHERE key = 'orders.auto_transfer'`).Scan(&value)
	t.Logf("إعدادٌ مع سقوط الأثر: الردُّ %d · النقطةُ أصابت %d · القيمةُ %q",
		got.Code, fired, value)

	if got.Code >= 400 {
		t.Errorf("**فعلٌ غيرُ حسّاسٍ سقط** (%d) — **والتدقيقُ التشغيليُّ "+
			"أفضلُ جهد، ولا يُسقَط بيعٌ لأنّ سطرَ سجلٍّ تعذّر.**", got.Code)
	}
	if !strings.Contains(value, "true") {
		t.Errorf("**الإعدادُ لم يُكتب** (%q) رغم أنّ أثرَه أفضلُ جهد", value)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الحارسُ البنيويُّ: لا فعلَ حسّاسٌ خارجَ التدقيق المعامليّ**
// ══════════════════════════════════════════════════════════════════════
//
// **والقائمةُ تُقرأ من الشيفرة لا تُكتب هنا** — **قائمةٌ ثانيةٌ تشيخ.**
// **فمن أضاف فعلاً ماليّاً غداً بتدقيقٍ أفضلِ جهدٍ سقط هذا الحارس.**
func TestAQ4_CriticalActionsUseTransactionalAudit(t *testing.T) {
	root := backendRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "internal/server/audit_tx.go"))
	if err != nil {
		t.Fatalf("قراءةُ معجم الأفعال الحسّاسة: %v", err)
	}
	block := string(src)
	i := strings.Index(block, "criticalAuditActions = map[string]bool{")
	if i < 0 {
		t.Fatal("**لا معجمَ للأفعال الحسّاسة** — ولا حارسَ بلا قائمة")
	}
	list := regexp.MustCompile(`"([a-z_.]+)":\s*true`).
		FindAllStringSubmatch(block[i:i+strings.Index(block[i:], "}")], -1)
	if len(list) == 0 {
		t.Fatal("**المعجمُ فارغ** — **وحارسٌ أعمى أسوأُ من لا حارس**")
	}

	// **ويُقرأ كلُّ استدعاءٍ لهذا الفعل في الخادم.**
	var files []string
	_ = filepath.Walk(filepath.Join(root, "internal/server"),
		func(p string, fi os.FileInfo, err error) error {
			if err == nil && !fi.IsDir() && strings.HasSuffix(p, ".go") &&
				!strings.HasSuffix(p, "_test.go") {
				files = append(files, p)
			}
			return nil
		})

	for _, m := range list {
		action := m[1]
		var bestEffort, transactional int
		for _, p := range files {
			b, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			body := string(b)
			bestEffort += strings.Count(body, `s.audit(r, "`+action+`"`)
			transactional += strings.Count(body, `"`+action+`"`) -
				strings.Count(body, `s.audit(r, "`+action+`"`)
		}
		t.Logf("%-24s معامليٌّ=%d · أفضلُ جهدٍ=%d", action, transactional, bestEffort)
		if bestEffort == 0 {
			continue
		}

		// ══════════════════════════════════════════════════════════
		// **وفعلٌ صنفُه يتقرّر بمعامله لا باسمه**
		// ══════════════════════════════════════════════════════════
		//
		// **`admin.setting_update` اسمٌ واحدٌ لفعلين**: **عمولةُ
		// المنصّة** و**نصُّ صفحة.** **والأوّلُ من الصنف `A`
		// والثاني لا** — بعقد دورةِ ٢١.
		//
		// **وكان هذا الحارسُ يقرأ الاسمَ صنفاً واحداً فيُدين الفرعَ
		// المشروع** (`XG-41A`) — **وحارسُ `XG-20` يعرف التفريع.**
		// **وحارسان يقرآن عقداً واحداً قراءتين أسوأُ من حارسٍ
		// واحد.**
		//
		// **فصار المصنِّفُ مكتوباً في الشيفرة مرّةً**
		// (`server.ConditionalAuditClassifier`) — **ويُقرأ منها.**
		if by := server.ConditionalAuditClassifier(action); by != "" {
			branched := false
			for _, p := range files {
				b, err := os.ReadFile(p)
				if err != nil {
					continue
				}
				body := string(b)
				if !strings.Contains(body, `s.audit(r, "`+action+`"`) {
					continue
				}
				// **والفرعُ يُقبَل بشرطين مقروءين في الملفّ نفسِه**:
				// **أن يحسم المصنِّفُ الكانونيُّ النداء**،
				// **وأن يكون للحسّاس فرعٌ معامليّ.**
				if strings.Contains(body, "!"+by+"(key)") &&
					strings.Contains(body, `s.auditTx(ctx, q, r, "`+action+`"`) {
					branched = true
					continue
				}
				branched = false
				break
			}
			if branched {
				t.Logf("  `%s`: **مفرَّعٌ بـ`%s`** — الحسّاسُ معامليٌّ "+
					"وغيرُه أفضلُ جهد", action, by)
				continue
			}
			t.Errorf("**`%s` فعلٌ مشروطٌ ونداؤه بأفضلِ جهدٍ بلا "+
				"`%s`** — **والفرعُ يُرى في الشيفرة أو لا يُفترَض.** "+
				"(`AQ-4`)", action, by)
			continue
		}

		t.Errorf("**`%s` فعلٌ حسّاسٌ يُدقَّق بأفضل جهد** — "+
			"**وسقوطُ أثرِه يترك مالاً تحرّك بلا من ولا متى.** (`PF-06`)",
			action)
	}
}
