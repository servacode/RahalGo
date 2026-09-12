package identity

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"time"
)

// عمليات إدارة المستخدمين — للوحة الأدمن (كلها مسجلة في سجل التدقيق).

var (
	ErrPhoneTaken  = httpx.NewError(http.StatusConflict, "phone_taken", "errors.phone_taken")
	ErrInvalidRole = httpx.NewError(http.StatusBadRequest, "invalid_role", "errors.invalid_role")
	ErrSelfAction  = httpx.NewError(http.StatusBadRequest, "self_action", "errors.self_action")
)

// AllRoles الأدوار السبعة المثبتة (PLAN.md §3).
var AllRoles = []string{"customer", "driver", "merchant", "sales", "ops", "finance", "admin"}

type UserPage struct {
	Users   []User `json:"users"`
	Total   int    `json:"total"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
}

func (s *Service) AdminListUsers(ctx context.Context, query, role string, onlineOnly bool, status string, page, perPage int) (*UserPage, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	users, total, err := s.repo.ListUsers(ctx, query, role, onlineOnly, status, perPage, (page-1)*perPage)
	if err != nil {
		return nil, err
	}
	return &UserPage{Users: users, Total: total, Page: page, PerPage: perPage}, nil
}

type CreateUserInput struct {
	Phone    string   `json:"phone"`
	FullName string   `json:"full_name"`
	Roles    []string `json:"roles"`
	Password string   `json:"password"`
}

func (s *Service) AdminCreateUser(ctx context.Context, actorID string, in CreateUserInput, ip string) (*User, error) {
	phone, ok := NormalizePhone(in.Phone)
	if !ok {
		return nil, ErrInvalidPhone
	}
	if len(in.Roles) == 0 {
		return nil, ErrInvalidRole
	}
	// **والدورُ المحميُّ لا يُخلَق به حسابٌ كذلك** — **وهذا المسلكُ
	// يأخذ الأدوارَ من مدخل الفاعل** وقدرتُه `users.status.manage` لا
	// `roles.manage`. **و`AllRoles` تحجبه اليومَ بالعرَض لا بقصد**
	// (رمزُه ليس فيها)، **ومن أضاف رمزاً إليها غداً فتح باباً لا
	// يعرف أنّه فتحه.**
	//
	// **والحمايةُ تُقاس قبل التحقّق الشكليّ** — **فخطأُ «دورٌ غير
	// صالح» يخبر الفاعلَ أنّ الرمزَ مجهول، وخطأُ الحماية يخبره أنّه
	// معروفٌ وممنوع** — **والثاني هو الحقّ، والأوّلُ يُخفي الحارسَ
	// فيُحسَب غائباً.**
	if err := s.guardProtectedRoles(ctx, actorID, in.Roles); err != nil {
		return nil, err
	}
	for _, r := range in.Roles {
		if !slices.Contains(AllRoles, r) {
			return nil, ErrInvalidRole
		}
	}
	// **ودورٌ واحدٌ ومعه الزبون** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	if err := checkOnePrimary(in.Roles); err != nil {
		return nil, err
	}
	// **ولا حسابَ متجرٍ بلا متجر** — يُنشأ مع متجره أو لا يُنشأ.
	for _, r := range in.Roles {
		if err := checkGrantable(r); err != nil {
			return nil, err
		}
	}
	if int64(len(in.Password)) < s.intSetting(ctx, "security.password_min_length", minPasswordLn) { // إلزامية — لا حساب موظف بلا كلمة مرور
		return nil, ErrWeakPassword
	}

	// **والإنشاءُ كلُّه معاملةٌ واحدة** — `PF-03` · `D15`.
	//
	// **كان ثلاثَ عمليّاتٍ متتالية**، **فسقوطُ الأخيرة يترك مستخدِماً
	// بلا كلمةٍ لا يدخل ورقمُه محجوز** — **والتعافي مسدود.**
	var hash string
	if in.Password != "" {
		var err error
		hash, err = auth.HashPassword(in.Password)
		if err != nil {
			return nil, err
		}
	}

	// **والكلمةُ مؤقّتةٌ لا نهائيّة**: الأدمنُ وضعها فمرّت بيدِ ثالثٍ كما
	// تمرّ كلمةُ المتجر بيد المندوب. **ومن اعتمدها نهائيّةً ترك موظّفَ
	// المنصّة بكلمةٍ يعرفها غيرُه إلى الأبد.**
	user, err := s.repo.AdminCreateUserFull(ctx, phone, in.FullName, in.Roles,
		hash, &actorID)
	if isUniqueViolation(err) {
		return nil, ErrPhoneTaken
	}
	if err != nil {
		return nil, err
	}

	s.repo.Audit(ctx, &actorID, "admin.user_create", "user", user.ID, ip,
		map[string]any{"phone": phone, "roles": in.Roles})
	user, _, err = s.repo.UserByID(ctx, user.ID)
	return user, err
}

type UpdateUserInput struct {
	FullName *string `json:"full_name"`
	Status   *string `json:"status"`
	// سبب الإيقاف/الحظر — إلزامي لغير active
	StatusReason *string `json:"status_reason"`
	// تغيير رقم الحساب (أدمن) — تنتقل الهوية بمحفظتها وسجلها
	Phone *string `json:"phone"`
	// معرف وسائط الصورة: غير مُرسل = بلا تغيير، "" = إزالة
	AvatarMediaID *string `json:"avatar_media_id"`
	// ملاحظات داخلية تراكمية (للموظفين فقط)
	AdminNotes *string `json:"admin_notes"`
}

func (s *Service) AdminUpdateUser(ctx context.Context, actorID, userID string, in UpdateUserInput, ip string) (*User, error) {
	if in.Status != nil {
		if *in.Status != "active" && *in.Status != "suspended" && *in.Status != "blocked" {
			return nil, errValidationErr
		}
		if userID == actorID && *in.Status != "active" {
			return nil, ErrSelfAction // لا يمكنك حظر نفسك
		}
		if *in.Status != "active" && (in.StatusReason == nil || *in.StatusReason == "") {
			return nil, errValidationErr // السبب إلزامي للإيقاف والحظر
		}
		if *in.Status == "active" && in.StatusReason == nil {
			empty := ""
			in.StatusReason = &empty // التفعيل يمسح السبب
		}
	}
	if in.Phone != nil {
		normalized, ok := NormalizePhone(*in.Phone)
		if !ok {
			return nil, ErrInvalidPhone
		}
		in.Phone = &normalized
	}
	// ══════════════════════════════════════════════════════════════
	// **والتبديلُ وأثرُه في معاملةٍ واحدة** — `XG-20` · `AQ-4`
	// ══════════════════════════════════════════════════════════════
	//
	// **«الإيقافُ والحظر» في نصّ العقد** — **وحظرٌ يقع وأثرُه يسقط لا
	// يُسأل عنه أحد.**
	//
	// **وإبطالُ الجلسات بعد التثبيت لا داخلَه**: **معاملتُه الخاصّةُ
	// في `RevokeAllTokens`** — ولا تُعشَّش. **وسقوطُها بعد تثبيتِ
	// الحظر لا يُبقي وصولاً**: **القاعدةُ تقول محظورٌ فيُرَدّ**
	// (`R16`).
	if err := func() error {
		tx, err := s.repo.pool().Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if err := s.repo.UpdateUserTx(ctx, tx, userID, in.FullName, in.Status,
			in.AvatarMediaID, in.StatusReason, in.Phone, in.AdminNotes); err != nil {
			return err
		}
		if err := AuditTx(ctx, tx, &actorID, "admin.user_update", "user", userID, ip,
			map[string]any{"full_name": in.FullName, "status": in.Status,
				"status_reason": in.StatusReason, "phone": in.Phone}); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}(); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPhoneTaken
		}
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.ErrNotFound
		}
		return nil, err
	}
	// ══════════════════════════════════════════════════════════════
	// **والإيقافُ العاديُّ لا يُبطل جلسة** — `XG-39`
	// ══════════════════════════════════════════════════════════════
	//
	// # ما كان
	//
	// **كلُّ حالٍ غيرِ `active` تُبطل كلَّ التوكنات.**
	//
	// **ولم يكن لذلك أثرٌ يُرى**: `RevokeAllTokens` تكتب القاعدةَ
	// **ولا تكتب مفتاحَ `Redis`** (بخلاف `revokeAllSessions`)،
	// **والوسيطُ كان يسأل `Redis` وحدَها** — **فبابُ دورةِ ١١ يعمل
	// بالمصادفة لا بالتصميم.**
	//
	// **ثمّ صارت القاعدةُ هي الحقيقة** (`R16`) — **فظهر التعارض**:
	// الإيقافُ يُبطل الجلسةَ فيُرَدُّ السائقُ بـ٤٠١ قبل أن يبلغ
	// استثناءَه، **والطلبُ الحيُّ يبقى معلَّقاً بمن لا يقدر.**
	//
	// # وعقدُ المالك (٢٠٢٦-٠٩-٠٧)
	//
	//	suspended  **يمنع نشاطاً جديداً ولا يترك طلباً حيّاً معلَّقاً**
	//	           — فلا إبطالَ للجلسات
	//	blocked    **حالُ أمنٍ استثنائيّة** — إبطالٌ فوريٌّ شامل
	//	deleted    **إزالةٌ** — إبطالٌ فوريٌّ شامل
	//	active     **لا تُبعَث جلسةٌ أُبطلت**
	//
	// # وحدُّ الأمان
	//
	// **التوثيقُ غيرُ التخويل**: **جلسةُ الموقوف تبقى قائمةً**،
	// **ووسيطُ دورةِ ١١ يمنعه من كلّ شيءٍ إلّا قائمةَ الاستمرار
	// المغلقة على طلبه الحيّ بعينه.** **فليست الجلسةُ الباقيةُ
	// إذناً.**
	//
	// **والإبطالُ الصريحُ باقٍ كما هو**: خروجٌ · إخراجٌ شامل ·
	// إعادةُ كلمةٍ · أحداثُ أمن. **وهذه تمسّ الحالَ وحدَها.**
	if in.Status != nil && (*in.Status == "blocked" || *in.Status == "deleted") {
		_, _ = s.repo.RevokeAllTokens(ctx, userID)
	}
	if in.Status != nil {
		s.invalidateStatusCache(ctx, userID)
	}
	user, _, err := s.repo.UserByID(ctx, userID)
	return user, err
}

func (s *Service) AdminGrantRole(ctx context.Context, actorID, userID, role, reason, ip string) error {
	// ══════════════════════════════════════════════════════════════
	// **والأدوارُ في القاعدة لا في قائمةٍ مُصرَّفة** — `ADG-2`
	// ══════════════════════════════════════════════════════════════
	//
	// **`AllRoles` قائمةٌ في الشيفرة** — **ودورٌ يُنشئه الأدمنُ غداً
	// ليس فيها، فيُردّ منحُه بأربعمئة.** **ويصير «الأدوارُ تُدار من
	// اللوحة» كلاماً.**
	//
	// **والقاعدةُ هي الحقيقة**: جدولُ `roles` — ومفتاحُه الأجنبيُّ في
	// `user_roles` يحرسه أيضاً.
	var known bool
	if err := s.repo.pool().QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM roles WHERE code = $1)`, role).
		Scan(&known); err != nil {
		return err
	}
	if !known {
		return ErrInvalidRole
	}
	// **ولا يُمنح دورُ المتجر بيد** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	if err := checkGrantable(role); err != nil {
		return err
	}
	// **ولا دورين أساسيّين لحسابٍ واحد** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	//
	// **والفحصُ قبل الكتابة لا بعدها**: منحٌ يقع ثمّ يُسحب يترك سطراً في
	// سجلّ التدقيق وإشعاراً وصل صاحبَه.
	if err := s.repo.ensureOnePrimary(ctx, userID, role); err != nil {
		return err
	}
	// **والمنحُ وأثرُه في معاملةٍ واحدة** — `XG-20` · `AQ-4`.
	//
	// **والحارسان قبلها**: `checkGrantable` و`ensureOnePrimary` —
	// **فحصٌ قبل الكتابة لا بعدها**، وقد وقعا سلفاً.
	return s.criticalRoleTx(ctx, actorID, userID, ip, "admin.role_grant",
		map[string]any{"role": role, "reason": reason},
		func(ctx context.Context, q dbtx.Querier) error {
			// **والدورُ المحميُّ لا يكفيه `roles.manage`** — **فحصٌ
			// داخلَ المعاملة قبل الكتابة**، انظر `protected_role.go`.
			// **وقِيس قبله**: أدمنٌ رقّى نفسَه مالكاً ⇒ `200 granted`.
			if err := guardProtectedGrant(ctx, q, actorID, role); err != nil {
				return err
			}
			_, err := q.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_code, granted_by)
				VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
				userID, role, &actorID)
			return err
		})
}

