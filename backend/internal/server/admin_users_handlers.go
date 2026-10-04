package server

import (
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.identity.AdminListUsersFiltered(r.Context(), identity.ListFilter{
		Query: q.Get("query"), Role: q.Get("role"), Online: q.Get("online") == "true",
		Status: q.Get("status"), PhoneSearch: hasCap(r, authz.UsersContactRead),
	}, page, perPage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	hideMoney(r, res)
	httpx.JSON(w, http.StatusOK, res)
}

// hideMoney يحجب الأرقامَ الماليّة عمّن لا يملكها — في المحرّك لا في الشاشة.
func hideMoney(r *http.Request, res *identity.UserPage) {
	if canSeeMoney(r) {
		return
	}
	res.MoneyHidden = true
	for i := range res.Users {
		res.Users[i].Balance, res.Users[i].OrdersSpent, res.Users[i].Commissions = 0, 0, 0
	}
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		identity.CreateUserInput
		// WelcomeApp **أيُّ رابطٍ في رسالة الدخول** — للموظّف يُمنَح دورُه بعد الإنشاء
		// فيصير الرابطُ بابَ اللوحة: `panel`. وفارغُه يُستنتج من صفة الحساب.
		WelcomeApp string `json:"welcome_app"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ولا كلمةَ يكتبها الموظّف** (قرارُ المالك ٢٠٢٦-١٠-٠٤) — النظامُ يولّدها ويرسلها.
	in := req.CreateUserInput
	in.Password = ""
	user, err := s.identity.AdminCreateUser(r.Context(), userIDFrom(r), in, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sent, expires, werr := s.issueWelcomeFor(r.Context(), userIDFrom(r), user.ID, clientIP(r), req.WelcomeApp)
	if werr != nil {
		s.logger.Error("تعذّر توليدُ كلمة الدخول للحساب الجديد", "user", user.ID, "error", werr)
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id": user.ID, "phone": user.Phone, "full_name": user.FullName, "roles": user.Roles,
		"welcome": map[string]any{"sent": sent, "expires_at": expires, "ok": werr == nil},
	})
}

// handleAdminUpdateUser تعديلُ حساب — والرقمُ مسارُه الخاصّ.
//
// **تغييرُ الرقم ينقل الحسابَ كلَّه بمحفظته** (قرارُ المالك ٢٠٢٦-١٠-٠٤): بكلمةِ سرِّ
// الموظّف (خطوةُ تحقّق)، ويُخرج كلَّ الجلسات، ويُبلَّغ الرقمُ القديم، ويُسجَّل القديمُ
// والجديد. **وفوق حدِّ رصيدٍ ينتظر موافقةَ شخصٍ ثانٍ.**
func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	req, err := decode[identity.UpdateUserInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id := chi.URLParam(r, "id")
	// **والملاحظاتُ سجلٌّ لا نصٌّ يُستبدَل** — لها بابُها (`/users/{id}/notes`).
	req.AdminNotes = nil
	newPhone := req.Phone
	req.Phone = nil
	var phoneResult map[string]any
	if newPhone != nil && strings.TrimSpace(*newPhone) != "" {
		phoneResult, err = s.changePhone(r, id, *newPhone)
		if err != nil {
			s.respondErr(w, err)
			return
		}
	}
	user, err := s.identity.AdminUpdateUser(r.Context(), userIDFrom(r), id, *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// صاحب الحساب يعرف بتغيّر حالته فوراً بدل أن يكتشفه عند أول رفض
	if req.Status != nil {
		title := notifTitles.accountActivated
		if *req.Status != "active" {
			title = notifTitles.accountSuspended
		}
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: user.ID, Kind: notifications.KindAccount,
			Title: title, Body: derefOr(req.StatusReason, ""),
			Entity: "user", EntityID: user.ID,
		})
		s.afterStatusChange(r, user.ID, *req.Status)
	}
	if phoneResult != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"user": user, "phone_change": phoneResult})
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (s *Server) handleAdminGrantRole(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Role   string `json:"role"`
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	target := chi.URLParam(r, "id")
	if err := s.identity.AdminGrantRole(r.Context(), userIDFrom(r), target, req.Role, req.Reason, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	// الصلاحية تغيّر ما يراه صاحب الحساب وما يشترك به من قنوات — يجب أن يعلم.
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: target, Kind: notifications.KindAccount,
		Title: notifTitles.roleGranted, Body: req.Reason,
		Entity: "user", EntityID: target, Href: "/",
	})
	s.touch("account", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"granted": true})
}

func (s *Server) handleAdminRevokeRole(w http.ResponseWriter, r *http.Request) {
	target := chi.URLParam(r, "id")
	reason := r.URL.Query().Get("reason")
	if err := s.identity.AdminRevokeRole(r.Context(), userIDFrom(r),
		target, chi.URLParam(r, "role"), reason, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: target, Kind: notifications.KindAccount,
		Title: notifTitles.roleRevoked, Body: reason,
		Entity: "user", EntityID: target, Href: "/",
	})
	s.touch("account", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"revoked": true})
}

// handleAdminUserRoleCounts أعداد الحسابات لكل دور — تغذي الكروت الذكية
// الفلترة أعلى شاشة الحسابات.
//
// ══════════════════════════════════════════════════════════════════════
//
//	**والبطاقةُ تعدّ ما تعرضه القائمةُ تحتها — وإلّا فهما رقمان**  `BOOK-06`
//
// ══════════════════════════════════════════════════════════════════════
//
// # ما قِيس (٢٠٢٦-٠٩-٣٠، شاشةُ الحسابات على التجهيز، رآها المالك)
//
//	بطاقةُ «موظّفو المنصّة»  تقول  **١**
//	وضغطُها يُخرج            **٥**   — مديرُ المنصّة والعمليّاتُ والماليّةُ
//	                                   وخدمةُ العملاء ومراقبُ المنصّة
//	وبطاقةُ «كلّ الحسابات»   تقول  **٥**   والجدولُ تحتها **ستّةُ صفوف**
//
// **ورقمان يتناقضان في شاشةٍ واحدةٍ أسوأُ من رقمٍ غائب**: من قرأ البطاقة
// ولم يضغطها استنتج أن لا موظّفَ في منصّته.
//
// # وثلاثةُ أسبابٍ في دالّةٍ واحدة — كلُّها من عائلةٍ أُصلحت مرّتين
//
//  1. **`role_code <> 'admin'`** في عدّ الأدوار — **وهو `BOOK-01` بعينه**:
//     إخفاءُ الأدمن من القراءة. وقد نُقض بقرار المالك ٢٠٢٦-٠٩-٣٠.
//  2. **`role_code IN ('ops','finance')`** للطاقم — **وهو `BOOK-05` بعينه**:
//     `ops` مُحالٌ إلى الإرث وصفرُ حاملين، و`operations` و`customer_support`
//     و`admin` و`platform_monitor` غائبون كلُّهم. **فالخمسةُ صاروا واحداً.**
//  3. **`WHERE NOT EXISTS (... 'admin')`** في المجموع — فالمجموعُ يطرح
//     حساباتِ الأدمن، **والقائمةُ تعرضها**. (قِيس: ٥ مقابل ٦.)
//
// # والإصلاحُ أن يُقرأ المُسنَدُ من موضعٍ واحد
//
// **`identity.ListUsers` هي التي تُخرج الصفوف** — فهذه الدالّةُ تعدّ بمُسنَدها
// نفسِه لا بمُسنَدٍ مشابه: **مجموعٌ بلا استثناء**، **وطاقمٌ بالنفي**
// (`role_code <> ALL(accountTypes)`)، **وعدُّ أدوارٍ بلا حجب.**
//
// **ولا تشيخ بدورٍ جديد**: من أضاف دورَ عملٍ غداً ظهر في البطاقة وفي القائمة
// معاً **بلا تعديلِ استعلامٍ في موضعين.**
func (s *Server) handleAdminUserRoleCounts(w http.ResponseWriter, r *http.Request) {
	// **وصفةُ الحساب ليست وظيفة** — وما عداها عملٌ. (`authz.ClassAccountType`)
	accountTypes := authz.RolesInClass(authz.ClassAccountType)

	rows, err := s.pg.Query(r.Context(),
		`SELECT role_code, count(*) FROM user_roles GROUP BY role_code`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var role string
		var n int
		if err := rows.Scan(&role, &n); err != nil {
			s.respondErr(w, err)
			return
		}
		counts[role] = n
	}
	var total, staff, online int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*),
		       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM user_roles sr
		           WHERE sr.user_id = u.id AND sr.role_code <> ALL($1))),
		       count(*) FILTER (WHERE u.last_seen_at > now() - interval '2 minutes')
		FROM users u`, accountTypes).
		Scan(&total, &staff, &online); err != nil {
		s.respondErr(w, err)
		return
	}
	counts["staff"] = staff
	counts["online"] = online
	// **و«الزبون» زبونٌ فقط** (قرارُ المالك ٢٠٢٦-١٠-٠٤) — بمُسنَد القائمة نفسِه (`role=customer`).
	// **و«موقوفٌ أو محظور» بطاقةٌ لها** — والقائمةُ تُرشَّح بـ`status=restricted`.
	var customers, restricted int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*) FILTER (WHERE EXISTS (SELECT 1 FROM user_roles c
		                                       WHERE c.user_id = u.id AND c.role_code = 'customer')
		                          AND NOT EXISTS (SELECT 1 FROM user_roles n
		                                       WHERE n.user_id = u.id AND n.role_code <> 'customer')),
		       count(*) FILTER (WHERE u.status IN ('suspended', 'blocked'))
		FROM users u`).Scan(&customers, &restricted); err != nil {
		s.respondErr(w, err)
		return
	}
	counts["customer"] = customers
	counts["restricted"] = restricted
	httpx.JSON(w, http.StatusOK, map[string]any{"total": total, "roles": counts})
}

// handleAdminGetUser صفحة تفاصيل الحساب: الملف + المحفظة + مؤشرات حسب أدواره.
func (s *Server) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	var out struct {
		ID           string   `json:"id"`
		Phone        string   `json:"phone"`
		FullName     string   `json:"full_name"`
		Status       string   `json:"status"`
		InviteCode   *string  `json:"invite_code"`
		AvatarThumb  *string  `json:"avatar_thumb_url"`
		StatusReason string   `json:"status_reason"`
		AdminNotes   string   `json:"admin_notes"`
		Sessions     int      `json:"active_sessions"`
		Roles        []string `json:"roles"`
		CreatedAt    string   `json:"created_at"`
		// LastSeenAt **آخرُ ظهور** — (قرارُ المالك ٢٠٢٦-٠٨-١٥: حُذف
		// من بطاقة الحسابات، **فموضعُه ملفُّه**).
		LastSeenAt *string `json:"last_seen_at"`
		// Referrals **كم دعا وكم قبض** — (قرارُ المالك ٢٠٢٦-٠٨-١٥.)
		//
		// **والمقبولةُ من رُوفئ عنه فعلاً** (`rewarded_at`) — لا من
		// سجّل برمزه: **المكافأةُ تُقيَّد حين يُسلَّم أوّلُ طلبٍ
		// للمدعوّ**، ومن عدّ المسجّلين وعد صاحبَه بمالٍ لم يستحقّه.
		ReferralsCount  int      `json:"referrals_count"`
		ReferralsEarned int64    `json:"referrals_earned"`
		Balance         int64    `json:"balance"`
		OrdersCount     int      `json:"orders_count"` // كزبون
		OrdersSpent     int64    `json:"orders_spent"` // إنفاقه المُسلَّم
		Merchants       []string `json:"merchants"`    // متاجر يملكها
		RepStores       int      `json:"rep_stores"`   // متاجر جلبها كمندوب
		Commissions     int64    `json:"commissions"`  // عمولاته كمندوب
		// OnShift **أعلى الدوام الآن؟** — (قرارُ المالك ٢٠٢٦-٠٨-١٥:
		// حُذفت من بطاقة الحسابات، **فموضعُها ملفُّه**).
		OnShift    bool  `json:"on_shift"`
		DriverCash int64 `json:"driver_cash"` // نقد بحوزته كسائق
		Deliveries int   `json:"deliveries"`  // توصيلاته المُسلَّمة
		// DeliveredToday **ما سلّمه اليوم** — (قرارُ المالك ٢٠٢٦-٠٨-١٥:
		// حُذف من البطاقة). **وهو غيرُ المجموع**: ذاك يقول «كم عمل في
		// عمره» وهذا يقول «أيعمل اليوم؟».
		DeliveredToday int `json:"delivered_today"`

		// ══════════════════════════════════════════════════════════════
		// **قسمُ الحسابات** (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
		// ══════════════════════════════════════════════════════════════
		//
		// MoneyHidden **الأرصدةُ والإنفاقُ والعمولاتُ تُحجب من المحرّك** عمّن لا يملك
		// المالَ ولا خدمةَ العملاء — **لا تُخفى في الشاشة وتصل في الردّ.**
		MoneyHidden bool `json:"money_hidden"`
		// TempPassword **كلمةٌ مؤقّتةٌ لم تُبدَّل** — «لم يدخل بعد · تنتهي بعد …».
		TempPending   bool    `json:"temp_password_pending"`
		TempExpiresAt *string `json:"temp_password_expires_at"`
		WelcomeSentAt *string `json:"welcome_sent_at"`
		EverLoggedIn  bool    `json:"ever_logged_in"`
		// **السائق**: المركبةُ · سقفُ النقد العامُّ والخاصّ · قفلُ ما بعد الحادث.
		VehicleType      string  `json:"vehicle_type"`
		VehiclePlate     string  `json:"vehicle_plate"`
		VehicleColor     string  `json:"vehicle_color"`
		CashLimit        int64   `json:"cash_limit"`
		CashLimitGeneral int64   `json:"cash_limit_general"`
		CashLimitCustom  *int64  `json:"cash_limit_override"`
		AccidentLocked   bool    `json:"accident_locked"`
		AccidentLockAt   *string `json:"accident_lock_at"`
		AccidentCleared  *string `json:"accident_cleared_at"`
		AccidentClearer  *string `json:"accident_cleared_by"`
		// **المندوب**: عمولاتٌ محجوزةٌ ما دام موقوفاً.
		HeldCommission  int64 `json:"held_commission"`
		HeldCommissions int   `json:"held_commissions_count"`
		// **تغييرُ رقمٍ ينتظر شخصاً ثانياً.**
		PendingPhone *phoneRequestRow `json:"pending_phone_change"`
		// **متاجرُه بمعرّفاتها** — للربط بملفّ المتجر.
		Stores []map[string]any `json:"stores"`
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT u.id, u.phone, u.full_name, u.status, u.invite_code, u.status_reason, u.admin_notes,
		       (SELECT count(*) FROM refresh_tokens rt WHERE rt.user_id = u.id AND rt.revoked_at IS NULL AND rt.expires_at > now()),
		       (SELECT m.thumb_path FROM media m WHERE m.id = u.avatar_media_id), u.created_at::text,
		       u.last_seen_at::text,
		       COALESCE((SELECT array_agg(role_code ORDER BY role_code) FROM user_roles WHERE user_id = u.id), '{}'),
		       COALESCE((SELECT count(*) FROM referrals rf
		                 WHERE rf.inviter_id = u.id AND rf.rewarded_at IS NOT NULL), 0),
		       COALESCE((SELECT sum(rf.reward_amount) FROM referrals rf
		                 WHERE rf.inviter_id = u.id AND rf.rewarded_at IS NOT NULL), 0),
		       COALESCE((SELECT balance FROM wallets WHERE user_id = u.id), 0),
		       (SELECT count(*) FROM orders o WHERE o.customer_id = u.id),
		       COALESCE((SELECT sum(o.total) FROM orders o WHERE o.customer_id = u.id AND o.status = 'delivered'), 0),
		       COALESCE((SELECT array_agg(m.name ORDER BY m.created_at) FROM merchants m WHERE m.owner_user_id = u.id), '{}'),
		       (SELECT count(*) FROM merchants m WHERE m.sales_rep_user_id = u.id),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t WHERE t.user_id = u.id AND t.kind = 'commission'), 0),
		       u.on_shift,
		       COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = u.id), 0),
		       (SELECT count(*) FROM orders o WHERE o.driver_id = u.id AND o.status = 'delivered'),
		       (SELECT count(*) FROM orders o WHERE o.driver_id = u.id AND o.status = 'delivered'
		         AND o.delivered_at >= date_trunc('day', now()))
		FROM users u WHERE u.id = $1`, id).
		Scan(&out.ID, &out.Phone, &out.FullName, &out.Status, &out.InviteCode, &out.StatusReason, &out.AdminNotes, &out.Sessions, &out.AvatarThumb, &out.CreatedAt, &out.LastSeenAt,
			&out.Roles, &out.ReferralsCount, &out.ReferralsEarned, &out.Balance, &out.OrdersCount, &out.OrdersSpent, &out.Merchants,
			&out.RepStores, &out.Commissions, &out.OnShift, &out.DriverCash, &out.Deliveries, &out.DeliveredToday)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	out.AvatarThumb = media.SignedURLPtr(out.AvatarThumb)
	ctx := r.Context()
	out.CashLimitGeneral = s.cashbox.Limit(ctx)
	_ = s.pg.QueryRow(ctx, `
		SELECT u.temp_password_expires_at::text, u.welcome_sent_at::text,
		       EXISTS (SELECT 1 FROM refresh_tokens rt WHERE rt.user_id = u.id),
		       u.vehicle_type, u.vehicle_plate, u.vehicle_color, u.cash_limit_override,
		       u.accident_lock_at IS NOT NULL
		         AND (u.accident_cleared_at IS NULL OR u.accident_cleared_at < u.accident_lock_at),
		       u.accident_lock_at::text, u.accident_cleared_at::text,
		       NULLIF(COALESCE(NULLIF(c.full_name, ''), c.phone, ''), '')
		FROM users u LEFT JOIN users c ON c.id = u.accident_cleared_by
		WHERE u.id = $1`, id).
		Scan(&out.TempExpiresAt, &out.WelcomeSentAt, &out.EverLoggedIn,
			&out.VehicleType, &out.VehiclePlate, &out.VehicleColor, &out.CashLimitCustom,
			&out.AccidentLocked, &out.AccidentLockAt, &out.AccidentCleared, &out.AccidentClearer)
	pending, _ := s.identity.TempPasswordPending(ctx, id)
	out.TempPending = pending
	if !pending {
		out.TempExpiresAt = nil
	}
	out.CashLimit = out.CashLimitGeneral
	if out.CashLimitCustom != nil {
		out.CashLimit = *out.CashLimitCustom
	}
	out.HeldCommission, out.HeldCommissions = s.orders.HeldCommissionTotal(ctx, id)
	var pr phoneRequestRow
	if err := s.pg.QueryRow(ctx, `
		SELECT pr.id::text, pr.user_id::text, pr.old_phone, pr.new_phone, pr.balance, pr.status,
		       pr.proposed_by::text, COALESCE(NULLIF(p.full_name, ''), p.phone, ''), pr.created_at
		FROM phone_change_requests pr JOIN users p ON p.id = pr.proposed_by
		WHERE pr.user_id = $1 AND pr.status = 'pending'`, id).
		Scan(&pr.ID, &pr.UserID, &pr.OldPhone, &pr.NewPhone, &pr.Balance, &pr.Status,
			&pr.ProposedBy, &pr.Proposer, &pr.CreatedAt); err == nil {
		out.PendingPhone = &pr
	}
	out.Stores = []map[string]any{}
	if rows, err := s.pg.Query(ctx, `SELECT id::text, name, status FROM merchants
		WHERE owner_user_id = $1 ORDER BY created_at`, id); err == nil {
		for rows.Next() {
			var mid, name, st string
			if rows.Scan(&mid, &name, &st) == nil {
				out.Stores = append(out.Stores, map[string]any{"id": mid, "name": name, "status": st})
			}
		}
		rows.Close()
	}
	if !canSeeMoney(r) {
		out.MoneyHidden = true
		out.Balance, out.OrdersSpent, out.Commissions, out.ReferralsEarned = 0, 0, 0, 0
		out.HeldCommission = 0
	}
	httpx.JSON(w, http.StatusOK, out)
}

