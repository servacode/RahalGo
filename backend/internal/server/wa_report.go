package server

// ══════════════════════════════════════════════════════════════════════
// **«وضع المنصة» على الواتساب** (طلبُ المالك ٢٠٢٦-١٠-٠٩)
// ══════════════════════════════════════════════════════════════════════
//
// «فيني اطلب تقرير عن طريق رقمي الواتس… اكتب شو وضع المنصة يعطيني شو في
// ما في.» — **والبوتُ على رقم المالك نفسِه**، فيكتب في محادثته مع نفسه
// («ملاحظة لنفسي») فيردّ البوتُ هناك. **أو من رقمٍ في قائمة
// `ops.report_phones`** فيردّ عليه.
//
// # الأمان
//
//   - **لا يردّ بالتقرير إلّا لمحادثة المالك مع نفسه أو لرقمٍ في القائمة** —
//     ومن كتب «تقرير» غيرُهم يمضي إلى المعالِج العاديّ كأنّه لم يكتب شيئاً.
//   - **لا اسمَ زبونٍ ولا رقمَه ولا عنوانَه** — أعدادٌ ومبالغُ عامّة.
//   - **لا يبادر البوتُ بشيء** — يردّ على من كتب، فلا خطرَ حظر.
//
// # والأرقامُ أرقامُ اللوحة نفسُها
//
// **يُنادى معالِجُ «الإحصاءات» نفسُه** (`handleAdminStats`) — **فلا يقول
// الواتساب رقماً وتقول اللوحةُ غيرَه.**

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/identity"
)

// waReportWords **الكلماتُ التي تطلب تقريراً** — وما يُعرض لكلٍّ منها.
var waReportWords = []struct {
	words []string
	kind  string
}{
	{[]string{"وضع المنصة", "وضع المنصه", "حالة المنصة", "حالة المنصه", "تقرير", "الوضع"}, "all"},
	{[]string{"طلبات", "الطلبات"}, "orders"},
	{[]string{"سائقين", "السائقين", "سواقين"}, "drivers"},
	{[]string{"مال", "المال", "مصاري", "فلوس"}, "money"},
}

func reportKind(text string) string {
	t := strings.TrimSpace(strings.Trim(text, "؟?!. "))
	for _, k := range waReportWords {
		for _, w := range k.words {
			if t == w {
				return k.kind
			}
		}
	}
	return ""
}

// ReportForSelf **محادثةُ المالك مع نفسه** — يُربط بالبوت في `main`.
func (s *Server) ReportForSelf(ctx context.Context, text string) string {
	kind := reportKind(text)
	if kind == "" {
		return ""
	}
	return s.platformReport(ctx, kind)
}

// ReportForPhone **رقمٌ في القائمة** — وفارغٌ لغيره فيمضي الواردُ إلى معالِجه العاديّ.
func (s *Server) ReportForPhone(ctx context.Context, from, text string) string {
	kind := reportKind(text)
	if kind == "" || !s.reportPhoneAllowed(ctx, from) {
		return ""
	}
	return s.platformReport(ctx, kind)
}

func (s *Server) reportPhoneAllowed(ctx context.Context, from string) bool {
	want, ok := identity.NormalizePhone(from)
	if !ok {
		return false
	}
	for _, p := range strings.FieldsFunc(s.settings.GetString(ctx, "ops.report_phones"), func(r rune) bool {
		return r == ',' || r == '،' || r == ' ' || r == '\n'
	}) {
		if n, ok := identity.NormalizePhone(p); ok && n == want {
			return true
		}
	}
	return false
}

type reportStats struct {
	OrdersOpen           int   `json:"orders_open"`
	OrdersStuck          int   `json:"orders_stuck"`
	OrdersUnassigned     int   `json:"orders_unassigned"`
	DriversOnShift       int   `json:"drivers_on_shift"`
	DriversBusy          int   `json:"drivers_busy"`
	DriversOverCash      int   `json:"drivers_over_cash"`
	MerchantsOpen        int   `json:"merchants_open"`
	PayoutsPending       int   `json:"payouts_pending"`
	TicketsOpen          int   `json:"tickets_open"`
	OrdersToday          int   `json:"orders_today"`
	DeliveredToday       int   `json:"delivered_today"`
	CancelledToday       int   `json:"cancelled_today"`
	SalesToday           int64 `json:"sales_today"`
	NetToday             int64 `json:"net_today"`
	CashHeldTotal        int64 `json:"cash_held_total"`
	LeadsNew             int   `json:"leads_new"`
	EmergenciesOpen      int   `json:"emergencies_open"`
	CompensationsPending int   `json:"compensations_pending"`
}

