package qa

// الفشلُ الجزئيّ — الحالُ والتدقيقُ والإشعار — **`P-6` البنود ٩…١٥ و١٧.**

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **١٥ · `AQ-4` — ذرّيّةُ التدقيق**
// ══════════════════════════════════════════════════════════════════════

// TestFAIL_AQ4_AuditAtomicity **البند ١٥ — العقدُ المعتمد.**
//
//	CRITICAL SUCCESS REQUIRES AUDIT SUCCESS
//
// **والقياسُ يقول ما تفعله الشيفرةُ اليوم** (`server/audit.go:42`):
//
//	«الكتابةُ في الخلفيّة ولا تُفشل الفعل»
//
// **فالفعلُ ينجح والقيدُ يسقط صامتاً** — والتعليقُ نفسُه يقول العاقبةَ:
// «من فتح سجلَّ الأحداث بعد شهرٍ لا يجد شيئاً — ويظنّ أنّ أحداً لم يفعلها».
func TestFAIL_AQ4_AuditAtomicity(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	admin := h.NewUser("admin")
	target := f.NewUserWith("customer")

	cases := []struct {
		name, path string
		body       map[string]any
		check      func(t *testing.T) (done bool, what string)
	}{
		{
			name: "قيدٌ يدويٌّ في محفظة", path: "/api/v1/admin/users/" + target.ID + "/wallet",
			body: map[string]any{"amount": 12_000, "kind": "topup", "note": "P-6"},
			check: func(t *testing.T) (bool, string) {
				var bal int64
				_ = h.Pool.QueryRow(ctxBG(),
					`SELECT COALESCE(balance,0) FROM wallets WHERE user_id = $1::uuid`,
					target.ID).Scan(&bal)
				return bal > 0, "رصيدٌ = " + itoa(int(bal))
			},
		},
		{
			name: "تعليقُ حساب", path: "/api/v1/admin/users/" + target.ID + "/status",
			body: map[string]any{"status": "suspended", "reason": "P-6"},
			check: func(t *testing.T) (bool, string) {
				var st string
				_ = h.Pool.QueryRow(ctxBG(),
					`SELECT status FROM users WHERE id = $1::uuid`, target.ID).Scan(&st)
				return st == "suspended", "الحالُ = " + st
			},
		},
	}

	broken := 0
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var auditBefore int
			_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log`).Scan(&auditBefore)

			fp := h.ArmAny("AQ-4/audit-write", "audit_log", "INSERT")
			got := h.POST(c.path, admin.Token, c.body)
			// **والتدقيقُ في خيطٍ منفصل** — يُنتظَر أن يبلغ النقطةَ.
			fired := fp.WaitFire(3 * time.Second)
			done, what := c.check(t)
			_ = fired

			var auditAfter int
			_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log`).Scan(&auditAfter)
			t.Logf("الردّ %d · %s · قيودُ التدقيق %d ⇒ %d · النقطةُ أصابت %d",
				got.Code, what, auditBefore, auditAfter, fp.Fired())

			if fp.Fired() == 0 {
				t.Skip("لم يُنادَ التدقيقُ في هذا المسار — لا شيءَ يُختبَر")
			}
			if done && auditAfter == auditBefore {
				broken++
				t.Logf("AQ-4 EXPECTED FAIL — الفعلُ الحسّاسُ وقع (%s) **ولا قيدَ تدقيقٍ له**", what)
				if got.Code < 400 {
					t.Logf("  ورُدَّ نجاحٌ — **فلا شيءَ يقول إنّ الأثرَ ضاع**")
				}
			} else if !done {
				t.Logf("الفعلُ لم يقع — والتدقيقُ حالَ دونه")
			}
			fp.Disarm()
		})
	}

	if broken > 0 {
		t.Logf("AQ-4 AUDIT ATOMICITY = EXPECTED FAIL — %d فعلٍ حسّاسٍ نجح بلا قيدِ تدقيق", broken)
		t.Logf("CONTRACT GAP — التدقيقُ اليومَ «أفضلُ جهد» والعقدُ يشترطه شرطاً للنجاح")
	} else {
		t.Logf("AQ-4 AUDIT ATOMICITY = PASS — لم ينجح فعلٌ حسّاسٌ بلا قيدِ تدقيق")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **١٠ · `R7` — قبولُ السائق**
// ══════════════════════════════════════════════════════════════════════

// TestFAIL_R7_DriverAcceptPartialState **البند ١٠.**
//
// **وخطواتُ القبول مقيسةٌ** (`driver_handlers.go:583…612`):
//
//	١ UPDATE orders SET driver_id …   ← شرطيٌّ ذرّيّ · يُردّ خطؤه
//	٢ UPDATE users SET last_assigned_at ← **خطؤه مُهمَلٌ ويُسجَّل فقط**
//	٣ Transition(assigned)            ← يكتب order_events ويبثّ
//
// **والثلاثُ بلا معاملةٍ واحدة** — فالسؤال: **أيبقى طلبٌ مملوكٌ بلا حدثٍ
// ولا انتقال؟**
func TestFAIL_R7_DriverAcceptPartialState(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	item := h.NewItem(1000)

	base := financialBaseline(t, h)
	oid := dispatchOrder(t, h, item)
	drv := onShiftDriver(t, f)

	// **النقطةُ على حدثِ الانتقال** — بعد أن صار الطلبُ مملوكاً.
	// **والنقطةُ على حدثِ الإسناد بعينه** — لا على أوّلِ حدثٍ يمرّ.
	// (**وكانت `order_id` فأصابت حدثاً سابقاً واستُهلكت** — فالتضييقُ
	// بالحال هو ما يجعلها تصيب الخطوةَ المقصودة.)
	fp := h.Arm("R7/step-3-order-event", "order_events", "INSERT", 1, "to_status", "assigned")
	got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil)
	t.Logf("الردّ: %d %s · النقطةُ أصابت %d", got.Code, got.Err(), fp.Fired())

	if fp.Fired() == 0 {
		t.Skip("لم يُكتب حدثٌ في هذا المسار — لا نافذةَ تُختبَر")
	}

	var driverID *string
	var status string
	var events int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT driver_id::text, status,
		       (SELECT count(*) FROM order_events WHERE order_id = $1::uuid)
		FROM orders WHERE id = $1::uuid`, oid).Scan(&driverID, &status, &events); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	owned := driverID != nil
	t.Logf("بعد الفشل: مملوكٌ=%v · الحالُ=%q · أحداثٌ=%d", owned, status, events)

	if owned && status == "dispatching" {
		t.Logf("R7 DRIVER ACCEPT = RISK CONFIRMED")
		t.Logf("  الطلبُ مملوكٌ لسائقٍ **وحالُه ما تزال dispatching** — **حالٌ غيرُ متطابقة**")
		t.Logf("  DEFECT CANDIDATE — يُعرَض على المالك ولا يُجمَّد (البند ٣١)")
	} else if owned && status == "assigned" {
		t.Logf("R7 = PASS — المِلكيّةُ والحالُ متطابقتان رغم سقوط الحدث")
	} else if !owned {
		t.Logf("R7 = PASS — لم يُملَّك الطلبُ أصلاً")
	}

	// ── والتعافي ────────────────────────────────────────────────────
	retry := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil)
	var status2 string
	var owner2 *string
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT status, driver_id::text FROM orders WHERE id = $1::uuid`, oid).Scan(&status2, &owner2)
	t.Logf("الإعادة: %d · الحالُ=%q · مملوكٌ=%v", retry.Code, status2, owner2 != nil)
	if owned && retry.Code >= 400 && status2 == "dispatching" {
		t.Logf("  RECOVERY = STUCK — الطلبُ مملوكٌ فلا يُقبَل ثانيةً، وحالُه لم تتقدّم")
		t.Logf("  ADMIN VISIBILITY = PARTIAL — يظهر في القوائم بحالٍ dispatching وله سائق")
	}
	assertNewViolations(t, h, base, "FI-06", "FI-05")
}

