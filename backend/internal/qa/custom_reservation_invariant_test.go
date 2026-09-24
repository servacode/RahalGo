package qa

// ══════════════════════════════════════════════════════════════════════
// **حجزُ الطلب المخصَّص — الحارسُ الماليُّ يراه فاسداً** — Batch 2a
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك: الثوابتُ مسنودةٌ باختباراتٍ سالبة — نحقن فساداً عمداً
//  فيجب أن يسقط الحارس؛ **وحارسٌ لا نراه يسقط لا نثق أنّه يحرس.**)
//
// # ما يُحقَن
//
//   - `FI-11.e`: `wallets.reserved` لا يطابق (سحوباتٌ + حجوزُ مخصَّص) — حجزٌ شائخ.
//   - `FI-11.l`: طلبٌ مخصَّصٌ انتهى وبقي له حجز.
//   - `FI-11.m`: حجزٌ موجبٌ لا يقابل تأكيداً حيّاً بنسخته الحاليّة.
//
// **ثمّ أساسٌ صحيحٌ نظيف** — الثوابتُ نفسُها لا تسقط على حالٍ سليمة.

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// crUser يزرع مستخدماً ويُرجع معرّفَه.
func crUser(t *testing.T, pool *pgxpool.Pool, phone string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (phone, full_name, status)
		VALUES ($1, 'زبونُ الحجز', 'active') RETURNING id::text`, phone).Scan(&id); err != nil {
		t.Fatalf("زرعُ مستخدم: %v", err)
	}
	return id
}

// ── الحجزُ الشائخ في المجموع يُسقط FI-11.e ──────────────────────────────
func TestCustomReservation_StaleAggregateFailsFININV(t *testing.T) {
	pool := fixtureDB(t, "rahalgo_custom_resv_e")
	ctx := context.Background()

	var uid string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (phone, full_name, status)
		VALUES ('0900555001', 'صاحبُ محفظة', 'active') RETURNING id::text`).Scan(&uid); err != nil {
		t.Fatalf("مستخدم: %v", err)
	}
	// **محفظةٌ محجوزٌ فيها 100 بلا سحبٍ ولا طلبٍ مخصَّصٍ يقابله** — حجزٌ شائخ.
	// (balance=100 كي لا يسقط FI-11.d بدل FI-11.e.)
	if _, err := pool.Exec(ctx, `
		INSERT INTO wallets (user_id, balance, reserved) VALUES ($1, 100, 100)
		ON CONFLICT (user_id) DO UPDATE SET balance = 100, reserved = 100`, uid); err != nil {
		t.Fatalf("محفظة: %v", err)
	}

	v, err := fininv.Run(ctx, pool, "FI-11.e")
	if err != nil {
		t.Fatalf("تشغيلُ FI-11.e: %v", err)
	}
	if len(v) == 0 {
		t.Fatal("**حجزٌ شائخٌ لم يُخرَق فيه FI-11.e** — فالحارسُ لا يرى الحجزَ غيرَ المنسوب")
	}
	t.Logf("FI-11.e أمسك %d خرقاً على حجزٍ شائخ", len(v))
}

// ── طلبٌ مخصَّصٌ انتهى وبقي له حجز يُسقط FI-11.l ─────────────────────────
func TestCustomReservation_TerminalResidueFailsFININV(t *testing.T) {
	pool := fixtureDB(t, "rahalgo_custom_resv_l")
	ctx := context.Background()
	cust := crUser(t, pool, "0900555002")

	// **طلبٌ ملغيٌّ ومع ذلك يحمل حجزاً** — القيدُ يسمح به (محفظةٌ مؤكَّدة)،
	// **والحارسُ الماليُّ وحدَه يراه.**
	if _, err := pool.Exec(ctx, `
		INSERT INTO orders (customer_id, kind, status, address_text, dropoff, payment_method,
		                    subtotal, delivery_fee, total,
		                    quote_version, quote_confirmed_at, quote_confirmed_total,
		                    quote_confirmed_version, custom_reserved_amount)
		VALUES ($1, 'custom', 'cancelled', 'الرقة',
		        ST_SetSRID(ST_MakePoint(39.0,35.95),4326)::geography, 'wallet',
		        50, 50, 50, 1, now(), 50, 1, 50)`, cust); err != nil {
		t.Fatalf("طلبٌ ملغيٌّ بحجز: %v", err)
	}

	v, err := fininv.Run(ctx, pool, "FI-11.l")
	if err != nil {
		t.Fatalf("تشغيلُ FI-11.l: %v", err)
	}
	if len(v) == 0 {
		t.Fatal("**طلبٌ انتهى وبقي له حجزٌ لم يُخرَق فيه FI-11.l**")
	}
	t.Logf("FI-11.l أمسك %d خرقاً على بقايا حجزٍ لطلبٍ منتهٍ", len(v))
}

