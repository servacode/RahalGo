package server

// ══════════════════════════════════════════════════════════════════════
//  **سلسلةُ تعريف المتجر بالإشعارات** (طلبُ المالك ٢٠٢٦-١٠-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// «بدي توصلهم بشكل تلقائي» — **كلُّ صاحبِ متجرٍ** يصله إشعاران في اليوم،
// **الساعةَ ١١ والرابعةَ بدمشق**، حتّى تنتهي القائمة:
//
//   - **الموجودون يوم تُشغَّل الميزة** يبدؤون في الموعد التالي.
//   - **ومن فُتح حسابه بعدها** يبدأ في الموعد التالي لفتحه.
//   - **والإشعارُ الموسومُ «إذا ما جرّب»** يُتخطّى عمّن طلب «لدي توصيلة».
//
// **وموعدٌ واحدٌ لا يحمل إشعارين**: `last_slot` يُحجز قبل الإرسال بشرطٍ
// (`WHERE last_slot <> slot`) — **فنبضتان متداخلتان لا ترسلان مرّتين.**
//
// **والعباراتُ من الإعدادات** (`merchants.onboarding_push_messages`) — يعدّلها
// المالكُ من اللوحة بلا نشر. **والمفتاحُ** `merchants.onboarding_push` يطفئها.

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// onboardingHours **موعدا الإرسال بتوقيت دمشق** — اختيارُ المالك: «بـ11 وبعد»، والعصر.
var onboardingHours = []int{11, 16}

type onboardingMsg struct {
	Title, Body  string
	OnlyIfUnused bool
}

// parseOnboarding سطرٌ لكلّ إشعار: «العنوان | النص [| إذا ما جرّب]».
func parseOnboarding(text string) []onboardingMsg {
	var out []onboardingMsg
	for _, line := range strings.Split(text, "\n") {
		parts := strings.Split(line, "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) < 2 || parts[0] == "" {
			continue
		}
		m := onboardingMsg{Title: parts[0], Body: parts[1]}
		if len(parts) > 2 && (strings.Contains(parts[2], "جرّب") || strings.Contains(parts[2], "جرب")) {
			m.OnlyIfUnused = true
		}
		out = append(out, m)
	}
	return out
}

// onboardingSlot **أيُّ موعدٍ الآن؟** — «2026-10-12/11» داخلَ ساعتِه، وفارغٌ خارجها.
func onboardingSlot(now time.Time) string {
	loc, err := time.LoadLocation("Asia/Damascus")
	if err != nil {
		loc = time.FixedZone("Damascus", 3*3600)
	}
	t := now.In(loc)
	for _, h := range onboardingHours {
		if t.Hour() == h {
			return t.Format("2006-01-02") + "/" + strconv.Itoa(h)
		}
	}
	return ""
}

// RunMerchantOnboarding **نبضةٌ كلَّ دقيقة** — تُرسل ما حان في ساعتَي الإرسال.
func (s *Server) RunMerchantOnboarding(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.merchantOnboardingTick(ctx, now)
		}
	}
}

func (s *Server) merchantOnboardingTick(ctx context.Context, now time.Time) int {
	if s.pg == nil || s.settings == nil || !s.settings.GetBool(ctx, "merchants.onboarding_push") {
		return 0
	}
	slot := onboardingSlot(now)
	if slot == "" {
		return 0
	}
	msgs := parseOnboarding(s.settings.GetString(ctx, "merchants.onboarding_push_messages"))
	if len(msgs) == 0 {
		return 0
	}
	rows, err := s.pg.Query(ctx, `
		SELECT DISTINCT u.id::text, COALESCE(p.next_step, 0)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'merchant'
		LEFT JOIN merchant_onboarding_push p ON p.user_id = u.id
		WHERE u.status = 'active'
		  AND COALESCE(p.next_step, 0) < $1
		  AND COALESCE(p.last_slot, '') <> $2
		LIMIT 1000`, len(msgs), slot)
	if err != nil {
		s.logger.Warn("سلسلةُ المتاجر: تعذّرت القراءة", "err", err)
		return 0
	}
	type due struct {
		id   string
		step int
	}
	var list []due
	for rows.Next() {
		var d due
		if rows.Scan(&d.id, &d.step) == nil {
			list = append(list, d)
		}
	}
	rows.Close()

	sent := 0
	for _, d := range list {
		step := d.step
		for step < len(msgs) && msgs[step].OnlyIfUnused && s.ownerUsedDelivery(ctx, d.id) {
			step++
		}
		next := step
		if step < len(msgs) {
			next = step + 1
		}
		// **يُحجز الموعدُ قبل الإرسال** — ومن سبقته نبضةٌ أخرى لا يُرسَل له.
		tag, err := s.pg.Exec(ctx, `
			INSERT INTO merchant_onboarding_push (user_id, next_step, last_slot, updated_at)
			VALUES ($1::uuid, $2, $3, now())
			ON CONFLICT (user_id) DO UPDATE
			   SET next_step = EXCLUDED.next_step, last_slot = EXCLUDED.last_slot, updated_at = now()
			 WHERE merchant_onboarding_push.last_slot <> EXCLUDED.last_slot`, d.id, next, slot)
		if err != nil || tag.RowsAffected() == 0 || step >= len(msgs) {
			continue
		}
		if s.notify != nil {
			s.notify.Notify(ctx, notifications.Input{
				UserID: d.id, Kind: notifications.KindAccount,
				Title: msgs[step].Title, Body: msgs[step].Body,
				Apps: []string{notifications.AppMerchant},
			})
		}
		sent++
	}
	if sent > 0 {
		s.logger.Info("سلسلةُ المتاجر: أُرسل", "slot", slot, "count", sent)
	}
	return sent
}

// ownerUsedDelivery **أطلب صاحبُ هذا الحساب «لدي توصيلة» مرّةً؟**
func (s *Server) ownerUsedDelivery(ctx context.Context, userID string) bool {
	var used bool
	_ = s.pg.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE m.owner_user_id = $1::uuid AND o.kind = 'merchant_delivery')`, userID).Scan(&used)
	return used
}

// MerchantOnboardingTickForTest **نبضةٌ بوقتٍ يُمرَّر** — لفحوص `qa` وحدَها.
func (s *Server) MerchantOnboardingTickForTest(ctx context.Context, now time.Time) int {
	return s.merchantOnboardingTick(ctx, now)
}

// IdentityForTest **خدمةُ الهويّة** — لفحوص `qa` وحدَها.
func (s *Server) IdentityForTest() *identity.Service { return s.identity }
