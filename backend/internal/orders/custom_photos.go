package orders

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// ══════════════════════════════════════════════════════════════════════
// **صورُ الطلب الخاصّ** — قرارُ المالك ٢٠٢٦-١١-١١ (هجرة ٠٥٤١)
// ══════════════════════════════════════════════════════════════════════
//
// الزبونُ يرفق صورةَ قائمة مشترياته أو الغرض (١–٣) بدل الكتابة وحدَها.
// **تُرفع قبل الطلب** (`POST /me/custom-photo`) فيُرسَل معرّفاتُها مع الطلب،
// **وتُربط في معاملة الإنشاء نفسِها.** لا مالَ هنا ولا دفتر.

// MaxCustomPhotos **سقفُ الصور للطلب الواحد.**
const MaxCustomPhotos = 3

// CustomPhotoOnlyRequest **نصُّ الطلب حين يُرسَل بالصور وحدَها** — فالعمودُ يبقى غيرَ فارغ
// وتقرؤه الإشعاراتُ والقوائمُ كما تقرأ أيَّ طلب.
const CustomPhotoOnlyRequest = "طلب بالصور المرفقة"

// ErrCustomPhotos **صورٌ لا تُقبل** — غريبةٌ أو مستعملةٌ أو فوق السقف.
var ErrCustomPhotos = httpx.NewError(http.StatusBadRequest, "custom_photos_invalid", "errors.validation")

// SignPhotoPaths **مساراتٌ إلى روابط موقَّعة** — ويرجع `[]` لا `nil`.
func SignPhotoPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			out = append(out, media.SignedURL(p))
		}
	}
	return out
}

// AttachCustomPhotosTx **يربط صورَ الزبون بطلبه** — في معاملة الإنشاء.
//
// **كلُّ صورةٍ يجب أن تكون: من صنف `custom_photo` · رفعَها هو · لم تُربط بطلب قبلَه.**
// وما خالف ذلك يُردّ كلُّه (`ErrCustomPhotos`) — لا ربطٌ جزئيّ.
func (s *Service) AttachCustomPhotosTx(ctx context.Context, q dbtx.Querier, orderID, customerID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > MaxCustomPhotos {
		return ErrCustomPhotos
	}
	seen := map[string]bool{}
	clean := make([]string, 0, len(ids))
	for _, id := range ids {
		u, err := uuid.Parse(id)
		if err != nil || seen[u.String()] {
			return ErrCustomPhotos
		}
		seen[u.String()] = true
		clean = append(clean, u.String())
	}
	var ok int
	if err := q.QueryRow(ctx, `
		SELECT count(*) FROM media m
		 WHERE m.id::text = ANY($1) AND m.kind = 'custom_photo' AND m.created_by = $2
		   AND NOT EXISTS (SELECT 1 FROM order_custom_photos p WHERE p.media_id = m.id)`,
		clean, customerID).Scan(&ok); err != nil {
		return err
	}
	if ok != len(clean) {
		return ErrCustomPhotos
	}
	for i, id := range clean {
		if _, err := q.Exec(ctx, `
			INSERT INTO order_custom_photos (order_id, media_id, position) VALUES ($1, $2, $3)`,
			orderID, id, i); err != nil {
			return err
		}
	}
	return nil
}

// CustomPhotoURLs **روابطُ صور طلباتٍ دفعةً واحدة** (لقائمة السائق) — `orderID ⇒ روابط`.
func (s *Service) CustomPhotoURLs(ctx context.Context, orderIDs []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(orderIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT p.order_id::text, m.path
		  FROM order_custom_photos p JOIN media m ON m.id = p.media_id
		 WHERE p.order_id::text = ANY($1) ORDER BY p.order_id, p.position`, orderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		out[id] = append(out[id], media.SignedURL(path))
	}
	return out, rows.Err()
}

// CustomPhotoURLsTx **صورُ طلبٍ واحد بالمنفّذ المُمرَّر** — لطلبٍ لم يُثبَّت بعد.
func (s *Service) CustomPhotoURLsTx(ctx context.Context, q dbtx.Querier, orderID string) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT m.path FROM order_custom_photos p JOIN media m ON m.id = p.media_id
		 WHERE p.order_id = $1 ORDER BY p.position`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	paths := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return SignPhotoPaths(paths), rows.Err()
}