// canSeeMoney **من يرى أرصدةَ الناس وإنفاقَهم وعمولاتِهم** (قرارُ المالك ٢٠٢٦-١٠-٠٤):
// مديرُ المنصّة والماليّة (`finance.read`) وخدمةُ العملاء (`support.manage`) — تتابع
// «رصيدي ناقص». **ومن سواهم يُحجب عنه في المحرّك.**
func canSeeMoney(r *http.Request) bool {
	return hasCap(r, authz.FinanceRead) || hasCap(r, authz.SupportManage)
}

// handleAdminResetPassword **إعادةُ كلمة المرور — يولّدها النظام ويرسلها** (قرارُ المالك
// ٢٠٢٦-١٠-٠٤). **ولا يكتبها الموظّف**: تُقطَع الجلساتُ، وتنتهي بعد ٧٢ ساعة، وتُبدَّل عند
// أوّل دخول. والجسمُ يُتجاهَل — **كلمةٌ يرسلها عميلٌ قديمٌ لا تُقبَل.**
func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **ولا يعيد الموظّفُ كلمتَه هو من ملفّه** — فيُخرج نفسَه. بابُه «حسابي».
	if id == userIDFrom(r) {
		s.respondErr(w, identity.ErrSelfAction)
		return
	}
	sent, expires, err := s.issueWelcome(r.Context(), userIDFrom(r), id, clientIP(r), true)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true, "sent": sent, "expires_at": expires})
}

