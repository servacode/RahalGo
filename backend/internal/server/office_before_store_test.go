package server

// **قبل المتجر يردّ المكتبُ على «الزبونُ طلب الإلغاء»** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «الإلغاءُ قبل
// المتجر الإدارةُ تقرّره، لأنّ الزبونَ لا يبقى عنده زرُّ إلغاء») — **قِيس على جهازه**: البلاغُ وصل،
// ولم يكن للمكتب إلّا الإلغاء؛ **«أكمل» ردّها المحرّك بـ٤٠٩** فبقي السائقُ لا يعرف أقُرئ بلاغُه.

import (
	"context"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func TestOffice_KeepOrderBeforeStore(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := f.armOps(t)
	d := f.drivers[0]
	ctx := context.Background()
	orderID := f.problemOrderAt(t, "assigned", d)

	w := f.call(f.srv.handleDriverReportOrStage, http.MethodPost,
		"/driver/orders/"+orderID+"/report", orderID, d, []string{"driver"},
		`{"code":"customer_cancelled_by_phone"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("البلاغُ ردّ %d: %s", w.Code, w.Body.String())
	}
	if v := f.srv.orders.DoorViewOf(ctx, orderID); v.ReportCode != "customer_cancelled_by_phone" {
		t.Fatalf("لوحةُ المكتب لا ترى البلاغ: %+v", v)
	}

	w = f.call(f.srv.handleDoorResolution, http.MethodPost,
		"/admin/orders/"+orderID+"/door-resolution", orderID, ops, []string{"ops"},
		`{"action":"deliver_now","note":"لم يُلغِ"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("«أكمل الطلب» ردّ %d: %s", w.Code, w.Body.String())
	}
	if v := f.srv.orders.DoorViewOf(ctx, orderID); v.Instruction != orders.DoorDeliverNow {
		t.Fatalf("أمرُ «أكمل الطلب» لم يُكتب على الطلب: %+v", v)
	}
	if got := orders.DoorKeepTitle(); got != "الإدارة: أكمل الطلب" {
		t.Fatalf("نصُّ الخبر قبل المتجر %q", got)
	}

	// **والأمرُ يخصّ مرحلته** — عند المتجر لا يُقرأ «أكمل» قيلت قبله.
	if _, err := f.srv.orders.Transition(ctx, d, []string{"driver"},
		orderID, orders.StAtPickup, ""); err != nil {
		t.Fatalf("الوصولُ إلى المتجر: %v", err)
	}
	var instr string
	if err := f.pool.QueryRow(ctx, `SELECT COALESCE(door_instruction, '') FROM orders WHERE id = $1`,
		orderID).Scan(&instr); err != nil {
		t.Fatal(err)
	}
	if instr != "" {
		t.Fatalf("أمرُ ما قبل المتجر %q بقي عند المتجر", instr)
	}
}
