package server

// ══════════════════════════════════════════════════════════════════════
// **إعدادُ المتاجر — مَن ضبط ومَن لم يضبط** (طلبُ المالك ٢٠٢٦-١٠-٠٨)
// ══════════════════════════════════════════════════════════════════════
//
// «يكون واضح مين ضبط الأوقات مين ماضبطهن مين غيّر مين ماغيّر، وهيك نعرف
// ونرسله المطلوب.» — **والتذكيرُ بزرٍّ يدويٍّ لا بالبوت**: الموظّفُ يفتح
// الواتساب برسالةٍ جاهزةٍ ويرسلها بيده، **فلا يصير البوتُ مزعجاً فيُحظر الرقم.**
// **وإشعارُ التطبيق** بزرٍّ ثانٍ — يصل صندوقَ صاحب المتجر ويرنّ في تطبيقه.
//
// # ما يُعدّ ناقصاً
//
//   - **الدوام**: متجرٌ بلا صفوفٍ في `merchant_hours` يُعامَل مفتوحاً ٢٤ ساعة
//     (`OpenNowSQL`) — **فيصله طلبٌ وهو مسكّر.**
//   - **العنوان**: `address_text` فارغ.
//   - **الموقع**: `location` فارغ — والسائقُ لا يعرف أين يقف.
//   - **طريقةُ المستحقّات والاسترداد**: لهما افتراض، **فالناقصُ أنّه لم يُسأل**
//     (`settlement_confirmed_at`/`returns_confirmed_at` فارغان).

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

type merchantSetupRow struct {
	ID                    string     `json:"id"`
	Name                  string     `json:"name"`
	Status                string     `json:"status"`
	Phone                 string     `json:"phone"`
	HasHours              bool       `json:"has_hours"`
	HasAddress            bool       `json:"has_address"`
	HasLocation           bool       `json:"has_location"`
	SettlementMethod      string     `json:"settlement_method"`
	SettlementConfirmedAt *time.Time `json:"settlement_confirmed_at"`
	SettlementConfirmedBy *string    `json:"settlement_confirmed_by"`
	AcceptsReturns        bool       `json:"accepts_returns"`
	ReturnsConfirmedAt    *time.Time `json:"returns_confirmed_at"`
	ReturnsConfirmedBy    *string    `json:"returns_confirmed_by"`
	// Missing **ما ينقصه بالترتيب** — hours/address/location/settlement/returns.
	Missing []string `json:"missing"`
	// Message **نصُّ التذكير على مقاسه** — يُرسَل كما هو بالواتساب أو بالإشعار.
	Message string `json:"message"`
	// WhatsAppURL رابطُ wa.me بالرسالة جاهزةً — **يفتحه الموظّفُ ويرسل بيده.**
	WhatsAppURL string `json:"whatsapp_url"`
}

func (m *merchantSetupRow) fill() {
	m.Missing = []string{}
	if !m.HasHours {
		m.Missing = append(m.Missing, "hours")
	}
	if !m.HasAddress {
		m.Missing = append(m.Missing, "address")
	}
	if !m.HasLocation {
		m.Missing = append(m.Missing, "location")
	}
	if m.SettlementConfirmedAt == nil {
		m.Missing = append(m.Missing, "settlement")
	}
	if m.ReturnsConfirmedAt == nil {
		m.Missing = append(m.Missing, "returns")
	}
	m.Message = setupMessage(m.Name, m.Missing)
	if p := strings.TrimLeft(strings.TrimSpace(m.Phone), "+"); p != "" && m.Message != "" {
		m.WhatsAppURL = "https://wa.me/" + p + "?text=" + url.QueryEscape(m.Message)
	}
}

