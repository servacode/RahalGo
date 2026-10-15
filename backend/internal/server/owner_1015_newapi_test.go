package server

// **قراراتُ المالك ٢٠٢٦-١٠-١٥ — ما يقرأ واجهاتٍ وُلدت معها** (الجولةُ ولفظُ المرحلة).
//
// **مفصولٌ عن `owner_1015_incentives_test.go`** — فذاك يُبنى على الشيفرة القديمة
// ويسقط فيها (دليلُ «ساقطٌ قبل الإصلاح»)، وهذا لا يُبنى عليها أصلاً.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/incentives"
)

// TestINC1015_PendingSweepPaysReactivatedDriver **والجولةُ تلتقط من عاد من غير شاشة الحسابات.**
func TestINC1015_PendingSweepPaysReactivatedDriver(t *testing.T) {
	f := newIncFixture(t)
	f.setTargets(t, map[string]string{
		"drivers.monthly_target": "1", "drivers.target_reward": "700",
		"drivers.target_2": "0", "drivers.reward_2": "0", "drivers.target_3": "0", "drivers.reward_3": "0",
	})
	ctx := context.Background()
	driver := f.kept(t, "driver")
	if _, err := f.pool.Exec(ctx, `UPDATE users SET status = 'suspended' WHERE id = $1`, driver); err != nil {
		t.Fatal(err)
	}
	f.deliveredOrder(t, driver)
	if paid := f.srv.incentives.GrantTargetIfReached(ctx, driver, "driver"); paid != 0 {
		t.Fatalf("صُرف لموقوف: %d", paid)
	}
	var held int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM incentive_pending_grants
		WHERE user_id = $1 AND paid_at IS NULL`, driver).Scan(&held); err != nil || held != 1 {
		t.Fatalf("**لم تُحفظ مكافأةُ الموقوف**: %d (%v)", held, err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE users SET status = 'active' WHERE id = $1`, driver); err != nil {
		t.Fatal(err)
	}
	var paid int64
	for _, p := range f.srv.incentives.PayPendingSweep(ctx) {
		if p.UserID == driver {
			paid += p.Amount
		}
	}
	if paid != 700 || f.balance(t, driver) != 700 {
		t.Fatalf("الجولةُ صرفت %d والرصيدُ %d — والمنتظَر 700", paid, f.balance(t, driver))
	}
}

// ── تحقّقٌ صغيرٌ من لفظ المرحلة بلا قاعدة ────────────────────────────────

func TestINC1015_StageTitleWording(t *testing.T) {
	if got := incentives.StageTitle(1, false); got != "وصلت للمرحلة 1 — نالتك مكافأتها" {
		t.Fatalf("المرحلةُ الأولى: %q", got)
	}
	if got := incentives.StageTitle(3, true); got != "أنجزت هدف الشهر — نالتك مكافأته" {
		t.Fatalf("الأخيرة: %q", got)
	}
}
