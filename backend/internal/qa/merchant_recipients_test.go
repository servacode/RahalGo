package qa

// ══════════════════════════════════════════════════════════════════════
// **مستلمو «لدي توصيلة» السابقون — يُقترحون وقتَ الكتابة** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «اذا رجع بعث مره ثانية بمجرد ما يكتب اسم الزبون يتعبى باقي المعلومات بشكل تلقائي
// عنوان موقع رقم وكلشي»
//
// **وما يُقاس هنا من الشبكة كما يناديه التطبيقُ والموقع**: كلُّ متجرٍ يرى مستلِميه وحدَهم
// (وموظّفوه معه) · صفٌّ لكلّ رقمٍ بصيغه كلِّها من آخر توصيلة · النقطةُ إن عُرفت وحدَها ·
// الاسمُ بلا همزةٍ ولا تاءٍ مربوطة · «شيلو من القائمة» ويرجع إن بعث له من جديد.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// mrSend **توصيلةٌ من باب المتجر بمستلِمٍ معلوم** — والنقطةُ اختياريّة.
func mrSend(t *testing.T, h *Harness, m mdFx, tok, storeID, name, phone, addr string, point bool) string {
	t.Helper()
	body := map[string]any{
		"recipient_name": name, "recipient_phone": phone, "address_text": addr,
		"fee_payer": "merchant_cash", "delivery_mode": "send",
	}
	if point {
		body["lat"], body["lng"] = m.dropLat, m.dropLng
	}
	r := h.POSTKey("/api/v1/merchant/stores/"+storeID+"/deliveries", tok, uniq("mr"), body)
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاء (%s): %d / %s", phone, r.Code, r)
	}
	id, _ := r.JSON()["id"].(string)
	return id
}

