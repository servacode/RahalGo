package server

// ══════════════════════════════════════════════════════════════════════
// **«رسالة للمتاجر» — واتساب وإشعارُ التطبيق لكلّ المتاجر** (قرارُ المالك ٢٠٢٦-١١-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// رسالةٌ واحدةٌ (تذكيرُ تحديث الأسعار) تصل صاحبَ كلّ متجرٍ فعّالٍ له رقمُ واتساب: يدوياً بزرّ
// «ابعت هلّق»، أو آلياً (يومياً/أسبوعياً بساعةٍ بتوقيت دمشق)، أو «تجربةً» لمتجرٍ واحد.
//
// # حمايةُ رقم البوت
//
// **البوتُ نفسُه يرسل رموزَ التسجيل — فحظرُ رقمه يوقف التسجيلَ كلَّه.** لذلك:
//   - الإرسالُ **متسلسلٌ** من عاملٍ واحدٍ في الخلفيّة، **بفاصلٍ عشوائيٍّ ٢٠–٤٥ ثانيةً** بين رسالتين؛
//   - **لا رسالتين لمتجرٍ في يومٍ واحد** — الحارسُ فهرسٌ فريدٌ في القاعدة (٠٥٤٩) لا شرطٌ في الشيفرة؛
//   - **البوتُ غيرُ جاهز ⇒ تتوقّف التشغيلةُ** وتُحذَّر الإدارة، **ولا تُعاد وحدَها** (لا طرقَ على باب قناةٍ مقطوعة).
//
// # وعند إعادة تشغيل الخادم
//
// التشغيلةُ والتسليماتُ صفوفٌ في القاعدة. **ما كان `pending` يكمل بعد الإقلاع** (العاملُ يقرأ من القاعدة
// لا من ذاكرة)، **وما كان `sending` لحظةَ الانقطاع يُوسم `failed`** ولا يُعاد: لا نعرف أوصل أم لا،
// **والتكرارُ على مستلمٍ أسوأُ من فقدِ رسالة.**
//
// # وإشعارُ التطبيق
//
// يُرسل **عند التخطيط** لكلّ متجرٍ حُجز له تسليم (لا يتأخّر ساعاتٍ مع فاصل الواتساب)، ولصاحب المتجر
// وحدَه في تطبيق المتجر.

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

const (
	// storeMsgGapMin/Max الفاصلُ العشوائيّ بين رسالتين — **حمايةُ رقم البوت.**
	storeMsgGapMin = 20 * time.Second
	storeMsgGapMax = 45 * time.Second
	// storeMsgPoll كم ينتظر العاملُ حين لا شيء عنده.
	storeMsgPoll = 5 * time.Second
	// storeMsgBodyMax سقفُ نصّ الإشعار (كالإعلان).
	storeMsgBodyMax = 500

	storeMsgBotNotReady = "bot_not_ready"
)

var (
	errStoreMsgNoWhatsApp = httpx.NewError(http.StatusUnprocessableEntity,
		"store_message_no_whatsapp", "errors.store_message_no_whatsapp")
	errStoreMsgBotNotReady = httpx.NewError(http.StatusConflict,
		"store_message_bot_not_ready", "errors.store_message_bot_not_ready")
	errStoreMsgEmpty = httpx.NewError(http.StatusUnprocessableEntity,
		"store_message_empty", "errors.store_message_empty")
)

// storeMsgRand مصدرُ العشوائيّة للفاصل.
func storeMsgRandomGap() time.Duration {
	span := int64(storeMsgGapMax - storeMsgGapMin)
	return storeMsgGapMin + time.Duration(rand.Int63n(span+1))
}

func (s *Server) storeMsgGap() time.Duration {
	if s.storeMsgGapFn != nil {
		return s.storeMsgGapFn()
	}
	return storeMsgRandomGap()
}

func (s *Server) storeMsgNow() time.Time {
	if s.storeMsgClock != nil {
		return s.storeMsgClock()
	}
	return time.Now()
}

func storeMsgLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Damascus")
	if err != nil {
		return time.FixedZone("Damascus", 3*3600)
	}
	return loc
}

// storeMsgDayOf يوم دمشق كنصّ تاريخ.
func storeMsgDayOf(now time.Time) string {
	return now.In(storeMsgLoc()).Format("2006-01-02")
}

