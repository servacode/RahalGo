package server

// **عدّاداتُ القائمة وشريطُ الانتباه** — قراءةٌ بقدرة كلّ رقم، ونفسُ أرقام الرئيسيّة.
//
//   - **الرقمُ بقدرة صفحته**: من لا يملك القدرةَ يصله `null` لا صفر.
//   - **العالقُ والبلا-سائق والنقدُ والتوسّعُ تساوي ما تعرضه الرئيسيّةُ** (لا شرطَ عدٍّ ثانٍ).
//   - **والمسارُ مستثنىً بسببٍ مكتوب** (لا قدرةَ واحدةً تحجبه عن موظّفٍ ما).

import (
	"encoding/json"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

func readMenuCounts(t *testing.T, srv *Server, caps []string) menuCounts {
	t.Helper()
	w := callAs(srv.handleMenuCounts, "GET", "/admin/menu-counts", "", "", caps)
	var m menuCounts
	if err := json.Unmarshal(dataOf(t, w), &m); err != nil {
		t.Fatalf("الردُّ لا يُقرأ: %v", err)
	}
	return m
}

func TestMenuCounts_ExemptWithReason(t *testing.T) {
	if why, ok := authz.IsExempt("/menu-counts"); !ok || why == "" {
		t.Fatal("`/menu-counts` يجب أن يكون مستثنىً بسببٍ مكتوب")
	}
}

func TestMenuCounts_NumbersFollowCapabilities(t *testing.T) {
	srv := overviewServer(t)
	l := srv.orders.StuckLimitsOf(t.Context())
	seedMonitorOrder(t, srv, orders.StPending, l.AcceptMin+30, -1)
	seedMonitorOrder(t, srv, orders.StDispatching, 5, l.DriverMin+20)

	// موظّفُ عمليّات: الطلبات ونبضها فقط — لا مالَ ولا شكاوى.
	ops := readMenuCounts(t, srv, []string{string(authz.OrdersRead)})
	if ops.Attention.StuckOrders == nil || ops.Attention.UnassignedOrders == nil {
		t.Fatal("العالقُ وبلا-السائق يجب أن يصلا من يملك orders.read")
	}
	if ops.Pulse == nil {
		t.Fatal("النبضُ لمن يملك orders.read")
	}
	if ops.Attention.CashOver != nil || ops.Attention.PayoutsPending != nil || ops.Money != nil {
		t.Fatal("**المالُ لا يصل من لا يملك finance.read**")
	}
	if ops.Attention.TicketsOpen != nil || ops.Attention.EmergenciesOpen != nil || ops.Attention.LeadsNew != nil {
		t.Fatal("شكاوى وطوارئ وانضمامٌ لا تصل من لا يملك قدرتَها")
	}

	// ماليّة: المالُ فقط.
	fin := readMenuCounts(t, srv, []string{string(authz.FinanceRead)})
	if fin.Attention.CashOver == nil || fin.Attention.PayoutsPending == nil || fin.Money == nil {
		t.Fatal("المالُ يجب أن يصل من يملك finance.read")
	}
	if fin.Attention.StuckOrders != nil || fin.Pulse != nil {
		t.Fatal("**الطلباتُ لا تصل من لا يملك orders.read**")
	}

	// بلا قدرة: لا شيء.
	none := readMenuCounts(t, srv, nil)
	if none.Pulse != nil || none.Money != nil || none.Attention.StuckOrders != nil {
		t.Fatal("لا قدرةَ ⇒ لا أرقام")
	}

	// **ونفسُ الرئيسيّة**: العالقُ وبلا-السائق والنقدُ فوق السقف.
	ov := srv.buildOverview(t.Context())
	both := readMenuCounts(t, srv, []string{string(authz.OrdersRead), string(authz.FinanceRead)})
	eq := func(name string, a, b num) {
		t.Helper()
		if a == nil || b == nil || *a != *b {
			t.Errorf("%s: القائمة %v والرئيسيّة %v — تعريفان", name, a, b)
		}
	}
	eq("العالق", both.Attention.StuckOrders, ov.Live.OrdersStuck)
	eq("بلا سائق", both.Attention.UnassignedOrders, ov.Awaiting.OrdersUnassigned)
	eq("نقد فوق السقف", both.Attention.CashOver, ov.Awaiting.DriversOverCash)
	eq("سحوب", both.Attention.PayoutsPending, ov.Awaiting.PayoutsPending)
	eq("توسّع", both.Attention.ExpansionWaiting, ov.Awaiting.ExpansionWaiting)
	eq("السائقون على الدوام", both.Pulse.DriversOnShift, ov.Live.DriversOnShift)
	eq("متاجر مفتوحة", both.Pulse.StoresOpenNow, ov.Live.StoresOpenNow)
	if len(both.Missing) != 0 {
		t.Fatalf("أقسامٌ لم تُقرأ: %v", both.Missing)
	}
}