// mrLegacyPhone **صفٌّ قديمٌ برقمٍ غيرِ موحّد** — كما لو كُتب قبل توحيد بوّابة الإنشاء أو من
// غير الباب. **فبوّابةُ الإنشاء توحّد كلَّ ما يمرّ بها**، ولا يُقاس الجمعُ إلّا بصفٍّ لم يمرّ.
func mrLegacyPhone(t *testing.T, h *Harness, orderID, phone string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET recipient_phone = $2 WHERE id = $1`, orderID, phone); err != nil {
		t.Fatalf("صفٌّ قديم: %v", err)
	}
}

// mrList **مستلمو المتجر لما كُتب** — ويُعاد الردُّ كاملاً ليُقرأ رمزُه.
func mrList(h *Harness, tok, storeID, q string) Res {
	p := "/api/v1/merchant/stores/" + storeID + "/delivery-recipients"
	if q != "" {
		p += "?q=" + url.QueryEscape(q)
	}
	return h.GET(p, tok)
}

// mrRows **صفوفُ القائمة** — ويسقط الفحصُ إن لم تُقرأ.
func mrRows(t *testing.T, h *Harness, tok, storeID, q string) []map[string]any {
	t.Helper()
	r := mrList(h, tok, storeID, q)
	if r.Code != http.StatusOK {
		t.Fatalf("القائمة (q=%q): %d / %s", q, r.Code, r)
	}
	raw, ok := r.JSON()["recipients"].([]any)
	if !ok {
		t.Fatalf("ردٌّ بلا recipients: %s", r)
	}
	out := make([]map[string]any, 0, len(raw))
	for _, x := range raw {
		row, _ := x.(map[string]any)
		out = append(out, row)
	}
	return out
}

func mrPhones(rows []map[string]any) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		p, _ := r["phone"].(string)
		out = append(out, p)
	}
	return out
}

func mrHas(rows []map[string]any, phone string) map[string]any {
	for _, r := range rows {
		if r["phone"] == phone {
			return r
		}
	}
	return nil
}

// mrStaffToken **موظّفٌ عند صاحب متجر** — رمزُه باسم صاحبه ويحمل الموظّف (`IssueAccessStaff`)،
// كما يصدره الدخول.
func mrStaffToken(t *testing.T, h *Harness, ownerID string) string {
	t.Helper()
	var staffID string
	if err := h.Pool.QueryRow(context.Background(), `
		INSERT INTO merchant_staff (owner_user_id, phone, name, password_hash)
		VALUES ($1::uuid, $2, 'موظف فحص', 'x') RETURNING id::text`,
		ownerID, uniqPhone()).Scan(&staffID); err != nil {
		t.Fatalf("موظّف: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `DELETE FROM merchant_staff WHERE id = $1`, staffID)
	})
	tok, _, err := h.tokens.IssueAccessStaff(ownerID, []string{"merchant"}, "", staffID)
	if err != nil {
		t.Fatalf("رمزُ الموظّف: %v", err)
	}
	return tok
}

// TestMR01_OwnStoreOnly **كلُّ متجرٍ يرى مستلِميه وحدَهم — وموظّفوه معه، ولا أحدَ غيرُهم.**
func TestMR01_OwnStoreOnly(t *testing.T) {
	h := New(t)
	a := newMDFx(t, h, "منطقةُ MR-01")
	b := a.f.Merchant()
	bTok := h.TokenFor(b.Owner.ID, "merchant")

	mrSend(t, h, a, a.fx.Tok, a.fx.M.ID, "أبو خالد", "0935111222", "شارع المنصور", true)
	mrSend(t, h, a, bTok, b.ID, "أم سمير", "0944333444", "حي الثكنة", false)

	aRows := mrRows(t, h, a.fx.Tok, a.fx.M.ID, "")
	if len(aRows) != 1 || mrHas(aRows, "+963935111222") == nil {
		t.Fatalf("**المتجرُ أ لا يرى مستلِمه وحدَه**: %v", mrPhones(aRows))
	}
	bRows := mrRows(t, h, bTok, b.ID, "")
	if len(bRows) != 1 || mrHas(bRows, "+963944333444") == nil {
		t.Fatalf("**المتجرُ ب لا يرى مستلِمه وحدَه**: %v", mrPhones(bRows))
	}
	// **وبحثٌ برقم الآخر لا يجده** — لا بالقائمة ولا بالنصّ.
	if rows := mrRows(t, h, bTok, b.ID, "0935"); len(rows) != 0 {
		t.Fatalf("**المتجرُ ب وجد مستلِمَ أ بالبحث**: %v", mrPhones(rows))
	}

	// ── الموظّفون: موظّفُ أ يرى، وموظّفُ ب لا ───────────────────────────
	aStaff := mrStaffToken(t, h, a.fx.M.Owner.ID)
	bStaff := mrStaffToken(t, h, b.Owner.ID)
	if rows := mrRows(t, h, aStaff, a.fx.M.ID, "خالد"); mrHas(rows, "+963935111222") == nil {
		t.Fatalf("**موظّفُ المتجر لا يرى مستلِمي متجره**: %v", mrPhones(rows))
	}

	// ── ومن لا يملك المتجرَ يُردّ ٤٠٣ — قراءةً وإخفاءً ───────────────────
	cust := h.Customer()
	for _, c := range []struct{ who, tok string }{
		{"صاحبُ متجرٍ آخر", bTok}, {"موظّفُ متجرٍ آخر", bStaff}, {"زبون", cust.Token},
	} {
		if r := mrList(h, c.tok, a.fx.M.ID, ""); r.Code != http.StatusForbidden {
			t.Fatalf("**%s قرأ مستلمي غيره**: %d / %s", c.who, r.Code, r)
		}
		if r := h.DEL("/api/v1/merchant/stores/"+a.fx.M.ID+"/delivery-recipients/0935111222", c.tok); r.Code != http.StatusForbidden {
			t.Fatalf("**%s أخفى مستلِمَ غيره**: %d / %s", c.who, r.Code, r)
		}
	}
	// **والإخفاءُ المردودُ لم يُكتب** — المستلِمُ باقٍ عند صاحبه.
	if rows := mrRows(t, h, a.fx.Tok, a.fx.M.ID, ""); mrHas(rows, "+963935111222") == nil {
		t.Fatalf("**إخفاءٌ مردودٌ كُتب**: %v", mrPhones(rows))
	}
}

// TestMR02_LatestPerPhone **صفٌّ لكلّ رقمٍ بصيغه كلِّها — من آخر توصيلةٍ إليه، والأحدثُ أوّلاً.**
func TestMR02_LatestPerPhone(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MR-02")
	tok, sid := m.fx.Tok, m.fx.M.ID

	// **الرقمُ نفسُه بثلاث صيغ مخزَّنة** — الأوّلُ من الباب (`+963…`)، والآخران صفّان قديمان
	// لم يمرّا بتوحيده: أرقامٌ هنديّةٌ بمسافات، و`00963…`. **والجمعُ بالمفتاح لا بالمخزَّن.**
	mrSend(t, h, m, tok, sid, "أبو خالد", "0935111222", "العنوان الأول", true)
	mrSend(t, h, m, tok, sid, "سامر", "0966000111", "عنوان سامر", false)
	mrLegacyPhone(t, h, mrSend(t, h, m, tok, sid, "أبو خالد الحلبي", "0935111222", "العنوان الثاني", false), "٠٩٣٥ ١١١ ٢٢٢")
	mrLegacyPhone(t, h, mrSend(t, h, m, tok, sid, "خالد", "0935111222", "العنوان الأخير", true), "00963935111222")

	rows := mrRows(t, h, tok, sid, "")
	if got := mrPhones(rows); len(got) != 2 || got[0] != "+963935111222" || got[1] != "+963966000111" {
		t.Fatalf("**المتوقَّعُ صفّان والأحدثُ أوّلاً** (+963935111222 ثمّ +963966000111): %v", got)
	}
	k := rows[0]
	if k["name"] != "خالد" || k["address_text"] != "العنوان الأخير" {
		t.Fatalf("**الاسمُ والعنوانُ ليسا من آخر توصيلة**: %v", k)
	}
	if n, _ := k["times"].(float64); n != 3 {
		t.Fatalf("**عددُ التوصيلات %v والمنتظَرُ ٣ — الصيغُ لم تُوحَّد**", k["times"])
	}
	lat, _ := k["lat"].(float64)
	lng, _ := k["lng"].(float64)
	if fmt.Sprintf("%.5f,%.5f", lat, lng) != fmt.Sprintf("%.5f,%.5f", m.dropLat, m.dropLng) {
		t.Fatalf("**نقطةُ آخر توصيلة لم تُعطَ**: %v,%v والمنتظَر %v,%v", k["lat"], k["lng"], m.dropLat, m.dropLng)
	}
	if s, _ := k["last_used_at"].(string); s == "" {
		t.Fatalf("**بلا last_used_at**: %v", k)
	} else if at, err := time.Parse(time.RFC3339Nano, s); err != nil || time.Since(at) > time.Hour {
		t.Fatalf("**last_used_at لا يُقرأ وقتاً قريباً**: %q (%v)", s, err)
	}

	// ── والبحثُ بالرقم بأيّ صيغة — والأرقامُ الهنديّة ────────────────────
	for _, q := range []string{"0935", "٠٩٣٥١١", "+963935", "00963935", "935111", "١١١٢٢٢"} {
		got := mrPhones(mrRows(t, h, tok, sid, q))
		if len(got) != 1 || got[0] != "+963935111222" {
			t.Fatalf("**البحثُ بـ%q لم يجد الرقمَ وحدَه**: %v", q, got)
		}
	}
	if got := mrPhones(mrRows(t, h, tok, sid, "0977")); len(got) != 0 {
		t.Fatalf("**رقمٌ لا وجودَ له وجد**: %v", got)
	}

	// ── وثمانيةٌ على الأكثر ──────────────────────────────────────────────
	for i := 0; i < 8; i++ {
		mrSend(t, h, m, tok, sid, fmt.Sprintf("زبون %d", i), fmt.Sprintf("09880000%02d", i), "عنوان", false)
	}
	rows = mrRows(t, h, tok, sid, "")
	if len(rows) != 8 || rows[0]["phone"] != "+963988000007" {
		t.Fatalf("**المتوقَّعُ ثمانيةٌ أحدثُها أوّلاً**: %v", mrPhones(rows))
	}

	// ── وكلُّ حقلٍ يقرؤه التطبيقُ موجودٌ في الردّ (`CONTRACT`) ─────────────
	for _, f := range kotlinFields(t, "merchant/DeliveriesApi.kt", "DeliveryRecipient") {
		if _, ok := rows[0][f]; !ok {
			t.Fatalf("**حقلُ التطبيق %q غائبٌ عن الردّ**: %v", f, rows[0])
		}
	}
}

// TestMR03_PointOnlyWhenKnown **النقطةُ إن حُدّدت وحدَها — وإلّا لا نقطة.**
//
// **وبلا نقطةٍ يُكتب في `dropoff` موقعُ المتجر نفسِه** — ونقطةٌ كهذه لو أُعطيت لأرسلت السائقَ
// إلى المتجر بدل المستلِم.
func TestMR03_PointOnlyWhenKnown(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MR-03")
	tok, sid := m.fx.Tok, m.fx.M.ID

	mrSend(t, h, m, tok, sid, "وليد", "0933000001", "عند الجامع", true)
	mrSend(t, h, m, tok, sid, "وليد", "0933000001", "عند الجامع", false) // **الأخيرةُ بلا نقطة**
	mrSend(t, h, m, tok, sid, "رامي", "0933000002", "قرب المدرسة", false)
	mrSend(t, h, m, tok, sid, "هند", "0933000003", "الدوار", true)

	rows := mrRows(t, h, tok, sid, "")
	for _, c := range []struct {
		phone string
		point bool
	}{{"+963933000001", false}, {"+963933000002", false}, {"+963933000003", true}} {
		r := mrHas(rows, c.phone)
		if r == nil {
			t.Fatalf("%s غائب: %v", c.phone, mrPhones(rows))
		}
		has := r["lat"] != nil && r["lng"] != nil
		if has != c.point {
			t.Fatalf("**%s: نقطة=%v والمنتظَر %v** — %v", c.phone, has, c.point, r)
		}
	}
}

// TestMR04_NameAndDigitMatch **الاسمُ بلا همزةٍ ولا تاءٍ مربوطةٍ ولا تشكيل — والرقمُ جزءاً منه.**
func TestMR04_NameAndDigitMatch(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MR-04")
	tok, sid := m.fx.Tok, m.fx.M.ID

	mrSend(t, h, m, tok, sid, "أحمد الطويلة", "0944100100", "حي الرميلة", false)
	mrSend(t, h, m, tok, sid, "مصطفى", "0944200200", "المشلب", false)
	mrSend(t, h, m, tok, sid, "محمود", "0935300300", "الفردوس", false)

	cases := []struct {
		q    string
		want []string
	}{
		{"احمد", []string{"+963944100100"}},
		{"أحْمَد", []string{"+963944100100"}},
		{"إحمد", []string{"+963944100100"}},
		{"الطويله", []string{"+963944100100"}},
		{"مصطفي", []string{"+963944200200"}},
		{"مـصـطـفـى", []string{"+963944200200"}},
		{"محمود", []string{"+963935300300"}},
		{"سليم", nil},
		// **والرقمُ جزءاً منه** — `0944` لاثنين، و`300` لواحد.
		{"0944", []string{"+963944200200", "+963944100100"}},
		{"300300", []string{"+963935300300"}},
		// **و`%` و`_` حرفان لا أداتا مطابقة.**
		{"%", nil},
		{"_", nil},
	}
	for _, c := range cases {
		got := mrPhones(mrRows(t, h, tok, sid, c.q))
		if fmt.Sprint(got) != fmt.Sprint(append([]string{}, c.want...)) {
			t.Fatalf("**البحثُ بـ%q ردّ %v والمنتظَر %v**", c.q, got, c.want)
		}
	}
}

// TestMR05_HideAndReturn **«شيلو من القائمة» — ويرجع وحدَه إن بعث له من جديد.**
func TestMR05_HideAndReturn(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MR-05")
	tok, sid := m.fx.Tok, m.fx.M.ID
	base := "/api/v1/merchant/stores/" + sid + "/delivery-recipients/"

	mrSend(t, h, m, tok, sid, "غلط", "0955123456", "عنوان غلط", false)
	mrSend(t, h, m, tok, sid, "صح", "0955654321", "عنوان صح", false)

	// **يُخفى برقمه بأيّ صيغة** — هنا بالأرقام الهنديّة.
	if r := h.DEL(base+url.PathEscape("٠٩٥٥١٢٣٤٥٦"), tok); r.Code != http.StatusOK {
		t.Fatalf("الإخفاء: %d / %s", r.Code, r)
	}
	rows := mrRows(t, h, tok, sid, "")
	if mrHas(rows, "+963955123456") != nil || mrHas(rows, "+963955654321") == nil {
		t.Fatalf("**المُخفى ظاهرٌ أو غيرُه اختفى**: %v", mrPhones(rows))
	}
	if got := mrPhones(mrRows(t, h, tok, sid, "غلط")); len(got) != 0 {
		t.Fatalf("**المُخفى يظهر بالبحث**: %v", got)
	}
	// **ومرّةً ثانيةً بصيغة `+963` المرمَّزة** — يجدّد ولا يُكرَّر.
	if r := h.DEL(base+url.PathEscape("+963955123456"), tok); r.Code != http.StatusOK {
		t.Fatalf("الإخفاءُ الثاني: %d / %s", r.Code, r)
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM merchant_recipients_hidden WHERE merchant_id = $1`, sid).Scan(&n)
	if n != 1 {
		t.Fatalf("**صفوفُ الإخفاء %d والمنتظَرُ واحد**", n)
	}

	// **ورقمٌ لم يبعث له قطّ لا يُكتب · وما ليس رقماً يُردّ.**
	if r := h.DEL(base+"0999888777", tok); r.Code != http.StatusNotFound {
		t.Fatalf("**رقمٌ غريبٌ أُخفي**: %d / %s", r.Code, r)
	}
	if r := h.DEL(base+"12", tok); r.Code != http.StatusBadRequest || r.Err() != "invalid_phone" {
		t.Fatalf("**رقمٌ فاسدٌ لم يُردّ**: %d / %s", r.Code, r)
	}

	// ── يرجع وحدَه بتوصيلةٍ جديدة — باسمها الجديد وعددِه كاملاً ─────────────
	mrSend(t, h, m, tok, sid, "صار صح", "+963955123456", "العنوان الجديد", true)
	rows = mrRows(t, h, tok, sid, "")
	r := mrHas(rows, "+963955123456")
	if r == nil {
		t.Fatalf("**المُخفى لم يرجع بعد توصيلةٍ جديدة**: %v", mrPhones(rows))
	}
	if r["name"] != "صار صح" || r["address_text"] != "العنوان الجديد" || r["times"] != float64(2) {
		t.Fatalf("**رجع بغير آخر توصيلة**: %v", r)
	}
	if rows[0]["phone"] != "+963955123456" {
		t.Fatalf("**الراجعُ ليس أوّلاً وهو الأحدث**: %v", mrPhones(rows))
	}
}