// storeMsgWakeCh قناةُ إيقاظ العامل — تُنشأ مرّةً.
func (s *Server) storeMsgWakeCh() chan struct{} {
	s.storeMsgOnce.Do(func() { s.storeMsgWake = make(chan struct{}, 1) })
	return s.storeMsgWake
}

func (s *Server) storeMsgPoke() {
	select {
	case s.storeMsgWakeCh() <- struct{}{}:
	default:
	}
}

// clipRunes يقصّ بالمحارف لا بالبايتات — **والعربيُّ بايتان، فقصُّ البايت يكسر الحرف.**
func clipRunes(v string, max int) string {
	v = strings.TrimSpace(v)
	r := []rune(v)
	if len(r) > max {
		return string(r[:max])
	}
	return v
}

// ── المتاجرُ المؤهَّلة ────────────────────────────────────────────────

type storeMsgTarget struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Owner string `json:"-"`
	Phone string `json:"-"`
}

// **مؤهَّلٌ = متجرٌ فعّالٌ وصاحبُه فعّالٌ وله رقمُ جوّالٍ سوريّ.**
//
// **الرقمُ**: واتساب الموثَّق لصاحبه أوّلاً، ثم هاتفُ صاحبه، ثم هاتفُ المتجر — **وأوّلُ ما هو جوّالٌ منها**
// (`09…` أو `+9639…`). **والخطُّ الأرضيّ (`022…`) رابطٌ ميّت على واتساب** فيُتخطّى بصمت
// (انظر `merchantPhones.WhatsApp`).
const storeMsgEligibleSQL = `
SELECT t.id, t.name, t.owner, t.phone FROM (
  SELECT m.id::text AS id, m.name::text AS name,
         COALESCE(m.owner_user_id::text, '') AS owner,
         COALESCE((
           SELECT regexp_replace(c.v, '[\s-]', '', 'g')
           FROM unnest(ARRAY[
                  CASE WHEN ou.whatsapp_verified_at IS NOT NULL THEN ou.whatsapp_phone::text END,
                  ou.phone::text, m.phone::text]) WITH ORDINALITY AS c(v, ord)
           WHERE regexp_replace(COALESCE(c.v, ''), '[\s-]', '', 'g') ~ '^(\+?963|0)9[0-9]{8}$'
           ORDER BY c.ord LIMIT 1), '') AS phone
  FROM merchants m
  LEFT JOIN users ou ON ou.id = m.owner_user_id
  WHERE m.status = 'active' AND (ou.id IS NULL OR ou.status = 'active')
) t
WHERE t.phone <> '' AND ($1 = '' OR t.id = $1)
ORDER BY t.name, t.id`

