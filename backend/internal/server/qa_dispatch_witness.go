package server

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **شاهدُ التوزيعِ الحيّ — عرضٌ حقيقيٌّ يصل جهازَ السائق** (E، ٢٠٢٦-٠٩-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك: «شاهدُ توزيعٍ حقيقيٌّ على التجهيز وحدَه — **بلا التفافٍ على
//  المصادقة، بلا سكِّ توكن أدمن، بلا حقنِ FCM مزيّف، بلا حالةِ سائقٍ وهميّةٍ
//  في الواجهة، وبالمحرّك الحقيقيّ، ومستحيلٌ في الإنتاج، ونظيفٌ عكوسٌ، وبلا
//  تلويثٍ للدفتر**».)
//
// # المفقودُ الوحيدُ — والباقي محرّكٌ قائم
//
// **الانتقالُ إلى `dispatching` يُطلق العرضَ الحقيقيَّ أصلاً**
// (`transitions.go` ⇒ `OfferNext` ⇒ `notifyOffer` ⇒ `FCM.Send`)، **وبذّارُ
// `order_advance` (هدفُه `dispatching`) على طلبِ زبون QA المخصّصِ النقديّ
// يسوق هذا الانتقالَ بعينِه.** فما ينقص شيءٌ واحدٌ: **أن يكون سائقُ QA
// الثابتُ في ورديّةٍ منتِجةٍ فيصله العرض** — وهذا ما يفعله هذا البذّار.
//
// # لماذا وضعُ «بالتساوي» مؤقّتاً
//
// **البثُّ في وضع «بالتساوي» يُرسل العرضَ إلى كلّ من في الورديّة** — **فيصل
// جهازَ سائقِ QA حتماً** بلا اعتمادٍ على ترتيبِ الدور. **عكوسٌ**: يُرجَع
// الوضعُ السابقُ في نداءِ الإطفاء. **وليست حالةً وهميّة**: `on_shift` هو
// العمودُ نفسُه الذي ترفعه نقطةُ الورديّة الحقيقيّة — **حالةُ توفّرٍ فعليّةٌ لا
// خدعةُ واجهة.**
//
// # لا أثرَ ماليّ
//
// **العرضُ يكتب `offered_driver_id`/`offer_expires_at` على `orders` فقط**
// (أو لا شيءَ في وضع البثّ) — **صفرُ قيدِ محفظةٍ أو صندوقٍ أو خزينة.**
// والطلبُ يبقى مخصّصاً نقديّاً (لا يُسوّى) فلا يمسّ الدفترَ. **وعلى التجهيز
// وحدَه** — الحارسُ في المنادي (`qaStagingEnabled` ⇒ ٤٠٤ في الإنتاج).

// qaDriverShift **يضع سائقَ QA الثابتَ في ورديّةٍ منتِجةٍ (أو يطفئها)** ويضبط
// وضعَ التوزيعَ ليصلَه العرض — عكوسٌ.
//
//	on=true  : on_shift + active + بدايةٌ الآن + لا إسنادَ سابق، ووضعٌ «بالتساوي».
//	on=false : إطفاءُ الورديّة، واستعادةُ الوضعِ السابقِ إن مُرّر في `restoreMode`
//	           (لا تخمين — قيمةٌ صريحةٌ صحيحةٌ فقط).
func (s *Server) qaDriverShift(w http.ResponseWriter, r *http.Request, on bool, restoreMode string) {
	ctx := r.Context()
	// **هويّةُ سائق QA الثابتة** — المسارُ الحقيقيّ، بلا توكن، بلا جلسة.
	uid, err := s.qaFixedUser(ctx, qaStagingDriverPhone, "driver", "سائق الاختبار QA", clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	const modeKey = "drivers.assignment_mode"

	var prevOnShift bool
	if err := s.pg.QueryRow(ctx,
		`SELECT on_shift FROM users WHERE id = $1::uuid`, uid).Scan(&prevOnShift); err != nil {
		s.respondErr(w, err)
		return
	}
	prevMode := s.settings.GetString(ctx, modeKey)

	if on {
		// **مرشّحٌ منتِجٌ حتميّاً**: هذه بعينُها شروطُ `OfferNext`/البثّ
		// (on_shift + status='active')، **و`last_assigned_at=NULL` يجعله في
		// مقدّمة الدور** لو كان الوضعُ «بالتناوب» رغم أنّنا نبثّ.
		if _, err := s.pg.Exec(ctx,
			`UPDATE users SET on_shift = true, status = 'active',
			        shift_started_at = now(), last_assigned_at = NULL
			  WHERE id = $1::uuid`, uid); err != nil {
			s.respondErr(w, err)
			return
		}
		// **بثٌّ لضمانِ الوصول** — عكوسٌ، ويُرجَع السابقُ للاستعادة.
		if err := s.settings.Set(ctx, modeKey, "queue", nil); err != nil {
			s.respondErr(w, err)
			return
		}
		s.logger.Warn("QA driver_shift ON (staging-only)",
			"driver", uid, "previous_on_shift", prevOnShift, "previous_mode", prevMode)
		httpx.JSON(w, http.StatusOK, map[string]any{
			"driver_id": uid, "on_shift": true, "assignment_mode": "queue",
			"previous_on_shift": prevOnShift, "previous_mode": prevMode,
		})
		return
	}

	// **إطفاء** — والوضعُ يُستعاد بقيمةٍ صريحةٍ صحيحةٍ فقط، لا تخمين.
	if _, err := s.pg.Exec(ctx,
		`UPDATE users SET on_shift = false WHERE id = $1::uuid`, uid); err != nil {
		s.respondErr(w, err)
		return
	}
	restored := prevMode
	if restoreMode == "queue" || restoreMode == "rotation" {
		if err := s.settings.Set(ctx, modeKey, restoreMode, nil); err != nil {
			s.respondErr(w, err)
			return
		}
		restored = restoreMode
	}
	s.logger.Warn("QA driver_shift OFF (staging-only)", "driver", uid, "restored_mode", restored)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"driver_id": uid, "on_shift": false, "assignment_mode": restored,
	})
}
