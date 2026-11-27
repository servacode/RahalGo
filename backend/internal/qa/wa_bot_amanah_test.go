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
	wbAmPayerQ   = "من سيدفع أجرة التوصيل؟\n1️⃣ أنا\n2️⃣ الشخص الآخر"
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
	id, status, dir, pickName, pickPhone, recName, recPhone, note, currency, fetchStatus, feePayer string
	bot, amanah, pickKnown, dropKnown                                                              bool
	collect, fetch                                                                                 *int64
	fee, total, cash                                                                               int64
}

func wbAmanahRow(t *testing.T, h *Harness, number int64) wbAmRow {
	t.Helper()
	var r wbAmRow
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT id::text, status, amanah_direction, pickup_name, COALESCE(pickup_phone, ''), recipient_name,
		       recipient_phone, parcel_note, amount_currency, COALESCE(fetch_amount_status, ''),
		       wa_bot, wa_amanah, pickup_override IS NOT NULL, dropoff_known,
		       collect_amount, fetch_amount, delivery_fee, total, cash_due, COALESCE(fee_payer, '')
		  FROM orders WHERE number = $1`, number).
		Scan(&r.id, &r.status, &r.dir, &r.pickName, &r.pickPhone, &r.recName, &r.recPhone, &r.note, &r.currency,
			&r.fetchStatus, &r.bot, &r.amanah, &r.pickKnown, &r.dropKnown, &r.collect, &r.fetch,
			&r.fee, &r.total, &r.cash, &r.feePayer); err != nil {
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
	wbExpect(t, wbSay(h, cust, "ليرة"), wbAmPayerQ, "العملة")
	if r := wbSay(h, cust, "3"); !strings.HasPrefix(r, wbUnclear) || !strings.Contains(r, wbAmPayerQ) {
		t.Fatalf("**دافعٌ غيرُ مفهوم**: %q", r)
	}
	// **الزبونُ يدفع وهو عند التسليم** ⇒ يدفعها عند التسليم كما كان.
	wbExpect(t, wbSay(h, cust, "أنا"), wbAmLocRecv, "الدافع")
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
		r.fetchStatus != "approved" || r.fee != 0 || r.feePayer != "recipient" {
		t.Fatalf("**الصفّ**: %+v", r)
	}
	if waLocReqCount(t, h, r.id, "pickup", other) != 1 {
		t.Fatal("**لم يصل الطرفَ الآخرَ رابطُ الموقع (نقطة الاستلام)**")
	}

	admin := h.NewUser("admin")
	b := wbBotAmanah(t, h, admin.Token, r.id)
	if b["direction"] != "receive" || b["other_name"] != "أبو خالد" || b["other_phone"] != other ||
		b["currency"] != "SYP" || b["other_located"] != false || b["customer_located"] != true ||
		b["fee_by"] != "customer" {
		t.Fatalf("**بطاقةُ اللوحة**: %v", b)
	}
	if amt, _ := b["amount"].(float64); amt != 5000 {
		t.Fatalf("**مبلغُ البطاقة**: %v", b["amount"])
	}

	// ── الموافقة ⇒ «مقبول» لا يبلغ الكباتن حتّى يصل موقعُ الطرف الآخر ──
	if res := h.POST(waURL+"/"+r.id+"/approve", admin.Token, map[string]any{"fee": 600}); res.Code != http.StatusOK {
		t.Fatalf("الموافقة: %d %s", res.Code, res.Body)
	}
	wbWaitMsg(t, n, cust, "تمت الموافقة على طلبك رقم #"+fmt.Sprint(num)+". أجرة التوصيل: 600 ل.س. نبحث")
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
	wbExpect(t, wbSay(h, cust, "٢"), wbAmPayerQ, "دولار")
	// **المستلِمُ يدفع وهو عند التسليم** ⇒ عند التسليم كما كان.
	wbExpect(t, wbSay(h, cust, "الشخص الآخر"), wbAmLocSend, "الدافع")
	num := wbDoneNumber(t, wbIn(h, notify.MenuInbound{From: cust, Kind: notify.MenuLocation,
		Loc: notify.InboundLocation{Lat: w.storeLat, Lng: w.storeLng}}).Text)

	r := wbAmanahRow(t, h, num)
	if r.status != "pending" || !r.bot || !r.amanah || r.dir != "send" || r.pickPhone != cust || !r.pickKnown ||
		r.recPhone != other || r.recName != "زبون" || r.dropKnown || r.collect == nil || *r.collect != 1250 ||
		r.currency != "USD" || r.fetch == nil || *r.fetch != 1250 || r.fetchStatus != "approved" ||
		r.feePayer != "recipient" {
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
		b["other_located"] != false || b["customer_located"] != true || b["fee_by"] != "other" {
		t.Fatalf("**بطاقةُ اللوحة**: %v", b)
	}

	if res := h.POST(waURL+"/"+r.id+"/approve", admin.Token, map[string]any{"fee": 600}); res.Code != http.StatusOK {
		t.Fatalf("الموافقة: %d %s", res.Code, res.Body)
	}
	wbWaitMsg(t, n, cust, "تمت الموافقة على طلبك رقم #"+fmt.Sprint(num)+
		". أجرة التوصيل: 600 ل.س، ويدفعها الشخص الآخر للكابتن عند استلامه الأمانة.")
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

// wbAmanahToPickup **أمانةُ بوتٍ إلى «مقبول» ثمّ النزول** — الأسئلةُ كلُّها بلا مبلغ، والدافعُ كما يُعطى،
// ثمّ الموافقةُ بأجرة ٦٠٠ وموقعُ الطرف الآخر. يردّ الصفَّ ورقمَ الطلب.
func wbAmanahToPickup(t *testing.T, h *Harness, w waFx, n *recNotifier, cust, other, dir, payer,
	wantApproved string) wbAmRow {
	t.Helper()
	wbSay(h, cust, "مرحبا")
	wbExpect(t, wbSay(h, cust, "7"), wbAmDirQ, "الخيار ٧")
	nameQ, locQ := wbAmNameRecv, wbAmLocRecv
	if dir == "1" {
		nameQ, locQ = wbAmNameSend, wbAmLocSend
	}
	wbExpect(t, wbSay(h, cust, dir), nameQ, "الاتّجاه")
	wbExpect(t, wbSay(h, cust, "سامر"), wbAmPhoneQ, "الاسم")
	wbExpect(t, wbSay(h, cust, wbLocal(other)), wbAmDescQ, "الرقم")
	wbExpect(t, wbSay(h, cust, "علبة"), wbAmPayQ, "الوصف")
	wbExpect(t, wbSay(h, cust, "لا"), wbAmPayerQ, "بلا مبلغ")
	wbExpect(t, wbSay(h, cust, payer), locQ, "الدافع")
	custLat, custLng, otherLat, otherLng := w.md.dropLat, w.md.dropLng, w.storeLat, w.storeLng
	if dir == "1" {
		custLat, custLng, otherLat, otherLng = otherLat, otherLng, custLat, custLng
	}
	num := wbDoneNumber(t, wbIn(h, notify.MenuInbound{From: cust, Kind: notify.MenuLocation,
		Loc: notify.InboundLocation{Lat: custLat, Lng: custLng}}).Text)
	r := wbAmanahRow(t, h, num)
	admin := h.NewUser("admin")
	if res := h.POST(waURL+"/"+r.id+"/approve", admin.Token, map[string]any{"fee": 600}); res.Code != http.StatusOK {
		t.Fatalf("الموافقة: %d %s", res.Code, res.Body)
	}
	wbWaitMsg(t, n, cust, fmt.Sprintf(wantApproved, num))
	if reply := h.API.HandleWALocation(ctxBG(), other, otherLat, otherLng, 6); reply == "" {
		t.Fatal("**لا ردَّ على موقع الطرف الآخر**")
	}
	if st := h.statusOf(r.id); st != "dispatching" && st != "assigned" {
		t.Fatalf("**لم تنزل بعد موقع الطرف الآخر**: %s", st)
	}
	return wbAmanahRow(t, h, num)
}

// wbStep **انتقالُ الكابتن** — ويسقط إن رُدّ.
func wbStep(t *testing.T, h *Harness, drv *User, id string, to ...string) {
	t.Helper()
	for _, s := range to {
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": s}); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s: %s", s, r)
		}
	}
}

// wbCashRows **سطورُ قبض الكابتن عن الطلب** — عددُها ومجموعُها.
func wbCashRows(t *testing.T, h *Harness, id string) (n int, sum int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), COALESCE(sum(amount), 0)::bigint FROM driver_cash_entries
		 WHERE ref = $1 AND kind = 'order_collection'`, id).Scan(&n, &sum); err != nil {
		t.Fatal(err)
	}
	return n, sum
}

