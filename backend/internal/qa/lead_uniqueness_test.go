package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **مرشَّحٌ واحدٌ لا يصير متجرين** — `XG-18` · `D25` · `C-03`
// ══════════════════════════════════════════════════════════════════════
//
// # الهويّةُ الكانونيّة — مُقاسةٌ لا مُخترَعة
//
// **ليست الهاتفَ ولا المالك**: **متجرٌ ثانٍ لصاحبٍ واحدٍ مسموحٌ عمداً**
// — «فالمتجر قد يكون فرعاً جديداً بحقّ» (`handlePublicJoin`)، **وإنّما
// يسقط عنه إسنادُ المندوب.**
//
// **وإنّما هي المرشَّح**: **طلبُ انضمامٍ واحدٌ يُنتج متجراً واحداً على
// الأكثر.** وهو ما يقوله حارسُ `convertLead` نفسُه: `merchantID != nil
// ⇒ return`. **وهو ما يحرسه `C-03`**:
//
//	ONE REAL MERCHANT IDENTITY MUST NOT BECOME
//	TWO PAYABLE MERCHANT/REP RELATIONSHIPS
//
// # ولماذا كان يُخرَق
//
// **قراءةٌ ثمّ كتابةٌ بلا قفل**: يقرأ النداءان `merchant_id` فيريانه
// فارغاً معاً، **ثمّ يُنشئ كلٌّ منهما متجراً.** **والفحصُ في الشيفرة
// لا يمنع سباقاً** — **من فحص ثمّ كتب ترك بينهما فجوةً.**

// leadFacts ما يقوله الدفترُ عن مرشَّحٍ بعد المحاولات.
type leadFacts struct {
	Merchants  int // متاجرُ نشأت عن هذا المرشَّح
	RepOwned   int // منها المنسوبُ للمندوب
	Owners     int // حساباتُ صاحبِ المتجر بهذا الهاتف
	LeadStatus string
	Rewards    int // قيودُ مكافأةِ هدفٍ للمندوب
}

func factsOf(t *testing.T, h *Harness, leadID, phone, repID string) leadFacts {
	t.Helper()
	var f leadFacts
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM merchants WHERE phone = $1),
		       (SELECT count(*) FROM merchants
		         WHERE phone = $1 AND sales_rep_user_id = $3::uuid),
		       (SELECT count(*) FROM users WHERE phone = $1),
		       COALESCE((SELECT status FROM merchant_leads WHERE id = $2::uuid), ''),
		       (SELECT count(*) FROM incentives
		         WHERE user_id = $3::uuid AND for_target)`,
		phone, leadID, repID).Scan(&f.Merchants, &f.RepOwned, &f.Owners,
		&f.LeadStatus, &f.Rewards); err != nil {
		t.Fatalf("قراءةُ الوقائع: %v", err)
	}
	return f
}

func newLeadFor(t *testing.T, h *Harness, repID, categoryID, phone string) string {
	t.Helper()
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO merchant_leads
		       (store_name, owner_name, phone, area, category_id, lat, lng,
		        status, sales_rep_user_id, owner_password_hash)
		VALUES ($1, $2, $3, 'الرقّة', $4::uuid, 35.95, 39.01, 'new', $5::uuid, 'x')
		RETURNING id::text`,
		uniq("متجرُ تفرّدٍ "), "صاحبُه", phone, categoryID, repID).Scan(&id); err != nil {
		t.Skipf("تعذّر تجهيزُ مرشَّح: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = h.Pool.Exec(c, `DELETE FROM merchant_leads WHERE id = $1::uuid`, id)
		_, _ = h.Pool.Exec(c, `DELETE FROM merchants WHERE phone = $1`, phone)
	})
	return id
}

