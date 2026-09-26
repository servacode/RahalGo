package server

// ══════════════════════════════════════════════════════════════════════
// **عتادُ شهادةِ تطبيق المتجر على الجهاز** — staging-only (Merchant E2E)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك: شهادةٌ حيّةٌ كاملةٌ لتطبيق المتجر على SM-A525F.)
//
// # ما يُهيّئه
//
// **مالكُ متجرٍ ثابتٌ برقمٍ وكلمةٍ معلومَين** (يدخل بهما على الجهاز)، **يملك
// متجرَين** (لشهود مبدّل الفروع وعزلها، B8)، **كلاهما في نقطةِ عنوانِ زبون
// QA نفسِها** — فيقعان في منطقته فيَطلب منهما، **ومفتوحان دائماً** (بلا صفوف
// ساعات ⇒ `OpenNowSQL` يعدّهما مفتوحَين) **نشطان**، **ولكلٍّ صنفٌ واحدٌ
// مُعتمَدٌ متاح** — فيُعرَض للزبون ويُطلَب.
//
// **ولا يمسّ إنتاجاً**: المسارُ غيرُ مسجَّلٍ إلّا على التجهيز (`qaStagingEnabled`
// في `handleQAStagingSeed`)، وكلُّ ما يبذره كياناتُ QA بأسماءٍ محصورة.
//
// # عكوسٌ
//
// `merchant_device_clear` يحذف المتجرَين وأصنافَهما وأقسامَهما (FK-safe).
// **والمالكُ حسابُ QA ثابتٌ يبقى** كأقرانه (زبونُ/مندوبُ QA) — لا سرَّ فيه.

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

const (
	// qaMerchantOwnerPhone **مالكُ متجرِ QA** — بعيدٌ عن أرقام البذور الأخرى
	// (زبون 555001/2، أدمن 555009، مندوب 555010/11، حذف 555999).
	qaMerchantOwnerPhone = "+963900555020"
	qaMerchantOwnerName  = "مالك متجر الاختبار QA"
	qaDeviceStoreA       = "QA-Merchant-Device-A"
	qaDeviceStoreB       = "QA-Merchant-Device-B"
	// نقطةُ الرقّة الاحتياطيّة — إن لم يكن لزبون QA عنوانٌ افتراضيّ بعد.
	qaFallbackLat = 35.9528
	qaFallbackLng = 39.0079
)

// qaMerchantDeviceSetup **يُهيّئ مالكَ متجرِ QA ومتجرَيه** — reusable، يُعيد
// المعرّفاتِ والدخول. — kind=merchant_device_setup (على التجهيز وحدَه).
func (s *Server) qaMerchantDeviceSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ① المالك — مستخدمٌ بدور `merchant` وكلمةٍ معلومة.
	owner, err := s.identity.EnsureUserWithRole(
		ctx, "", qaMerchantOwnerPhone, "merchant", qaMerchantOwnerName, qaCustomerPassword, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وتُضبط الكلمةُ صراحةً** — فلو كان قائماً بكلمةٍ أخرى صار الدخولُ معلوماً.
	if err := s.identity.QASetPassword(ctx, owner.ID, qaCustomerPassword); err != nil {
		s.respondErr(w, err)
		return
	}

	// ② نقطةُ المتجرَين — عنوانُ زبون QA الافتراضيّ (فيقعان في منطقته)، أو الرقّة.
	lat, lng := qaFallbackLat, qaFallbackLng
	_ = s.pg.QueryRow(ctx, `
		SELECT ST_Y(a.location::geometry), ST_X(a.location::geometry)
		FROM addresses a JOIN users u ON u.id = a.user_id
		WHERE u.phone = $1
		ORDER BY a.is_default DESC, a.created_at DESC
		LIMIT 1`, qaStagingPhone).Scan(&lat, &lng)

	storeA, err := s.qaEnsureDeviceStore(ctx, owner.ID, qaDeviceStoreA, lat, lng)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	storeB, err := s.qaEnsureDeviceStore(ctx, owner.ID, qaDeviceStoreB, lat, lng)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	s.logger.Warn("QA merchant device set up (staging-only)",
		"owner", owner.ID, "store_a", storeA, "store_b", storeB, "lat", lat, "lng", lng)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"owner_id": owner.ID, "phone": qaMerchantOwnerPhone, "password": qaCustomerPassword,
		"store_a": storeA, "store_b": storeB, "lat": lat, "lng": lng,
	})
}

// qaEnsureDeviceStore **يُنشئ (أو يُعيد) متجرَ جهازٍ مملوكاً** — نشطٌ، بنقطةٍ،
// بلا صفوفِ ساعاتٍ (مفتوحٌ دائماً)، وبصنفٍ واحدٍ مُعتمَدٍ متاح.
func (s *Server) qaEnsureDeviceStore(ctx context.Context, ownerID, name string, lat, lng float64) (string, error) {
	// **متكرّرٌ آمن**: يُعاد إن كان قائماً، ويُثبَّت مالكُه ونقطتُه ونشاطُه.
	var mid string
	err := s.pg.QueryRow(ctx, `SELECT id::text FROM merchants WHERE name = $1`, name).Scan(&mid)
	if err == nil {
		if _, uerr := s.pg.Exec(ctx, `
			UPDATE merchants SET owner_user_id = $2::uuid, status = 'active',
			   emergency_closed = false,
			   location = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography
			WHERE id = $1::uuid`, mid, ownerID, lng, lat); uerr != nil {
			return "", uerr
		}
		return mid, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := tx.QueryRow(ctx, `
		INSERT INTO merchants (name, owner_user_id, category_id, commission_percent, status, location)
		VALUES ($1, $2::uuid, (SELECT id FROM categories ORDER BY id LIMIT 1), 10, 'active',
		        ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography)
		RETURNING id::text`, name, ownerID, lng, lat).Scan(&mid); err != nil {
		return "", err
	}
	var secID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name)
		VALUES ($1::uuid, 'الرئيسية') RETURNING id::text`, mid).Scan(&secID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO menu_items
		  (merchant_id, section_id, platform_section_id, name, merchant_price, price, available, approved)
		VALUES ($1::uuid, $2::uuid,
		        (SELECT id FROM platform_sections WHERE active ORDER BY sort_order LIMIT 1),
		        'QA صنف الجهاز', 10000, 10000, true, true)`, mid, secID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return mid, nil
}

// qaMerchantDeviceClear **يحذف متجرَي الجهاز وأصنافَهما وأقسامَهما** — FK-safe.
// **ولا يمسّ المالكَ** (حسابُ QA ثابت). — kind=merchant_device_clear.
func (s *Server) qaMerchantDeviceClear(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	names := []string{qaDeviceStoreA, qaDeviceStoreB}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx,
		`DELETE FROM menu_items WHERE merchant_id IN (SELECT id FROM merchants WHERE name = ANY($1))`, names); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM menu_sections WHERE merchant_id IN (SELECT id FROM merchants WHERE name = ANY($1))`, names); err != nil {
		s.respondErr(w, err)
		return
	}
	tag, err := tx.Exec(ctx, `DELETE FROM merchants WHERE name = ANY($1)`, names)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA merchant device cleared (staging-only)", "deleted", tag.RowsAffected())
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted_merchants": tag.RowsAffected()})
}
