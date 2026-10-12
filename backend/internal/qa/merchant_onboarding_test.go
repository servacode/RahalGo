package qa

// **سلسلةُ تعريف المتجر بالإشعارات** (طلبُ المالك ٢٠٢٦-١٠-١٢): إشعاران في اليوم
// (١١ والرابعة بدمشق)، **لا اثنان في موعدٍ واحد، ولا شيءَ خارج الموعدين،
// و«لسا ما جرّبتها؟» لا يصل من طلب «لدي توصيلة».**

import (
	"testing"
	"time"
)

func obInbox(t *testing.T, h *Harness, userID string) []string {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `SELECT title FROM notifications WHERE user_id = $1 ORDER BY created_at, title`, userID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func obAt(day, hour int) time.Time {
	loc := time.FixedZone("Damascus", 3*3600)
	return time.Date(2031, 3, day, hour, 5, 0, 0, loc)
}

func TestOB01_TwoADayInOrderThenStops(t *testing.T) {
	h := New(t)
	h.Setting("merchants.onboarding_push", "true")
	h.Setting("merchants.onboarding_push_messages", `"أول | ١\nتاني | ٢\nتالت | ٣"`)
	f := h.Factory()
	m := f.Merchant()
	ctx := ctxBG()

	if n := h.API.MerchantOnboardingTickForTest(ctx, obAt(1, 9)); n != 0 {
		t.Fatalf("**أُرسل خارج الموعد** — %d", n)
	}
	h.API.MerchantOnboardingTickForTest(ctx, obAt(1, 11))
	h.API.MerchantOnboardingTickForTest(ctx, obAt(1, 11)) // **نفسُ الموعد — لا ثانٍ**
	if got := obInbox(t, h, m.Owner.ID); len(got) != 1 || got[0] != "أول" {
		t.Fatalf("بعد موعد ١١: %v", got)
	}
	h.API.MerchantOnboardingTickForTest(ctx, obAt(1, 16))
	h.API.MerchantOnboardingTickForTest(ctx, obAt(2, 11))
	h.API.MerchantOnboardingTickForTest(ctx, obAt(2, 16)) // **انتهت القائمة**
	got := obInbox(t, h, m.Owner.ID)
	if len(got) != 3 || got[1] != "تاني" || got[2] != "تالت" {
		t.Fatalf("**الترتيبُ أو العدد خطأ** — %v", got)
	}
}

func TestOB02_SkipsTryReminderForUsers(t *testing.T) {
	h := New(t)
	h.Setting("merchants.onboarding_push", "true")
	h.Setting("merchants.onboarding_push_messages", `"أول | ١\nلسا ما جرّبتها؟ | جرّب | إذا ما جرّب"`)
	md := newMDFx(t, h, "OB02")
	md.create(t, h, "merchant", uniq("ob02"))
	owner := md.fx.M.Owner.ID
	ctx := ctxBG()
	h.API.MerchantOnboardingTickForTest(ctx, obAt(1, 11))
	h.API.MerchantOnboardingTickForTest(ctx, obAt(1, 16))
	for _, title := range obInbox(t, h, owner) {
		if title == "لسا ما جرّبتها؟" {
			t.Fatal("**«لسا ما جرّبتها؟» وصل من طلب توصيلةً**")
		}
	}
}

func TestOB03_OffSendsNothing(t *testing.T) {
	h := New(t)
	h.Setting("merchants.onboarding_push", "false")
	if n := h.API.MerchantOnboardingTickForTest(ctxBG(), obAt(1, 11)); n != 0 {
		t.Fatalf("**أُرسل والمفتاحُ مطفأ** — %d", n)
	}
}

// TestOB03_ReachesBeyondTheFirstThousand **كلُّ المتاجر لا أوّلُ ألفٍ منها** (٢٠٢٦-١٠-١٢) —
// كان `LIMIT 1000` بلا ترتيبٍ ولا تكرار، فأسقط `OB01` في قاعدةٍ فيها أكثرُ من ألف متجر.
func TestOB03_ReachesBeyondTheFirstThousand(t *testing.T) {
	h := New(t)
	h.Setting("merchants.onboarding_push", "true")
	h.Setting("merchants.onboarding_push_messages", `"أول | ١\nتاني | ٢"`)
	ctx := ctxBG()
	var ids []string
	rows, err := h.Pool.Query(ctx, `
		WITH u AS (
			INSERT INTO users (phone, full_name, created_at)
			SELECT '+96388' || lpad(($1::int * 10000 + g)::text, 7, '0'), 'زحمة ' || g,
			       now() - interval '1 day'
			FROM generate_series(1, 1005) g
			RETURNING id)
		INSERT INTO user_roles (user_id, role_code) SELECT id, 'merchant' FROM u RETURNING user_id::text`,
		int(time.Now().UnixNano()%900)+1)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM users WHERE id = ANY($1::uuid[])`, ids)
	})
	m := h.Factory().Merchant()
	h.API.MerchantOnboardingTickForTest(ctx, obAt(1, 11))
	if got := obInbox(t, h, m.Owner.ID); len(got) != 1 || got[0] != "أول" {
		t.Fatalf("**المتجرُ بعد الألف لم يصله شيء**: %v", got)
	}
}
