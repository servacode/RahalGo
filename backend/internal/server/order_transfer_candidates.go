package server

// ══════════════════════════════════════════════════════════════════════
// **المتاجرُ المرشّحةُ لتحويل الطلب — مرتّبةً بما تقدّمه منه** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
// كان الموظّفُ يختار متجراً من قائمةٍ بالأسماء **ثمّ يكتشف بعد الضغط** أنّ صنفاً
// لا يطابق بالحرف — «رز مصري» هنا و«أرز مصري» هناك. **فيعيد ويجرّب غيرَه أعمى.**
//
// والمالك: «لازم نلاقي حلّ يكون نفس المتجر يقدّم نفس الأصناف». **فتُعرض المتاجرُ
// مرتّبةً بكم صنفاً من أصناف الطلب تقدّم** («٣ من ٣») **ثمّ بقربها** — من موضع
// السائق إن كان معه الطلب، **وإلّا من المتجر الحاليّ.**
//
// **ولكلّ متجرٍ مقابلٌ مقترحٌ لكلّ صنف** بالمطابقة الذكيّة (`itemmatch`) —
// **يقترح ولا يقرّر**: الموظّفُ يؤكّده أو يختار غيرَه من قائمة المتجر،
// **وتلك القائمةُ تأتي حين يُطلب متجرٌ بعينه** (`?merchant_id=`) لا مع كلّ متجر.

import (
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/itemmatch"
)

type transferOrderItem struct {
	OrderItemID   string `json:"order_item_id"`
	Name          string `json:"name"`
	Qty           int    `json:"qty"`
	MerchantPrice int64  `json:"merchant_price"`
}

type transferMenuItem struct {
	MenuItemID    string `json:"menu_item_id"`
	Name          string `json:"name"`
	MerchantPrice int64  `json:"merchant_price"`
}

type transferProposal struct {
	OrderItemID string `json:"order_item_id"`
	// **و`null` لا مقابل** — والموظّفُ يختار بيده إن أراد.
	Match *transferMenuItem `json:"match"`
	Score float64           `json:"score"`
}

type transferCandidate struct {
	MerchantID string             `json:"merchant_id"`
	Name       string             `json:"name"`
	Matched    int                `json:"matched"`
	Total      int                `json:"total"`
	DistanceM  *float64           `json:"distance_m"`
	Items      []transferProposal `json:"items"`
	// **قائمةُ المتجر كاملةً** — حين يُطلب وحدَه فقط.
	Menu []transferMenuItem `json:"menu,omitempty"`
}