// setupMessage **رسالةُ التذكير — تذكر الناقصَ وحدَه ولماذا يلزم.** وفارغةٌ لمتجرٍ مكتمل.
func setupMessage(name string, missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	has := func(k string) bool {
		for _, x := range missing {
			if x == k {
				return true
			}
		}
		return false
	}
	var b strings.Builder
	b.WriteString("أهلاً " + name + " 👋\n")
	var fix []string
	if has("hours") {
		fix = append(fix, "• *ساعات الدوام*: بدونها متجرك بيبين فاتح ٢٤ ساعة، وممكن يجيك طلب وإنت مسكّر.")
	}
	if has("address") {
		fix = append(fix, "• *عنوان المتجر*: حتى السائق والزبون يعرفوا وين محلّك.")
	}
	if has("location") {
		fix = append(fix, "• *موقعك على الخريطة*: حتى السائق يوصلك بالضبط بلا ما يضيع.")
	}
	if len(fix) > 0 {
		b.WriteString("حتى توصلك الطلبات صح، ناقص عندك:\n")
		b.WriteString(strings.Join(fix, "\n"))
		b.WriteString("\nظبّطهن من تطبيقك ← «متجري».\n")
	}
	var ask []string
	if has("settlement") {
		ask = append(ask, "بدك تستلم مصاريك *نقدي من السائق طلب بطلب* ولا *على محفظتك بالتطبيق*؟")
	}
	if has("returns") {
		ask = append(ask, "إذا الزبون ما استلم الطلب، بتقبل *ترجيع الغرض* إلك؟")
	}
	if len(ask) > 0 {
		if len(fix) > 0 {
			b.WriteString("\n")
		}
		if len(ask) == 1 {
			b.WriteString("وسؤال:\n" + ask[0] + "\n")
		} else {
			b.WriteString("وسؤالين:\n١. " + ask[0] + "\n٢. " + ask[1] + "\n")
		}
		b.WriteString("فيك تختار من تطبيقك ← «متجري»، أو ردّ علينا هون 🙏")
	}
	return strings.TrimRight(b.String(), "\n")
}

const merchantSetupSelect = `
	SELECT m.id, m.name, m.status,
	       COALESCE(NULLIF(u.whatsapp_phone::text, ''), NULLIF(u.phone::text, ''), m.phone::text, ''),
	       EXISTS (SELECT 1 FROM merchant_hours h WHERE h.merchant_id = m.id),
	       btrim(m.address_text) <> '', m.location IS NOT NULL,
	       m.settlement_method, m.settlement_confirmed_at, m.settlement_confirmed_by,
	       m.accepts_returns, m.returns_confirmed_at, m.returns_confirmed_by,
	       m.owner_user_id
	FROM merchants m
	LEFT JOIN users u ON u.id = m.owner_user_id`

func (s *Server) merchantSetupRows(ctx context.Context, where string, args ...any) ([]merchantSetupRow, []*string, error) {
	rows, err := s.pg.Query(ctx, merchantSetupSelect+` `+where+` ORDER BY m.created_at DESC`, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []merchantSetupRow{}
	owners := []*string{}
	for rows.Next() {
		var m merchantSetupRow
		var owner *string
		if err := rows.Scan(&m.ID, &m.Name, &m.Status, &m.Phone, &m.HasHours, &m.HasAddress,
			&m.HasLocation, &m.SettlementMethod, &m.SettlementConfirmedAt, &m.SettlementConfirmedBy,
			&m.AcceptsReturns, &m.ReturnsConfirmedAt, &m.ReturnsConfirmedBy, &owner); err != nil {
			return nil, nil, err
		}
		m.fill()
		out = append(out, m)
		owners = append(owners, owner)
	}
	return out, owners, rows.Err()
}

// handleMerchantSetupStatus **كلُّ المتاجر وما ينقص كلّاً منها.**
func (s *Server) handleMerchantSetupStatus(w http.ResponseWriter, r *http.Request) {
	list, _, err := s.merchantSetupRows(r.Context(), ``)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"stores": list})
}

// handleMerchantSetupReminder **إشعارٌ في تطبيق صاحب المتجر بما ينقصه** — بيد الموظّف.
func (s *Server) handleMerchantSetupReminder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	list, owners, err := s.merchantSetupRows(r.Context(), `WHERE m.id = $1`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(list) == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	m, owner := list[0], owners[0]
	if m.Message == "" || owner == nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"sent": false})
		return
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: *owner, Kind: notifications.KindAccount,
		Title: "متجرك بحاجة لإكمال الإعداد", Body: m.Message,
		Entity: "merchant", EntityID: m.ID, Apps: []string{notifications.AppMerchant},
	})
	s.auditCtx(r.Context(), userIDFrom(r), clientIP(r), "ops.merchant_setup_reminder", "merchant", m.ID,
		map[string]any{"missing": m.Missing})
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}