// handleAdminUserActivity سجل نشاط الحساب: ما فعله وما فُعل به (من سجل التدقيق).
func (s *Server) handleAdminUserActivity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// **وسجلُّ النشاط يُرقَّم** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	//
	// **وهو أسرعُ ما ينمو في الحساب**: كلُّ دخولٍ وكلُّ تعديلٍ سطر. **ومئةٌ
	// صامتةٌ تعني أنّ ما قبل الأسبوع الماضي محجوبٌ عمّن يراجع** — وهو ما
	// يُبحث عنه بالضبط حين يُشتكى على حساب.
	pg := pagingOf(r, 25)
	// ══════════════════════════════════════════════════════════════════
	// **وتجديدُ الجلسة يُخفى — تكتبه الساعةُ لا الإنسان**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٥ بعد قياسٍ على الخادم: **ثلاثون من ثمانيةٍ
	//  وخمسين سطراً `auth.refresh`** — أكثرُ من نصف السجلّ.)
	//
	// **وهاتفٌ تُرك مفتوحاً يكتب سطراً كلَّ دورة تجديدٍ إلى الأبد** —
	// **فتُدفَع الأفعالُ الحقيقيّةُ خارجَ الصفحة الأولى** بعد يومٍ من
	// الاستعمال، **وسجلٌّ يُفتح لتجد فيه شيئاً واحداً يملؤه المؤقّت.**
	//
	// **ولا يُحذف من القاعدة** — هو أثرُ أمانٍ بعنوانٍ ووقت: **يُخفى
	// ويُطلب.** وشاشةُ «آخر دخولاتك» عند المستخدم تستثنيه هكذا من قبل،
	// **والسابقةُ كانت موجودةً واللوحةُ لا تتبعها.**
	q := r.URL.Query()
	action := q.Get("action")
	withRefresh := q.Get("refresh") == "true"

	// **والشرطُ واحدٌ للعدّ وللقائمة ولجرد الأفعال** — ثلاثةُ نصوصٍ تفترق
	// يوماً **فيقول العدّادُ رقماً وتعرض القائمةُ غيرَه.**
	const scope = `
		FROM audit_log a
		CROSS JOIN (SELECT id FROM users WHERE id = $1) u
		LEFT JOIN users au ON au.id = a.actor_user_id
		WHERE (a.actor_user_id = u.id OR (a.entity = 'user' AND a.entity_id = u.id::text))
		  AND ($2 = '' OR a.action = $2)
		  AND ($3 OR a.action <> 'auth.refresh')`

	// **وجردُ أفعاله قبل الترشيح** — **وقائمةٌ تُبنى من المعروض تخسر
	// خياراتِها كلَّما رُشِّحت**، فلا يُرجَع منها إلى ما قبلها.
	//
	// **وما ليس عنده لا يُعرض عليه**: قائمةٌ بثمانين فعلاً لحسابٍ فيه
	// ثلاثة **تُبحث ولا تُقرأ.**
	kinds := []map[string]any{}
	if kRows, err := s.pg.Query(r.Context(), `
		SELECT a.action, count(*)`+scope+`
		GROUP BY a.action ORDER BY count(*) DESC`, id, "", true); err == nil {
		for kRows.Next() {
			var act string
			var n int
			if kRows.Scan(&act, &n) == nil {
				kinds = append(kinds, map[string]any{"action": act, "count": n})
			}
		}
		kRows.Close()
	}

	var count int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*)`+scope, id, action, withRefresh).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT a.action, a.entity, COALESCE(a.entity_id, ''), COALESCE(a.ip, ''),
		       COALESCE(a.details::text, ''), a.created_at,
		       NULLIF(COALESCE(au.full_name, au.phone::text), ''),
		       (a.actor_user_id IS NOT DISTINCT FROM u.id) AS by_self`+scope+`
		ORDER BY a.id DESC LIMIT $4 OFFSET $5`,
		id, action, withRefresh, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type entry struct {
		Action    string    `json:"action"`
		Entity    string    `json:"entity"`
		EntityID  string    `json:"entity_id"`
		IP        string    `json:"ip"`
		Details   string    `json:"details"`
		ByName    *string   `json:"by_name"`
		BySelf    bool      `json:"by_self"`
		CreatedAt time.Time `json:"created_at"`
	}
	out := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.Action, &e.Entity, &e.EntityID, &e.IP, &e.Details,
			&e.CreatedAt, &e.ByName, &e.BySelf); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, e)
	}
	res := paged("activity", out, count, pg)
	// **وأفعالُه المتاحةُ معه** — تُبنى منها قائمةُ الترشيح في الشاشة.
	res["kinds"] = kinds
	httpx.JSON(w, http.StatusOK, res)
}