// handleTransferCandidates — `GET /admin/orders/{id}/transfer-candidates[?merchant_id=]`.
func (s *Server) handleTransferCandidates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orderID := chi.URLParam(r, "id")
	if !isUUID(orderID) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	only := r.URL.Query().Get("merchant_id")
	if only != "" && !isUUID(only) {
		s.respondErr(w, errValidation)
		return
	}

	// **نقطةُ القياس**: موضعُ السائق إن كان الطلبُ معه وموضعُه معروف —
	// **فهو من سيقود إلى المتجر الجديد** — وإلّا المتجرُ الحاليّ.
	var current, from string
	if err := s.pg.QueryRow(ctx, `
		SELECT o.merchant_id::text,
		       CASE WHEN u.last_location IS NOT NULL THEN 'driver'
		            WHEN m.location IS NOT NULL THEN 'store' ELSE '' END
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		LEFT JOIN users u ON u.id = o.driver_id
		WHERE o.id = $1`, orderID).Scan(&current, &from); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	items := []transferOrderItem{}
	rows, err := s.pg.Query(ctx, `
		SELECT id::text, name, qty, merchant_price FROM order_items
		WHERE order_id = $1 ORDER BY name, id`, orderID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var x transferOrderItem
		if err := rows.Scan(&x.OrderItemID, &x.Name, &x.Qty, &x.MerchantPrice); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		items = append(items, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}

	// **المتاجرُ القائمة**: فعّالةٌ، **ولا إغلاقَ طارئاً عليها**، وليست متجرَ الطلب.
	stores := []*transferCandidate{}
	byID := map[string]*transferCandidate{}
	srows, err := s.pg.Query(ctx, `
		WITH origin AS (
			SELECT COALESCE(u.last_location, m.location) AS pt
			FROM orders o
			JOIN merchants m ON m.id = o.merchant_id
			LEFT JOIN users u ON u.id = o.driver_id
			WHERE o.id = $1)
		SELECT m.id::text, m.name, ST_Distance(m.location, origin.pt)
		FROM merchants m, origin
		WHERE m.status = 'active' AND NOT m.emergency_closed AND m.id <> $2::uuid
		  AND ($3 = '' OR m.id::text = $3)`, orderID, current, only)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for srows.Next() {
		c := &transferCandidate{Total: len(items), Items: []transferProposal{}}
		if err := srows.Scan(&c.MerchantID, &c.Name, &c.DistanceM); err != nil {
			srows.Close()
			s.respondErr(w, err)
			return
		}
		stores = append(stores, c)
		byID[c.MerchantID] = c
	}
	srows.Close()
	if err := srows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	if only != "" && len(stores) == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	// **الأصنافُ المتاحةُ في المتاجر المرشّحة** — مرّةً واحدة لا لكلّ متجر.
	// **والمُقرّةُ وحدَها** (`approved`): صنفٌ لم تراجعه المنصّةُ لا يُعرض للزبون،
	// **فلا يُشترى له.**
	menus := map[string][]itemmatch.Candidate{}
	mrows, err := s.pg.Query(ctx, `
		SELECT mi.merchant_id::text, mi.id::text, mi.name, mi.merchant_price
		FROM menu_items mi
		JOIN merchants m ON m.id = mi.merchant_id
		WHERE mi.available AND mi.approved AND m.status = 'active' AND NOT m.emergency_closed
		  AND m.id <> $1::uuid AND ($2 = '' OR m.id::text = $2)
		ORDER BY mi.name, mi.id`, current, only)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for mrows.Next() {
		var mid string
		var c itemmatch.Candidate
		if err := mrows.Scan(&mid, &c.ID, &c.Name, &c.Price); err != nil {
			mrows.Close()
			s.respondErr(w, err)
			return
		}
		menus[mid] = append(menus[mid], c)
	}
	mrows.Close()
	if err := mrows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}

	for id, menu := range menus {
		st := byID[id]
		if st == nil {
			continue
		}
		for _, it := range items {
			p := transferProposal{OrderItemID: it.OrderItemID}
			if i, sc := itemmatch.Best(it.Name, it.MerchantPrice, menu); i >= 0 {
				p.Match = &transferMenuItem{
					MenuItemID: menu[i].ID, Name: menu[i].Name, MerchantPrice: menu[i].Price,
				}
				p.Score = sc
				st.Matched++
			}
			st.Items = append(st.Items, p)
		}
		if only != "" {
			for _, c := range menu {
				st.Menu = append(st.Menu, transferMenuItem{
					MenuItemID: c.ID, Name: c.Name, MerchantPrice: c.Price,
				})
			}
		}
	}
	// **ومتجرٌ بلا قائمةٍ متاحةٍ يُعرض بلا مقابل** — لا يُخفى: قد يُختار بيدٍ لسببٍ آخر.
	for _, st := range stores {
		if len(st.Items) == 0 {
			for _, it := range items {
				st.Items = append(st.Items, transferProposal{OrderItemID: it.OrderItemID})
			}
		}
		if only != "" && st.Menu == nil {
			st.Menu = []transferMenuItem{}
		}
	}

	// **الأكثرُ تقديماً أوّلاً، ثمّ الأقرب، ثمّ الاسم** — ومجهولُ الموضع آخراً.
	sort.SliceStable(stores, func(i, j int) bool {
		a, b := stores[i], stores[j]
		if a.Matched != b.Matched {
			return a.Matched > b.Matched
		}
		if (a.DistanceM == nil) != (b.DistanceM == nil) {
			return a.DistanceM != nil
		}
		if a.DistanceM != nil && *a.DistanceM != *b.DistanceM {
			return *a.DistanceM < *b.DistanceM
		}
		return a.Name < b.Name
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"distance_from": from,
		"items":         items,
		"stores":        stores,
	})
}
