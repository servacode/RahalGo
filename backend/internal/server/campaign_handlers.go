package server

// ══════════════════════════════════════════════════════════════════════
// **مركزُ الإشعارات — يُعايَن ثمّ يُرسَل** (`NT`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// # ولا قدرةَ جديدة
//
// **و`content.manage` تحرس الإعلانَ والعروضَ واللافتاتِ والأكواد منذ
// زمن** — **وهي قدرةُ من يخاطب الناسَ باسم المنصّة.** **ودورُ
// `marketing_content` مبذورٌ بها وحدَها** (الهجرة ٠١٣٨)، **فلا يُوسَّع
// أحدٌ ولا تُخترَع قدرةٌ لتُضاف إلى أدوارٍ لا تحتاجها.**
//
// # والمعاينةُ لا تُرسل
//
// **ونداءُ العدّ يقرأ ولا يكتب** — **ولا صفَّ لإنسانٍ منه** (`NT-04`):
// **ومن عاين فوجد رسالتَه قد وصلت لا يملك أن يسحبها.**

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/campaigns"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/offers"
)

// handleCampaignList **تاريخُ الحملات.**
func (s *Server) handleCampaignList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.campaigns.List(r.Context(), 50)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"campaigns": rows})
}

// handleCampaignPreview **من سيصلهم؟ — ولا يُرسَل شيء.**
//
// **وهو نداءُ قراءةٍ محض** — **`GET` لأنّه لا يغيّر شيئاً**: **ومن
// جعله `POST` أغرى بأن يكتب فيه يوماً.**
func (s *Server) handleCampaignPreview(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("audience_type")
	ref := r.URL.Query().Get("audience_ref")
	n, err := s.campaigns.Count(r.Context(), kind, ref)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ويُقال إن كانت ستُؤجَّل** — **فيعرف قبل أن يضغط** (البند ١١).
	quiet := campaigns.DefaultQuiet
	httpx.JSON(w, http.StatusOK, map[string]any{
		"count":       n,
		"quiet_now":   quiet.InQuiet(timeNow()),
		"quiet_until": quiet.NextAllowed(timeNow()),
	})
}

// handleCampaignCreate **يكتب مسوّدةً أو مجدولة.**
func (s *Server) handleCampaignCreate(w http.ResponseWriter, r *http.Request) {
	in, err := decode[campaigns.Input](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **وهويّةُ المحاولة من ترويسة المنصّة نفسِها**
	// ══════════════════════════════════════════════════════════════════
	//
	// **و`Idempotency-Key` عقدٌ قائمٌ في بناء الطلبات** — **ولا
	// يُخترَع ثانٍ.** **وبلا مفتاحٍ يُشتقّ من المحتوى**: **فضغطتان
	// على النصّ نفسِه لصاحب الحملة نفسِه محاولةٌ واحدة.**
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		key = campaignFingerprint(*in)
	}
	c, err := s.campaigns.Create(r.Context(), userIDFrom(r), key, *in)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ولا نصَّ الرسالةِ في السجلّ** — **العنوانُ يكفي للمراجعة**،
	// **ولا قوائمَ متلقّين ولا أرقام** (البند ١٦).
	s.audit(r, "ops.campaign_create", "campaign", c.ID, map[string]any{
		"audience": c.AudienceType + ":" + c.AudienceRef,
		"status":   c.Status, "title": c.Title,
	})
	httpx.JSON(w, http.StatusOK, c)
}

// handleCampaignSend **يُرسل الآن — ومرّةً واحدة.**
func (s *Server) handleCampaignSend(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, err := s.campaigns.Get(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	c, err := s.campaigns.Send(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ولا يُقيَّد إرسالٌ لم يقع** — **والنداءُ الثاني على مُرسَلةٍ
	// يردّها كما هي، فلا يُكتب في السجلّ فعلٌ ثانٍ لم يكن.**
	if before.Status != campaigns.StatusSent {
		s.audit(r, "ops.campaign_send", "campaign", c.ID, map[string]any{
			"audience": c.AudienceType + ":" + c.AudienceRef,
			"targeted": c.Targeted, "inbox_created": c.InboxCreated,
			"deferred": c.Deferred, "status": c.Status,
		})
	}
	httpx.JSON(w, http.StatusOK, c)
}

// handleCampaignCancel **يُلغي ما لم يبدأ.**
func (s *Server) handleCampaignCancel(w http.ResponseWriter, r *http.Request) {
	c, err := s.campaigns.Cancel(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.campaign_cancel", "campaign", c.ID, map[string]any{
		"audience": c.AudienceType + ":" + c.AudienceRef,
	})
	httpx.JSON(w, http.StatusOK, c)
}

// ══════════════════════════════════════════════════════════════════════
// **الجسرُ إلى محرّك الإشعارات** — **واجهةٌ ضيّقةٌ لا محرّكٌ ثانٍ**
// ══════════════════════════════════════════════════════════════════════
//
// **والحملةُ لا تملك إلّا أن تُرسل خبراً لإنسان** — **ولا تبلغ `FCM`
// ولا صفوفَ النقل**: **تلك حقيقةُ `push` وحدَها** (`PF-09`).
type campaignNotifier struct{ n *notifications.Service }

func (c campaignNotifier) Notify(ctx context.Context, userID, kind, title, body, entity, entityID string) {
	c.n.Notify(ctx, notifications.Input{
		UserID: userID, Kind: kind, Title: title, Body: body,
		Entity: entity, EntityID: entityID,
		// **وتطبيقُ الزبون وحدَه يرنّ بالعروض** — **وصاحبُ المتجر
		// يحمل تطبيقين**، **وخبرُ عرضٍ في تطبيق متجره ضجيج.**
		Apps: []string{notifications.AppCustomer},
	})
}

// timeNow **ساعةُ الخادم** — **ولا تُقرأ من ترويسةٍ ولا من جسم.**
func timeNow() time.Time { return time.Now() }

// campaignFingerprint **هويّةٌ تُشتقّ من المحتوى حين لا تُرسَل ترويسة.**
//
// **ولوحةٌ قديمةٌ لا ترسل `Idempotency-Key`** — **وضغطتاها على النصّ
// نفسِه محاولةٌ واحدةٌ لا اثنتان.** **ومن أراد حملةً ثانيةً بالنصّ
// نفسِه يرسل مفتاحاً صريحاً.**
func campaignFingerprint(in campaigns.Input) string {
	at := ""
	if in.ScheduledAt != nil {
		at = in.ScheduledAt.UTC().Format(time.RFC3339)
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(in.Title), strings.TrimSpace(in.Body),
		in.AudienceType, strings.TrimSpace(in.AudienceRef),
		in.DestType, in.DestID, at,
	}, "\x00")))
	return hex.EncodeToString(sum[:16])
}

