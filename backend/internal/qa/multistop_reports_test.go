package qa

// ══════════════════════════════════════════════════════════════════════
// **التقاريرُ والعدّاداتُ لطلبٍ بعدّة محطّات — كلُّ متجرٍ بقسمه** (الدفعة P9)
// ══════════════════════════════════════════════════════════════════════

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

// TestMultiStop_RepClientDetailPerStore **مندوبُ المتجر الثاني يرى طلبَه ببضاعة متجره ونصيبه عنه** — `T41` (C8).
//
// **كان الطلبُ لا يظهر لمندوب المتجر الثاني أصلاً** — لا في تفصيل عميله ولا في عدّاداته —
// **ولمندوب الأوّل بمجموع الطلب كلّه.**
func TestMultiStop_RepClientDetailPerStore(t *testing.T) {
	s := newMSRepShop(t)
	oid := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, oid)
	var number int64
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT number FROM orders WHERE id = $1::uuid`, oid).Scan(&number); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		label string
		rep   *Rep
		m     *Merchant
		goods int64
		share int64
	}{{"الأوّل", s.r1, s.mA, 1000, 10}, {"الثاني", s.r2, s.mB, 2000, 40}} {
		r := s.h.GET("/api/v1/rep/merchants/"+c.m.ID, c.rep.Token)
		if r.Code != http.StatusOK {
			t.Fatalf("تفصيلُ العميل %s: %s", c.label, r)
		}
		j := r.JSON()
		var row map[string]any
		list, _ := j["orders"].([]any)
		for _, v := range list {
			if m, ok := v.(map[string]any); ok && num(m["number"]) == number {
				row = m
			}
		}
		if row == nil {
			t.Errorf("**المتجر %s**: الطلبُ غائبٌ عن تفصيل عميل مندوبه", c.label)
			continue
		}
		if num(row["total"]) != c.goods || num(row["subtotal"]) != c.goods || num(row["delivery_fee"]) != 0 {
			t.Errorf("**المتجر %s**: المجموع %v والبضاعة %v والتوصيل %v — يُنتظر بضاعتُه %d وتوصيلٌ صفر",
				c.label, row["total"], row["subtotal"], row["delivery_fee"], c.goods)
		}
		if num(row["my_share"]) != c.share {
			t.Errorf("**المتجر %s**: نصيبُ المندوب %v — يُنتظر %d", c.label, row["my_share"], c.share)
		}
		sum, _ := j["summary"].(map[string]any)
		if num(sum["delivered"]) != 1 || num(sum["delivered_sales"]) != c.goods || num(sum["my_earnings"]) != c.share {
			t.Errorf("**ملخّصُ المتجر %s** %v — يُنتظر مسلَّمٌ واحد ومبيعٌ %d وكسبٌ %d", c.label, sum, c.goods, c.share)
		}

		// **وقائمةُ عملائه وعدّاداتُه.**
		lr := s.h.GET("/api/v1/rep/merchants", c.rep.Token)
		if lr.Code != http.StatusOK {
			t.Fatalf("قائمةُ العملاء: %s", lr)
		}
		var card map[string]any
		var env struct {
			Data []any `json:"data"`
		}
		if err := json.Unmarshal(lr.Body, &env); err != nil {
			t.Fatalf("قائمةُ العملاء ليست مصفوفة: %v", err)
		}
		for _, v := range env.Data {
			if m, ok := v.(map[string]any); ok && m["id"] == c.m.ID {
				card = m
			}
		}
		if card == nil {
			t.Fatalf("المتجر %s غائبٌ عن قائمة عملاء مندوبه: %s", c.label, lr)
		}
		if num(card["delivered_orders"]) != 1 || num(card["my_commission"]) != c.share {
			t.Errorf("**بطاقةُ المتجر %s** %v — يُنتظر طلبٌ مسلَّم وعمولةٌ %d", c.label, card, c.share)
		}
		me := s.h.GET("/api/v1/rep/me", c.rep.Token)
		if me.Code != http.StatusOK || num(me.JSON()["delivered_orders"]) != 1 {
			t.Errorf("**لوحةُ مندوب المتجر %s** تعدّ %v طلباً مسلَّماً — يُنتظر 1: %s", c.label,
				me.JSON()["delivered_orders"], me)
		}
	}
}

// TestMultiStop_ExportStoreColumnAndFilter **ملفُّ الطلبات: «أ + ب» في عمود المتجر، وفلترُ المتجر
// الثاني يجده** (§10).
func TestMultiStop_ExportStoreColumnAndFilter(t *testing.T) {
	s := newMSRepShop(t)
	oid := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, oid)
	var number int64
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT number FROM orders WHERE id = $1::uuid`, oid).Scan(&number); err != nil {
		t.Fatal(err)
	}
	r := s.h.GET("/api/v1/admin/orders/export?merchant_id="+s.mB.ID, s.admin.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("التصدير: %s", r)
	}
	recs, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(r.Body), "\ufeff"))).ReadAll()
	if err != nil || len(recs) == 0 {
		t.Fatalf("الملفّ: %v", err)
	}
	col := -1
	for i, h := range recs[0] {
		if h == "المتجر" {
			col = i
		}
	}
	var store string
	found := false
	for _, rec := range recs[1:] {
		if rec[0] == strconv.FormatInt(number, 10) {
			found, store = true, rec[col]
		}
	}
	if !found {
		t.Fatalf("**فلترُ المتجر الثاني لا يجد الطلب** — %d سطراً", len(recs)-1)
	}
	if want := s.mA.Name + " + " + s.mB.Name; store != want {
		t.Errorf("**عمودُ المتجر** %q — يُنتظر %q", store, want)
	}
}