// ── حجزٌ لا يقابل تأكيداً حيّاً بنسخته الحاليّة يُسقط FI-11.m ─────────────
func TestCustomReservation_StaleConfirmationFailsFININV(t *testing.T) {
	pool := fixtureDB(t, "rahalgo_custom_resv_m")
	ctx := context.Background()
	cust := crUser(t, pool, "0900555003")

	// **نسخةُ العرض 2، والتأكيدُ على 1** — حجزٌ حيٌّ لتأكيدٍ شائخ.
	if _, err := pool.Exec(ctx, `
		INSERT INTO orders (customer_id, kind, status, address_text, dropoff, payment_method,
		                    subtotal, delivery_fee, total,
		                    quote_version, quote_confirmed_at, quote_confirmed_total,
		                    quote_confirmed_version, custom_reserved_amount)
		VALUES ($1, 'custom', 'assigned', 'الرقة',
		        ST_SetSRID(ST_MakePoint(39.0,35.95),4326)::geography, 'wallet',
		        50, 50, 50, 2, now(), 50, 1, 50)`, cust); err != nil {
		t.Fatalf("طلبٌ بتأكيدٍ شائخ: %v", err)
	}

	v, err := fininv.Run(ctx, pool, "FI-11.m")
	if err != nil {
		t.Fatalf("تشغيلُ FI-11.m: %v", err)
	}
	if len(v) == 0 {
		t.Fatal("**حجزٌ لا يطابق تأكيداً حيّاً لم يُخرَق فيه FI-11.m**")
	}
	t.Logf("FI-11.m أمسك %d خرقاً على تأكيدٍ شائخ", len(v))
}

// ── أساسٌ صحيح: طلبُ محفظةٍ حيٌّ مؤكَّدٌ ومحجوزٌ مطابق — لا خرق ────────────
func TestCustomReservation_ValidLiveOrderClean(t *testing.T) {
	pool := fixtureDB(t, "rahalgo_custom_resv_ok")
	ctx := context.Background()
	cust := crUser(t, pool, "0900555004")

	// **محفظةٌ محجوزٌ فيها 50 يقابلها طلبٌ حيٌّ مؤكَّدٌ بنسخته الحاليّة.**
	if _, err := pool.Exec(ctx, `
		INSERT INTO wallets (user_id, balance, reserved) VALUES ($1, 1000, 50)
		ON CONFLICT (user_id) DO UPDATE SET balance = 1000, reserved = 50`, cust); err != nil {
		t.Fatalf("محفظة: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO orders (customer_id, kind, status, address_text, dropoff, payment_method,
		                    subtotal, delivery_fee, total,
		                    quote_version, quote_confirmed_at, quote_confirmed_total,
		                    quote_confirmed_version, custom_reserved_amount)
		VALUES ($1, 'custom', 'assigned', 'الرقة',
		        ST_SetSRID(ST_MakePoint(39.0,35.95),4326)::geography, 'wallet',
		        50, 50, 50, 1, now(), 50, 1, 50)`, cust); err != nil {
		t.Fatalf("طلبٌ حيٌّ مؤكَّد: %v", err)
	}

	v, err := fininv.Run(ctx, pool, "FI-11.e", "FI-11.j", "FI-11.k", "FI-11.l", "FI-11.m")
	if err != nil {
		t.Fatalf("تشغيلُ الثوابت: %v", err)
	}
	for _, x := range v {
		t.Errorf("**خرقٌ على أساسٍ صحيح** — %s", x)
	}
}
