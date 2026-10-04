package server

// ══════════════════════════════════════════════════════════════════════
// **شريطُ الطوارئ الأحمر أعلى كلّ صفحة — وزرُّ «استلمتها»**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤: «شريطٌ أحمرُ أعلى كلّ صفحة… حتّى يضغط موظّفٌ
//  استلمتها».)
//
// # ما يُعرض فيه
//
//   - **طارئُ سائقٍ مفتوحٌ لم يستلمه أحد** — يزول بضغطة «استلمتها» ويبقى
//     مفتوحاً في صفحة الطوارئ حتّى يُحلّ.
//   - **متجرٌ أُغلق طارئاً ولم يُستلَم إغلاقُه** — كذلك.
//   - **وتوقّفُ المنصّة** — حالٌ لا واقعة: **يبقى ما دامت المنصّةُ متوقّفة**،
//     ولا يُستلَم فيُخفى وهي لا تستقبل طلباً.
//
// # والاستلامُ غيرُ الإغلاق
//
// **من ضغط «استلمتها» قال «أنا عليه»** — والطارئُ لا يُغلق إلّا بحلٍّ مكتوب
// (`handleResolveEmergency`). **وخلطُهما يجعل الضغطةَ الأولى تمحو السؤالَ
// «ماذا جرى؟».**

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

type bannerDriverEmergency struct {
	ID          string    `json:"id"`
	DriverName  string    `json:"driver_name"`
	OrderNumber *int64    `json:"order_number"`
	CreatedAt   time.Time `json:"created_at"`
}

type bannerStoreClosure struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	ClosedAt *time.Time `json:"closed_at"`
}

type emergencyBanner struct {
	Drivers []bannerDriverEmergency `json:"drivers"`
	Stores  []bannerStoreClosure    `json:"stores"`
	// PlatformHalted **المنصّةُ موقوفةٌ الآن بيد المالك** (`service_closure`).
	PlatformHalted bool   `json:"platform_halted"`
	HaltMessage    string `json:"halt_message"`
}

// emergencyBannerData **ما لم يُستلَم بعد** — للشريط ولرئيسيّة المدير معاً.
func (s *Server) emergencyBannerData(ctx context.Context) (emergencyBanner, error) {
	b := emergencyBanner{Drivers: []bannerDriverEmergency{}, Stores: []bannerStoreClosure{}}
	rows, err := s.pg.Query(ctx, `
		SELECT e.id::text, COALESCE(NULLIF(u.full_name, ''), u.phone::text, ''),
		       o.number, e.created_at
		FROM driver_emergencies e
		JOIN users u ON u.id = e.driver_id
		LEFT JOIN orders o ON o.id = e.order_id
		WHERE e.status = 'open' AND e.acknowledged_at IS NULL
		ORDER BY e.created_at`)
	if err != nil {
		return b, err
	}
	for rows.Next() {
		var x bannerDriverEmergency
		if err := rows.Scan(&x.ID, &x.DriverName, &x.OrderNumber, &x.CreatedAt); err != nil {
			rows.Close()
			return b, err
		}
		b.Drivers = append(b.Drivers, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return b, err
	}

	rows, err = s.pg.Query(ctx, `
		SELECT m.id::text, m.name, m.emergency_closed_at
		FROM merchants m
		WHERE m.emergency_closed AND m.emergency_ack_at IS NULL
		ORDER BY m.emergency_closed_at NULLS FIRST, m.name`)
	if err != nil {
		return b, err
	}
	for rows.Next() {
		var x bannerStoreClosure
		if err := rows.Scan(&x.ID, &x.Name, &x.ClosedAt); err != nil {
			rows.Close()
			return b, err
		}
		b.Stores = append(b.Stores, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return b, err
	}

	if s.platform != nil {
		st, err := s.platform.State(ctx, s.pg)
		if err != nil {
			return b, err
		}
		if st.Reason == platform.ReasonTemporarilyUnavailable {
			b.PlatformHalted = true
			b.HaltMessage = st.Message
		}
	}
	return b, nil
}

// handleEmergencyBanner **الشريطُ الأحمر** — لكلّ من يملك الطوارئ.
func (s *Server) handleEmergencyBanner(w http.ResponseWriter, r *http.Request) {
	b, err := s.emergencyBannerData(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

// handleAckEmergency **«استلمتها» على طارئ سائق.**
//
// **ومرّةً واحدة**: من ضغطها ثانيةً أو بعد غيره يُردّ ٤٠٤ — **فلا يُكتب
// اسمُ الثاني فوق الأوّل**، والسجلُّ يقول من استلم أوّلاً.
func (s *Server) handleAckEmergency(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE driver_emergencies
		SET acknowledged_at = now(), acknowledged_by = $2
		WHERE id = $1 AND status = 'open' AND acknowledged_at IS NULL`,
		id, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **وإغلاقُ المتجر في الصندوق نفسِه** (0280) — فيُستلَم في الموضعين معاً.
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE merchants m SET emergency_ack_at = now(), emergency_ack_by = $2
		  FROM driver_emergencies e
		 WHERE e.id = $1 AND e.kind = 'store_closure' AND m.id = e.merchant_id
		   AND m.emergency_ack_at IS NULL`, id, userIDFrom(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.emergency_ack", "emergency", id, map[string]any{})
	s.touch("emergency", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"acknowledged": true})
}

// handleAckStoreEmergency **«استلمتها» على إغلاقٍ طارئٍ لمتجر.**
func (s *Server) handleAckStoreEmergency(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE merchants
		SET emergency_ack_at = now(), emergency_ack_by = $2
		WHERE id = $1 AND emergency_closed AND emergency_ack_at IS NULL`,
		id, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE driver_emergencies SET acknowledged_at = now(), acknowledged_by = $2
		 WHERE merchant_id = $1 AND kind = 'store_closure' AND status = 'open'
		   AND acknowledged_at IS NULL`, id, userIDFrom(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.store_emergency_ack", "merchant", id, map[string]any{})
	s.touch("emergency", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"acknowledged": true})
}
