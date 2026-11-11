package orders_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// TestCustomPhotos_AttachRules — **صورُ الطلب الخاصّ**: تُربط لصاحبها وحدَه، مرّةً واحدة،
// وبسقف ثلاث، وتظهر في قراءة الطلب والحمولةِ لكلّ من الزبون والسائق والمكتب.
func TestCustomPhotos_AttachRules(t *testing.T) {
	pool := testdb.Pool(t)
	store := settings.NewStore(pool)
	svc := orders.NewService(pool, nil, wallet.NewService(pool),
		cashbox.NewService(pool, store), nil,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	svc.SetSettings(store)
	owner := testdb.NewUser(t, pool, "customer")
	other := testdb.NewUser(t, pool, "customer")
	ctx := context.Background()

	mk := func(by, kind string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO media (kind, path, thumb_path, width, height, bytes, created_by)
			VALUES ($1, 'x/'||gen_random_uuid()||'.jpg', 'x/'||gen_random_uuid()||'_t.jpg', 10, 10, 10, $2)
			RETURNING id::text`, kind, by).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	newOrder := func() string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO orders (kind, customer_id, status, address_text, dropoff,
				payment_method, custom_request, subtotal, delivery_fee, total, cash_due,
				snap_merchant_commission_percent, snap_rep_commission_percent,
				snap_commission_source, snap_activation_orders)
			VALUES ('custom', $1, 'pending', 'عنوان اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				'cash', 'طلب', 0, 0, 0, 0, `+qaSnapSQLX()+`)
			RETURNING id::text`, owner).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}

	order := newOrder()
	a, b := mk(owner, "custom_photo"), mk(owner, "custom_photo")
	if err := svc.AttachCustomPhotosTx(ctx, pool, order, owner, []string{a, b}); err != nil {
		t.Fatalf("ربطُ صورتَي صاحبِها يجب أن ينجح: %v", err)
	}
	o, err := svc.GetByID(ctx, order)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.CustomPhotos) != 2 || !strings.HasPrefix(o.CustomPhotos[0], "/media/x/") {
		t.Fatalf("الصورتان لم تظهرا في قراءة الطلب: %v", o.CustomPhotos)
	}
	for _, aud := range []orders.Audience{orders.AudienceCustomer, orders.AudienceDriver, orders.AudienceOps} {
		if v := orders.ViewFor(aud, o); v["custom_photos"] == nil {
			t.Errorf("الحمولةُ لـ%s بلا صور", aud)
		}
	}
	if v := orders.ViewFor(orders.AudienceMerchant, o); v["custom_photos"] != nil {
		t.Errorf("صورُ الطلب الخاصّ وصلت المتجر")
	}

	// صورةٌ مستعملةٌ، وصورةٌ لغير صاحبها، وصورةٌ من صنفٍ آخر، وفوقَ الثلاث، ومكرَّرة.
	o2 := newOrder()
	cases := map[string][]string{
		"مستعملة":    {a},
		"غريبة":      {mk(other, "custom_photo")},
		"صنف آخر":    {mk(owner, "avatar")},
		"فوق السقف":  {mk(owner, "custom_photo"), mk(owner, "custom_photo"), mk(owner, "custom_photo"), mk(owner, "custom_photo")},
		"معرّف فاسد": {"not-a-uuid"},
	}
	for name, ids := range cases {
		if err := svc.AttachCustomPhotosTx(ctx, pool, o2, owner, ids); err == nil {
			t.Errorf("%s: كان يجب أن تُردّ", name)
		}
	}
	dup := mk(owner, "custom_photo")
	if err := svc.AttachCustomPhotosTx(ctx, pool, o2, owner, []string{dup, dup}); err == nil {
		t.Errorf("مكرَّرة: كان يجب أن تُردّ")
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM order_custom_photos WHERE order_id = $1`, o2).Scan(&n)
	if n != 0 {
		t.Errorf("ربطٌ جزئيٌّ بعد الرفض: %d", n)
	}
}
