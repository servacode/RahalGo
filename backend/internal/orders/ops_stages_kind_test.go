package orders_test

// **مسارُ المكتب بنوع الطلب** (ملاحظةُ المالك ٢٠٢٦-١٠-١٣ على #1003): الأمانةُ «استلام» لا
// «شراء»، وأجرةُ المنصّة بلا «توثيق»، و«لدي توصيلة» بلا «تجهيزٍ في المتجر».

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func has(list []orders.Stage, st orders.Stage) bool {
	for _, s := range list {
		if s == st {
			return true
		}
	}
	return false
}

func TestOpsStages_PerKind(t *testing.T) {
	list, at := orders.OpsStagesOf(orders.KindCustom, "assigned", false, orders.CustomModeAmanah, true)
	if has(list, orders.StageAgreeing) || has(list, orders.StageBuying) || !has(list, orders.StageTaking) {
		t.Fatalf("أمانةٌ بأجرة المنصّة ⇒ %v — والمنتظَرُ «استلام» بلا «توثيق» ولا «شراء»", list)
	}
	if at < 0 || list[at] != orders.StageTaking {
		t.Fatalf("أمانةٌ مُسنَدة عند %d (%v) — والمنتظَرُ «استلام»", at, list)
	}

	list, at = orders.OpsStagesOf(orders.KindCustom, "assigned", false, orders.CustomModePurchase, true)
	if has(list, orders.StageAgreeing) || !has(list, orders.StageBuying) || list[at] != orders.StageBuying {
		t.Fatalf("مشترياتٌ بأجرة المنصّة ⇒ %v عند %d", list, at)
	}

	list, at = orders.OpsStagesOf(orders.KindCustom, "assigned", false, orders.CustomModePurchase, false)
	if !has(list, orders.StageAgreeing) || list[at] != orders.StageAgreeing {
		t.Fatalf("مشترياتٌ بأجرة السائق قبل التوثيق ⇒ %v عند %d", list, at)
	}

	list, at = orders.OpsStagesOf(orders.KindMerchantDelivery, "accepted", false, "", false)
	if has(list, orders.StagePreparing) || list[at] != orders.StageSeekingDriver {
		t.Fatalf("«لدي توصيلة» ⇒ %v عند %d — والمنتظَرُ بلا «تجهيز»", list, at)
	}

	list, _ = orders.OpsStagesOf(orders.KindStandard, "preparing", false, "", false)
	if !has(list, orders.StagePreparing) {
		t.Fatalf("العاديُّ فقد «قيد التجهيز»: %v", list)
	}
}
