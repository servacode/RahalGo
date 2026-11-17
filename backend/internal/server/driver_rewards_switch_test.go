package server

// ══════════════════════════════════════════════════════════════════════
// **مفتاحُ «مكافآت السائقين» الواحد** — قرارُ المالك ٢٠٢٦-١١-١٨
// ══════════════════════════════════════════════════════════════════════
//
// مطفأٌ ⇒ لا مكافأةَ هدفٍ آليّة، ولا طلبَ مكافأةٍ يدويّ (اقتراحاً أو موافقة)، وشاشةُ السائق
// بلا وعد. **والعقوبةُ تبقى**، **والمندوبُ لا يتأثّر.**

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestDriverRewardsOff_TargetNotPaidAndScreenSilent **الهدفُ الآليّ لا يُصرف والشاشةُ صفر — ثمّ
// يعود المفتاحُ فيُصرف كما ضُبط.**
func TestDriverRewardsOff_TargetNotPaidAndScreenSilent(t *testing.T) {
	f := newIncFixture(t)
	ctx := context.Background()
	f.setTargets(t, map[string]string{
		"drivers.monthly_target": "1", "drivers.target_reward": "1000",
		"drivers.target_2": "0", "drivers.reward_2": "0", "drivers.target_3": "0", "drivers.reward_3": "0",
		"drivers.rewards_enabled": "false",
	})
	driver := f.kept(t, "driver")
	f.deliveredOrder(t, driver)

	paid, err := f.srv.incentives.GrantTargetIfReachedTx(ctx, f.pool, driver, "driver")
	if err != nil || paid != 0 || f.targetRows(t, driver) != 0 {
		t.Fatalf("**صُرفت مكافأةُ الهدف والمفتاحُ مطفأ**: %d (%v) صفوف=%d", paid, err, f.targetRows(t, driver))
	}
	if p := f.srv.incentives.SweepMonth(ctx, "driver"); len(p) != 0 {
		t.Fatalf("الجولةُ صرفت والمفتاحُ مطفأ: %+v", p)
	}
	w := f.call(driver, http.MethodGet, "/me/incentives", nil, ``, f.srv.handleMyIncentives)
	if w.Code != http.StatusOK {
		t.Fatalf("شاشتي: %d %s", w.Code, w.Body.String())
	}
	var res struct {
		Data struct {
			TargetReward int64 `json:"target_reward"`
			Levels       []struct {
				Reward int64 `json:"reward"`
			} `json:"levels"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Data.TargetReward != 0 {
		t.Fatalf("**الشاشةُ تعده بـ%d والمكافآتُ موقّفة**", res.Data.TargetReward)
	}
	for _, l := range res.Data.Levels {
		if l.Reward != 0 {
			t.Fatalf("مرحلةٌ بمكافأة %d والمكافآتُ موقّفة: %s", l.Reward, w.Body.String())
		}
	}

	// **ويعود المفتاحُ فيُصرف ما ضُبط** — لم يُمسّ رقمُ المرحلة.
	f.setTargets(t, map[string]string{"drivers.rewards_enabled": "true"})
	paid, err = f.srv.incentives.GrantTargetIfReachedTx(ctx, f.pool, driver, "driver")
	if err != nil || paid != 1000 {
		t.Fatalf("بعد التشغيل: %d (%v) — والمنتظَر 1000", paid, err)
	}
}

// TestDriverRewardsOff_ManualRewardRefusedPenaltyKept **طلبُ المكافأة يُردّ عند الاقتراح وعند
// الموافقة، والعقوبةُ تمرّ، ومكافأةُ المندوب تمرّ.**
func TestDriverRewardsOff_ManualRewardRefusedPenaltyKept(t *testing.T) {
	f := newIncFixture(t)
	driver := f.kept(t, "driver")
	rep := f.kept(t, "sales")
	propose := func(uid, kind string) (int, string, string) {
		w := f.call(f.proposer, http.MethodPost, "/admin/users/"+uid+"/incentive",
			map[string]string{"id": uid},
			`{"kind":"`+kind+`","amount":500,"reason":"اختبار المفتاح"}`, f.srv.handleIncentiveGrant)
		var res struct {
			Data struct {
				RequestID string `json:"request_id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		return w.Code, w.Body.String(), res.Data.RequestID
	}

	// **اقتُرحت والمفتاحُ شغّال، ثمّ أُطفئ قبل الموافقة.**
	f.setTargets(t, map[string]string{"drivers.rewards_enabled": "true"})
	code, body, reqID := propose(driver, "reward")
	if code != http.StatusCreated || reqID == "" {
		t.Fatalf("الاقتراحُ والمفتاحُ شغّال: %d %s", code, body)
	}
	f.setTargets(t, map[string]string{"drivers.rewards_enabled": "false"})

	code, body, _ = propose(driver, "reward")
	if code != http.StatusConflict || !strings.Contains(body, "driver_rewards_off") {
		t.Fatalf("**قُبل اقتراحُ مكافأةٍ والمكافآتُ موقّفة**: %d %s", code, body)
	}
	ok := f.call(f.approver, http.MethodPost, "/admin/incentive-requests/x/approve",
		map[string]string{"id": reqID}, `{}`, f.srv.handleDecideIncentiveRequest(true))
	if ok.Code != http.StatusConflict || !strings.Contains(ok.Body.String(), "driver_rewards_off") {
		t.Fatalf("**وُوفق على مكافأةٍ والمكافآتُ موقّفة**: %d %s", ok.Code, ok.Body.String())
	}
	if b := f.balance(t, driver); b != 0 {
		t.Fatalf("تحرّك رصيدُ السائق %d", b)
	}

	// **والعقوبةُ لا تتأثّر، ولا مكافأةُ المندوب.**
	if code, body, _ := propose(driver, "penalty"); code != http.StatusCreated {
		t.Fatalf("رُدّت العقوبةُ والمفتاحُ للمكافآت وحدَها: %d %s", code, body)
	}
	if code, body, _ := propose(rep, "reward"); code != http.StatusCreated {
		t.Fatalf("رُدّت مكافأةُ المندوب والمفتاحُ للسائقين وحدَهم: %d %s", code, body)
	}
}
