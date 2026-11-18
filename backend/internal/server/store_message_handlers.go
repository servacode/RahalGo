package server

// واجهةُ صفحة «رسالة للمتاجر» — للإدارة بقدرة المحتوى كالإعلان (`content.manage`).
// المنطقُ كلُّه في `store_message.go`؛ هنا قراءةُ الحال وحفظُ الإعدادات وزرّا الإرسال.

import (
	"net/http"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type storeMsgConfig struct {
	Title       string `json:"title"`
	Text        string `json:"text"`
	AutoEnabled bool   `json:"auto_enabled"`
	Frequency   string `json:"auto_frequency"`
	Weekday     int    `json:"auto_weekday"`
	Hour        int    `json:"auto_hour"`
}

func (s *Server) storeMsgConfig(r *http.Request) storeMsgConfig {
	ctx := r.Context()
	return storeMsgConfig{
		Title:       s.settings.GetString(ctx, "store_message.title"),
		Text:        s.settings.GetString(ctx, "store_message.text"),
		AutoEnabled: s.settings.GetBool(ctx, "store_message.auto_enabled"),
		Frequency:   s.settings.GetString(ctx, "store_message.auto_frequency"),
		Weekday:     int(s.settings.GetInt(ctx, "store_message.auto_weekday")),
		Hour:        int(s.settings.GetInt(ctx, "store_message.auto_hour")),
	}
}

// handleStoreMessageState الحالُ كلُّه في نداءٍ واحد: الإعدادات، عددُ من سيصلهم، جاهزيّةُ البوت، والسجلّ.
func (s *Server) handleStoreMessageState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stores, err := s.storeMsgEligible(ctx, "")
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والعددُ الذي سيصل فعلاً** = المؤهَّلون ناقصَ من وصلهم اليومَ.
	var already int
	_ = s.pg.QueryRow(ctx, `
		SELECT count(*) FROM store_message_deliveries WHERE day = $1::date AND NOT is_test`,
		storeMsgDayOf(s.storeMsgNow())).Scan(&already)
	rows, err := s.pg.Query(ctx, `
		SELECT id::text, trigger, status, stop_reason, target_count, sent_count, failed_count,
		       started_at, finished_at
		FROM store_message_runs ORDER BY started_at DESC LIMIT 30`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type run struct {
		ID         string     `json:"id"`
		Trigger    string     `json:"trigger"`
		Status     string     `json:"status"`
		StopReason string     `json:"stop_reason"`
		Target     int        `json:"target"`
		Sent       int        `json:"sent"`
		Failed     int        `json:"failed"`
		StartedAt  time.Time  `json:"started_at"`
		FinishedAt *time.Time `json:"finished_at"`
	}
	runs := []run{}
	for rows.Next() {
		var x run
		if err := rows.Scan(&x.ID, &x.Trigger, &x.Status, &x.StopReason, &x.Target,
			&x.Sent, &x.Failed, &x.StartedAt, &x.FinishedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		runs = append(runs, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"config":         s.storeMsgConfig(r),
		"eligible":       len(stores),
		"already_today":  already,
		"bot_ready":      s.merchantReady(),
		"runs":           runs,
		"default_gap_ms": []int64{storeMsgGapMin.Milliseconds(), storeMsgGapMax.Milliseconds()},
	})
}

// handleStoreMessageStores قائمةُ المتاجر المؤهَّلة — لاختيار متجر «التجربة».
func (s *Server) handleStoreMessageStores(w http.ResponseWriter, r *http.Request) {
	stores, err := s.storeMsgEligible(r.Context(), "")
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if stores == nil {
		stores = []storeMsgTarget{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"stores": stores})
}

// handleStoreMessageConfig يحفظ النصَّ والعنوانَ والجدولةَ — **كلُّها تمرّ على `settings.Validate`**.
func (s *Server) handleStoreMessageConfig(w http.ResponseWriter, r *http.Request) {
	req, err := decode[storeMsgConfig](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if strings.TrimSpace(req.Text) == "" || strings.TrimSpace(req.Title) == "" {
		s.respondErr(w, errStoreMsgEmpty)
		return
	}
	by := userIDFrom(r)
	var actor *string
	if by != "" {
		actor = &by
	}
	for _, kv := range []struct {
		key string
		val any
	}{
		{"store_message.title", strings.TrimSpace(req.Title)},
		{"store_message.text", strings.TrimSpace(req.Text)},
		{"store_message.auto_enabled", req.AutoEnabled},
		{"store_message.auto_frequency", req.Frequency},
		{"store_message.auto_weekday", req.Weekday},
		{"store_message.auto_hour", req.Hour},
	} {
		if err := s.settings.Set(r.Context(), kv.key, kv.val, actor); err != nil {
			s.respondErr(w, errValidation)
			return
		}
	}
	s.audit(r, "ops.store_message_config", "store_message", "", map[string]any{
		"auto_enabled": req.AutoEnabled, "frequency": req.Frequency,
		"weekday": req.Weekday, "hour": req.Hour,
	})
	httpx.JSON(w, http.StatusOK, map[string]any{"config": s.storeMsgConfig(r)})
}

// handleStoreMessageSend «ابعت هلّق» — يخطّط التشغيلةَ ويردّ كم حُجز؛ والعاملُ يرسل في الخلفيّة.
func (s *Server) handleStoreMessageSend(w http.ResponseWriter, r *http.Request) {
	if !s.merchantReady() {
		s.respondErr(w, errStoreMsgBotNotReady)
		return
	}
	runID, n, err := s.storeMsgPlan(r.Context(), s.storeMsgNow(), "manual", "", userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.store_message_send", "store_message", runID, map[string]any{"trigger": "manual", "target": n})
	httpx.JSON(w, http.StatusOK, map[string]any{"run_id": runID, "target": n})
}

// handleStoreMessageTest «ابعت تجربة لمتجر واحد».
func (s *Server) handleStoreMessageTest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		MerchantID string `json:"merchant_id"`
	}](r)
	if err != nil || strings.TrimSpace(req.MerchantID) == "" {
		s.respondErr(w, errValidation)
		return
	}
	if !s.merchantReady() {
		s.respondErr(w, errStoreMsgBotNotReady)
		return
	}
	runID, n, err := s.storeMsgPlan(r.Context(), s.storeMsgNow(), "test", strings.TrimSpace(req.MerchantID), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.store_message_send", "store_message", runID, map[string]any{
		"trigger": "test", "merchant": req.MerchantID})
	httpx.JSON(w, http.StatusOK, map[string]any{"run_id": runID, "target": n})
}
