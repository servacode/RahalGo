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

type addressInput struct {
	Label       string   `json:"label"`
	AddressText string   `json:"address_text"`
	Lat         *float64 `json:"lat"`
	Lng         *float64 `json:"lng"`
	IsDefault   *bool    `json:"is_default"`
}

type address struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	AddressText string  `json:"address_text"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	IsDefault   bool    `json:"is_default"`
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
		SELECT id, label, address_text,
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
		if err := rows.Scan(&a.ID, &a.Label, &a.AddressText, &a.Lat, &a.Lng, &a.IsDefault); err != nil {
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
	if strings.TrimSpace(req.Label) == "" || strings.TrimSpace(req.AddressText) == "" ||
		req.Lat == nil || req.Lng == nil {
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
	var id string
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO user_addresses (user_id, label, address_text, location, is_default)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($5, $4), 4326)::geography, $6)
		RETURNING id`,
		uid, clip(req.Label, 60), clip(req.AddressText, 300), *req.Lat, *req.Lng, makeDefault).
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
