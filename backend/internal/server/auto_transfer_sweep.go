package server

import (
	"context"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **تقدّمٌ لازمٌ لا يعتمد على خيطٍ يموت** — `PF-08` · `R21` · `XOB-7`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **`go autoTransfer(...)` بعد تثبيت الطلب** — **والخيطُ يموت بموت
// العمليّة.** فإن سقط الخادمُ بين التثبيت والإنزال بقي الطلبُ حيث
// وقف: **مقبولٌ بلا سائقٍ يُعرَض عليه، ولا أحدَ يعلم.**
//
// **و`sweepAutoAccept` يلتقط ما بقي `pending`** — **ولا يلتقط ما بلغ
// `accepted` ولم يُنزَل.**
//
// # ومصدرُ العمل حالُ الطلب نفسُها
//
// **ولا جدولَ ثانٍ**: **حقيقةُ الطلب تقول كلَّ شيء** — `accepted` بلا
// سائقٍ وغيرُ مغلقٍ **هو عملٌ معلَّقٌ بذاته.** **وجدولُ عملٍ ثانٍ
// حقيقةٌ ثانيةٌ تشيخ وتُناقض الأولى.**
//
// # والخيطُ السريعُ يبقى
//
// **لا يُحذَف بل يُنزَع عنه الاعتماد**: **تعجيلٌ لا مصدرَ حقيقة.**
// **ومن حذفه أبطأ كلَّ طلبٍ ثلاثين ثانيةً بلا سبب.**

// autoTransferSweepAge **كم يُترَك الطلبُ قبل أن يُعَدَّ عالقاً.**
//
// **والخيطُ السريعُ يُنجزه في أجزاء الثانية** — **فدقيقةٌ حدٌّ لا
// يبلغه إلّا من مات خيطُه.** **وأقصرُ منها يسابق الخيطَ فيتنازعان،
// وأطولُ يترك الزبونَ ينتظر بلا سائق.**
const autoTransferSweepAge = time.Minute

// RunAutoTransferSweeper يستأنف التقدّمَ اللازمَ للطلبات العالقة.
//
// **ويُشغَّل مع الراصد** — ودورتُه دورتُه، **فلا تُخترَع مهلةٌ ثانية.**
func (s *Server) RunAutoTransferSweeper(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.SweepAutoTransfer(ctx)
		}
	}
}

// SweepAutoTransfer جولةٌ واحدة — **ويُنادى من الفحص مباشرةً.**
//
// # الشروط
//
//	`accepted` · بلا سائق · غيرُ مغلق · مضى عليه حدٌّ · والإعدادُ فعّال
//
// **والحالُ النهائيّةُ ليست منها** — **ولا يُبعَث ميت.**
//
// # والقفل
//
// **`FOR UPDATE SKIP LOCKED`** — **فكانسان يقتسمان العملَ ولا
// يتنازعان عليه**، ولا قفلَ موزَّعٌ يُخترَع.
//
// # والإتمام
//
// **حالُ الطلب هي الإتمام**: **بلغ `dispatching` فخرج من الشرط.**
// **ولا يُوسَم منتهياً قبل أن يقع عملُه** — **ومن وسم ثمّ سقط فقد
// العملَ إلى الأبد.**
//
// # وإعادةُ المحاولة
//
// **الدورةُ التالية** — **ولا حلقةَ ساخنة**: من بقي عالقاً يُلتقَط
// بعد ثلاثين ثانيةً لا بعد ميلي.
func (s *Server) SweepAutoTransfer(ctx context.Context) (picked, progressed int) {
	if !s.settings.GetBool(ctx, "orders.auto_transfer") {
		return 0, 0
	}
	rows, err := s.pg.Query(ctx, `
		SELECT id::text FROM orders
		 WHERE status = 'accepted'
		   AND driver_id IS NULL
		   AND closed_at IS NULL
		   AND updated_at < now() - $1::interval
		 ORDER BY updated_at
		 LIMIT 50
		 FOR UPDATE SKIP LOCKED`, autoTransferSweepAge.String())
	if err != nil {
		s.logger.Warn("كنسُ التحويل: تعذّرت القراءة", "error", err)
		return 0, 0
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			s.logger.Warn("كنسُ التحويل: تعذّر المسح", "error", err)
			return 0, 0
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) == 0 {
		return 0, 0
	}

	// ══════════════════════════════════════════════════════════════
	// **وأثرٌ تشغيليٌّ يُقرأ**
	// ══════════════════════════════════════════════════════════════
	//
	// **ولا منظومةَ قياسٍ تُبنى** — **سطرٌ يقول كم عَلِق وكم عمرُ
	// أقدمِه يكفي من أراد أن يعرف.** **وعملٌ لا يُرى عالقٌ لا يُصلَح.**
	var oldest float64
	_ = s.pg.QueryRow(ctx, `
		SELECT COALESCE(EXTRACT(epoch FROM now() - min(updated_at)), 0)
		  FROM orders
		 WHERE status = 'accepted' AND driver_id IS NULL AND closed_at IS NULL
		   AND updated_at < now() - $1::interval`,
		autoTransferSweepAge.String()).Scan(&oldest)
	s.logger.Info("كنسُ التحويل: طلباتٌ عالقة",
		"count", len(ids), "oldest_seconds", int64(oldest))

	picked = len(ids)
	for _, id := range ids {
		// **ويُستأنَف بالمسار نفسِه لا بنسخةٍ ثانية** — **وتحوّلٌ
		// مكتوبٌ بيدٍ يفقد الحدثَ والتدقيقَ والإشعارَ والبثّ.**
		//
		// **و`autoTransfer` تفحص الحالَ بنفسها فتصير عمليّاً مرّةً
		// واحدةً وإن نُوديت مرّتين**: من بلغ `dispatching` خرج.
		if s.resumeAutoTransfer(ctx, id) {
			progressed++
		}
	}
	return picked, progressed
}

// resumeAutoTransfer يستأنف طلباً بلغ `accepted` ولم يُنزَل.
//
// **ولا يُعاد قبولُه** — **هو مقبولٌ فعلاً**، وإنّما يُنزَل.
func (s *Server) resumeAutoTransfer(ctx context.Context, orderID string) bool {
	if err := s.orders.AutoDispatch(ctx, "", orderID); err != nil {
		// **ولا يُبتلَع الخطأ صامتاً** — **وطلبٌ يفشل إنزالُه مرّةً
		// بعد مرّةٍ يجب أن يُقرأ في السجلّ.**
		s.logger.Error("كنسُ التحويل: تعثّر الإنزال", "order", orderID, "error", err)
		return false
	}
	s.touch("order", "ops")
	return true
}