// TestMR06_HideIsPerStore **إخفاءُ رقمٍ عند متجرٍ لا يُخفيه عند غيره** — ولو بعثا للرقم نفسِه.
func TestMR06_HideIsPerStore(t *testing.T) {
	h := New(t)
	a := newMDFx(t, h, "منطقةُ MR-06")
	b := a.f.Merchant()
	bTok := h.TokenFor(b.Owner.ID, "merchant")

	mrSend(t, h, a, a.fx.Tok, a.fx.M.ID, "زبون مشترك", "0977555444", "عند أ", false)
	mrSend(t, h, a, bTok, b.ID, "زبون مشترك", "0977555444", "عند ب", false)

	if r := h.DEL("/api/v1/merchant/stores/"+a.fx.M.ID+"/delivery-recipients/0977555444", a.fx.Tok); r.Code != http.StatusOK {
		t.Fatalf("الإخفاء عند أ: %d / %s", r.Code, r)
	}
	if rows := mrRows(t, h, a.fx.Tok, a.fx.M.ID, ""); mrHas(rows, "+963977555444") != nil {
		t.Fatalf("**لم يُخفَ عند أ**: %v", mrPhones(rows))
	}
	r := mrHas(mrRows(t, h, bTok, b.ID, ""), "+963977555444")
	if r == nil || r["address_text"] != "عند ب" {
		t.Fatalf("**إخفاءُ أ أخفاه عند ب**: %v", r)
	}
}
