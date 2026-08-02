package support_test

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// setup طلبٌ مغلقٌ منذ `ago`، وخدمةُ دعمٍ تقرأ إعداداتها.
func setup(t *testing.T, status string, ago time.Duration) (*support.Service, string, string) {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()

	svc := support.NewService(pool, nil, wallet.NewService(pool))
	svc.SetSettings(settings.NewStore(pool))

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات في القاعدة: %v", err)
	}
	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, owner_user_id)
		VALUES ('متجر اختبار الشكاوى', $1, 10, $2) RETURNING id`,
		categoryID, testdb.NewUser(t, pool, "merchant")).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	customer := testdb.NewUser(t, pool, "customer")
	var orderID string
	closed := any(nil)
	if ago >= 0 {
		closed = time.Now().Add(-ago)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due, closed_at)
		VALUES ($1, $2, $3, 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 100000, 10000, 110000, 0, 110000, $4)
		RETURNING id`, customer, merchantID, status, closed).Scan(&orderID); err != nil {
		t.Fatalf("تعذّر إنشاء طلب: %v", err)
	}
	return svc, customer, orderID
}

// TestComplaint_Opens الزبونُ يفتح شكواه بنفسه — **وهو البابُ الذي لم يكن.**
func TestComplaint_Opens(t *testing.T) {
	svc, customer, orderID := setup(t, "delivered", time.Hour)
	tk, err := svc.Complaint(context.Background(), customer, orderID, "not_received", "انتظرتُ ولم يصل")
	if err != nil {
		t.Fatalf("تعذّر فتح الشكوى: %v", err)
	}
	if tk.Number == 0 {
		t.Error("شكوى بلا رقم — لا يملك صاحبُها أن يسأل عنها")
	}
	if tk.Reason != "not_received" {
		t.Errorf("السبب = %q، والمتوقّع not_received", tk.Reason)
	}
	if !tk.OpenedByCustomer {
		t.Error("لم تُنسب إلى صاحبها")
	}
	// **والنصُّ يُحفظ بجانب السبب**: القائمةُ تُصنّف والنصُّ يشرح.
	if len(tk.Replies) != 1 {
		t.Errorf("عددُ الردود = %d، والمتوقّع 1 (شرحُ الزبون)", len(tk.Replies))
	}
}

// TestComplaint_OnlyOnce طلبٌ لا يُشتكى منه مرّتين وشكواه الأولى مفتوحة.
//
// **بلا هذا يفتح الزبونُ عشراً بضغطاتٍ متتالية** حين لا يأتيه ردٌّ فوريّ،
// فيُغرق مكتبَ الدعم بشكوًى واحدةٍ مكرّرة **ويضيع بينها ما يستحقّ النظر.**
func TestComplaint_OnlyOnce(t *testing.T) {
	svc, customer, orderID := setup(t, "delivered", time.Hour)
	ctx := context.Background()
	if _, err := svc.Complaint(ctx, customer, orderID, "late", ""); err != nil {
		t.Fatalf("الأولى فشلت: %v", err)
	}
	if _, err := svc.Complaint(ctx, customer, orderID, "quality", ""); err != support.ErrComplaintOpen {
		t.Errorf("الثانية مرّت أو ردّت خطأً آخر: %v", err)
	}
}

// TestComplaint_WindowPasses شكوى بعد انقضاء المهلة تُردّ.
//
// **لأن الذاكرةَ تُنسى والدليلَ يذهب**: السائقُ لا يذكر والبضاعةُ ذهبت، **ولا
// يبقى إلّا كلمةٌ ضدّ كلمة** — فتُقبل بلا بيّنة أو تُردّ بلا بيّنة.
func TestComplaint_WindowPasses(t *testing.T) {
	svc, customer, orderID := setup(t, "delivered", 30*time.Hour)
	if _, err := svc.Complaint(context.Background(), customer, orderID, "late", ""); err != support.ErrComplaintWindow {
		t.Errorf("قُبلت بعد المهلة أو رُدّت بخطأ آخر: %v", err)
	}
}

// TestComplaint_ReasonFollowsStatus «لم يصلني طلبي» لا تُقبل على طلبٍ لم يُسلَّم.
//
// **حالتُه تقول ذلك أصلاً**، وسؤالُ المستخدم عمّا نعرفه يجعله يشكّ فيما نعرف.
func TestComplaint_ReasonFollowsStatus(t *testing.T) {
	svc, customer, orderID := setup(t, "cancelled", time.Hour)
	ctx := context.Background()
	if _, err := svc.Complaint(ctx, customer, orderID, "not_received", ""); err != support.ErrBadReason {
		t.Errorf("قُبل سببٌ لا يخصّ الحالة: %v", err)
	}
	// **وما يخصّ كلَّ الحالات يمرّ**: من أُلغي طلبُه بلا سببٍ يفهمه يشكو.
	if _, err := svc.Complaint(ctx, customer, orderID, "other", "أُلغي بلا سبب"); err != nil {
		t.Errorf("رُدّ سببٌ عامّ: %v", err)
	}
}

// TestComplaint_NotOnRunningOrder لا شكوى على طلبٍ يجري — **يُتابَع لا يُشتكى منه.**
func TestComplaint_NotOnRunningOrder(t *testing.T) {
	svc, customer, orderID := setup(t, "on_the_way", -1)
	if _, err := svc.Complaint(context.Background(), customer, orderID, "late", ""); err != support.ErrOrderNotClosed {
		t.Errorf("قُبلت على طلبٍ يجري: %v", err)
	}
}
