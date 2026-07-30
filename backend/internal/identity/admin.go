package identity

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/httpx"
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
	for _, r := range in.Roles {
		if !slices.Contains(AllRoles, r) {
			return nil, ErrInvalidRole
		}
	}
	if len(in.Password) < minPasswordLn { // إلزامية — لا حساب موظف بلا كلمة مرور
		return nil, ErrWeakPassword
	}

	user, err := s.repo.CreateUserWithRole(ctx, phone, in.FullName, in.Roles[0])
	if isUniqueViolation(err) {
		return nil, ErrPhoneTaken
	}
	if err != nil {
		return nil, err
	}
	for _, r := range in.Roles[1:] {
		if err := s.repo.GrantRole(ctx, user.ID, r, &actorID); err != nil {
			return nil, err
		}
	}
	if in.Password != "" {
		hash, err := auth.HashPassword(in.Password)
		if err != nil {
			return nil, err
		}
		if err := s.repo.SetPassword(ctx, user.ID, hash); err != nil {
			return nil, err
		}
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
	if err := s.repo.UpdateUser(ctx, userID, in.FullName, in.Status, in.AvatarMediaID, in.StatusReason, in.Phone, in.AdminNotes); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPhoneTaken
		}
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.ErrNotFound
		}
		return nil, err
	}
	// إيقاف/حظر: أبطل كل الجلسات فوراً وامسح كاش الحالة — قطع وصول كامل
	if in.Status != nil && *in.Status != "active" {
		_, _ = s.repo.RevokeAllTokens(ctx, userID)
	}
	if in.Status != nil {
		s.invalidateStatusCache(ctx, userID)
	}
	s.repo.Audit(ctx, &actorID, "admin.user_update", "user", userID, ip,
		map[string]any{"full_name": in.FullName, "status": in.Status,
			"status_reason": in.StatusReason, "phone": in.Phone})
	user, _, err := s.repo.UserByID(ctx, userID)
	return user, err
}

func (s *Service) AdminGrantRole(ctx context.Context, actorID, userID, role, ip string) error {
	if !slices.Contains(AllRoles, role) {
		return ErrInvalidRole
	}
	if err := s.repo.GrantRole(ctx, userID, role, &actorID); err != nil {
		return err
	}
	s.repo.Audit(ctx, &actorID, "admin.role_grant", "user", userID, ip, map[string]any{"role": role})
	return nil
}

func (s *Service) AdminRevokeRole(ctx context.Context, actorID, userID, role, ip string) error {
	if userID == actorID && role == "admin" {
		return ErrSelfAction // لا يمكنك سحب دور الأدمن من نفسك
	}
	if err := s.repo.RevokeRole(ctx, userID, role); err != nil {
		return err
	}
	s.repo.Audit(ctx, &actorID, "admin.role_revoke", "user", userID, ip, map[string]any{"role": role})
	return nil
}

var errValidationErr = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// AdminLogoutAll يُبطل كل جلسات الحساب فوراً (توكنات التجديد) — لقطع وصول موقوف.
func (s *Service) AdminLogoutAll(ctx context.Context, actorID, userID, ip string) (int, error) {
	n, err := s.repo.RevokeAllTokens(ctx, userID)
	if err != nil {
		return 0, err
	}
	s.repo.Audit(ctx, &actorID, "admin.logout_all", "user", userID, ip, map[string]any{"sessions": n})
	return n, nil
}
