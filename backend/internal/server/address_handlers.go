package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// عناوين الزبون المحفوظة — يكتبها مرّة ويستعملها دائماً.

// سقف العناوين لكل مستخدم — صار إعداداً (`customers.max_addresses`).
//
// ليس ترقيماً بل حاجزُ معنى: من له عشرون عنواناً لا يجد عنوانه بينها، فتنقلب
// الميزة على نفسها. والعشرة سخيّة لأي استعمال واقعي — لكنّ من يعرف زبائنه
// أَولى بتقديرها من مبرمجٍ كتب رقماً.

var errTooManyAddresses = httpx.NewError(http.StatusConflict, "too_many_addresses", "errors.too_many_addresses")

// addressInput **ما يُرسَل — أجزاءٌ لا سطر.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٨.)
//
// **ولا `address_text` فيه**: يُركَّب في المحرّك من الأجزاء —
// **ولو أُرسل لَصار للعنوان مصدران يفترقان**: يعدّل صاحبُه الشارعَ
// وتبقى النسخةُ المركَّبةُ على الشارع القديم.
type addressInput struct {
	AreaBuilding string `json:"area_building"`
	Street       string `json:"street"`
	Floor        string `json:"floor"`
	// Kind **رمزٌ لا كلمة** — `home` أو `work` أو `other`.
	Kind      string   `json:"kind"`
	Lat       *float64 `json:"lat"`
	Lng       *float64 `json:"lng"`
	IsDefault *bool    `json:"is_default"`
}