// TestMultiStop_SecondStoreInMapRatingAndEmergencyRoom **المتجرُ الثاني في الخريطة وغرفة الطوارئ،
// ومندوبُه لا يقيّم طلباً اشتراه منه** (§11).
//
// **كانت كلُّها تقرأ المتجرَ الأوّلَ وحدَه**: الثاني بلا طلبٍ جارٍ على الخريطة، وإغلاقُه الطارئُ
// لا يُري المكتبَ الطلبَ الذي ينتظر بضاعتَه، ومندوبُه يقيّم متجرَه بطلبٍ اشتراه.
func TestMultiStop_SecondStoreInMapRatingAndEmergencyRoom(t *testing.T) {
	s := newMSRepShop(t)
	oid := s.placeAccepted(t, "")

	// ١ · الخريطة: طلبٌ جارٍ على المتجر الثاني، وفلترُه يجده.
	ms, err := opsmap.Merchants(ctxBG(), s.h.Pool, nil, opsmap.MerchantFilter{Search: s.mB.Name}, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	var active = -1
	for _, m := range ms {
		if m.ID == s.mB.ID {
			active = m.ActiveN
		}
	}
	if active != 1 {
		t.Errorf("**الخريطة**: المتجرُ الثاني عليه %d طلباً جارياً — يُنتظر 1", active)
	}
	os, err := opsmap.ActiveOrders(ctxBG(), s.h.Pool, nil, opsmap.OrderFilter{MerchantID: s.mB.ID}, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range os {
		found = found || o.ID == oid
	}
	if !found {
		t.Errorf("**الخريطة**: فلترُ المتجر الثاني لا يجد طلبَه الجاري")
	}

	// ٢ · غرفةُ الطوارئ: إغلاقُ المتجر الثاني يُري الطلبَ الذي لم تُستلم بضاعتُه منه.
	var eid string
	if err := s.h.Pool.QueryRow(ctxBG(), `
		INSERT INTO driver_emergencies (kind, merchant_id) VALUES ('store_closure', $1::uuid)
		RETURNING id::text`, s.mB.ID).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = s.h.Pool.Exec(ctxBG(), `DELETE FROM driver_emergencies WHERE id = $1::uuid`, eid) })
	d := s.h.GET("/api/v1/admin/emergencies/"+eid, s.admin.Token)
	if d.Code != http.StatusOK {
		t.Fatalf("صفحةُ الطارئ: %s", d)
	}
	list, _ := d.JSON()["store_orders"].([]any)
	inRoom := false
	for _, v := range list {
		if m, ok := v.(map[string]any); ok && m["id"] == oid {
			inRoom = true
		}
	}
	if !inRoom {
		t.Errorf("**غرفةُ الطوارئ**: طلبُ المتجر الثاني غائبٌ عن طلباته المفتوحة: %s", d)
	}

	// ٣ · التقييم: مندوبُ المتجر الثاني يشتري من المتجرين ثمّ يقيّم ⇒ مرفوض.
	buyer := *s.r2.User
	buyer.Token = s.h.TokenFor(buyer.ID, "customer", "sales")
	s.cust = &buyer
	bought := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, bought)
	r := s.h.POST("/api/v1/orders/"+bought+"/rating", buyer.Token, map[string]any{"platform_stars": 5})
	if r.Code != http.StatusForbidden || r.Err() != "rate_own_client" {
		t.Errorf("**مندوبُ المتجر الثاني يقيّم طلباً اشتراه منه**: %s — يُنتظر 403 rate_own_client", r)
	}
}
