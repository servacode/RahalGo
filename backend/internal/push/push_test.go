package push

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

// fakeTransport ناقلٌ يسجّل ما وصله ويردّ ما يُملى عليه.
type fakeTransport struct {
	sent []string
	msg  Message
	dead []string
	err  error
}

func (f *fakeTransport) Platform() string { return PlatformAndroid }
func (f *fakeTransport) Send(_ context.Context, tokens []string, msg Message) ([]string, error) {
	f.sent = append(f.sent, tokens...)
	f.msg = msg
	return f.dead, f.err
}

// ══════════════════════════════════════════════════════════════════════
// **الرمزُ ينتقل مع الجهاز — لا يبقى عند صاحبه القديم**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا أخطرُ ما في الجدول.** هاتفٌ يُسلَّم لسائقٍ آخرَ — وهو أمرٌ واقعٌ
// في أسطولٍ صغير — **يجب أن ينقطع عن حساب الأوّل فوراً.**
//
// **ولو بقي الصفّان معاً** لَوصل إشعارُ السائق الأوّل — وفيه اسمُ الزبون
// وعنوانُه — **إلى هاتفٍ صار عند غيره.** وهو تسريبٌ لا يُكتشف: كلُّ شيءٍ
// يبدو سليماً عند الطرفين.
func TestPush_TokenMovesToNewOwner(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := New(pool, quietLogger())

	first := testdb.NewUser(t, pool, "driver")
	second := testdb.NewUser(t, pool, "driver")
	const token = "tok-shared-phone"

	if err := svc.Register(ctx, first, token, PlatformAndroid, "1.0"); err != nil {
		t.Fatalf("تسجيلُ الأوّل: %v", err)
	}
	if err := svc.Register(ctx, second, token, PlatformAndroid, "1.0"); err != nil {
		t.Fatalf("تسجيلُ الثاني: %v", err)
	}

	var owner string
	var rows int
	if err := pool.QueryRow(ctx,
		`SELECT user_id::text, count(*) OVER () FROM device_tokens WHERE token = $1`,
		token).Scan(&owner, &rows); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	if rows != 1 {
		t.Fatalf("الرمزُ في %d صفّاً — **فيصل إشعارُ السائق الأوّل إلى هاتفٍ صار عند غيره**", rows)
	}
	if owner != second {
		t.Fatal("الرمزُ لم ينتقل إلى صاحبه الجديد — **فيبقى الهاتفُ يستقبل إشعاراتِ من سلّمه**")
	}
}

