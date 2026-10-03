package orders_test

// **مشكلةُ المتجر يقرّرها المكتب** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «ينتظر الإدارة تحلّ المشكلة») —
// السائقُ يُبلّغ ولا يُنهي، **والمكتبُ يعيد الطلبَ لتبديل المتجر** من لوحته (`ResolveDoor`).

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// officeStoreBlock **المكتبُ يقرّر «حوّل لمتجرٍ آخر»** بسبب البلاغ — بذنب المتجر.
func officeStoreBlock(t *testing.T, svc *orders.Service, pool *pgxpool.Pool, orderID, reason string) (*orders.Order, error) {
	t.Helper()
	ops := testdb.NewUser(t, pool, "ops")
	return svc.ResolveDoor(context.Background(), ops, []string{"ops"}, orderID,
		orders.DoorResolution{Action: orders.DoorReturnToOffice, Fault: orders.FaultMerchant,
			Reason: reason, Note: "اتّصلنا بالمتجر — مغلق"}, nil)
}