// TestWABOT12_AmanahSendSenderPaysAtPickup **«إرسال أمانة» والزبونُ يدفع** (قرارُ المالك ٢٠٢٦-١١-٢٧) — الزبونُ
// عند نقطة الاستلام ⇒ `merchant_cash`: **يُقبض منه عند الاستلام ويُقيَّد في الصندوق ساعتَها، ولا يُقبض ثانيةً
// عند التسليم**، والكابتنُ يرى «اقبض عند الاستلام» بآليّة «أنا نقداً» القائمة. والدفترُ والفحوصُ سليمة.
func TestWABOT12_AmanahSendSenderPaysAtPickup(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-BOT-12")
	wbOpen(t, h)
	n := armFeedback(t, h, true)
	base := mdaBaseline(t, h)
	cust, other := uniqPhone(), uniqPhone()

	r := wbAmanahToPickup(t, h, w, n, cust, other, "1", "1",
		"تمت الموافقة على طلبك رقم #%d. أجرة التوصيل: 600 ل.س. نبحث")
	if r.dir != "send" || r.feePayer != "merchant_cash" || r.cash != 600 || r.pickPhone != cust {
		t.Fatalf("**الصفّ**: %+v", r)
	}
	admin := h.NewUser("admin")
	if b := wbBotAmanah(t, h, admin.Token, r.id); b["fee_by"] != "customer" {
		t.Fatalf("**بطاقةُ اللوحة**: %v", b)
	}
	drv := h.driverOf(r.id)
	if d := driverOrderOf(t, h, drv.Token, r.id); d["fee_payer"] != "merchant_cash" || d["cash_due"] != float64(600) {
		t.Fatalf("**الكابتنُ لا يرى القبضَ عند الاستلام**: fee_payer=%v cash_due=%v", d["fee_payer"], d["cash_due"])
	}
	wbStep(t, h, drv, r.id, "at_pickup")
	if c, _ := wbCashRows(t, h, r.id); c != 0 {
		t.Fatalf("**قُبض قبل الاستلام**: %d", c)
	}
	wbStep(t, h, drv, r.id, "picked_up")
	if c, sum := wbCashRows(t, h, r.id); c != 1 || sum != 600 {
		t.Fatalf("**لم يُقبض عند الاستلام**: %d سطر بمجموع %d", c, sum)
	}
	wbStep(t, h, drv, r.id, "on_the_way", "at_dropoff")
	h.authorizeProofExempt(r.id)
	wbStep(t, h, drv, r.id, "delivered")
	if c, sum := wbCashRows(t, h, r.id); c != 1 || sum != 600 {
		t.Fatalf("**قُبض ثانيةً عند التسليم**: %d سطر بمجموع %d", c, sum)
	}
	if r = wbAmanahRow(t, h, wbNumber(t, h, r.id)); r.status != "delivered" || r.fee != 600 || r.cash != 600 {
		t.Fatalf("**بعد التسليم**: %+v", r)
	}
	if led := waLedgerOf(t, h, r.id); led["platform_profit"] != 60 || led["driver_earning"] != 540 {
		t.Fatalf("**الدفتر**: %v", led)
	}
	mdaNoNewViolations(t, h, base)
}