func (s *Service) AdminRevokeRole(ctx context.Context, actorID, userID, role, reason, ip string) error {
	if userID == actorID && role == "admin" {
		return ErrSelfAction // لا يمكنك سحب دور الأدمن من نفسك
	}
	// **والسحبُ وأثرُه في معاملةٍ واحدة** — `XG-20` · `AQ-4`.
	//
	// **وسحبٌ ينجح وأثرُه يسقط لا يُسأل عنه أحد** — **ولا يُعرَف من
	// سحبه ولا متى ولا لماذا.**
	//
	// **وهو يمسّ التخويلَ في اللحظة** (`R15`) — **فأولى أن يُقيَّد.**
	return s.criticalRoleTx(ctx, actorID, userID, ip, "admin.role_revoke",
		map[string]any{"role": role, "reason": reason},
		func(ctx context.Context, q dbtx.Querier) error {
			// **ونزعُ الدور المحميِّ من مالكٍ، وما دام يبقى مالك.**
			if err := guardProtectedRevoke(ctx, q, actorID, role); err != nil {
				return err
			}
			_, err := q.Exec(ctx,
				`DELETE FROM user_roles WHERE user_id = $1 AND role_code = $2`,
				userID, role)
			return err
		})
}

// criticalRoleTx **فعلٌ حسّاسٌ وأثرُه في معاملةٍ واحدة** — `XG-20`.
//
// **ولا نداءَ خارجيٌّ داخلَها**: **قفلٌ ينتظر شبكةً قفلٌ ينتظر الأبد.**
func (s *Service) criticalRoleTx(ctx context.Context, actorID, userID, ip,
	action string, details map[string]any,
	do func(context.Context, dbtx.Querier) error) error {
	tx, err := s.repo.pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := do(ctx, tx); err != nil {
		return err
	}
	if err := AuditTx(ctx, tx, &actorID, action, "user", userID, ip, details); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var errValidationErr = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// AdminResetPassword **إعادةُ كلمةٍ إداريّةٌ — فعلُ استرداد** (`R13`).
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا انتقلت من المعالِج إلى هنا**
// ══════════════════════════════════════════════════════════════════════
//
// **كانت جملةَ `UPDATE` في `handleAdminResetPassword` تتجاوز خدمةَ
// الهويّة كلَّها**: لا حدَّ معاملةٍ ولا إبطالَ جلسة. **فتُعاد الكلمةُ
// لحسابٍ اختُرق ويبقى صاحبُ الوصول القديمِ داخلاً** — **ورمزُ تجديده
// يدور إلى الأبد، فالمهلةُ لا تُنقذ.**
//
// **وقيس**: بعد الإعادة ⇒ الوصولُ 200 والتجديدُ 200 وصفوفٌ حيّةٌ=1.
//
// # وثلاثةُ أفعالٍ في معاملةٍ واحدة
//
//	١ البصمةُ الجديدةُ و`must_change_password`
//	٢ **حِقبةُ الجلسات** — `sessions_revoked_at = now()`
//	٣ إبطالُ كلّ رموز التجديد الحيّة
//
// **فإن سقط أحدُها لم يقع شيء** — **وإعادةُ كلمةٍ بلا إبطالٍ أسوأُ من
// لا إعادة**: تُطمئن من طلبها وهي لم تسترجع شيئاً.
//
// # ولماذا الحِقبةُ مع الإبطال
//
// **`Refresh` تُبطل الرمزَ المعروضَ ثمّ تُصدر بديلَه** — **وبينهما
// تقع الإعادةُ فلا تجد ما تُبطله، ثمّ يُدرَج البديلُ فيُفلت.**
// **والحِقبةُ تقتل العائلةَ كلَّها بأصلها لا بصفوفها** — **فلا يعبرها
// إدراجٌ متأخّر.**
//
// # والبثُّ والخبيئةُ بعد التثبيت
//
// **مفاتيحُ `Redis` تسريعُ رفضٍ لا مصدرُ حقيقة** (`R16`) — **فسقوطُها
// لا يُسقط الاسترداد.**
func (s *Service) AdminResetPassword(ctx context.Context, actorID, userID, hash, ip string) error {
	tx, err := s.repo.pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE users
		   SET password_hash = $2, must_change_password = true,
		       sessions_revoked_at = now(),
		       -- **وأقوى الإبطالين يغلب**: لا استثناءَ من جولةٍ
		       -- سابقةٍ يُنجي جلسةً وجب قطعُها.
		       sessions_kept_session_id = NULL,
		       updated_at = now()
		 WHERE id = $1::uuid`, userID, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}

	rows, err := tx.Query(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		 WHERE user_id = $1::uuid AND revoked_at IS NULL AND expires_at > now()
		RETURNING session_id::text`, userID)
	if err != nil {
		return err
	}
	var sids []string
	for rows.Next() {
		var sid string
		if rows.Scan(&sid) == nil {
			sids = append(sids, sid)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// **وبعد التثبيت**: مُسرِّعُ الرفض ثمّ خبيئةُ الحال ثمّ التدقيق.
	for _, sid := range sids {
		s.rdb.Set(ctx, sessionRevokedKey(sid), "1", s.tokens.AccessTTL()+time.Minute)
	}
	s.invalidateStatusCache(ctx, userID)
	s.repo.Audit(ctx, &actorID, "admin.password_reset", "user", userID, ip,
		map[string]any{"sessions_revoked": len(sids)})
	return nil
}

// AdminLogoutAll يُبطل كل جلسات الحساب فوراً (توكنات التجديد) — لقطع وصول موقوف.
func (s *Service) AdminLogoutAll(ctx context.Context, actorID, userID, ip string) (int, error) {
	sids, err := s.repo.ActiveSessionIDs(ctx, userID)
	if err != nil {
		return 0, err
	}
	n := len(sids)
	// الإبطال في القاعدة وفي قائمة Redis معاً — وإلا بقيت توكنات الوصول
	// القائمة صالحة حتى انتهاء مهلتها رغم "إنهاء الجلسات".
	if err := s.revokeAllSessions(ctx, userID); err != nil {
		return 0, err
	}
	s.repo.Audit(ctx, &actorID, "admin.logout_all", "user", userID, ip, map[string]any{"sessions": n})
	return n, nil
}