// TestPush_UnregisterBoundToOwner **ولا يُسكت أحدٌ إشعاراتِ غيره.**
//
// **ولولا قيدُ `user_id`** لَاستطاع أيُّ داخلٍ يعرف رمزاً أن يحذفه —
// **فيصمت هاتفُ سائقٍ منافسٍ ولا يعرف لماذا.**
func TestPush_UnregisterBoundToOwner(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := New(pool, quietLogger())

	owner := testdb.NewUser(t, pool, "driver")
	other := testdb.NewUser(t, pool, "driver")
	const token = "tok-mine"

	if err := svc.Register(ctx, owner, token, PlatformAndroid, "1.0"); err != nil {
		t.Fatalf("تسجيل: %v", err)
	}
	if err := svc.Unregister(ctx, other, token); err != nil {
		t.Fatalf("حذفٌ من غير صاحبه: %v", err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM device_tokens WHERE token = $1`, token).Scan(&n)
	if n != 1 {
		t.Fatal("غيرُ صاحبِ الرمز حذفه — **فيصمت هاتفُ غيره ولا يعرف لماذا**")
	}

	if err := svc.Unregister(ctx, owner, token); err != nil {
		t.Fatalf("حذفٌ من صاحبه: %v", err)
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM device_tokens WHERE token = $1`, token).Scan(&n)
	if n != 0 {
		t.Fatal("صاحبُ الرمز لم يستطع حذفَه — **فيبقى الهاتفُ يستقبل بعد الخروج**")
	}
}

// TestPush_DeadTokensDropped **والمرفوضُ يُحذف — ولا يُنادى عليه إلى الأبد.**
func TestPush_DeadTokensDropped(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "driver")

	tr := &fakeTransport{dead: []string{"tok-dead"}}
	svc := New(pool, quietLogger(), tr)
	if err := svc.Register(ctx, uid, "tok-dead", PlatformAndroid, "1.0"); err != nil {
		t.Fatalf("تسجيل: %v", err)
	}
	if err := svc.Register(ctx, uid, "tok-live", PlatformAndroid, "1.0"); err != nil {
		t.Fatalf("تسجيل: %v", err)
	}

	svc.SendToUser(ctx, uid, Message{Title: "طلبٌ جديد", Urgent: true})

	if len(tr.sent) != 2 {
		t.Fatalf("وصل الناقلَ %d رمزاً من ٢ — **فجهازٌ من أجهزته لا يُنبَّه**", len(tr.sent))
	}
	if !tr.msg.Urgent {
		t.Fatal("الرسالةُ العاجلةُ وصلت عاديّةً — **فيؤجّلها `Doze` إلى نافذة صيانةٍ قد تبعد ساعة**")
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM device_tokens WHERE token = 'tok-dead'`).Scan(&n)
	if n != 0 {
		t.Fatal("الرمزُ الميّتُ بقي — **يُنادى عليه في كلّ حدثٍ إلى الأبد ويستهلك حصّةً**")
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM device_tokens WHERE token = 'tok-live'`).Scan(&n)
	if n != 1 {
		t.Fatal("حُذف رمزٌ حيٌّ مع الميّت")
	}
}

// TestPush_DisabledIsSilent **وبلا ناقلٍ لا تسقط ولا تُرسل.**
//
// **وهو عهدُ الحزمة**: المنصّةُ عملت بلا دفعٍ حتّى ٢٠٢٦-٠٨-١١، **وجهازُ
// المطوّر بلا مفتاح.** ولو ذُعرت هنا لَسقط كلُّ نداءٍ يُنشئ إشعاراً.
func TestPush_DisabledIsSilent(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "customer")
	svc := New(pool, quietLogger())

	if svc.Enabled() {
		t.Fatal("خدمةٌ بلا ناقلٍ تقول إنّها مهيّأة")
	}
	svc.SendToUser(ctx, uid, Message{Title: "خبر"}) // لا يجوز أن يذعر
}

// TestPush_NilServiceIsSafe **وخدمةٌ فارغةٌ تمرّ** — الاختباراتُ تبنيها كذلك.
func TestPush_NilServiceIsSafe(t *testing.T) {
	var svc *Service
	ctx := context.Background()
	svc.SendToUser(ctx, "u", Message{Title: "x"})
	if err := svc.Register(ctx, "u", "t", PlatformAndroid, ""); err != nil {
		t.Fatalf("تسجيلٌ على خدمةٍ فارغة: %v", err)
	}
	if svc.Enabled() {
		t.Fatal("خدمةٌ فارغةٌ تقول إنّها مهيّأة")
	}
}

// TestPush_PlatformClosedList **والقائمةُ مغلقة** — كما نوعُ الجلسة.
func TestPush_PlatformClosedList(t *testing.T) {
	for _, raw := range []string{"", "IOS", "web", "huawei", "android"} {
		got := normalizePlatform(raw)
		if got != PlatformAndroid && got != PlatformIOS {
			t.Fatalf("منصّةٌ مجهولةٌ %q صارت %q — **فيُخزَّن نصٌّ لا ناقلَ له "+
				"ولا يصل صاحبَه شيءٌ أبداً**", raw, got)
		}
	}
	if normalizePlatform(PlatformIOS) != PlatformIOS {
		t.Fatal("آيفون صار أندرويد — **فيُرسَل إلى ناقلٍ لا يعرفه**")
	}
}