// ══════════════════════════════════════════════════════════════════════
// **١١ · `R8` — مطالبةٌ يتيمةٌ تحت فشلٍ حقيقيّ**
// ══════════════════════════════════════════════════════════════════════

// TestFAIL_R8_OrphanClaimUnderRealFailure **البند ١١.**
//
// **و`P-5` أثبت الحالَ بصنعها في القاعدة.** **وهذا يُثبتها بفشلٍ محقونٍ
// في المسار نفسِه**: المطالبةُ تُكتب، **ثمّ يسقط العملُ**، **ولا
// `releaseIdempotency` يُنادى إلّا من ردٍّ ≥٤٠٠ في الطلب نفسِه.**
func TestFAIL_R8_OrphanClaimUnderRealFailure(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	key := uniq("k")
	body := orderBody(item, 1)

	// **النقطةُ على إنشاء الطلب** — بعد أن تُكتب المطالبةُ في الوسيط.
	fp := h.ArmAny("R8/order-insert", "orders", "INSERT")
	first := h.POSTKey("/api/v1/orders", cust.Token, key, body)
	t.Logf("الأوّل: %d %s · النقطةُ أصابت %d", first.Code, first.Err(), fp.Fired())
	fp.MustFire(t)

	done, _, found := idemRow(t, h, cust.ID, ordersEndpoint, key)
	t.Logf("صفُّ المفتاح بعد الفشل: موجودٌ=%v · مُنجَزٌ=%v", found, done)

	retry := h.POSTKey("/api/v1/orders", cust.Token, key, body)
	t.Logf("الإعادةُ بالمفتاح نفسِه: %d %s", retry.Code, retry.Err())

	orders := h.CountOrders(cust.ID)
	t.Logf("طلباتُ الزبون: %d", orders)

	switch {
	case !found && retry.Code < 400:
		t.Logf("R8 = PASS — الردُّ ≥400 أطلق المطالبةَ · والإعادةُ نجحت · وطلبٌ واحدٌ")
		t.Logf("  **والمسارُ الطبيعيُّ يتعافى**: الخطأُ المُعادُ يُطلق المفتاح")
	case found && !done && retry.Code == 409:
		t.Logf("R8 ORPHAN IDEMPOTENCY CLAIM = RISK CONFIRMED")
		t.Logf("  المطالبةُ بقيت غيرَ مُنجَزةٍ والإعادةُ 409 — **والزبونُ محجوبٌ حتّى التقليم**")
		t.Logf("  DEFECT CANDIDATE — يُعرَض على المالك (البند ٣١)")
	default:
		t.Logf("R8 = PARTIAL — موجودٌ=%v · مُنجَزٌ=%v · الإعادةُ %d", found, done, retry.Code)
	}
	if orders > 1 {
		t.Errorf("أثرٌ مزدوج: %d طلباً", orders)
	}
	t.Logf("PROCESS-CRASH PROOF REQUIRES LATER ENVIRONMENT — " +
		"وهذا خطأٌ مُعادٌ لا موتُ عمليّة (البند ١٩)")
}

