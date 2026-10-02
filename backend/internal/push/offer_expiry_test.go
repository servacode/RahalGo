package push

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// **عرضُ الطلب يُدفَع عاجلاً بأجله — ولا يُرسَل بعد موته** (٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **العرضُ صار صفّاً** (`order_offer`) ليصل هاتفاً تطبيقُه مغلق. **وصفٌّ يُدفَع
// بلا أجلٍ يُسلَّم بعد دقائق** — فيضغط السائقُ على طلبٍ ذهب لغيره.

// recTransport **يحفظ رسالةَ كلّ رمز** — القاعدةُ مشتركة، **وجولةٌ واحدةٌ قد
// تحمل إشعاراتِ غيرنا**، فلا يُقرأ «آخرُ ما أُرسل».
type recTransport struct{ got map[string]Message }

func (r *recTransport) Platform() string { return PlatformAndroid }
func (r *recTransport) Send(_ context.Context, tokens []string, msg Message) ([]string, error) {
	for _, t := range tokens {
		r.got[t] = msg
	}
	return nil, nil
}

func offerRow(t *testing.T, svc *Service, user, collapse string, ttlSec int) string {
	t.Helper()
	var id string
	if err := svc.db.QueryRow(context.Background(), `
		INSERT INTO notifications (user_id, kind, title, body, entity, entity_id,
		            href, push_pending, push_apps, expires_at, collapse_key)
		VALUES ($1, 'order_offer', 'عرض', 'نص', 'order', gen_random_uuid()::text,
		        '', true, '{}', now() + make_interval(secs => $2::bigint), $3)
		RETURNING id::text`, user, ttlSec, collapse).Scan(&id); err != nil {
		t.Fatalf("تعذّر إدراجُ العرض: %v", err)
	}
	return id
}

// TestOfferPush_UrgentWithTTLAndCollapse **يصل عاجلاً بما بقي من مهلته ومفتاحِ طيّه.**
func TestOfferPush_UrgentWithTTLAndCollapse(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	ft := &recTransport{got: map[string]Message{}}
	svc := New(pool, quietLogger(), ft)

	driver := testdb.NewUser(t, pool, "driver")
	tok := "tok-offer-" + driver[:8]
	if err := svc.Register(ctx, driver, tok, PlatformAndroid, "driver", "1.0"); err != nil {
		t.Fatalf("تسجيلُ الجهاز: %v", err)
	}
	offerRow(t, svc, driver, "offer:x", 60)

	svc.DeliverOnce(ctx)
	msg, ok := ft.got[tok]
	if !ok {
		t.Fatal("العرضُ لم يُرسَل — **والتطبيقُ مغلقٌ لا يعلم بطلبٍ معروضٍ عليه**")
	}
	if !msg.Urgent {
		t.Error("العرضُ أُرسل عادياً — **وأندرويد يؤجّل العاديَّ في السبات فيفوت العرض**")
	}
	if msg.TTL <= 0 || msg.TTL > 61*time.Second {
		t.Errorf("مهلةُ العرض عند المزوّد %v — والمتوقّعُ ما بقي من الدقيقة", msg.TTL)
	}
	if msg.Collapse != "offer:x" {
		t.Errorf("مفتاحُ الطيّ %q — **وعرضان للطلب نفسِه يتراكمان في الشريط**", msg.Collapse)
	}
	if msg.Data["kind"] != "order_offer" {
		t.Errorf("النوعُ في الحمولة %q — والتطبيقُ يعرف `order_offer` عاجلاً", msg.Data["kind"])
	}
}

// TestOfferPush_ExpiredIsNotSent **عرضٌ مات لا يُرسَل.**
func TestOfferPush_ExpiredIsNotSent(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	ft := &recTransport{got: map[string]Message{}}
	svc := New(pool, quietLogger(), ft)

	driver := testdb.NewUser(t, pool, "driver")
	tok := "tok-dead-offer-" + driver[:8]
	if err := svc.Register(ctx, driver, tok, PlatformAndroid, "driver", "1.0"); err != nil {
		t.Fatalf("تسجيلُ الجهاز: %v", err)
	}
	id := offerRow(t, svc, driver, "offer:y", 60)
	// **يُفرَّع وهو حيّ، ثمّ يموت قبل أن يُرسَل** — وجهُ التأخّر الحقيقيّ.
	svc.fanOut(ctx)
	if _, err := pool.Exec(ctx,
		`UPDATE notifications SET expires_at = now() - interval '1 second' WHERE id = $1`, id); err != nil {
		t.Fatalf("إنضاجُ الأجل: %v", err)
	}
	svc.DeliverOnce(ctx)
	if _, sent := ft.got[tok]; sent {
		t.Fatal("عرضٌ انقضى أجلُه أُرسل — **فيضغط السائقُ على طلبٍ ذهب لغيره**")
	}
	var state, class string
	if err := pool.QueryRow(ctx, `
		SELECT state, last_error_class FROM notification_deliveries
		WHERE notification_id = $1`, id).Scan(&state, &class); err != nil {
		t.Fatalf("قراءةُ صفّ النقل: %v", err)
	}
	if state != "failed" || class != classExpired {
		t.Errorf("صفُّ النقل %s/%s — والمتوقّعُ failed/%s", state, class, classExpired)
	}

	// **وما مات قبل التفريع لا يُفرَّع أصلاً** — وتُطفأ علامتُه.
	id2 := offerRow(t, svc, driver, "offer:z", 60)
	if _, err := pool.Exec(ctx,
		`UPDATE notifications SET expires_at = now() - interval '1 second' WHERE id = $1`, id2); err != nil {
		t.Fatalf("إنضاجُ الأجل: %v", err)
	}
	svc.DeliverOnce(ctx)
	var pending bool
	var targets int
	if err := pool.QueryRow(ctx, `
		SELECT n.push_pending,
		       (SELECT count(*) FROM notification_deliveries d WHERE d.notification_id = n.id)
		FROM notifications n WHERE n.id = $1`, id2).Scan(&pending, &targets); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	if pending || targets != 0 {
		t.Errorf("عرضٌ ميّتٌ قبل التفريع: معلَّق=%v · أهداف=%d", pending, targets)
	}
}
