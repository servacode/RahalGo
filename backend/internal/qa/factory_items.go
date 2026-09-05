package qa

// أصنافٌ لمتجرٍ من المصنع — **وشرطٌ سابقٌ لثوابت `P-4`.**
//
// **و`NewItem` القائمةُ تنشئ متجرَها بنفسها** — فلا تصلح لسيناريو يشترط
// متجراً بمندوبٍ أو بنسبةِ عمولةٍ بعينها. **وهذا يكملها ولا يبدّلها**
// (البند ٤): **لا فكسچرَ ماليٌّ يدويٌّ متناثرٌ إن كان المصنعُ يكفي.**

import (
	"context"
	"fmt"
)

// NewItemFor صنفٌ في متجرٍ من المصنع — سعرُ بيعِه سعرُ كلفتِه (**بلا هامش**).
func (h *Harness) NewItemFor(m *Merchant, cost int64) *Item {
	h.T.Helper()
	return h.NewItemPriced(m, cost, cost)
}

// NewItemPriced صنفٌ بكلفةٍ وسعرِ بيعٍ مستقلَّين — **والفرقُ هو الهامش.**
//
// **وعليه تُبنى مصفوفةُ مصدر العمولة** (البند ٨): الهامشُ
// `unit_price − merchant_price` هو ما تحسب منه عمولةُ المندوب
// (`OrderMarginSQL` في `sources.go:342`).
func (h *Harness) NewItemPriced(m *Merchant, cost, sale int64) *Item {
	h.T.Helper()
	ctx := context.Background()

	var sectionID string
	if err := h.Pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, active)
		VALUES ($1, true) RETURNING id::text`, uniq("قسم QA ")).Scan(&sectionID); err != nil {
		h.T.Fatalf("qa: تعذّر إنشاءُ قسم: %v", err)
	}

	var itemID string
	name := uniq("صنف QA ")
	if err := h.Pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, platform_section_id, name,
		                        price, merchant_price, available, approved)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, true, true) RETURNING id::text`,
		m.ID, sectionID, name, sale, cost).Scan(&itemID); err != nil {
		h.T.Fatalf("qa: تعذّر إنشاءُ صنف: %v", err)
	}

	h.T.Cleanup(func() {
		_, _ = h.Pool.Exec(ctx, `DELETE FROM menu_items WHERE id = $1::uuid`, itemID)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM platform_sections WHERE id = $1::uuid`, sectionID)
	})
	return &Item{ID: itemID, SectionID: sectionID, MerchantID: m.ID, Name: name, Cost: cost}
}

// HeldOf محتجَزُ صندوقِ سائق — **يُقرأ من القاعدة لا يُفترَض.**
func (f *Factory) HeldOf(driverID string) (int64, error) {
	var held int64
	err := f.h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = $1::uuid), 0)`,
		driverID).Scan(&held)
	if err != nil {
		return 0, fmt.Errorf("قراءةُ المحتجَز: %w", err)
	}
	return held, nil
}

// Treasury يضمن خزينةَ المنصّة — **وثوابتُ الحفظ كلُّها مشروطةٌ بها.**
//
// **ولماذا لم تكن؟** لأنّ `wallet.TreasuryID` تعيّنها لأوّلِ إداريٍّ **عند
// أوّلِ نداء** (wallet.go:322)، **والمِسنَدُ لا ينادِيها.** فكانت القاعدةُ
// الاختباريّةُ بلا خزينةٍ أبداً — **وكلُّ ثابتِ حفظٍ يُتخطّى بصمت.**
//
// **وهي واحدةٌ للقاعدة كلِّها** — فهرسٌ فريدٌ يشترط ذلك
// (`wallets_single_treasury_idx`)، **فلا تُنظَّف مع السيناريو**: صاحبُها
// حسابٌ ثابتُ الهاتف يُعاد استعمالُه.
func (f *Factory) Treasury() string {
	f.h.T.Helper()
	ctx := ctxBG()
	var id string
	if err := f.h.Pool.QueryRow(ctx,
		`SELECT user_id::text FROM wallets WHERE is_treasury LIMIT 1`).Scan(&id); err == nil {
		return id
	}
	const phone = "+963900000000"
	if _, err := f.h.Pool.Exec(ctx, `
		INSERT INTO users (phone, full_name, password_hash, status)
		VALUES ($1, 'خزينة QA', 'x', 'active') ON CONFLICT (phone) DO NOTHING`, phone); err != nil {
		f.h.T.Fatalf("qa: تعذّر إنشاءُ حاملِ الخزينة: %v", err)
	}
	if err := f.h.Pool.QueryRow(ctx,
		`SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&id); err != nil {
		f.h.T.Fatalf("qa: تعذّرت قراءةُ حاملِ الخزينة: %v", err)
	}
	if _, err := f.h.Pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, 'admin')
		ON CONFLICT DO NOTHING`, id); err != nil {
		f.h.T.Fatalf("qa: دورُ حاملِ الخزينة: %v", err)
	}
	if _, err := f.h.Pool.Exec(ctx, `
		INSERT INTO wallets (user_id, is_treasury) VALUES ($1::uuid, true)
		ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, id); err != nil {
		f.h.T.Fatalf("qa: تعذّر تعيينُ الخزينة: %v", err)
	}
	return id
}