// ══════════════════════════════════════════════════════════════════════
// **١ · نداءان متوازيان على المرشَّح نفسِه**
// ══════════════════════════════════════════════════════════════════════
//
// **تداخلٌ حقيقيٌّ مقيس** — لا نداءان متتاليان. **والمِسنَدُ `P-5`
// يقيس التداخلَ ويُسقط الجولةَ إن لم يقع.**
func TestUNIQ_ConcurrentLeadConversionMakesOneMerchant(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	admin := h.NewUser("admin")

	var categoryID string
	if err := h.Pool.QueryRow(ctxBG(),
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("تصنيفُ تفرّدٍ ")).Scan(&categoryID); err != nil {
		t.Fatalf("تصنيف: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM categories WHERE id = $1::uuid`, categoryID)
	})

	// **وجولةُ تسخينٍ خارجَ القياس.**
	//
	// **أوّلُ نداءٍ بعد إقلاع الحاوية يبني الاتّصالاتِ ويُترجم
	// الاستعلامات** — **فيتجاوز مهلةَ السباق ويُقرأ عَلَقاً وليس
	// كذلك.** (وقع مرّتين، وعلى قاعدةٍ دافئةٍ يمضي في أجزاء الثانية.)
	warm := newLeadFor(t, h, f.RepAccount().ID, categoryID, f.NS.Phone())
	_ = h.POST("/api/v1/admin/leads/"+warm+"/status", admin.Token,
		map[string]any{"status": "converted"})

	base := financialBaseline(t, h)
	const rounds = 6
	var dup, dupOwner, dupRep int

	for i := 0; i < rounds; i++ {
		rep := f.RepAccount()
		phone := f.NS.Phone()
		leadID := newLeadFor(t, h, rep.ID, categoryID, phone)
		body := map[string]any{"status": "converted"}

		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "إداريّ-أ", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
			}},
			Actor{Name: "إداريّ-ب", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
			}},
		)
		if r.TimedOut {
			t.Fatalf("**الجولة %d عَلِقت** — ونداءان يتزاحمان على مرشَّحٍ "+
				"لا يجوز أن يحبس أحدُهما الآخرَ إلى الأبد.\n%s", i+1, r)
		}
		if r.Probe.Max() < 2 {
			t.Errorf("الجولة %d: تداخلٌ مقيسٌ %d — **والسيناريو يشترط تزامناً**",
				i+1, r.Probe.Max())
		}

		got := factsOf(t, h, leadID, phone, rep.ID)
		t.Logf("الجولة %d: متاجرُ=%d (للمندوب %d) · أصحابُ=%d · المرشَّحُ=%q · "+
			"مكافآتُ=%d · نجح %d من 2",
			i+1, got.Merchants, got.RepOwned, got.Owners, got.LeadStatus,
			got.Rewards, r.CountOK())

		if got.Merchants > 1 {
			dup++
		}
		if got.Owners > 1 {
			dupOwner++
		}
		if got.RepOwned > 1 {
			dupRep++
		}
		// **والنتيجةُ حتميّة** — **ولا `500` لأحدهما.**
		//
		// **والعقدُ القائمُ يجعل التحويلَ المكرَّرَ `200`**: من ناداه
		// على مرشَّحٍ محوَّلٍ يُردّ عليه «تمّ» ولا يُنشأ شيء
		// (`convertLead`: `merchantID != nil ⇒ return nil`).
		// **وهو `canonical existing result` لا خطأ** — **والحكمُ على
		// الأثر لا على رمز الردّ**: متجرٌ واحدٌ لا اثنان.
		for _, o := range r.Outcomes {
			if res, ok := o.Value.(Res); ok && res.Code >= 500 {
				t.Errorf("الجولة %d: %s ردّ %d — **وسباقٌ متوقَّعٌ لا يُردّ "+
					"بعطبٍ داخليّ**", i+1, o.Name, res.Code)
			}
		}
		if got.LeadStatus != "converted" {
			t.Errorf("الجولة %d: حالُ المرشَّح %q — **وواحدٌ نجح فيجب أن يثبت**",
				i+1, got.LeadStatus)
		}
	}

	if dup > 0 {
		t.Errorf("**متجران من مرشَّحٍ واحد** في %d من %d جولة — "+
			"`XG-18` · `D25` · `C-03`", dup, rounds)
	}
	if dupOwner > 0 {
		t.Errorf("**حسابا صاحبِ متجرٍ لهاتفٍ واحد** في %d جولة", dupOwner)
	}
	if dupRep > 0 {
		t.Errorf("**نسبتان للمندوب** في %d جولة — **وهي مصدرُ عمولةٍ مضاعفة**",
			dupRep)
	}
	assertNewViolations(t, h, base, "FI-02", "FI-03", "FI-05")
}

