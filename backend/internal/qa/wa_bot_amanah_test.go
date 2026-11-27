package qa

// ══════════════════════════════════════════════════════════════════════
// **أمانةٌ من قائمة بوت الواتساب** (الخيار ٧، قرارُ المالك ٢٠٢٦-١١-٢٧) — `server/wa_bot_menu.go`
// ══════════════════════════════════════════════════════════════════════
//
//	١ الأسئلةُ واحداً واحداً: الاتّجاه ← الاسم ← الرقم (يُطبَّع، ولا رقمُ الزبون) ← الوصف ← مبلغٌ؟ ← العملة ← الموقع
//	٢ طلبٌ معلّقٌ (`wa_bot`) بعلامة الأمانة واتّجاهها، ورابطُ موقعٍ للطرف الآخر
//	٣ الموافقةُ بأجرة ⇒ «مقبول» لا يبلغ الكباتن حتّى تصل نقطةُ الطرف الآخر، ثمّ ينزل وحدَه
//	٤ المالُ: المبلغُ ثمنُ بضاعةٍ خارجَ الدفتر (`collect_amount`)، والمقبوضُ في الصندوق والدفتر الأجرةُ وحدَها
//
// **والأرقامُ كلُّها وهميّةٌ** (`uniqPhone`).

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notify"
)

const (
	wbAmDirQ     = "هل تريد إرسال أمانة أم استلام أمانة؟\n1️⃣ إرسال أمانة\n2️⃣ استلام أمانة"
	wbAmNameSend = "يرجى كتابة اسم الشخص الذي ستُرسَل إليه الأمانة."
	wbAmNameRecv = "يرجى كتابة اسم الشخص الذي ستُستلَم منه الأمانة."
	wbAmPhoneQ   = "يرجى كتابة رقم هاتف هذا الشخص (ويُفضَّل أن يكون عليه واتساب)، لنرسل إليه رابطاً يشارك به موقعه."
	wbAmPhoneBad = "عذراً، رقم الهاتف غير صحيح."
	wbAmSame     = "عذراً، هذا رقمك أنت."
	wbAmDescQ    = "يرجى كتابة وصف مختصر للأمانة (مثلاً: ظرف أوراق، كيس ملابس)."
	wbAmPayQ     = "هل يحتاج الكابتن إلى دفع مبلغ عند استلام الأمانة، يستوفيه عند تسليمها؟ أجب بـ: نعم أو لا."
	wbAmAmountQ  = "يرجى كتابة المبلغ بالأرقام فقط."
	wbAmCurQ     = "ما عملة المبلغ؟\n1️⃣ ليرة سورية\n2️⃣ دولار"
	wbAmSYPWhole = "عذراً، المبلغ بالليرة السورية يُكتب دون فواصل عشرية."
	wbAmLocSend  = "يرجى إرسال موقعك (العادي أو المباشر)، وهو مكان استلام الأمانة منك."
	wbAmLocRecv  = "يرجى إرسال موقعك (العادي أو المباشر)، وهو مكان تسليم الأمانة إليك."
	wbAmDoneHead = "شكراً لك، تم تسجيل طلب الأمانة برقم #"
)

// wbLocal **الرقمُ بصيغته المحلّيّة** (09…) — كما يكتبه الزبون.
func wbLocal(p string) string { return "0" + strings.TrimPrefix(p, "+963") }

func wbExpect(t *testing.T, got, want, what string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: الردّ %q والمنتظَر %q", what, got, want)
	}
}

