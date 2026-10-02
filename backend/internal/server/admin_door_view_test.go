package server

// **لوحةُ «عند باب الزبون» تقرأ ما تحتاجه من ردّ الإدارة** (قرارُ المالك مساءَ
// ٢٠٢٦-١٠-٠٢، البند ١): آخرُ بلاغٍ وكم ينتظر السائقُ وأمرُ المكتب — **وتنبيهُ
// البلاغ يشير إلى الطلب بعينه.**

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func TestAdminOrders_DoorViewAndAlertHref(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := f.armOps(t)
	d := f.drivers[0]
	ctx := context.Background()
	orderID := f.problemOrderAt(t, "on_the_way", d)
	if _, err := f.srv.orders.Transition(ctx, d, []string{"driver"},
		orderID, orders.StAtDropoff, ""); err != nil {
		t.Fatalf("الوصول: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `
		UPDATE order_events SET created_at = now() - interval '9 minutes'
		WHERE order_id = $1 AND to_status = 'at_dropoff'`, orderID); err != nil {
		t.Fatal(err)
	}
	var number int64
	if err := f.pool.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, orderID).
		Scan(&number); err != nil {
		t.Fatal(err)
	}

	w := f.call(f.srv.handleDriverReportOrStage, http.MethodPost,
		"/driver/orders/"+orderID+"/report", orderID, d, []string{"driver"},
		`{"code":"customer_refused","note":"قال لم أطلب"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("البلاغُ ردّ %d: %s", w.Code, w.Body.String())
	}

	// **والتنبيهُ يشير إلى الطلب بعينه** لا إلى القائمة عامّةً.
	var href string
	if err := f.pool.QueryRow(ctx, `
		SELECT href FROM notifications WHERE user_id = $1 AND entity_id = $2`,
		ops, orderID).Scan(&href); err != nil {
		t.Fatalf("قراءةُ التنبيه: %v", err)
	}
	if want := "/dashboard/orders?q=" + strconv.FormatInt(number, 10); href != want {
		t.Fatalf("وجهةُ التنبيه %q — والمتوقّعُ %q", href, want)
	}

	readDoor := func() *orders.DoorView {
		t.Helper()
		w := f.call(f.srv.handleListOrders, http.MethodGet,
			"/admin/orders?query="+strconv.FormatInt(number, 10), "", ops, []string{"ops"}, "")
		if w.Code != http.StatusOK {
			t.Fatalf("القائمةُ ردّت %d: %s", w.Code, w.Body.String())
		}
		// **والردُّ ملفوفٌ في `data`** — كسائر ردود الإدارة.
		var env struct {
			Data orders.OrderPage `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		page := env.Data
		for _, o := range page.Orders {
			if o.ID == orderID {
				if o.Door == nil {
					t.Fatal("طلبٌ عند الباب بلا `door` في ردّ الإدارة")
				}
				return o.Door
			}
		}
		t.Fatalf("الطلبُ غائبٌ عن القائمة: %s", w.Body.String())
		return nil
	}

	v := readDoor()
	if v.ReportCode != "customer_refused" || v.ReportNote != "قال لم أطلب" ||
		v.ReportAt == nil || v.SuggestedFault != orders.FaultCustomer {
		t.Fatalf("البلاغُ في الردّ %+v", v)
	}
	if v.WaitedMin != 9 || v.ArrivedAt == nil {
		t.Fatalf("الانتظارُ %d (وصل %v) — والمتوقّعُ ٩ دقائق", v.WaitedMin, v.ArrivedAt)
	}
	if v.Instruction != "" {
		t.Fatalf("أمرٌ %q ولم يُرسَل شيء", v.Instruction)
	}

	w = f.call(f.srv.handleDoorResolution, http.MethodPost,
		"/admin/orders/"+orderID+"/door-resolution", orderID, ops, []string{"ops"},
		`{"action":"deliver_now","note":"الزبون سيفتح"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("«سلّم الآن» ردّ %d: %s", w.Code, w.Body.String())
	}
	v = readDoor()
	if v.Instruction != orders.DoorDeliverNow || v.InstructionNote != "الزبون سيفتح" ||
		v.InstructionAt == nil {
		t.Fatalf("الأمرُ في الردّ %+v", v)
	}
}

// TestAdminOrders_DoorViewOnlyAtDoor **ولا `door` لطلبٍ ليس عند الباب.**
func TestAdminOrders_DoorViewOnlyAtDoor(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := f.armOps(t)
	orderID := f.problemOrderAt(t, "on_the_way", f.drivers[0])
	w := f.call(f.srv.handleListOrders, http.MethodGet, "/admin/orders?status=on_the_way",
		"", ops, []string{"ops"}, "")
	var env struct {
		Data orders.OrderPage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	// **ويُشترط أنّ الطلبَ وُجد** — وإلّا مرّ الاختبارُ على قائمةٍ فارغةٍ لا على الحكم.
	found := false
	for _, o := range env.Data.Orders {
		if o.ID == orderID {
			found = true
			if o.Door != nil {
				t.Fatalf("طلبٌ في الطريق يحمل `door`: %+v", o.Door)
			}
		}
	}
	if !found {
		t.Fatalf("الطلبُ غائبٌ عن القائمة: %s", w.Body.String())
	}
}