type address struct {
	ID           string `json:"id"`
	AreaBuilding string `json:"area_building"`
	Street       string `json:"street"`
	Floor        string `json:"floor"`
	Kind         string `json:"kind"`
	// AddressText **السطرُ المركَّب** — يقرؤه السائقُ ويُحفظ لقطةً في
	// الطلب. **مشتقٌّ لا مكتوب.**
	AddressText string  `json:"address_text"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	IsDefault   bool    `json:"is_default"`
}

// addressLine **يركّب السطرَ من أجزائه — في موضعٍ واحد.**
//
// **والسائقُ يقرأ سطراً لا ثلاثةَ حقول** — وهو ما يُحفظ في الطلب.
//
// **وما فرغ يسقط ولا يترك فاصلةً معلَّقة**: «المزة، بناء ١٤ · ·» تُقرأ
// عنواناً ناقصاً لا عنواناً بلا طابق.
func addressLine(area, street, floor string) string {
	parts := []string{}
	for _, p := range []string{strings.TrimSpace(area), strings.TrimSpace(street)} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if f := strings.TrimSpace(floor); f != "" {
		parts = append(parts, "الطابق "+f)
	}
	return strings.Join(parts, " · ")
}

// addressKind **يقبل الأنواعَ الثلاثةَ ويردّ ما عداها إلى `other`.**
//
// **ورمزٌ مجهولٌ لا يُسقط حفظَ عنوان** — يُقرأ «أخرى» ويبقى العنوانُ
// صالحاً، **وعميلٌ قديمٌ لا يعرف الأنواعَ لا يُمنع من الحفظ.**
func addressKind(k string) string {
	switch strings.TrimSpace(k) {
	case "home", "work":
		return strings.TrimSpace(k)
	default:
		return "other"
	}
}

func (s *Server) handleMyAddresses(w http.ResponseWriter, r *http.Request) {
	s.writeAddresses(w, r, userIDFrom(r))
}

// handleAdminUserAddresses عناوينُ زبونٍ بعينه — **في ملفّه لا في بحثٍ عنه.**
//
// من يتّصل به زبونٌ يقول «طلبي لم يصل» يحتاج أن يرى **أين يسكن** قبل أن يسأل.
// **وكانت لا تُقرأ إلّا من حساب صاحبها** — فتُقرأ من الطلب وحدَه، ومن لا طلبَ
// له اليومَ لا عنوانَ له عندنا.
//
// **وقراءةٌ لا كتابة**: عنوانُ بيتِ إنسانٍ يكتبه هو.
func (s *Server) handleAdminUserAddresses(w http.ResponseWriter, r *http.Request) {
	s.writeAddresses(w, r, chi.URLParam(r, "id"))
}

// writeAddresses **استعلامٌ واحدٌ لموضعين** — ولو نُسخ لَافترقا حين يُزاد حقل.
func (s *Server) writeAddresses(w http.ResponseWriter, r *http.Request, userID string) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT id, area_building, street, floor, kind, address_text,
		       ST_Y(location::geometry), ST_X(location::geometry), is_default
		FROM user_addresses WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC`, userID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []address{}
	for rows.Next() {
		var a address
		if err := rows.Scan(&a.ID, &a.AreaBuilding, &a.Street, &a.Floor, &a.Kind,
			&a.AddressText, &a.Lat, &a.Lng, &a.IsDefault); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, a)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateAddress(w http.ResponseWriter, r *http.Request) {
	req, err := decode[addressInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	uid := userIDFrom(r)
	// **والمنطقةُ والمبنى وحدَهما إلزاميّان** — الشارعُ يُنسى في أحياءٍ
	// بلا لافتات، **والطابقُ لا يخصّ بيتاً أرضيّا.**
	if strings.TrimSpace(req.AreaBuilding) == "" || req.Lat == nil || req.Lng == nil {
		s.respondErr(w, errValidation)
		return
	}

	var count int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM user_addresses WHERE user_id = $1`, uid).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	if int64(count) >= s.settings.GetInt(r.Context(), "customers.max_addresses") {
		s.respondErr(w, errTooManyAddresses)
		return
	}

	// أول عنوان يصير الافتراضي بلا سؤال — من له واحد لا يُخيَّر فيه
	makeDefault := count == 0 || (req.IsDefault != nil && *req.IsDefault)

	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	// إسقاط الافتراضي السابق **داخل المعاملة**: الفهرس الفريد يرفض اثنين، فلو
	// وقع الإسقاط خارجها لبقي المستخدم بلا افتراضي إن فشل الإدراج.
	if makeDefault {
		if _, err := tx.Exec(r.Context(),
			`UPDATE user_addresses SET is_default = false WHERE user_id = $1 AND is_default`, uid); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	area := clip(strings.TrimSpace(req.AreaBuilding), 160)
	street := clip(strings.TrimSpace(req.Street), 120)
	floor := clip(strings.TrimSpace(req.Floor), 20)
	var id string
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO user_addresses
		    (user_id, label, area_building, street, floor, kind, address_text, location, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7,
		        ST_SetSRID(ST_MakePoint($9, $8), 4326)::geography, $10)
		RETURNING id`,
		// **و`label` يبقى للعمود القديم** — مطلوبٌ غيرَ فارغٍ في القاعدة،
		// **ولا يُقرأ في شاشة**: النوعُ في `kind` والعنوانُ في أجزائه.
		uid, addressKind(req.Kind), area, street, floor, addressKind(req.Kind),
		addressLine(area, street, floor), *req.Lat, *req.Lng, makeDefault).
		Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

// handleUpdateAddress **يعدّل عنواناً محفوظاً.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٨: «قسمُ حسابي يُظهر العناوينَ المحفوظةَ ويمكن
//
//	تعديلها أو حذفها».)
//
// **ولم يكن له باب**: من أخطأ في طابقه حذف العنوانَ وأنشأه من جديد —
// **فيفقد كونَه الافتراضيَّ ويعيد التقاطَ نقطته على الخريطة.**
//
// **والنقطةُ اختياريّةٌ في التعديل**: أكثرُ التصحيح نصٌّ (طابقٌ أو شارع)،
// **ومن أُلزم بإعادة فتح الخريطة لتصحيح حرفٍ لا يصحّح.**
func (s *Server) handleUpdateAddress(w http.ResponseWriter, r *http.Request) {
	req, err := decode[addressInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if strings.TrimSpace(req.AreaBuilding) == "" {
		s.respondErr(w, errValidation)
		return
	}
	area := clip(strings.TrimSpace(req.AreaBuilding), 160)
	street := clip(strings.TrimSpace(req.Street), 120)
	floor := clip(strings.TrimSpace(req.Floor), 20)

	// **والنقطةُ تُبدَّل إن أُرسلت وحدَها** — `COALESCE` على الموضع لا
	// يصلح مع `geography`، فيُفصَل الشرطُ في الاستعلام.
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE user_addresses SET
		    label         = $3,
		    kind          = $3,
		    area_building = $4,
		    street        = $5,
		    floor         = $6,
		    address_text  = $7,
		    location      = CASE WHEN $8::double precision IS NULL THEN location
		                         ELSE ST_SetSRID(ST_MakePoint($9, $8), 4326)::geography END,
		    updated_at    = now()
		WHERE id = $1 AND user_id = $2`,
		chi.URLParam(r, "id"), userIDFrom(r), addressKind(req.Kind),
		area, street, floor, addressLine(area, street, floor), req.Lat, req.Lng)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (s *Server) handleDeleteAddress(w http.ResponseWriter, r *http.Request) {
	tag, err := s.pg.Exec(r.Context(),
		`DELETE FROM user_addresses WHERE id = $1 AND user_id = $2`,
		chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// handleSetDefaultAddress يجعل عنواناً هو الافتراضي.
func (s *Server) handleSetDefaultAddress(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	id := chi.URLParam(r, "id")

	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if _, err := tx.Exec(r.Context(),
		`UPDATE user_addresses SET is_default = false WHERE user_id = $1 AND is_default`, uid); err != nil {
		s.respondErr(w, err)
		return
	}
	tag, err := tx.Exec(r.Context(), `
		UPDATE user_addresses SET is_default = true, updated_at = now()
		WHERE id = $1 AND user_id = $2`, id, uid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
