package identity

// ══════════════════════════════════════════════════════════════════════
//  **تهيئةُ أوّلِ مالك — الكتابةُ هنا والحارسُ في الأداة**
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا هنا لا في `cmd/`
//
// **كلُّ كتابةٍ في `user_roles` تمرّ بهذه الحزمة** — `AdminGrantRole`
// و`GrantRoleTx` وهذه. **ومن كتب صفّاً من `cmd/` مباشرةً تجاوز كلَّ
// ما بُني هنا**: حدَّ المعاملة وقيدَ التدقيق وحارسَ السباق.
//
// **والأداةُ تبقى حارساً ونداءً** (`cmd/ownerbootstrap`): بيئةٌ
// وإذنٌ وتأكيدُ رقم — **ولا منطقَ كتابةٍ فيها.**
//
// # والمرّةُ الواحدةُ بالثابت لا بعلَم
//
// **ولا صفَّ «قد نُفِّذت» يُحفَظ** — **وعلَمٌ يُمحى تُعاد به الأداة.**
// **والشرطُ نفسُه هو الضمان**: **صفرُ مالكين.** فبعد أوّلِ نجاحٍ صار
// العددُ واحداً، فتردّ نفسَها إلى الأبد.
//
// # ويُقفَل ثمّ يُعَدّ
//
// **وأمران متزامنان يقرأ كلٌّ منهما «صفراً» فيكتبان معاً ⇒ مالكان.**
// **والقفلُ على الجدول لا على الصفّ** — **فالصفُّ غيرُ موجودٍ بعد،
// ولا يُقفَل ما ليس هناك.**

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ErrOwnerAlreadyExists **صار للمنصّة مالكٌ — ولا تهيئةَ ثانية.**
var ErrOwnerAlreadyExists = errors.New("owner_bootstrap: للمنصّة مالكٌ بالفعل")

// OwnerBootstrapAction **اسمُ الفعل في سجلّ التدقيق.**
const OwnerBootstrapAction = "admin.owner_bootstrap"

// OwnerBootstrapDB **ما يلزم للتهيئة** — بادئُ معاملةٍ لا أكثر.
type OwnerBootstrapDB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// BootstrapFirstOwner **يمنح أوّلَ مالكٍ — الصفُّ وأثرُه في معاملةٍ واحدة.**
//
// **ولا يُنشئ حساباً ولا يمسّ كلمةً ولا رمزَ لوحةٍ ولا هاتفاً** —
// **صفٌّ واحدٌ في `user_roles` وقيدٌ في `audit_log`، لا غير.**
func BootstrapFirstOwner(ctx context.Context, db OwnerBootstrapDB, userID, phone string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("owner_bootstrap: بدءُ المعاملة: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **يُقفَل الجدولُ ثمّ يُعَدّ** — انظر رأسَ الملفّ.
	if _, err := tx.Exec(ctx, `LOCK TABLE user_roles IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("owner_bootstrap: قفلُ الجدول: %w", err)
	}
	var owners int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM user_roles WHERE role_code = $1`,
		authz.RoleOwnerSuperAdmin).Scan(&owners); err != nil {
		return fmt.Errorf("owner_bootstrap: عدُّ المالكين: %w", err)
	}
	if owners != 0 {
		return ErrOwnerAlreadyExists
	}

	// **والحسابُ يجب أن يكون قائماً وفعّالاً** — ولا يُنشأ هنا.
	var status string
	if err := tx.QueryRow(ctx,
		`SELECT status FROM users WHERE id = $1::uuid`, userID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("owner_bootstrap: لا حسابَ بهذا المعرّف — **ولا يُنشَأ هنا**")
		}
		return fmt.Errorf("owner_bootstrap: قراءةُ الحساب: %w", err)
	}
	if status != "active" {
		return fmt.Errorf("owner_bootstrap: حالُ الحساب %q — ولا يُرقّى غيرُ الفعّال", status)
	}

	// **والعلاقةُ القائمةُ نفسُها** — `user_roles`، ولا جدولَ موازٍ.
	ct, err := tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_code, granted_by)
		VALUES ($1::uuid, $2, NULL) ON CONFLICT DO NOTHING`,
		userID, authz.RoleOwnerSuperAdmin)
	if err != nil {
		return fmt.Errorf("owner_bootstrap: كتابةُ الدور: %w", err)
	}
	if ct.RowsAffected() != 1 {
		return fmt.Errorf("owner_bootstrap: لم يُكتب صفٌّ واحدٌ (%d) — ولا يُكمَل على غموض",
			ct.RowsAffected())
	}

	// **والفاعلُ لا حسابَ له** — **الأمرُ يُنفَّذ على الخادم لا من
	// جلسة**، فـ`actor_user_id` فارغٌ والتفاصيلُ تقول من نفّذ وكيف.
	if err := AuditTx(ctx, tx, nil, OwnerBootstrapAction, "user", userID, "", map[string]any{
		"role":   authz.RoleOwnerSuperAdmin,
		"phone":  phone,
		"via":    "cmd/ownerbootstrap",
		"reason": "تهيئةُ أوّلِ مالكٍ — صفرُ مالكين قبلها",
	}); err != nil {
		return err
	}
	// **ولا يُرجَع أثرُ التدقيق مباشرةً** — **كان `return AuditTx(...)`
	// فلا يُنادى `Commit`**، **فتُرجِع الدالّةُ نجاحاً والمعاملةُ
	// تُرجَع كلُّها بالمؤجَّل**: **صفرُ مالكين وصفرُ آثارٍ ورمزُ
	// نجاح.** (أمسكه `OBS-2` و`OBS-3`، ٢٠٢٦-٠٩-١٢.)
	return tx.Commit(ctx)
}