// ══════════════════════════════════════════════════════════════════════
// **١·ب · وصاحبُ المتجر له حسابٌ من قبل**
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا هذا هو السيناريو الحقيقيّ
//
// **الحمايةُ القائمةُ عرَضيّةٌ لا مقصودة**: النداءان يُنشئان حسابَ
// صاحبِ المتجر، **فيصطدمان بتفرّد `users.phone`** فيسقط أحدُهما.
//
// **ومن كان له حسابٌ من قبلُ لا اصطدامَ فيه** — **يقرأ النداءان
// الحسابَ القائمَ ويمضيان.** **وهي الحالُ الشائعة**: صاحبُ المتجر
// زبونٌ عندنا أصلاً، أو سُجّل مرشَّحاً ورُدّ ثمّ عاد.
//
// **فالقيدُ الحقيقيُّ لم يُوجَد بعد** — والذي يحمي اليومَ يحمي بالصدفة.
func TestUNIQ_ConcurrentConversionWithExistingOwner(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	admin := h.NewUser("admin")

	var categoryID string
	_ = h.Pool.QueryRow(ctxBG(),
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("تصنيفُ صاحبٍ ")).Scan(&categoryID)
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM categories WHERE id = $1::uuid`, categoryID)
	})

	base := financialBaseline(t, h)
	const rounds = 6
	var dup, dupRep int

	for i := 0; i < rounds; i++ {
		rep := f.RepAccount()
		phone := f.NS.Phone()

		// **حسابُ صاحبِ المتجر قائمٌ قبل التحويل** — بدوره.
		if _, err := h.Pool.Exec(ctxBG(), `
			WITH u AS (
			  INSERT INTO users (phone, full_name) VALUES ($1, 'صاحبٌ قائم')
			  RETURNING id)
			INSERT INTO user_roles (user_id, role_code)
			SELECT id, 'merchant' FROM u`, phone); err != nil {
			t.Fatalf("تجهيزُ صاحبٍ قائم: %v", err)
		}
		leadID := newLeadFor(t, h, rep.ID, categoryID, phone)
		t.Cleanup(func() {
			_, _ = h.Pool.Exec(context.Background(), `DELETE FROM users WHERE phone = $1`, phone)
		})

		body := map[string]any{"status": "converted"}
		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "إداريّ-أ", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
			}},
			Actor{Name: "إداريّ-ب", Do: func(ctx context.Context) any {
				return h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
			}},
		)
		if r.TimedOut {
			t.Fatalf("الجولة %d عَلِقت — %s", i+1, r)
		}
		got := factsOf(t, h, leadID, phone, rep.ID)
		t.Logf("الجولة %d: متاجرُ=%d (للمندوب %d) · أصحابُ=%d · المرشَّحُ=%q · نجح %d من 2 — %s",
			i+1, got.Merchants, got.RepOwned, got.Owners, got.LeadStatus, r.CountOK(), r)
		if got.Merchants > 1 {
			dup++
		}
		if got.RepOwned > 1 {
			dupRep++
		}
	}

	if dup > 0 {
		t.Errorf("**متجران من مرشَّحٍ واحدٍ لصاحبٍ قائم** في %d من %d جولة — "+
			"**والحمايةُ كانت تفرّدَ الهاتف لا تفرّدَ المرشَّح.**", dup, rounds)
	}
	if dupRep > 0 {
		t.Errorf("**نسبتان للمندوب** في %d جولة — **عمولةٌ مضاعفةٌ عن جلبةٍ واحدة**",
			dupRep)
	}
	assertNewViolations(t, h, base, "FI-02", "FI-03", "FI-05")
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · عطبٌ ثمّ إعادة — ولا متجرَ ثانٍ**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو الدليلُ الذي كان `D2` يُنتجه**: يسقط التثبيتُ فتُعاد المحاولةُ
// **فيُنشأ متجرٌ ثانٍ.** **وقد صارت العمليّةُ ذرّيّةً في دورةِ ٤** —
// **وهذا يحرس أنّ الإعادةَ لا تُخرج مخرجاً ثانياً حتّى لو عاد العطب.**
func TestUNIQ_FailureThenRetryMakesOneMerchant(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	admin := h.NewUser("admin")

	var categoryID string
	_ = h.Pool.QueryRow(ctxBG(),
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("تصنيفُ إعادةٍ ")).Scan(&categoryID)
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM categories WHERE id = $1::uuid`, categoryID)
	})

	rep := f.RepAccount()
	phone := f.NS.Phone()
	leadID := newLeadFor(t, h, rep.ID, categoryID, phone)
	body := map[string]any{"status": "converted"}

	fp := h.Arm("D2/step-6-commit-conversion", "merchant_leads", "UPDATE",
		1, "status", "converted")
	first := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
	fp.MustFire(t)
	afterFail := factsOf(t, h, leadID, phone, rep.ID)
	t.Logf("بعد العطب: الردُّ %d · متاجرُ=%d · أصحابُ=%d · المرشَّحُ=%q",
		first.Code, afterFail.Merchants, afterFail.Owners, afterFail.LeadStatus)

	retry := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
	after := factsOf(t, h, leadID, phone, rep.ID)
	t.Logf("بعد الإعادة: الردُّ %d · متاجرُ=%d · أصحابُ=%d · المرشَّحُ=%q · مكافآتُ=%d",
		retry.Code, after.Merchants, after.Owners, after.LeadStatus, after.Rewards)

	if after.Merchants > 1 {
		t.Errorf("**الإعادةُ أنشأت متجراً ثانياً** (%d)", after.Merchants)
	}
	if after.Owners > 1 {
		t.Errorf("**الإعادةُ أنشأت صاحباً ثانياً** (%d)", after.Owners)
	}
	if retry.Code >= 400 {
		t.Errorf("**الإعادةُ سقطت** (%d) — والتعافي يجب أن يكون ممكناً", retry.Code)
	}
	if after.LeadStatus != "converted" {
		t.Errorf("المرشَّحُ %q بعد إعادةٍ ناجحة", after.LeadStatus)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · القاعدةُ ترفض ولو سقط حارسُ الشيفرة**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُوثَق بفحصٍ في الشيفرة وحدَه** — **من فحص ثمّ كتب ترك بينهما
// فجوة.** **والقيدُ في المخطَّط لا فجوةَ فيه.**
func TestUNIQ_DatabaseRefusesSecondMerchantForSameLead(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	admin := h.NewUser("admin")

	var categoryID string
	_ = h.Pool.QueryRow(ctxBG(),
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("تصنيفُ قيدٍ ")).Scan(&categoryID)
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM categories WHERE id = $1::uuid`, categoryID)
	})

	rep := f.RepAccount()
	phone := f.NS.Phone()
	leadID := newLeadFor(t, h, rep.ID, categoryID, phone)
	if got := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token,
		map[string]any{"status": "converted"}); got.Code >= 400 {
		t.Fatalf("التحويلُ الأوّل: %d %s", got.Code, got.Err())
	}

	var madeFrom string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(lead_id::text, '') FROM merchants WHERE phone = $1`,
		phone).Scan(&madeFrom); err != nil {
		t.Fatalf("قراءةُ نسبِ المتجر: %v", err)
	}
	if madeFrom != leadID {
		t.Fatalf("**المتجرُ لا يقول من أيّ مرشَّحٍ جاء**: %q", madeFrom)
	}

	// **محاولةٌ مباشرةٌ تتخطّى كلَّ حارسٍ في الشيفرة.**
	var owner string
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT owner_user_id::text FROM merchants WHERE phone = $1`, phone).Scan(&owner)
	_, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO merchants (name, category_id, phone, owner_user_id, lead_id)
		VALUES ($1, $2::uuid, $3, $4::uuid, $5::uuid)`,
		uniq("متجرٌ ثانٍ "), categoryID, phone, owner, leadID)
	if err == nil {
		t.Error("**القاعدةُ قبلت متجراً ثانياً للمرشَّح نفسِه** — " +
			"**والحارسُ في الشيفرة وحدَه لا يمنع سباقاً.**")
	} else {
		t.Logf("القاعدةُ ردّت الثانيَ — الحارسُ في المخطَّط")
	}
}