// TestWABOT13_AmanahReceiveOtherPaysAtPickup **«استلام أمانة» والشخصُ الآخرُ يدفع** — وهو عند نقطة الاستلام ⇒
// يُقبض منه عند الاستلام، **ورسالةُ الموافقة تقول إنّه يدفعها عند تسليمه الأمانة.** ثمّ فشلٌ بعد الاستلام
// بغير ذنب الزبون ⇒ **النقدُ يُردّ لدافعه بسطرٍ يقابله** فلا يبقى بذمّة الكابتن، والفحوصُ سليمة.
func TestWABOT13_AmanahReceiveOtherPaysAtPickup(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-BOT-13")
	wbOpen(t, h)
	n := armFeedback(t, h, true)
	base := mdaBaseline(t, h)
	cust, other := uniqPhone(), uniqPhone()

	r := wbAmanahToPickup(t, h, w, n, cust, other, "2", "الشخص الآخر",
		"تمت الموافقة على طلبك رقم #%d. أجرة التوصيل: 600 ل.س، ويدفعها الشخص الآخر للكابتن عند تسليمه الأمانة.")
	if r.dir != "receive" || r.feePayer != "merchant_cash" || r.cash != 600 || r.pickPhone != other {
		t.Fatalf("**الصفّ**: %+v", r)
	}
	admin := h.NewUser("admin")
	if b := wbBotAmanah(t, h, admin.Token, r.id); b["fee_by"] != "other" {
		t.Fatalf("**بطاقةُ اللوحة**: %v", b)
	}
	drv := h.driverOf(r.id)
	wbStep(t, h, drv, r.id, "at_pickup", "picked_up")
	if c, sum := wbCashRows(t, h, r.id); c != 1 || sum != 600 {
		t.Fatalf("**لم يُقبض عند الاستلام**: %d سطر بمجموع %d", c, sum)
	}
	wbStep(t, h, drv, r.id, "on_the_way")
	if res := h.POST("/api/v1/admin/orders/"+r.id+"/door-resolution", admin.Token, map[string]any{
		"action": "return_to_office", "fault": "platform", "return_to": "office", "note": "أُلغيت الأمانة",
	}); res.Code >= 400 {
		t.Fatalf("**إنهاءُ المكتب بعد الاستلام رُدّ**: %d / %s", res.Code, res.Err())
	}
	if st := h.statusOf(r.id); st != "failed" {
		t.Fatalf("الحال %s والمنتظَر failed", st)
	}
	var held int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0)::bigint FROM driver_cash_entries
		WHERE ref = $1 AND kind IN ('order_collection', 'recipient_cash_returned')`, r.id).Scan(&held); err != nil {
		t.Fatal(err)
	}
	if held != 0 {
		t.Fatalf("**نقدُ الدافع باقٍ بذمّة الكابتن بعد الفشل**: %d", held)
	}
	mdaNoNewViolations(t, h, base)
}

// wbNumber **رقمُ الطلب بمعرّفه.**
func wbNumber(t *testing.T, h *Harness, id string) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT number FROM orders WHERE id = $1`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
