package identity

// ══════════════════════════════════════════════════════════════════════
// **لا يكون أحدٌ سائقاً ومندوباً معاً — برمزه ونصّه** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// **كان يُردّ بالتعارض العامّ** (`role_conflict`) — فلا يُعرف أنّ هذا الجمعَ ممنوعٌ بقرار.
// **فيُقاس الرمزُ نفسُه في الأبواب الثلاثة**: الإنشاءُ والمنحُ من اللوحة والمنحُ في معاملة.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestDriverSales_RefusedWithOwnCode **الجمعُ يُردّ بـ`role_driver_sales_conflict` بالاتّجاهين.**
func TestDriverSales_RefusedWithOwnCode(t *testing.T) {
	// ── الإنشاء ──
	for _, roles := range [][]string{{"driver", "sales"}, {"sales", "driver", "customer"}} {
		if err := checkOnePrimary(roles); !errors.Is(err, ErrDriverSalesConflict) {
			t.Fatalf("إنشاءٌ بـ%v ردّ %v — والمنتظَر role_driver_sales_conflict", roles, err)
		}
	}
	// **والتعارضُ العامُّ باقٍ لغيرهما.**
	if err := checkOnePrimary([]string{"merchant", "driver"}); !errors.Is(err, ErrRoleConflict) {
		t.Fatalf("متجرٌ وسائقٌ ردّ %v — والمنتظَر role_conflict", err)
	}

	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()

	// ── المنحُ من اللوحة ──
	driver := testdb.NewUser(t, pool, "driver")
	if err := repo.ensureOnePrimary(ctx, driver, "sales"); !errors.Is(err, ErrDriverSalesConflict) {
		t.Fatalf("سائقٌ يُمنح المندوبَ ردّ %v — والمنتظَر role_driver_sales_conflict", err)
	}
	rep := testdb.NewUser(t, pool, "sales")
	if err := repo.ensureOnePrimary(ctx, rep, "driver"); !errors.Is(err, ErrDriverSalesConflict) {
		t.Fatalf("مندوبٌ يُمنح السائقَ ردّ %v — والمنتظَر role_driver_sales_conflict", err)
	}

	// ── المنحُ في معاملة ──
	if err := GrantRoleTx(ctx, pool, rep, "driver", nil); !errors.Is(err, ErrDriverSalesConflict) {
		t.Fatalf("GrantRoleTx ردّ %v — والمنتظَر role_driver_sales_conflict", err)
	}
	var both bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_code = 'driver')`,
		rep).Scan(&both); err != nil {
		t.Fatal(err)
	}
	if both {
		t.Fatal("المندوبُ صار سائقاً رغم الردّ")
	}
}