// wbWaitMsg **رسالةٌ لرقمٍ تبدأ بنصٍّ** — والإرسالُ بعد الموافقة في خيطٍ مستقلّ.
func wbWaitMsg(t *testing.T, n *recNotifier, phone, prefix string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, m := range n.to(phone) {
			if strings.HasPrefix(m.text, prefix) {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("**لم تصل %s رسالةٌ تبدأ بـ %q**: %+v", phone, prefix, n.to(phone))
}

type wbAmRow struct {
	id, status, dir, pickName, pickPhone, recName, recPhone, note, currency, fetchStatus string
	bot, amanah, pickKnown, dropKnown                                                    bool
	collect, fetch                                                                       *int64
	fee, total, cash                                                                     int64
}

func wbAmanahRow(t *testing.T, h *Harness, number int64) wbAmRow {
	t.Helper()
	var r wbAmRow
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT id::text, status, amanah_direction, pickup_name, COALESCE(pickup_phone, ''), recipient_name,
		       recipient_phone, parcel_note, amount_currency, COALESCE(fetch_amount_status, ''),
		       wa_bot, wa_amanah, pickup_override IS NOT NULL, dropoff_known,
		       collect_amount, fetch_amount, delivery_fee, total, cash_due
		  FROM orders WHERE number = $1`, number).
		Scan(&r.id, &r.status, &r.dir, &r.pickName, &r.pickPhone, &r.recName, &r.recPhone, &r.note, &r.currency,
			&r.fetchStatus, &r.bot, &r.amanah, &r.pickKnown, &r.dropKnown, &r.collect, &r.fetch,
			&r.fee, &r.total, &r.cash); err != nil {
		t.Fatal(err)
	}
	return r
}

func wbDoneNumber(t *testing.T, reply string) int64 {
	t.Helper()
	if !strings.HasPrefix(reply, wbAmDoneHead) || !strings.Contains(reply, "بانتظار موافقة الإدارة") {
		t.Fatalf("**ردُّ الإنشاء**: %q", reply)
	}
	var n int64
	if _, err := fmt.Sscanf(strings.TrimPrefix(reply, wbAmDoneHead), "%d", &n); err != nil || n <= 0 {
		t.Fatalf("رقمُ الطلب من %q: %v", reply, err)
	}
	return n
}

// wbBotAmanah **بطاقةُ أمانة البوت في قائمة اللوحة.**
func wbBotAmanah(t *testing.T, h *Harness, tok, id string) map[string]any {
	t.Helper()
	list := h.GET(waURL, tok)
	if list.Code != http.StatusOK {
		t.Fatalf("القائمة: %d %s", list.Code, list.Body)
	}
	rows, _ := list.JSON()["orders"].([]any)
	for _, x := range rows {
		if m, _ := x.(map[string]any); m != nil && m["id"] == id {
			if m["wa_bot"] != true {
				t.Fatalf("**ليست بعلامة البوت**: %v", m)
			}
			b, _ := m["bot_amanah"].(map[string]any)
			if b == nil {
				t.Fatalf("**لا تفاصيلَ أمانة**: %v", m)
			}
			return b
		}
	}
	t.Fatalf("**الأمانةُ ليست في القائمة**")
	return nil
}

// TestWABOT10_AmanahReceiveSYP **«استلام أمانة» بمبلغ بالليرة** — الأسئلةُ وحرّاسُها، ثمّ طلبٌ معلّقٌ ورابطُ موقعٍ
// للطرف الآخر، ثمّ الموافقةُ ⇒ «مقبول» حتّى يصل موقعُه ⇒ ينزل ⇒ التسليمُ بالأجرة وحدَها في الدفتر.
func TestWABOT10_AmanahReceiveSYP(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-BOT-10")
	wbOpen(t, h)
	n := armFeedback(t, h, true)
	base := mdaBaseline(t, h)
	cust, other := uniqPhone(), uniqPhone()

	wbMust(t, wbSay(h, cust, "مرحبا"), wbMenuHead, "القائمة")
	if r := wbSay(h, cust, "مرحبا"); !strings.Contains(r, "1 إلى 7") {
		t.Fatalf("**التوضيحُ لا يذكر السابع**: %q", r)
	}
	wbExpect(t, wbSay(h, cust, "7️⃣"), wbAmDirQ, "الخيار ٧")
	if r := wbSay(h, cust, "3"); !strings.HasPrefix(r, wbUnclear) || !strings.Contains(r, wbAmDirQ) {
		t.Fatalf("**اتّجاهٌ غيرُ مفهوم**: %q", r)
	}
	wbExpect(t, wbSay(h, cust, "٢"), wbAmNameRecv, "استلام")
	if r := wbIn(h, notify.MenuInbound{From: cust, Kind: notify.MenuMedia}).Text; !strings.Contains(r, wbAmNameRecv) {
		t.Fatalf("**صوتٌ مكانَ الاسم**: %q", r)
	}
	wbExpect(t, wbSay(h, cust, "أبو خالد"), wbAmPhoneQ, "الاسم")
	if r := wbSay(h, cust, "١٢٣"); !strings.HasPrefix(r, wbAmPhoneBad) {
		t.Fatalf("**رقمٌ فاسد**: %q", r)
	}
	if r := wbSay(h, cust, wbLocal(cust)); !strings.HasPrefix(r, wbAmSame) {
		t.Fatalf("**رقمُ الزبون نفسُه**: %q", r)
	}
	wbExpect(t, wbSay(h, cust, wbLocal(other)), wbAmDescQ, "الرقم")
	wbExpect(t, wbSay(h, cust, "كيس أوراق"), wbAmPayQ, "الوصف")
	wbExpect(t, wbSay(h, cust, "نعم"), wbAmAmountQ, "نعم")
	if r := wbSay(h, cust, "مبلغ"); !strings.HasPrefix(r, wbUnclear) {
		t.Fatalf("**مبلغٌ بلا أرقام**: %q", r)
	}
	wbExpect(t, wbSay(h, cust, "٥٠٠٠٫٥"), wbAmCurQ, "مبلغٌ بكسر")
	if r := wbSay(h, cust, "1"); !strings.HasPrefix(r, wbAmSYPWhole) || !strings.Contains(r, wbAmAmountQ) {
		t.Fatalf("**ليرةٌ بكسرٍ عشريّ**: %q", r)
	}
	wbExpect(t, wbSay(h, cust, "٥٠٠٠ ليرة"), wbAmCurQ, "المبلغ")
	wbExpect(t, wbSay(h, cust, "ليرة"), wbAmLocRecv, "العملة")
	if r := wbSay(h, cust, "عند الجامع"); !strings.HasPrefix(r, wbUnclear) || !strings.Contains(r, wbAmLocRecv) {
		t.Fatalf("**نصٌّ مكانَ الموقع**: %q", r)
	}
	num := wbDoneNumber(t, wbIn(h, notify.MenuInbound{From: cust, Kind: notify.MenuLocation,
		Loc: notify.InboundLocation{Lat: w.md.dropLat, Lng: w.md.dropLng}}).Text)

	// ── الصفّ: معلّقٌ بعلامة البوت والأمانة، والطرفُ الآخرُ «الاستلام من»، والمبلغُ موثَّق ──
	r := wbAmanahRow(t, h, num)
	if r.status != "pending" || !r.bot || !r.amanah || r.dir != "receive" || r.pickName != "أبو خالد" ||
		r.pickPhone != other || r.pickKnown || r.recPhone != cust || !r.dropKnown || r.note != "كيس أوراق" ||
		r.collect == nil || *r.collect != 5000 || r.currency != "SYP" || r.fetch == nil || *r.fetch != 5000 ||
		r.fetchStatus != "approved" || r.fee != 0 {
		t.Fatalf("**الصفّ**: %+v", r)
	}
	if waLocReqCount(t, h, r.id, "pickup", other) != 1 {
		t.Fatal("**لم يصل الطرفَ الآخرَ رابطُ الموقع (نقطة الاستلام)**")
	}

	admin := h.NewUser("admin")
	b := wbBotAmanah(t, h, admin.Token, r.id)
	if b["direction"] != "receive" || b["other_name"] != "أبو خالد" || b["other_phone"] != other ||
		b["currency"] != "SYP" || b["other_located"] != false || b["customer_located"] != true {
		t.Fatalf("**بطاقةُ اللوحة**: %v", b)
	}
	if amt, _ := b["amount"].(float64); amt != 5000 {
		t.Fatalf("**مبلغُ البطاقة**: %v", b["amount"])
	}

	// ── الموافقة ⇒ «مقبول» لا يبلغ الكباتن حتّى يصل موقعُ الطرف الآخر ──
	if res := h.POST(waURL+"/"+r.id+"/approve", admin.Token, map[string]any{"fee": 600}); res.Code != http.StatusOK {
		t.Fatalf("الموافقة: %d %s", res.Code, res.Body)
	}
	wbWaitMsg(t, n, cust, "تمت الموافقة على طلبك رقم #"+fmt.Sprint(num))
	if st := h.statusOf(r.id); st != "accepted" {
		t.Fatalf("**نزلت بلا نقطة استلام**: %s", st)
	}
	if reply := h.API.HandleWALocation(ctxBG(), other, w.storeLat, w.storeLng, 6); reply == "" {
		t.Fatal("**لا ردَّ على موقع الطرف الآخر**")
	}
	if st := h.statusOf(r.id); st != "dispatching" && st != "assigned" {
		t.Fatalf("**لم تنزل بعد موقع الطرف الآخر**: %s", st)
	}

	// ── التسليم: الأجرةُ وحدَها في الصندوق والدفتر، والمبلغُ عرضٌ يقبضه الكابتن ──
	waDeliver(t, h, r.id, 0)
	r = wbAmanahRow(t, h, num)
	if r.status != "delivered" || r.fee != 600 || r.total != 600 || r.cash != 600 ||
		r.collect == nil || *r.collect != 5000 {
		t.Fatalf("**بعد التسليم**: %+v", r)
	}
	if c := waCashOf(t, h, r.id); c != 600 {
		t.Fatalf("**مقبوضُ الكابتن في الصندوق %d** والمنتظَر الأجرةُ ٦٠٠", c)
	}
	led := waLedgerOf(t, h, r.id)
	if led["platform_profit"] != 60 {
		t.Fatalf("**حصّةُ المنصّة**: %v", led)
	}
	for k, v := range led {
		if v == 5000 || v == -5000 {
			t.Fatalf("**المبلغُ دخل الدفتر** (%s = %d)", k, v)
		}
	}
	mdaNoNewViolations(t, h, base)
}

// TestWABOT11_AmanahSendUSD **«إرسال أمانة» بمبلغٍ بالدولار** — الاستلامُ من نقطة الزبون والتسليمُ للطرف الآخر،
// **ورسالةُ الموافقة والمتابعةُ للزبون المرسِل** لا للمستلِم، **ولا تنزل للكباتن قبل موقع المستلِم.**
func TestWABOT11_AmanahSendUSD(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-BOT-11")
	wbOpen(t, h)
	n := armFeedback(t, h, true)
	base := mdaBaseline(t, h)
	cust, other := uniqPhone(), uniqPhone()

	wbSay(h, cust, "مرحبا")
	wbExpect(t, wbSay(h, cust, "7"), wbAmDirQ, "الخيار ٧")
	wbExpect(t, wbSay(h, cust, "1"), wbAmNameSend, "إرسال")
	wbExpect(t, wbSay(h, cust, "لا يوجد"), wbAmPhoneQ, "بلا اسم")
	wbExpect(t, wbSay(h, cust, other), wbAmDescQ, "الرقم الدوليّ")
	wbExpect(t, wbSay(h, cust, "ظرف أوراق رسميّة"), wbAmPayQ, "الوصف")
	if r := wbSay(h, cust, "ربما"); !strings.HasPrefix(r, wbUnclear) || !strings.Contains(r, wbYesNo) {
		t.Fatalf("**غيرُ نعم/لا**: %q", r)
	}
	wbExpect(t, wbSay(h, cust, "اي"), wbAmAmountQ, "نعم")
	wbExpect(t, wbSay(h, cust, "12.5$"), wbAmCurQ, "المبلغ")
	wbExpect(t, wbSay(h, cust, "٢"), wbAmLocSend, "دولار")
	num := wbDoneNumber(t, wbIn(h, notify.MenuInbound{From: cust, Kind: notify.MenuLocation,
		Loc: notify.InboundLocation{Lat: w.storeLat, Lng: w.storeLng}}).Text)

	r := wbAmanahRow(t, h, num)
	if r.status != "pending" || !r.bot || !r.amanah || r.dir != "send" || r.pickPhone != cust || !r.pickKnown ||
		r.recPhone != other || r.recName != "زبون" || r.dropKnown || r.collect == nil || *r.collect != 1250 ||
		r.currency != "USD" || r.fetch == nil || *r.fetch != 1250 || r.fetchStatus != "approved" {
		t.Fatalf("**الصفّ**: %+v", r)
	}
	if waLocReqCount(t, h, r.id, "dropoff", other) != 1 {
		t.Fatal("**لم يصل المستلِمَ رابطُ الموقع (نقطة التسليم)**")
	}
	// **والمرسِلُ يتابع طلبَه برقمه.**
	wbSay(h, cust, "0")
	wbSay(h, cust, "2")
	if rep := wbSay(h, cust, fmt.Sprint(num)); !strings.Contains(rep, "قيد المراجعة لدى الإدارة") {
		t.Fatalf("**المرسِلُ لا يتابع أمانتَه**: %q", rep)
	}

	admin := h.NewUser("admin")
	b := wbBotAmanah(t, h, admin.Token, r.id)
	if b["direction"] != "send" || b["other_phone"] != other || b["currency"] != "USD" ||
		b["other_located"] != false || b["customer_located"] != true {
		t.Fatalf("**بطاقةُ اللوحة**: %v", b)
	}

	if res := h.POST(waURL+"/"+r.id+"/approve", admin.Token, map[string]any{"fee": 600}); res.Code != http.StatusOK {
		t.Fatalf("الموافقة: %d %s", res.Code, res.Body)
	}
	wbWaitMsg(t, n, cust, "تمت الموافقة على طلبك رقم #"+fmt.Sprint(num))
	for _, m := range n.to(other) {
		if strings.HasPrefix(m.text, "تمت الموافقة") {
			t.Fatalf("**رسالةُ الموافقة ذهبت للمستلِم**: %q", m.text)
		}
	}
	if st := h.statusOf(r.id); st != "accepted" {
		t.Fatalf("**نزلت بلا موقع المستلِم**: %s", st)
	}
	if tr := h.POST("/api/v1/admin/orders/"+r.id+"/transition", admin.Token,
		map[string]any{"to": "dispatching"}); tr.Code < 400 {
		t.Fatalf("**المكتبُ أنزلها قبل موقع المستلِم**: %d", tr.Code)
	}
	if reply := h.API.HandleWALocation(ctxBG(), other, w.md.dropLat, w.md.dropLng, 6); reply == "" {
		t.Fatal("**لا ردَّ على موقع المستلِم**")
	}
	if st := h.statusOf(r.id); st != "dispatching" && st != "assigned" {
		t.Fatalf("**لم تنزل بعد موقع المستلِم**: %s", st)
	}

	waDeliver(t, h, r.id, 0)
	r = wbAmanahRow(t, h, num)
	if r.status != "delivered" || r.fee != 600 || r.total != 600 || r.cash != 600 ||
		r.collect == nil || *r.collect != 1250 || r.currency != "USD" {
		t.Fatalf("**بعد التسليم**: %+v", r)
	}
	if c := waCashOf(t, h, r.id); c != 600 {
		t.Fatalf("**مقبوضُ الكابتن في الصندوق %d** والمنتظَر الأجرةُ ٦٠٠", c)
	}
	if led := waLedgerOf(t, h, r.id); led["platform_profit"] != 60 {
		t.Fatalf("**حصّةُ المنصّة**: %v", led)
	}
	mdaNoNewViolations(t, h, base)
}
