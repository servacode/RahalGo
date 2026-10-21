package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/campaigns"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **«انضمّ متجر جديد إلى عائلة رحّال غو»** (قرارُ المالك ٢٠٢٦-١٠-٢١)
// ══════════════════════════════════════════════════════════════════════
//
// **حين يصير المتجرُ ظاهراً للزبائن أوّلَ مرّة** — نشطاً بموقعٍ ودوامٍ وصنفٍ متوفّرٍ
// موافَقٍ عليه — يُعلَن عنه لكلّ الزبائن، **مرّةً واحدةً** (`announced_at`، هجرة ٠٤٧١).
// لا لحظةَ التسجيل: من فتح متجراً فارغاً لا يعود.
//
// **ويمرّ بالحملات** (`campaigns`) لا بإشعارٍ مباشر: صنفُ «تفاعل» فيُؤجَّل في ساعات
// الهدوء إلى الصباح، ويُحسب في سقف اليوم، ولا يصل أحداً مرّتين. **ونقرُه يفتح صفحةَ المتجر.**

// storeAnnounceCond **متجرٌ ظاهرٌ لم يُعلَن عنه بعد.**
const storeAnnounceCond = `m.announced_at IS NULL AND NOT m.announce_skip AND m.status = 'active'
	AND m.location IS NOT NULL
	AND EXISTS (SELECT 1 FROM merchant_hours h WHERE h.merchant_id = m.id AND NOT h.closed)
	AND EXISTS (SELECT 1 FROM menu_items i WHERE i.merchant_id = m.id AND i.available AND i.approved)`

// storeAnnounceText **نصُّ الإعلان** — رسميٌّ قصير (طلبُ المالك: «صياغة احترافية»).
func storeAnnounceText(name string) (title, body string) {
	return "🎉 انضمّ " + name + " إلى عائلة رحّال غو",
		"أصبح بإمكانك الطلب من " + name + " عبر تطبيقنا. اكتشف منتجاته الآن"
}

// SweepStoreAnnounce **يعلن عن كلّ متجرٍ صار ظاهراً** — ويردّ كم أُعلن.
func (s *Server) SweepStoreAnnounce(ctx context.Context) int {
	if s.campaigns == nil {
		return 0
	}
	actor := s.systemActorID(ctx)
	if actor == "" {
		return 0
	}
	rows, err := s.pg.Query(ctx, `SELECT m.id::text FROM merchants m WHERE `+storeAnnounceCond+` LIMIT 20`)
	if err != nil {
		s.logger.Warn("store announce sweep", "err", err)
		return 0
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	n := 0
	for _, id := range ids {
		if s.announceStore(ctx, actor, id) {
			n++
		}
	}
	return n
}

// announceStore **يثبّت الإعلانَ ثمّ يرسله** — والتثبيتُ أوّلاً: عاملان معاً لا يعلنان مرّتين.
func (s *Server) announceStore(ctx context.Context, actor, merchantID string) bool {
	var name string
	if err := s.pg.QueryRow(ctx, `UPDATE merchants m SET announced_at = now()
		WHERE m.id = $1 AND `+storeAnnounceCond+` RETURNING m.name`, merchantID).Scan(&name); err != nil {
		return false
	}
	title, body := storeAnnounceText(name)
	c, err := s.campaigns.Create(ctx, actor, "store-announce:"+merchantID, campaigns.Input{
		Title: title, Body: body,
		AudienceType: campaigns.AudienceRole, AudienceRef: "customer",
		DestType: campaigns.DestMerchant, DestID: merchantID,
	})
	if err != nil {
		s.logger.Warn("store announce create", "merchant", merchantID, "err", err)
		return false
	}
	if _, err := s.campaigns.Send(ctx, c.ID); err != nil {
		s.logger.Warn("store announce send", "merchant", merchantID, "err", err)
	}
	return true
}

// RunStoreAnnounce **نبضةٌ دوريّة** — عشرُ دقائق تكفي: الإعلانُ لا يستعجل.
func (s *Server) RunStoreAnnounce(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepStoreAnnounce(ctx)
		}
	}
}

// handleAdminStoreAnnounce **حالُ إعلان المتجر** — أأُعلن عنه، وهل عليه «لا ترسل».
// GET /admin/merchants/{id}/announce
func (s *Server) handleAdminStoreAnnounce(w http.ResponseWriter, r *http.Request) {
	var at *time.Time
	var skip bool
	err := s.pg.QueryRow(r.Context(), `SELECT announced_at, announce_skip FROM merchants WHERE id::text = $1`,
		chi.URLParam(r, "id")).Scan(&at, &skip)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"announced_at": at, "skip": skip})
}

// handleAdminSetStoreAnnounce **«لا ترسل إشعاراً» لمتجرٍ بعينه** — تجريبيٌّ مثلاً، فلا يصل
// الزبائنَ الحقيقيّين خبرُ متجرٍ ليس حقيقيّاً. PATCH /admin/merchants/{id}/announce {skip}
func (s *Server) handleAdminSetStoreAnnounce(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Skip *bool `json:"skip"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Skip == nil {
		s.respondErr(w, errValidation)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `UPDATE merchants SET announce_skip = $2 WHERE id::text = $1`,
		chi.URLParam(r, "id"), *req.Skip)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.handleAdminStoreAnnounce(w, r)
}
