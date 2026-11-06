package orders

import (
	"context"
	"testing"
	"time"
)

type legRouter struct{ calls int }

func (r *legRouter) Seconds(_ context.Context, fromLat, _, toLat, _ float64) int {
	r.calls++
	return int((toLat - fromLat) * 1000)
}

// TestMultiStop_ToStoreETAAndLimit **زمنُ الطريق إلى المتاجر مجموعُ سيقانه، ومهلتُه تزيد بكلّ محطّة**
// — `T37` (D12): **وساقٌ لم تُقَس تُسقط القياس**، **وطلبُ المتجر الواحد لا يُمسّ.**
func TestMultiStop_ToStoreETAAndLimit(t *testing.T) {
	r := &legRouter{}
	pts := []legPoint{{lat: 0}, {lat: 0.3}, {lat: 0.5}}
	if got := sumLegSeconds(context.Background(), r, pts); got != 500 || r.calls != 2 {
		t.Errorf("مجموعُ السيقان %d بنداءات %d — يُنتظر ٥٠٠ بنداءين", got, r.calls)
	}
	if got := sumLegSeconds(context.Background(), r, []legPoint{{lat: 0.5}, {lat: 0.2}}); got != 0 {
		t.Errorf("ساقٌ بلا قياسٍ أعطت %d — يُنتظر صفر", got)
	}

	eta := 600
	evs := []Event{{ToStatus: StAssigned, CreatedAt: time.Now().Add(-15 * time.Minute)},
		{ToStatus: StPickedUp, CreatedAt: time.Now()}}
	lim := StageLimits{Handover: 10, RouteMarginPct: 0}
	single := &Order{Kind: KindStandard, Status: StPickedUp, Stops: 1, ToStoreETASec: &eta}
	single.SetStage()
	multi := &Order{Kind: KindStandard, Status: StPickedUp, Stops: 2, ToStoreETASec: &eta,
		Parts: []OpsPart{{Status: PartAccepted}, {Status: PartAccepted}}}
	multi.SetStage()
	if got := multi.stopsStageExtra(10); got != 10*time.Minute {
		t.Errorf("زيادةُ المحطّة %v — يُنتظر ١٠ دقائق", got)
	}
	removed := &Order{Stops: 2, Parts: []OpsPart{{Status: PartAccepted}, {Status: PartRemoved}}}
	if got := removed.stopsStageExtra(10); got != 0 {
		t.Errorf("محطّةٌ انشالت زادت المهلة %v", got)
	}
	if got := single.stopsStageExtra(10); got != 0 {
		t.Errorf("**طلبُ المتجر الواحد زادت مهلتُه** %v", got)
	}
	single.SetStageTimes(evs, lim)
	multi.SetStageTimes(evs, lim)
	late := func(o *Order) *bool {
		for i, st := range o.OpsStages {
			if st == StageOnTheWay && i < len(o.OpsStageLate) {
				return o.OpsStageLate[i]
			}
		}
		return nil
	}
	if l := late(single); l == nil || !*l {
		t.Errorf("المتجرُ الواحد بعد ١٥ دقيقةً من ١٠ — يُنتظر متأخّراً: %v", l)
	}
	if l := late(multi); l == nil || *l {
		t.Errorf("المحطّتان بعد ١٥ دقيقةً من ١٠+١٠ — يُنتظر غيرَ متأخّر: %v", l)
	}
}