// RunCampaignWorker **حلقةُ الحملات المستحقّة** — **تُنادى عند الإقلاع.**
func (s *Server) RunCampaignWorker(ctx context.Context, interval time.Duration) {
	s.campaigns.RunWorker(ctx, interval)
}

// ══════════════════════════════════════════════════════════════════════
// **أبوابٌ للفحص — ما لا مسارَ شبكةٍ له**
// ══════════════════════════════════════════════════════════════════════
//
// **وجولةُ العامل لا تُنادى بنداءٍ شبكيّ** — **ولا يُفتَح لها مسارٌ
// ليُنادى في فحص**: **بابٌ يُفتح للفحص يُنادى يوماً من غيره.**

// CampaignsDueOnce **جولةُ المستحقّ مرّةً** — **كما يفعل العامل.**
func (s *Server) CampaignsDueOnce(ctx context.Context) int {
	return s.campaigns.DueOnce(ctx)
}

// NotifyForTest **خبرٌ معامليٌّ يُرسَل بالمحرّك نفسِه.**
//
// **ولا يمرّ بسياسة الإزعاج** — **وهو ما يُراد إثباتُه**: **أنّ
// المعاملةَ لا تُعدّ في سقفٍ ولا تُؤجَّل بهدوء.**
func (s *Server) NotifyForTest(ctx context.Context, userID, kind, title string) {
	s.notify.Notify(ctx, notifications.Input{UserID: userID, Kind: kind, Title: title})
}

// ══════════════════════════════════════════════════════════════════════
// **أوصلت الخدمةُ إلى هذا الهدف؟** — **يُسأل قبل خبر الوصول** (`SI-N-06`)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُخترَع حقيقةٌ ثانيةٌ للخدمة** — **يُسأل محرّكُ التوفّر نفسُه**
// (`AvailabilityAt`) **الذي يقرؤه الزبونُ قبل أن يطلب.**
//
// **ونقطةُ القياس مركزُ الهدف**: **مركزُ المدينة لمشترِكيها، ومركزُ
// المربّع لمن خارجَها.** **ومن قاس بنقطةٍ من عنده قاس شيئاً آخر.**
func (s *Server) targetServiceable(ctx context.Context, targetKey string) (bool, error) {
	lat, lng, ok := s.targetPoint(ctx, targetKey)
	if !ok {
		// **وهدفٌ لا يُعرَف موضعُه لا يُقال عنه «وصلت»** — **والمنعُ
		// بجهلٍ هنا صوابٌ**: **الوعدُ الكاذبُ أسوأُ من صمت.**
		return false, nil
	}
	av, err := s.orders.AvailabilityAt(ctx, s.pg, s.orderGates(ctx), nil, lat, lng)
	if err != nil {
		return false, err
	}
	return av.Available, nil
}

// targetPoint **مركزُ الهدف** — **من `cities` أو من المربّع نفسِه.**
func (s *Server) targetPoint(ctx context.Context, targetKey string) (float64, float64, bool) {
	switch {
	case strings.HasPrefix(targetKey, "city:"):
		var lat, lng float64
		err := s.pg.QueryRow(ctx, `
			SELECT ST_Y(center::geometry), ST_X(center::geometry)
			  FROM cities WHERE id = $1::uuid AND active`,
			strings.TrimPrefix(targetKey, "city:")).Scan(&lat, &lng)
		return lat, lng, err == nil
	case strings.HasPrefix(targetKey, "cell:"):
		parts := strings.SplitN(strings.TrimPrefix(targetKey, "cell:"), ",", 2)
		if len(parts) != 2 {
			return 0, 0, false
		}
		lat, err1 := strconv.ParseFloat(parts[0], 64)
		lng, err2 := strconv.ParseFloat(parts[1], 64)
		return lat, lng, err1 == nil && err2 == nil
	}
	return 0, 0, false
}

// offerIsLive **أهذا العرضُ سارٍ الآن؟** — **بشرط السريان نفسِه.**
//
// **ولا يُنسَخ الشرطُ هنا** — **`offers.LiveCond` مصدرٌ واحدٌ**:
// **ونسختان تفترقان فيُعلَن عن عرضٍ لا يُسعَّر به.**
func (s *Server) offerIsLive(ctx context.Context, offerID string) (bool, error) {
	var live bool
	err := s.pg.QueryRow(ctx,
		`SELECT `+offers.LiveCond+` FROM offers o WHERE o.id = $1::uuid`,
		offerID).Scan(&live)
	if err != nil {
		return false, err
	}
	return live, nil
}