// handleAdminLogoutAll إنهاء كل جلسات الحساب فوراً.
func (s *Server) handleAdminLogoutAll(w http.ResponseWriter, r *http.Request) {
	// **ولا يُخرج الموظّفُ نفسَه من ملفّه** — بابُه «حسابي».
	if chi.URLParam(r, "id") == userIDFrom(r) {
		s.respondErr(w, identity.ErrSelfAction)
		return
	}
	n, err := s.identity.AdminLogoutAll(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"revoked_sessions": n})
}

// handleAdminUserFeedback الشكاوى والتقييمات المرتبطة بالحساب:
// تذاكره كزبون، تقييماته الممنوحة، والواردة عليه (كسائق أو صاحب متاجر).
func (s *Server) handleAdminUserFeedback(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// **وثلاثُ قوائمَ في ردٍّ واحدٍ لا تُرقَّم برقمٍ واحد** — (٢٠٢٦-٠٨-١٠).
	//
	// **فلكلٍّ صفحتُها**: `t_page` للتذاكر، و`g_page` للممنوحة، و`r_page`
	// للواردة. **ورقمٌ واحدٌ لثلاثتها يقلّب ما لم يُطلب** — يبحث في تذاكره
	// فتقفز تقييماتُه معها.
	q := r.URL.Query()
	tp := pagingFrom(q.Get("t_page"), 10)
	ap := pagingFrom(q.Get("a_page"), 10)
	gp := pagingFrom(q.Get("g_page"), 10)
	rp := pagingFrom(q.Get("r_page"), 10)

	out := struct {
		Tickets []map[string]any `json:"tickets"`
		// Against **الشكاوى عليه** — (قرارُ المالك ٢٠٢٦-٠٨-١٥).
		Against []map[string]any `json:"tickets_against"`
		Given   []map[string]any `json:"ratings_given"`
		Recv    []map[string]any `json:"ratings_received"`
		AvgRecv *float64         `json:"avg_received"`
		// ══════════════════════════════════════════════════════════════
		// **وتقييمُ السائقين لمتجره — كان يُكتب ولا يُقرأ**
		// ══════════════════════════════════════════════════════════════
		//
		// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
		//
		// **جدولُ `merchant_ratings` يُكتب فيه بعد كلّ تسليم** — السائقُ
		// يقيّم المتجرَ بنجمتين: سرعةُ التجهيز وحُسنُ التعامل.
		//
		// **ولا شاشةَ في المنصّة كلِّها تقرؤه** — لا إدارةٌ ولا بوّابةُ
		// متجرٍ ولا تقرير. **جدولٌ يُكتب فيه ولا يُقرأ منه أبداً.**
		//
		// **والسائقُ يُسأل بعد كلّ تسليم** — فيُنفَق وقتُه على رأيٍ لا
		// يبلغ أحداً. **وهذا أسوأُ من غياب الميزة**: غيابُها يُعرف،
		// **وهذه تبدو موجودةً وهي معطّلة.**
		ByDrivers      []map[string]any `json:"by_drivers"`
		ByDriversCount int              `json:"by_drivers_count"`
		AvgSpeed       *float64         `json:"avg_speed"`
		AvgConduct     *float64         `json:"avg_conduct"`
		// **وعددُ كلٍّ منها** — والمعروضُ صفحةٌ منه.
		TicketsCount int `json:"tickets_count"`
		AgainstCount int `json:"tickets_against_count"`
		GivenCount   int `json:"ratings_given_count"`
		RecvCount    int `json:"ratings_received_count"`
		PerPage      int `json:"per_page"`
	}{Tickets: []map[string]any{}, Against: []map[string]any{}, Given: []map[string]any{}, Recv: []map[string]any{}, ByDrivers: []map[string]any{}, PerPage: tp.PerPage}

	// ══════════════════════════════════════════════════════════════════
	// **وشكاواه وشكاوى عليه قائمتان لا قائمة**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٥.)
	//
	// **كان الشرطُ `customer_id` وحدَه** — وهو يقول «تذكرةُ أيّ طلبٍ هذه»
	// **لا «من كتبها»**. فبلاغُ السائق على الزبون يقع في القائمة نفسِها
	// **جنبَ شكاوى الزبون** — ولا يفرّقهما إلّا نصُّ العنوان.
	//
	// **والحكمان متناقضان**: «عليه ثلاثُ شكاوى» و«اشتكى ثلاثاً». **ومن
	// يقرأ هذه الشاشةَ هو من يقرّر الإنذارَ أو الحظر.**
	//
	// **وكان المحرّكُ يحفظ الفرقَ في ثلاثة أعمدة ولا يُقرأ منها واحد.**
	//
	// **وبلاغُ السائق يظهر في ملفّه أيضاً** (`created_by`) — وكان يسكن
	// ملفَّ زبونٍ آخر، **فمن فتح ملفَّ سائقٍ رفع عشرةَ بلاغاتٍ قرأ «لا
	// شكاوى».**
	const ticketCols = `
		SELECT t.id::text, t.number, t.subject, t.status, t.compensation,
		       COALESCE(t.reason, ''),
		       -- **ومن فتحها يُقال** — والعنوانُ وحدَه لا يقوله.
		       COALESCE(NULLIF(cb.full_name, ''), cb.phone::text, ''),
		       t.created_at
		FROM tickets t
		LEFT JOIN users cb ON cb.id = t.created_by`
	// **وما يخصّ طلباته** — فتحها هو أو المكتبُ عنه أو سائقٌ على متجرٍ فيها.
	// **وما كان عليه يُستثنى** فلا يُعدّ مرّتين.
	const mineCond = `
		WHERE (t.created_by = $1 OR t.customer_id = $1)
		  AND t.against_user_id IS DISTINCT FROM $1`
	const againstCond = ` WHERE t.against_user_id = $1`

	readTickets := func(cond string, pg Paging, into *[]map[string]any, count *int) error {
		if err := s.pg.QueryRow(r.Context(),
			`SELECT count(*) FROM tickets t`+cond, id).Scan(count); err != nil {
			return err
		}
		rows, err := s.pg.Query(r.Context(), ticketCols+cond+`
			ORDER BY t.number DESC LIMIT $2 OFFSET $3`, id, pg.PerPage, pg.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var tid, subject, status, reason, byName string
			var num, comp int64
			var at time.Time
			if err := rows.Scan(&tid, &num, &subject, &status, &comp,
				&reason, &byName, &at); err == nil {
				*into = append(*into, map[string]any{
					"id": tid, "number": num, "subject": subject, "status": status,
					"compensation": comp, "reason": reason, "by_name": byName,
					"created_at": at})
			}
		}
		return rows.Err()
	}
	if err := readTickets(mineCond, tp, &out.Tickets, &out.TicketsCount); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := readTickets(againstCond, ap, &out.Against, &out.AgainstCount); err != nil {
		s.respondErr(w, err)
		return
	}
	var rows pgx.Rows
	var err error

	// ══════════════════════════════════════════════════════════════════
	// **والمتجرُ يُضمّ يساراً — ثلاثَ مرّاتٍ في هذا المعالِج**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شهده المالك ٢٠٢٦-٠٨-١٠: «في صفحة المستخدمين التقييماتُ والشكاوى لا
	//  تظهر».)
	//
	// **الطلبُ الخاصُّ لا متجرَ له** — و`JOIN merchants` صلبٌ **يُسقط الصفَّ
	// كلَّه بلا خطأ ولا سطرٍ في سجلّ.** فأربعةُ تقييماتٍ في القاعدة تُقرأ
	// صفراً في الشاشة، **وتُقرأ «لم يقيّمه أحد» لا «الاستعلامُ يكذب».**
	//
	// **وهي العائلةُ نفسُها التي أمسكها المشيُ الحيُّ خمسَ مرّاتٍ من قبل**:
	// `GetByID` ومهامُّ السائق والتقييمُ والإشعاراتُ وبطاقةُ الإدارة.
	// **وكلُّ مرّةٍ تُصلَح واحدةً ويبقى الباقي** — لأنّها تُكتب في كلّ
	// استعلامٍ بيده.
	//
	// **واسمُ المتجر فارغٌ فيه** — والشاشةُ تعرض نصَّ الطلب مكانَه.
	_ = s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM order_ratings WHERE customer_id = $1`, id).Scan(&out.GivenCount)
	rows, err = s.pg.Query(r.Context(), `
		SELECT o.number, COALESCE(m.name, ''), rt.platform_stars, rt.driver_stars,
		       rt.created_at
		FROM order_ratings rt
		JOIN orders o ON o.id = rt.order_id
		LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE rt.customer_id = $1
		ORDER BY rt.created_at DESC LIMIT $2 OFFSET $3`, id, gp.PerPage, gp.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var num int64
		var mName string
		var ms int
		var ds *int
		var at time.Time
		if err := rows.Scan(&num, &mName, &ms, &ds, &at); err == nil {
			out.Given = append(out.Given, map[string]any{
				"order_number": num, "merchant_name": mName,
				"platform_stars": ms, "driver_stars": ds, "created_at": at})
		}
	}
	rows.Close()

	// الواردة: كسائق (نجوم السائق على طلباته) + كصاحب متاجر (نجوم متاجره)
	rows, err = s.pg.Query(r.Context(), `
		SELECT o.number, COALESCE(m.name, ''), rt.driver_stars, rt.created_at, 'driver'
		FROM order_ratings rt
		JOIN orders o ON o.id = rt.order_id
		LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE o.driver_id = $1 AND rt.driver_stars IS NOT NULL
		UNION ALL
		-- **وهذا الضمُّ صلبٌ بحقّ** — شرطُه `+"`m.owner_user_id`"+` نفسُه،
		-- **فلا صفَّ بلا متجرٍ يُطلب هنا أصلاً.**
		SELECT o.number, m.name, rt.platform_stars, rt.created_at, 'platform'
		FROM order_ratings rt
		JOIN orders o ON o.id = rt.order_id
		JOIN merchants m ON m.id = o.merchant_id
		WHERE m.owner_user_id = $1
		ORDER BY 4 DESC LIMIT $2 OFFSET $3`, id, rp.PerPage, rp.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **والمتوسّطُ على كلّ التقييمات لا على الصفحة المعروضة**
	// ══════════════════════════════════════════════════════════════════
	//
	// **كان يُجمَع من الأسطر المعروضة** — وكان صحيحاً حين تُعرض كلُّها.
	// **ومع الترقيم يصير متوسّطَ عشرةٍ لا متوسّطَ سائق**: يُقلَّب إلى الصفحة
	// الثانية **فيتبدّل تقييمُه أمام عين من يقرأ.**
	//
	// **وهي عائلةُ العطب نفسِها التي أمسكتها النزاعات**: رقمٌ يُشتقّ من صفحةٍ
	// وهو عن الكلّ.
	const recvAgg = `
		SELECT count(*), COALESCE(avg(s), 0) FROM (
			SELECT rt.driver_stars AS s
			FROM order_ratings rt JOIN orders o ON o.id = rt.order_id
			WHERE o.driver_id = $1 AND rt.driver_stars IS NOT NULL
			UNION ALL
			SELECT rt.platform_stars
			FROM order_ratings rt
			JOIN orders o ON o.id = rt.order_id
			JOIN merchants m ON m.id = o.merchant_id
			WHERE m.owner_user_id = $1
		) x`
	var recvAvg, avgSpeed, avgConduct float64
	if err := s.pg.QueryRow(r.Context(), recvAgg, id).Scan(&out.RecvCount, &recvAvg); err == nil &&
		out.RecvCount > 0 {
		out.AvgRecv = &recvAvg
	}

	for rows.Next() {
		var num int64
		var mName, as string
		var stars int
		var at time.Time
		if err := rows.Scan(&num, &mName, &stars, &at, &as); err == nil {
			out.Recv = append(out.Recv, map[string]any{
				"order_number": num, "merchant_name": mName, "stars": stars,
				"created_at": at, "as": as})
		}
	}
	rows.Close()

	// ══════════════════════════════════════════════════════════════════
	// **وما قاله السائقون عن متجره — يُقرأ لأوّل مرّة**
	// ══════════════════════════════════════════════════════════════════
	//
	// **والمتوسّطان على الكلّ لا على المعروض** — **ومتوسّطُ عشرةٍ ليس
	// متوسّطَ متجر**، ويتبدّل بتقليب الصفحة أمام عين من يقرأ.
	_ = s.pg.QueryRow(r.Context(), `
		SELECT count(*), COALESCE(avg(mr.speed_stars), 0), COALESCE(avg(mr.conduct_stars), 0)
		FROM merchant_ratings mr
		JOIN merchants m ON m.id = mr.merchant_id
		WHERE m.owner_user_id = $1`, id).
		Scan(&out.ByDriversCount, &avgSpeed, &avgConduct)
	if out.ByDriversCount > 0 {
		out.AvgSpeed, out.AvgConduct = &avgSpeed, &avgConduct
	}
	// **والسطورُ بأسماء قائليها** — **وتقييمٌ بلا قائلٍ لا يُراجَع ولا
	// يُحتجّ به**، ولا يُعرف أسائقٌ واحدٌ كرّرها أم عشرة.
	dRows, err := s.pg.Query(r.Context(), `
		SELECT o.number, m.name,
		       mr.speed_stars, mr.conduct_stars, mr.created_at,
		       COALESCE(NULLIF(dr.full_name, ''), dr.phone::text, '')
		FROM merchant_ratings mr
		JOIN merchants m ON m.id = mr.merchant_id
		JOIN orders o ON o.id = mr.order_id
		LEFT JOIN users dr ON dr.id = mr.driver_id
		WHERE m.owner_user_id = $1
		ORDER BY mr.created_at DESC LIMIT $2 OFFSET $3`, id, rp.PerPage, rp.Offset)
	if err == nil {
		for dRows.Next() {
			var num int64
			var mName, driver string
			var speed, conduct int
			var at time.Time
			if dRows.Scan(&num, &mName, &speed, &conduct, &at, &driver) == nil {
				out.ByDrivers = append(out.ByDrivers, map[string]any{
					"order_number": num, "merchant_name": mName,
					"speed_stars": speed, "conduct_stars": conduct,
					"created_at": at, "driver": driver})
			}
		}
		dRows.Close()
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleAdminUsersExport **تصديرُ الحسابات المرشَّحة** — لمديرِ المنصّة وحدَه (`users.export`).
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤): **عربيٌّ كلُّه** — العناوينُ والأدوارُ والحال، **ومحميٌّ من
// صيغ إكسل** (اسمٌ يبدأ بـ`=` كان يُنفَّذ حين يُفتح الملفّ)، **ولا قصَّ صامتاً**: يُقرأ
// كلُّ ما طابق الترشيحَ صفحةً بعد صفحة.
func (s *Server) handleAdminUsersExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := identity.ListFilter{Query: q.Get("query"), Role: q.Get("role"),
		Online: q.Get("online") == "true", Status: q.Get("status"),
		PhoneSearch: hasCap(r, authz.UsersContactRead)}
	var all []identity.User
	for page := 1; ; page++ {
		res, err := s.identity.AdminListUsersFiltered(r.Context(), filter, page, 100)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		hideMoney(r, res)
		all = append(all, res.Users...)
		if len(res.Users) < res.PerPage || len(all) >= res.Total {
			break
		}
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="accounts.csv"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM لعرض العربية في Excel
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"الاسم", "الهاتف", "الأدوار", "المتاجر", "الحالة", "كود الدعوة",
		"آخر ظهور", "تاريخ التسجيل"})
	for _, u := range all {
		lastSeen := ""
		if u.LastSeenAt != nil {
			lastSeen = u.LastSeenAt.Format("2006-01-02 15:04")
		}
		invite := ""
		if u.InviteCode != nil {
			invite = *u.InviteCode
		}
		roles := make([]string, 0, len(u.Roles))
		for _, rc := range u.Roles {
			roles = append(roles, roleLabelAr(rc))
		}
		_ = cw.Write([]string{
			csvSafe(u.FullName), csvSafe(u.Phone), strings.Join(roles, " + "),
			csvSafe(strings.Join(u.StoreNames, " · ")), statusLabelAr(u.Status), csvSafe(invite),
			lastSeen, u.CreatedAt.Format("2006-01-02"),
		})
	}
	cw.Flush()
}

// csvSafe **يُبطل صيغَ إكسل** — خليّةٌ تبدأ بـ`= + - @` أو جدولةٍ أو رجوعٍ تُسبَق بفاصلةٍ عليا
// فتُقرأ نصّاً. **والرقمُ الدوليُّ يبدأ بـ`+`** فيُحمى كذلك ويبقى مقروءاً.
func csvSafe(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	}
	return v
}

// roleLabelAr اسمُ الدور بالعربيّة في الملفّ — والمجهولُ يبقى رمزَه.
func roleLabelAr(code string) string {
	if l, ok := map[string]string{
		"customer": "زبون", "driver": "سائق", "merchant": "صاحب متجر", "sales": "مندوب",
		"admin": "مدير المنصة", "owner_super_admin": "المالك", "finance": "المالية",
		"operations": "العمليات", "ops": "العمليات", "customer_support": "دعم العملاء",
		"trust_safety": "الثقة والسلامة", "analytics": "التحليلات",
		"marketing_content": "التسويق والمحتوى", "driver_verification": "التحقق من السائقين",
		"merchant_verification": "التحقق من المتاجر", "platform_monitor": "مراقب المنصة",
	}[code]; ok {
		return l
	}
	return code
}

// statusLabelAr حالُ الحساب بالعربيّة.
func statusLabelAr(st string) string {
	switch st {
	case "active":
		return "فعّال"
	case "suspended":
		return "موقوف"
	case "blocked":
		return "محظور"
	}
	return st
}

// derefOr يقرأ مؤشراً نصياً بأمان.
func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

// minPasswordLen **طولُ كلمة المرور من اللوحة — مصدرٌ واحدٌ لكلّ المداخل.**
//
// (قرارُ المالك 2026-08-09: «طولُ كلمة المرور يجب أن تكون موحّدةً بكلّ
//
//	البرنامج» — وكانت أربعةُ مداخلَ تقرأ الإعدادَ وثلاثةٌ تكتب ٨ بيدها.)
//
// **ورقمٌ مكتوبٌ في مدخلٍ يتجاهل ما ضبطه المالك**: يرفع الحدَّ إلى اثني عشر
// **فتقبل إعادةُ التعيين ثمانيةً** — ولا يظهر الفرقُ إلّا لمن جرّب المدخلين.
//
// **ولا تُلزم حرفاً ولا رقماً** (قرارُ المالك: «هو حرٌّ في الاختيار») —
// والطولُ وحدَه شرط.
func (s *Server) minPasswordLen(ctx context.Context) int {
	n := s.settings.GetInt(ctx, "security.password_min_length")
	if n <= 0 {
		return 8
	}
	return int(n)
}