func (s *Server) storeMsgEligible(ctx context.Context, onlyID string) ([]storeMsgTarget, error) {
	rows, err := s.pg.Query(ctx, storeMsgEligibleSQL, onlyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storeMsgTarget
	for rows.Next() {
		var t storeMsgTarget
		if err := rows.Scan(&t.ID, &t.Name, &t.Owner, &t.Phone); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ── التخطيط ───────────────────────────────────────────────────────────

// storeMsgPlan **يفتح تشغيلةً ويحجز لكلّ متجرٍ مؤهَّلٍ تسليمَه** — والمحجوزُ له اليومَ يُتخطّى
// (الفهرسُ الفريد). **ويردّ عددَ من حُجز لهم فعلاً**؛ صفرٌ ⇒ لا تشغيلةَ (تُمحى) ولا شيءَ يُرسل.
func (s *Server) storeMsgPlan(ctx context.Context, now time.Time, trigger, onlyMerchant, createdBy string) (runID string, target int, err error) {
	title := clipRunes(s.settings.GetString(ctx, "store_message.title"), 80)
	body := strings.TrimSpace(s.settings.GetString(ctx, "store_message.text"))
	if body == "" {
		return "", 0, errStoreMsgEmpty
	}
	stores, err := s.storeMsgEligible(ctx, onlyMerchant)
	if err != nil {
		return "", 0, err
	}
	if onlyMerchant != "" && len(stores) == 0 {
		return "", 0, errStoreMsgNoWhatsApp
	}
	day := storeMsgDayOf(now)
	var by any
	if createdBy != "" {
		by = createdBy
	}
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO store_message_runs (trigger, title, body, day, created_by)
		VALUES ($1, $2, $3, $4::date, $5::uuid) RETURNING id::text`,
		trigger, title, body, day, by).Scan(&runID); err != nil {
		return "", 0, err
	}
	isTest := trigger == "test"
	for _, st := range stores {
		tag, e := s.pg.Exec(ctx, `
			INSERT INTO store_message_deliveries (run_id, merchant_id, phone, day, is_test)
			VALUES ($1::uuid, $2::uuid, $3, $4::date, $5)
			ON CONFLICT DO NOTHING`, runID, st.ID, st.Phone, day, isTest)
		if e != nil {
			s.logger.Warn("رسالةُ المتاجر: تعذّر حجزُ تسليم", "merchant", st.ID, "err", e)
			continue
		}
		if tag.RowsAffected() == 0 {
			continue // **سبق أن وصلته اليوم.**
		}
		target++
		if st.Owner != "" && s.notify != nil {
			s.notify.Notify(ctx, notifications.Input{
				UserID: st.Owner, Kind: notifications.KindAccount,
				Title: title, Body: clipRunes(body, storeMsgBodyMax),
				Apps: []string{notifications.AppMerchant},
			})
		}
	}
	if target == 0 {
		_, _ = s.pg.Exec(ctx, `DELETE FROM store_message_runs WHERE id = $1::uuid`, runID)
		return "", 0, nil
	}
	if _, err := s.pg.Exec(ctx,
		`UPDATE store_message_runs SET target_count = $2 WHERE id = $1::uuid`, runID, target); err != nil {
		return runID, target, err
	}
	s.storeMsgPoke()
	return runID, target, nil
}

// ── العاملُ ───────────────────────────────────────────────────────────

// RunStoreMessageWorker **عاملُ الإرسال الواحد** — يُقلَع مرّةً مع الخادم.
//
// **يُقرأ من القاعدة لا من الذاكرة**: فمن أُعيد تشغيلُ الخادم وسطَ تشغيلةٍ أكملها.
func (s *Server) RunStoreMessageWorker(ctx context.Context) {
	if s.pg == nil {
		return
	}
	s.storeMsgRecover(ctx)
	for {
		if ctx.Err() != nil {
			return
		}
		did, wasTest := s.storeMsgStep(ctx)
		if did && wasTest {
			continue // **التجريبيُّ والتوقّفُ بلا فاصل.**
		}
		if did {
			// **الفاصلُ بعد كلّ رسالةٍ حقيقيّة** — لا يُقصَّر بإيقاظ.
			select {
			case <-ctx.Done():
				return
			case <-time.After(s.storeMsgGap()):
			}
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(storeMsgPoll):
		case <-s.storeMsgWakeCh():
		}
	}
}

// storeMsgRecover عند الإقلاع: ما كان `sending` لا نعرف مصيرَه ⇒ failed ولا يُعاد.
func (s *Server) storeMsgRecover(ctx context.Context) {
	_, _ = s.pg.Exec(ctx, `
		UPDATE store_message_deliveries SET status = 'failed', error = 'interrupted'
		WHERE status = 'sending'`)
	rows, err := s.pg.Query(ctx, `SELECT id::text FROM store_message_runs WHERE status = 'running'`)
	if err != nil {
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		s.storeMsgSettle(ctx, id)
	}
}

// storeMsgSettle يعيد عدّ تشغيلةٍ ويُنهيها إن لم يبقَ ما ينتظر.
func (s *Server) storeMsgSettle(ctx context.Context, runID string) {
	_, _ = s.pg.Exec(ctx, `
		UPDATE store_message_runs r SET
		  sent_count   = (SELECT count(*) FROM store_message_deliveries d WHERE d.run_id = r.id AND d.status = 'sent'),
		  failed_count = (SELECT count(*) FROM store_message_deliveries d WHERE d.run_id = r.id AND d.status = 'failed')
		WHERE r.id = $1::uuid`, runID)
	_, derr := s.pg.Exec(ctx, `
		UPDATE store_message_runs r SET status = 'done', finished_at = now()
		WHERE r.id = $1::uuid AND r.status = 'running'
		  AND NOT EXISTS (SELECT 1 FROM store_message_deliveries d
		                   WHERE d.run_id = r.id AND d.status IN ('pending', 'sending'))`, runID)
	if derr != nil {
		s.logger.Warn("رسالةُ المتاجر: تعذّر إنهاءُ التشغيلة", "run", runID, "err", derr)
	}
}

// storeMsgStopAll **البوتُ غيرُ جاهز: تتوقّف كلُّ تشغيلةٍ جاريةٍ** ويُحذف ما لم يُرسل (pending) —
// **فيبقى المتجرُ غيرَ محجوزٍ له اليومُ ويُعاد بزرٍّ لاحقاً.** ثم تنبيهٌ لمكتب الإدارة.
func (s *Server) storeMsgStopAll(ctx context.Context, reason string) {
	rows, err := s.pg.Query(ctx, `
		UPDATE store_message_runs SET status = 'stopped', stop_reason = $1, finished_at = now()
		WHERE status = 'running' RETURNING id::text`, reason)
	if err != nil {
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		_, _ = s.pg.Exec(ctx, `
			DELETE FROM store_message_deliveries WHERE run_id = $1::uuid AND status = 'pending'`, id)
		_, _ = s.pg.Exec(ctx, `
			UPDATE store_message_runs r SET
			  sent_count   = (SELECT count(*) FROM store_message_deliveries d WHERE d.run_id = r.id AND d.status = 'sent'),
			  failed_count = (SELECT count(*) FROM store_message_deliveries d WHERE d.run_id = r.id AND d.status = 'failed')
			WHERE r.id = $1::uuid`, id)
	}
	if len(ids) > 0 {
		s.logger.Warn("رسالةُ المتاجر: توقّفت التشغيلةُ — بوتُ واتساب غيرُ جاهز", "runs", len(ids))
		if s.notify != nil {
			s.notify.NotifyRoles(ctx, notifications.OpsDesk, notifications.Input{
				Kind:  notifications.KindAccount,
				Title: "توقّفت رسالة المتاجر — بوت الواتساب مو جاهز",
				Body:  "ما انبعتت باقي الرسائل. تأكد من اتصال البوت وارجع اضغط «ابعت هلّق» من صفحة «رسالة للمتاجر».",
				Href:  "/dashboard/store-message",
			})
		}
	}
}

// storeMsgStep **يرسل تسليماً واحداً إن وُجد** — ويردّ: أعمل شيئاً؟ وأكان تجريبياً؟
//
// **والتجريبيُّ أوّلاً** (مرّةٌ واحدةٌ لا تنتظر فاصلاً خلف مئة رسالة)، ثم الأقدم.
func (s *Server) storeMsgStep(ctx context.Context) (did, wasTest bool) {
	var (
		id, runID, phone, body string
		isTest                 bool
	)
	err := s.pg.QueryRow(ctx, `
		SELECT d.id::text, d.run_id::text, d.phone, d.is_test, r.body
		FROM store_message_deliveries d
		JOIN store_message_runs r ON r.id = d.run_id
		WHERE d.status = 'pending' AND r.status = 'running'
		ORDER BY d.is_test DESC, r.started_at, d.created_at, d.id
		LIMIT 1`).Scan(&id, &runID, &phone, &isTest, &body)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warn("رسالةُ المتاجر: تعذّرت القراءة", "err", err)
		}
		return false, false
	}
	if !s.merchantReady() {
		s.storeMsgStopAll(ctx, storeMsgBotNotReady)
		return true, true // **لا فاصلَ بعد توقّف.**
	}
	tag, err := s.pg.Exec(ctx, `
		UPDATE store_message_deliveries SET status = 'sending' WHERE id = $1::uuid AND status = 'pending'`, id)
	if err != nil || tag.RowsAffected() == 0 {
		return false, false
	}
	sendErr := s.merchant.SendText(ctx, phone, body)
	if sendErr != nil {
		s.logger.Warn("رسالةُ المتاجر: تعذّر الإرسال", "delivery", id, "err", sendErr)
		_, _ = s.pg.Exec(ctx, `
			UPDATE store_message_deliveries SET status = 'failed', error = $2 WHERE id = $1::uuid`,
			id, clipRunes(sendErr.Error(), 300))
	} else {
		_, _ = s.pg.Exec(ctx, `
			UPDATE store_message_deliveries SET status = 'sent', sent_at = now() WHERE id = $1::uuid`, id)
	}
	s.storeMsgSettle(ctx, runID)
	// **والبوتُ سقط أثناء هذه الرسالة**: لا نكمل على قناةٍ مقطوعة.
	if sendErr != nil && !s.merchantReady() {
		s.storeMsgStopAll(ctx, storeMsgBotNotReady)
		return true, true
	}
	return true, isTest
}

// ── المجدوِل ──────────────────────────────────────────────────────────

// RunStoreMessageScheduler نبضةٌ كلَّ دقيقة — تفتح تشغيلةً آليّةً داخلَ ساعتها فقط.
func (s *Server) RunStoreMessageScheduler(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.storeMessageSchedTick(ctx, now)
		}
	}
}

// storeMessageSchedTick **هل حان الموعد؟** — مفتاحُ التشغيل، **وساعةُ دمشق**، **ويومُ الأسبوع إن كان أسبوعياً**.
// **وتشغيلةٌ آليّةٌ واحدةٌ باليوم** (`day`) فلا تعيد النبضاتُ الستّون داخلَ الساعة.
func (s *Server) storeMessageSchedTick(ctx context.Context, now time.Time) bool {
	if s.pg == nil || s.settings == nil || !s.settings.GetBool(ctx, "store_message.auto_enabled") {
		return false
	}
	local := now.In(storeMsgLoc())
	if local.Hour() != int(s.settings.GetInt(ctx, "store_message.auto_hour")) {
		return false
	}
	if s.settings.GetString(ctx, "store_message.auto_frequency") == "weekly" &&
		int(local.Weekday()) != int(s.settings.GetInt(ctx, "store_message.auto_weekday")) {
		return false
	}
	day := local.Format("2006-01-02")
	var exists bool
	if err := s.pg.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM store_message_runs WHERE trigger = 'auto' AND day = $1::date)`,
		day).Scan(&exists); err != nil || exists {
		return false
	}
	_, n, err := s.storeMsgPlan(ctx, now, "auto", "", "")
	if err != nil {
		s.logger.Warn("رسالةُ المتاجر: تعذّر التخطيطُ الآليّ", "err", err)
		return false
	}
	return n > 0
}

// ── الاختبارُ ─────────────────────────────────────────────────────────

// StoreMessageRunForTest يخطّط تشغيلةً. (للفحوص وحدَها.)
func (s *Server) StoreMessageRunForTest(ctx context.Context, trigger, merchantID string) (string, int, error) {
	return s.storeMsgPlan(ctx, s.storeMsgNow(), trigger, merchantID, "")
}

// StoreMessageStepForTest يرسل تسليماً واحداً بلا فاصل. (للفحوص وحدَها.)
func (s *Server) StoreMessageStepForTest(ctx context.Context) bool {
	did, _ := s.storeMsgStep(ctx)
	return did
}

// StoreMessageSchedTickForTest نبضةُ المجدوِل بوقتٍ يُمرَّر. (للفحوص وحدَها.)
func (s *Server) StoreMessageSchedTickForTest(ctx context.Context, now time.Time) bool {
	return s.storeMessageSchedTick(ctx, now)
}

// SetStoreMessageClockForTest يُثبّت ساعةَ التخطيط. (للفحوص وحدَها.)
func (s *Server) SetStoreMessageClockForTest(f func() time.Time) { s.storeMsgClock = f }

// SetStoreMessageGapForTest يحقن الفاصلَ. (للفحوص وحدَها.)
func (s *Server) SetStoreMessageGapForTest(f func() time.Duration) { s.storeMsgGapFn = f }

// StoreMessageGapForTest الفاصلُ الفعليّ الآن. (للفحوص وحدَها.)
func (s *Server) StoreMessageGapForTest() time.Duration { return s.storeMsgGap() }

// StoreMessageGapRandomForTest الفاصلُ العشوائيّ الحقيقيّ. (للفحوص وحدَها.)
func StoreMessageGapRandomForTest() time.Duration { return storeMsgRandomGap() }

// StoreMessageRecoverForTest ما يفعله العاملُ عند الإقلاع. (للفحوص وحدَها.)
func (s *Server) StoreMessageRecoverForTest(ctx context.Context) { s.storeMsgRecover(ctx) }
