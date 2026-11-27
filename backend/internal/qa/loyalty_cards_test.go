package qa

// **بطاقة رحّال غو للمتجر** (قرارُ المالك ٢٠٢٦-١١-٢٧) — «لدي توصيلة» المسلَّمةُ تختم بطاقةَ المتجر
// وحدَها (لا بطاقةَ صاحبه زبوناً)، والتي بعد امتلائها مجانيّةٌ والكابتنُ بأجره كاملاً والخزينةُ
// تتحمّل، **ومتجرٌ آخرُ لا يرى بطاقتَه.**

import (
	"fmt"
	"net/http"
	"testing"
)

func TestLoyalty_StoreCardSeparateAndGuarded(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ بطاقة رحّال غو")
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE free_delivery_campaigns SET active = false`)
	h.Setting("stamp_card.enabled", "true")
	h.Setting("stamp_card.size", "2")
	h.Setting("daily_card.enabled", "false")

	// ── متجرٌ آخرُ لا يرى بطاقتَه ─────────────────────────────────────
	other := newOfferFx(t, h, m.f, 1000)
	if r := h.GET(m.url("/stamp-card"), other.Tok); r.Code != http.StatusForbidden {
		t.Fatalf("متجرٌ آخرُ قرأ البطاقة: %d — والمنتظَر 403", r.Code)
	}

	base := mdaBaseline(t, h)
	for i := 0; i < 2; i++ {
		id, drv := mdmCreate(t, h, m, "recipient", "", fmt.Sprintf("loy%d", i))
		mdDriveTo(t, h, id, drv, "at_dropoff")
		mdmDeliver(t, h, id, drv)
	}
	card := h.GET(m.url("/stamp-card"), m.fx.Tok)
	if card.Code != http.StatusOK || card.JSON()["stamps"] != float64(2) || card.JSON()["free_ready"] != true {
		t.Fatalf("بطاقةُ المتجر بعد توصيلتين: %s", card)
	}
	// **وبطاقةُ صاحبه زبوناً لم تُختم.**
	if mine := h.GET("/api/v1/my/stamp-card", m.fx.Tok); mine.Code != http.StatusOK || mine.JSON()["stamps"] != float64(0) {
		t.Fatalf("بطاقةُ الزبون خُتمت بتوصيلات المتجر: %s", mine)
	}
	// ── المتجرُ يرى الأجرةَ مجانيّةً قبل الطلب ─────────────────────────
	q := h.GET(fmt.Sprintf("%s?lat=%f&lng=%f", m.url("/delivery-quote"), m.dropLat, m.dropLng), m.fx.Tok)
	if q.JSON()["fee"] != float64(0) || q.JSON()["free_waived"] != float64(500) || q.JSON()["free_source"] != "stamp" {
		t.Fatalf("التسعيرة: %s", q)
	}
	// ── الثالثةُ مجانيّة: الكابتنُ ٤٥٠ والخزينةُ −٤٥٠ ──────────────────────
	id, drv := mdmCreate(t, h, m, "recipient", "", "loy-free")
	var fee, cash, driverFee, waived int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee, cash_due, driver_fee, promo_delivery_waived
		FROM orders WHERE id = $1`, id).Scan(&fee, &cash, &driverFee, &waived); err != nil {
		t.Fatal(err)
	}
	if fee != 0 || cash != 0 || waived != 500 || driverFee != 450 {
		t.Fatalf("الجائزة: أجرة %d · نقد %d · معفى %d · السائق %d — والمنتظَر 0 · 0 · 500 · 450",
			fee, cash, waived, driverFee)
	}
	before := walletOf(t, h, drv.ID)
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)
	if got := walletOf(t, h, drv.ID) - before; got != 450 {
		t.Fatalf("أجرُ الكابتن %d والمنتظَر 450 — بأجره كاملاً", got)
	}
	if got := mdmProfit(t, h, id); got != -450 {
		t.Fatalf("**الخزينةُ %d والمنتظَر −450** — هي من تحمّل الجائزة", got)
	}
	mdaNoNewViolations(t, h, base)
	if after := h.GET(m.url("/stamp-card"), m.fx.Tok); after.JSON()["stamps"] != float64(0) || after.JSON()["free_ready"] != false {
		t.Fatalf("بعد الجائزة: %s — والمنتظَر بطاقةٌ فارغة", after)
	}
}