func (s *Server) reportStats(ctx context.Context) (*reportStats, error) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	s.handleAdminStats(rec, req)
	if rec.Code != http.StatusOK {
		return nil, fmt.Errorf("stats: %d", rec.Code)
	}
	// **والردُّ مغلَّفٌ** (`{"data": …}`) — كما يصل اللوحة.
	var env struct {
		Data reportStats `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// arNum **أرقامٌ عربيّة بفواصل الآلاف** — كما تُقرأ في الواتساب.
func arNum(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteRune('٬')
		}
		b.WriteRune('٠' + (c - '0'))
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func (s *Server) platformReport(ctx context.Context, kind string) string {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	st, err := s.reportStats(ctx)
	if err != nil {
		return "⚠️ ما قدرت أقرأ أرقام المنصة هلق — قاعدة البيانات ما ردّت. جرّب بعد دقيقة."
	}
	n := func(v int) string { return arNum(int64(v)) }
	loc, _ := time.LoadLocation("Asia/Damascus")
	now := time.Now()
	if loc != nil {
		now = now.In(loc)
	}
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }

	orders := func() {
		w("*الطلبات اليوم:* %s · تسلّم %s · انلغى %s · جارية هلق %s",
			n(st.OrdersToday), n(st.DeliveredToday), n(st.CancelledToday), n(st.OrdersOpen))
		if st.OrdersStuck > 0 {
			w("⚠️ متأخّر عن مهلته: *%s*", n(st.OrdersStuck))
		}
		if st.OrdersUnassigned > 0 {
			w("⚠️ بلا سائق: *%s*", n(st.OrdersUnassigned))
		}
	}
	drivers := func() {
		w("*السائقين:* %s على الدوام · %s بتوصيلة · %s فاضيين",
			n(st.DriversOnShift), n(st.DriversBusy), n(max(st.DriversOnShift-st.DriversBusy, 0)))
		if st.DriversOverCash > 0 {
			w("⚠️ تجاوزوا سقف النقد: %s", n(st.DriversOverCash))
		}
	}
	money := func() {
		w("*المال اليوم:* مبيعات %s · صافي المنصة %s · نقد بإيد السائقين %s",
			arNum(st.SalesToday), arNum(st.NetToday), arNum(st.CashHeldTotal))
	}

	w("📊 *وضع رحّال غو* — %s", now.Format("2006-01-02 15:04"))
	w("")
	switch kind {
	case "orders":
		orders()
	case "drivers":
		drivers()
	case "money":
		money()
	default:
		orders()
		w("")
		drivers()
		w("*المتاجر الفاتحة:* %s", n(st.MerchantsOpen))
		w("")
		w("*ناطرين قرارك:* %s طلب انضمام · %s شكوى/تذكرة · %s سحب · %s تعويض",
			n(st.LeadsNew), n(st.TicketsOpen), n(st.PayoutsPending), n(st.CompensationsPending))
		if st.EmergenciesOpen > 0 {
			w("🚨 حالات طوارئ مفتوحة: *%s*", n(st.EmergenciesOpen))
		}
		w("")
		money()
		w("")
		w("*الصحة:* %s", s.healthLine(ctx))
	}
	w("")
	w("_اكتب: تقرير · طلبات · سائقين · مال_")
	return strings.TrimRight(b.String(), "\n")
}

func (s *Server) healthLine(ctx context.Context) string {
	ok := func(b bool) string {
		if b {
			return "✅"
		}
		return "❌"
	}
	dbOK := s.pg != nil && s.pg.Ping(ctx) == nil && !dbDownFlag.Load()
	redisOK := s.rdb != nil && s.rdb.Ping(ctx).Err() == nil
	return "القاعدة " + ok(dbOK) + " · Redis " + ok(redisOK) + " · البوت " + ok(s.merchantReady())
}
