package identity

import (
	"context"
	"fmt"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// حذف الحساب — حق صاحبه، لكنه ليس زرّ محو.
//
// مبدآن يحكمانه:
//  1. **لا يُمحى تاريخ لا يخصّه وحده**: طلبٌ اشتراه من متجر، ونقدٌ بحوزة سائق،
//     وعمولة قُيّدت لمندوب — كلها قيود محاسبية لأطراف أخرى. فالحساب يُجرَّد من
//     هويته ويُقفَل، وتبقى القيود منسوبة إلى «حساب محذوف».
//  2. **لا يُحذف حساب عليه أو له مال**: الحذف ليس مهرباً من التزام ولا تنازلاً
//     عن حق. تُسوّى الأرصدة أولاً ثم يُحذف.

var (
	ErrWalletNotEmpty  = httpx.NewError(http.StatusConflict, "wallet_not_empty", "errors.wallet_not_empty")
	ErrCashNotSettled  = httpx.NewError(http.StatusConflict, "cash_not_settled", "errors.cash_not_settled")
	ErrOwnsMerchants   = httpx.NewError(http.StatusConflict, "owns_merchants", "errors.owns_merchants")
	ErrHasOpenOrders   = httpx.NewError(http.StatusConflict, "open_orders", "errors.open_orders")
	ErrPayoutPending   = httpx.NewError(http.StatusConflict, "payout_pending", "errors.payout_pending")
	ErrCannotDeleteOwn = httpx.NewError(http.StatusConflict, "admin_cannot_delete", "errors.admin_cannot_delete")
)

// RequestAccountDeletion يرسل رمز تأكيد إلى هاتف صاحب الحساب.
// نطلب رمزاً لا كلمة مرور: كثير من الزبائن دخلوا برمز ولا كلمة مرور لهم، والرمز
// أقوى تأكيداً على كل حال — يثبت أن الطالب يملك الرقم لا الجلسة فقط.
func (s *Service) RequestAccountDeletion(ctx context.Context, userID string) error {
	user, _, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.checkDeletable(ctx, user); err != nil {
		return err // لا نرسل رمزاً لطلب سيُرفض حتماً
	}
	return s.sendOTPFor(ctx, user.Phone, "delete", "otp:del:")
}

// checkDeletable يمنع الحذف عند وجود التزام أو حق قائم.
func (s *Service) checkDeletable(ctx context.Context, user *User) error {
	for _, r := range user.Roles {
		if r == "admin" {
			return ErrCannotDeleteOwn // مدير المنصة لا يحذف نفسه فتبقى بلا مدير
		}
	}
	db := s.repo.pool()

	var balance, cashHeld int64
	var pendingPayouts, activeMerchants, openOrders int
	err := db.QueryRow(ctx, `
		SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1), 0),
		       COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = $1), 0),
		       (SELECT count(*) FROM payout_requests WHERE user_id = $1 AND status = 'pending'),
		       (SELECT count(*) FROM merchants WHERE owner_user_id = $1 AND status = 'active'),
		       (SELECT count(*) FROM orders
		        WHERE (customer_id = $1 OR driver_id = $1) AND closed_at IS NULL)`,
		user.ID).Scan(&balance, &cashHeld, &pendingPayouts, &activeMerchants, &openOrders)
	if err != nil {
		return err
	}

	switch {
	case balance > 0:
		return ErrWalletNotEmpty // رصيده حقّه: يسحبه أو ينفقه قبل الحذف
	case cashHeld > 0:
		return ErrCashNotSettled // نقد المنصة بحوزته: يُسلّمه للمالية أولاً
	case pendingPayouts > 0:
		return ErrPayoutPending
	case activeMerchants > 0:
		return ErrOwnsMerchants // متجر فعّال بلا مالك يعني طلبات بلا مسؤول
	case openOrders > 0:
		return ErrHasOpenOrders // طلب جارٍ: لا يُترك زبونه ولا سائقه معلّقين
	}
	return nil
}

// ConfirmAccountDeletion يتحقق من الرمز ثم يجرّد الحساب ويقفله.
func (s *Service) ConfirmAccountDeletion(ctx context.Context, userID, code, ip string) error {
	user, _, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.checkDeletable(ctx, user); err != nil {
		return err // نعيد الفحص: قد يكون شيء تغيّر منذ طلب الرمز
	}
	valid, err := s.repo.ConsumeOTP(ctx, user.Phone, s.hashOTP(user.Phone, code), "delete")
	if err != nil {
		return err
	}
	if !valid {
		return ErrOTPInvalid
	}

	// نسجّل التدقيق **قبل** التجريد كي يبقى الرقم الأصلي في السجل
	s.repo.Audit(ctx, &user.ID, "user.self_delete", "user", user.ID, ip,
		map[string]any{"phone": user.Phone})

	if err := s.repo.AnonymizeUser(ctx, user.ID); err != nil {
		return err
	}
	// إقفال فوري: إبطال الجلسات ومسح كاش الحالة كي يُرفض أي توكن قائم
	if err := s.revokeAllSessions(ctx, user.ID); err != nil {
		return err
	}
	s.invalidateStatusCache(ctx, user.ID)
	return nil
}

// AnonymizeUser يجرّد الحساب من هويته ويقفله — الصفّ يبقى لتماسك القيود.
func (r *Repo) AnonymizeUser(ctx context.Context, userID string) error {
	// إنهاء نسبة المتاجر إليه كمندوب: لو بقيت، لظلّت عمولات الطلبات القادمة
	// تُقيَّد لمحفظة حساب محذوف — مال يتراكم لمن لا يستطيع الوصول إليه.
	// العمولات السابقة تبقى في الدفتر: تاريخ لا يُمحى.
	if _, err := r.db.Exec(ctx,
		`UPDATE merchants SET sales_rep_user_id = NULL WHERE sales_rep_user_id = $1`,
		userID); err != nil {
		return err
	}
	// الهاتف يُستبدل برمز مجهول: يتحرّر الرقم الأصلي فيستطيع صاحبه التسجيل
	// من جديد، ولا يبقى رقم شخصي في قاعدة بيانات حساب محذوف.
	//
	// الاسم يُستبدل بوسم لا يُترك فارغاً: طلباته القديمة تبقى معروضة للمتجر
	// والإدارة، واسمٌ فارغ فيها يبدو خللاً لا حذفاً. الوسم يقول ما جرى صراحة.
	_, err := r.db.Exec(ctx, fmt.Sprintf(`
		UPDATE users SET
			status          = 'deleted',
			deleted_at      = now(),
			full_name       = %s,
			phone           = 'deleted-' || id::text,
			password_hash   = NULL,
			avatar_media_id = NULL,
			invite_code     = NULL,
			admin_notes     = '',
			status_reason   = %s,
			updated_at      = now()
		WHERE id = $1`, "'حساب محذوف'", "'حُذف بطلب صاحبه'"), userID)
	return err
}