// ══════════════════════════════════════════════════════════════════════
// **١٣ · `R22` — وسمُ الإنذار قبل الإشعار**
// ══════════════════════════════════════════════════════════════════════

// TestFAIL_R22_WatchdogMarkerBeforeNotify **البند ١٣.**
//
// **والترتيبُ مقيسٌ** (`orders/watchdog.go:135`):
//
//	UPDATE orders SET alerted_at = now() WHERE alerted_at IS NULL   ← الوسم
//	if RowsAffected == 0 { continue }                                ← الحارس
//	s.notify.NotifyOps(…)                                            ← الإشعار
//
// **فالوسمُ يسبق الإشعار، والحارسُ يمنع الإعادة** — والسؤال: **إن سقط
// الإشعارُ، أيبقى الإنذارُ مكتوماً إلى الأبد؟**
func TestFAIL_R22_WatchdogMarkerBeforeNotify(t *testing.T) {
	h := New(t)
	item := h.NewItem(1000)
	oid := dispatchOrder(t, h, item)

	// **طلبٌ عالقٌ منذ زمن** — يُزاح الوقتُ في البيانة لا بانتظار.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET created_at = now() - interval '2 hours',
		                  updated_at = now() - interval '2 hours',
		                  alerted_at = NULL
		WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("إزاحةُ الزمن: %v", err)
	}

	// **النقطةُ على كتابة الإشعار** — بعد الوسم.
	fp := h.ArmAny("R22/notify-write", "notifications", "INSERT")

	// **ويُنادى الراصدُ من بابه الإداريّ** — لا بانتظار دورته.
	got := h.POST("/api/v1/admin/orders/watchdog/run", h.NewUser("admin").Token, nil)
	t.Logf("نداءُ الراصد: %d · النقطةُ أصابت %d", got.Code, fp.Fired())
	if got.Code == 404 || got.Code == 405 {
		t.Skipf("لا بابَ إداريٌّ للراصد — REQUIRES TESTABILITY SEAM (البند ١)")
	}

	var alerted bool
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT alerted_at IS NOT NULL FROM orders WHERE id = $1::uuid`, oid).Scan(&alerted)
	var notes int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE entity_id = $1`, oid).Scan(&notes)
	t.Logf("موسومٌ=%v · إشعاراتٌ=%d", alerted, notes)

	if alerted && notes == 0 {
		t.Logf("R22 WATCHDOG ALERT = RISK CONFIRMED")
		t.Logf("  الوسمُ كُتب والإشعارُ سقط — **والحارسُ يمنع الإعادةَ إلى الأبد**")
	} else if !alerted {
		t.Logf("R22 = PASS — لم يُوسَم حتّى نجح الإشعار")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٩ · `D15` — إنشاءُ مستخدمٍ إداريّاً**
// ══════════════════════════════════════════════════════════════════════

// TestFAIL_D15_AdminCreateUserPartial **البند ٩.**
//
// **وخطواتُه مقيسةٌ** (`identity/admin.go:80…95`): إنشاءُ المستخدم **ثمّ**
// منحُ كلِّ دورٍ في نداءٍ مستقلّ — **بلا معاملةٍ واحدة.**
func TestFAIL_D15_AdminCreateUserPartial(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	phone := uniqPhone()

	fp := h.ArmAny("D15/grant-role", "user_roles", "INSERT")
	got := h.POST("/api/v1/admin/users", admin.Token, map[string]any{
		"phone": phone, "full_name": "مستخدمُ QA", "password": "Passw0rd!234",
		"roles": []string{"driver"},
	})
	t.Logf("الردّ: %d %s · النقطةُ أصابت %d", got.Code, got.Err(), fp.Fired())
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `DELETE FROM users WHERE phone = $1`, phone)
	})

	if fp.Fired() == 0 {
		t.Skip("لم يُمنَح دورٌ في هذا المسار — لا نافذةَ تُختبَر")
	}

	var users, roles int
	var pwSet bool
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM users WHERE phone = $1),
		       (SELECT count(*) FROM user_roles r JOIN users u ON u.id = r.user_id
		        WHERE u.phone = $1),
		       COALESCE((SELECT COALESCE(password_hash,'') <> '' FROM users WHERE phone = $1), false)`,
		phone).Scan(&users, &roles, &pwSet)
	t.Logf("مستخدمون=%d · أدوارٌ=%d · كلمةٌ=%v", users, roles, pwSet)

	if users == 1 && roles == 0 {
		t.Logf("D15 ADMIN CREATE USER = DEFECT REPRODUCED")
		t.Logf("  مستخدمٌ بلا دورٍ واحد — **يدخل ولا يرى شيئاً، ولا يُنشأ ثانيةً**")
		retry := h.POST("/api/v1/admin/users", admin.Token, map[string]any{
			"phone": phone, "full_name": "مستخدمُ QA", "password": "Passw0rd!234",
			"roles": []string{"driver"},
		})
		t.Logf("  الإعادة: %d %s", retry.Code, retry.Err())
		if retry.Code >= 400 {
			t.Logf("  RETRY = BLOCKED — الهاتفُ محجوزٌ بمستخدمٍ ناقص · **ولا مسارَ إصلاح**")
			t.Logf("  ADMIN VISIBILITY = PARTIAL — يظهر في القائمة بلا أدوار")
		}
	} else if users == 0 {
		t.Logf("D15 = PASS — لم يبقَ مستخدمٌ بعد فشل منح الدور")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **١٧ · `XOB-7` — التحويلُ التلقائيُّ في خيطٍ منفصل**
// ══════════════════════════════════════════════════════════════════════

// TestFAIL_XOB7_AutoTransferFireAndForget **البند ١٧.**
//
// **والملاحظةُ مقيسةٌ** (`customer_handlers.go:362`):
//
//	go s.autoTransfer(context.WithoutCancel(r.Context()), o.ID, …)
//
// **فالردُّ يعود قبل أن يبدأ العمل** — **وسقوطُه لا يظهر لأحد.**
func TestFAIL_XOB7_AutoTransferFireAndForget(t *testing.T) {
	h := New(t)
	item := h.NewItem(1000)
	h.Setting("orders.auto_transfer_amount", "1")

	// **النقطةُ على حدثِ الطلب** — التحويلُ يكتب حدثاً حين ينجح.
	fp := h.ArmAny("XOB-7/auto-transfer-event", "order_events", "INSERT")
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	t.Logf("إنشاءُ الطلب: %d · النقطةُ أصابت %d", made.Code, fp.Fired())
	if made.Code >= 400 {
		t.Skipf("الإنشاءُ رُدّ: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	var status string
	var events, notes int
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT status,
		       (SELECT count(*) FROM order_events WHERE order_id = $1::uuid),
		       (SELECT count(*) FROM notifications WHERE entity_id = $1)`,
		oid).Scan(&status, &events, &notes)
	t.Logf("الحالُ=%q · أحداثٌ=%d · إشعاراتٌ=%d", status, events, notes)
	t.Logf("والردُّ للزبون كان %d — **بلا علاقةٍ بما جرى في الخيط**", made.Code)

	if fp.Fired() > 0 {
		t.Logf("XOB-7 = DEFECT CANDIDATE — عملُ الخيط سقط **والزبونُ رأى نجاحاً**")
		t.Logf("  ولا صفَّ دائمٌ ولا إعادةَ محاولةٍ — **يضيع بلا أثر**")
	} else {
		t.Logf("XOB-7 = UNPROVEN محلّيّاً — لم يبلغ الخيطُ النقطةَ قبل انتهاء الاختبار")
		t.Logf("  **وهذا وجهُ الملاحظة نفسِه**: عملٌ لا يُنتظَر ولا يُرصَد")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **١٤ · `R23` — دفعُ FCM · وما يمنع إثباتَه**
// ══════════════════════════════════════════════════════════════════════

// TestFAIL_P7SeamRequired_FCMTransport **البند ١ — ولا تُضاف خلسة.**
//
// **`push.Transport` واجهةٌ صالحةٌ للبديل** — **لكنّ الخدمةَ تُركَّب داخل
// `server.New` من متغيّرات البيئة** (`server.go:128`):
//
//	pushSvc := push.New(pg, logger)
//	if fcm, err := push.NewFCM(logger); err != nil { … } else { pushSvc = push.New(pg, logger, fcm) }
//
// **فلا مِعراضَ لحقن بديلٍ من اختبار** — والمِسنَدُ لا يمرّر ناقلاً.
//
// **ولا يُضاف مِعراضٌ من تلقائي** (البند ١): يُسجَّل ويُعرَض.
func TestFAIL_P7SeamRequired_FCMTransport(t *testing.T) {
	src, err := os.ReadFile("../server/server.go")
	if err != nil {
		t.Fatalf("قراءةُ المصدر: %v", err)
	}
	body := string(src)
	hasEnvWiring := strings.Contains(body, "push.NewFCM(")
	hasSeam := strings.Contains(body, "WithPushTransport")

	t.Logf("تركيبٌ من البيئة داخل server.New = %v · مِعراضُ حقنٍ = %v",
		hasEnvWiring, hasSeam)

	switch {
	case hasSeam && hasEnvWiring:
		t.Logf("SEAM ADDED — بإذن المالك ٢٠٢٦-٠٩-٠٥ (BEHAVIOR-PRESERVING)")
		t.Logf("  والافتراضُ ما يزال push.NewFCM من البيئة — والإنتاجُ لا يمرّر خياراً")
		t.Logf("  R23 صار يُثبَت — انظر TestEV_R23PushFailureIsLost")
		t.Logf("  والحارسُ في TestSeam_ProductionWiringUnchanged")
	case hasEnvWiring:
		t.Errorf("TESTABILITY SEAM REQUIRED — R23 لا يُثبَت بلا موضعِ حقن")
	default:
		t.Errorf("تركيبُ الدفع تبدّل — يُعاد القياسُ قبل الحكم")
	}
}

// TestFAIL_D15_Reconciliation **مصالحةُ `D15` — بقرار المالك قبل `P-7`.**
//
// # التعارض
//
//	المراجعةُ الساكنة : «حسابٌ بأدواره بلا كلمة مرور · والإعادةُ تردّ phone_taken»
//	اختبارُ P-6      : users = 0 · لا مستخدمَ جزئيٌّ بقي · PASS
//
// # والسببُ مقيسٌ: **اختباري ضرب خطوةً أخرى**
//
// **خطواتُ `AdminCreateUser` أربعٌ** (`identity/admin.go:78…101`):
//
//	١ CreateUserWithRole — **مستخدمٌ + الدورُ الأوّلُ في معاملةٍ واحدة**
//	٢ GrantRole لبقيّة الأدوار — **ولا تُنادى إن كان الدورُ واحداً**
//	٣ SetTempPassword — **خارجَ أيّ معاملة** (`repo.go:396`)
//	٤ Audit
//
// **واختبارُ `P-6` سلّح نقطةً على `user_roles`** — فأصابت الإدراجَ داخلَ
// معاملة الخطوة الأولى **فرُجعت كلُّها**، وذلك سليمٌ ومتوقَّع.
// **والدورُ كان واحداً فلم تُنادَ الخطوةُ الثانيةُ أصلاً.**
//
// **فما اختُبر ليس ما يدّعيه `D15`.**
func TestFAIL_D15_Reconciliation(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")

	create := func(phone string, roles ...string) Res {
		return h.POST("/api/v1/admin/users", admin.Token, map[string]any{
			"phone": phone, "full_name": "مستخدمُ QA", "password": "Passw0rd!234",
			"roles": roles,
		})
	}
	read := func(phone string) (users, roles int, pwSet bool) {
		_ = h.Pool.QueryRow(ctxBG(), `
			SELECT (SELECT count(*) FROM users WHERE phone = $1),
			       (SELECT count(*) FROM user_roles r JOIN users u ON u.id = r.user_id
			        WHERE u.phone = $1),
			       COALESCE((SELECT COALESCE(password_hash,'') <> '' FROM users WHERE phone = $1), false)`,
			phone).Scan(&users, &roles, &pwSet)
		return
	}

	// ── أ · الخطوةُ الثالثة — **ما يدّعيه `D15` بعينه** ──────────────
	t.Run("فشلُ كلمة المرور", func(t *testing.T) {
		phone := uniqPhone()
		t.Cleanup(func() {
			_, _ = h.Pool.Exec(context.Background(), `DELETE FROM users WHERE phone = $1`, phone)
		})
		// **النقطةُ على التحديث الذي يضع كلمةَ المرور وحدَه** —
		// يُميَّز بأنّه يُجبر التبديل.
		fp := h.Arm("D15/set-temp-password", "users", "UPDATE", 1,
			"must_change_password", "true")
		got := create(phone, "driver")
		users, roles, pwSet := read(phone)
		t.Logf("الردّ %d %s · النقطةُ أصابت %d", got.Code, got.Err(), fp.Fired())
		t.Logf("مستخدمون=%d · أدوارٌ=%d · كلمةٌ=%v", users, roles, pwSet)

		if fp.Fired() == 0 {
			t.Skip("لم تُصَب خطوةُ كلمة المرور — يُعاد القياسُ قبل الحكم")
		}
		if users == 1 && roles >= 1 && !pwSet {
			t.Logf("D15 STILL VALID — **حسابٌ بأدواره بلا كلمة مرور**")
			retry := create(phone, "driver")
			t.Logf("الإعادة: %d %s", retry.Code, retry.Err())
			if retry.Err() == "phone_taken" {
				t.Logf("  وRETRY = BLOCKED بـphone_taken — **كما قال السجلُّ حرفاً**")
			}
		} else if users == 0 {
			t.Logf("D15 DISPROVEN في هذا المسار — لا مستخدمَ بقي")
		}
		fp.Disarm()
	})

	// ── ب · الخطوةُ الثانية — **دوران لا دورٌ واحد** ────────────────
	t.Run("فشلُ منح الدور الثاني", func(t *testing.T) {
		phone := uniqPhone()
		t.Cleanup(func() {
			_, _ = h.Pool.Exec(context.Background(), `DELETE FROM users WHERE phone = $1`, phone)
		})
		// **دورٌ أوّلُ ينجح · وثانٍ يسقط** — النقطةُ تصيب الثاني.
		//
		// **وكان الثاني `ops` يُطلَب في جسم الإنشاء** — **وبابُ الإنشاء
		// صار لصفةِ الحساب وحدَها** (٢٠٢٦-٠٩-١٢)، **فالنداءُ يُردّ قبل
		// أيّ كتابةٍ فلا تُصيب النقطةُ ويصير الفحصُ تخطّياً أبديّاً.**
		//
		// **والنافذةُ نفسُها باقيةٌ بغير طلب**: **`driver` يُمنَح معه
		// `customer` تلقائيّاً** (قرارُ المالك ٢٠٢٦-٠٨-١٠) — **فصفّان
		// يُكتبان، والنقطةُ على الثاني.**
		fp := h.Arm("D15/grant-second-role", "user_roles", "INSERT", 1, "role_code", "customer")
		got := create(phone, "driver")
		users, roles, pwSet := read(phone)
		t.Logf("الردّ %d %s · النقطةُ أصابت %d", got.Code, got.Err(), fp.Fired())
		t.Logf("مستخدمون=%d · أدوارٌ=%d · كلمةٌ=%v", users, roles, pwSet)
		if fp.Fired() == 0 {
			t.Skip("لم يُطلَب الدورُ الثاني — لا نافذةَ")
		}
		if users == 1 && roles == 1 && !pwSet {
			t.Logf("D15 STILL VALID (وجهٌ ثانٍ) — **حسابٌ بدورٍ واحدٍ من اثنين وبلا كلمة مرور**")
		}
		fp.Disarm()
	})

	// ── ج · الخطوةُ الأولى — **وهي ما اختبره `P-6`** ────────────────
	t.Run("فشلُ الإنشاء نفسِه", func(t *testing.T) {
		phone := uniqPhone()
		t.Cleanup(func() {
			_, _ = h.Pool.Exec(context.Background(), `DELETE FROM users WHERE phone = $1`, phone)
		})
		fp := h.Arm("D15/create-user", "user_roles", "INSERT", 1, "role_code", "driver")
		got := create(phone, "driver")
		users, roles, _ := read(phone)
		t.Logf("الردّ %d · مستخدمون=%d · أدوارٌ=%d · النقطةُ أصابت %d",
			got.Code, users, roles, fp.Fired())
		if users == 0 {
			t.Logf("STEP-1 = ATOMIC — المعاملةُ في CreateUserWithRole تُرجع الاثنين معاً")
		}
		fp.Disarm()
	})
}
