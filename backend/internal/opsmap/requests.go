package opsmap

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **طلباتُ التغطية — البندان ١٦ و٣٥**
// ══════════════════════════════════════════════════════════════════════
//
// # وأصدقُ إشارةِ طلبٍ عندنا
//
// **من فتح التطبيقَ فوجد أنّنا لا نصله ثمّ ضغط «اطلب تغطية منطقتي»**
// قال لنا ما لا يقوله أيُّ تقرير: **يريد الخدمةَ ولا يملكها.**
//
// **ولا `CRM` كامل** (البند ٣٥) — عرضٌ وترشيحٌ وفتحٌ وتبديلُ حالٍ وملاحظة.

// RequestStatus حالُ طلب التغطية.
const (
	ReqNew       = "new"
	ReqReviewing = "reviewing"
	ReqPlanned   = "planned"
	ReqCovered   = "covered"
	ReqRejected  = "rejected"
)

// requestStates الحالاتُ المقبولة — **وهي عينُ قيد القاعدة.**
//
// **ومن أضاف حالاً هنا ونسي القيدَ سقط النداءُ بـ`SQLSTATE`** — ويُسقطه
// `TestOpsMap_RequestStatesMatchConstraint`.
var requestStates = []string{ReqNew, ReqReviewing, ReqPlanned, ReqCovered, ReqRejected}

// ValidRequestStatus **أحالٌ معروفة؟**
func ValidRequestStatus(s string) bool {
	for _, x := range requestStates {
		if x == s {
			return true
		}
	}
	return false
}

// RequestStates الحالاتُ كلُّها بترتيبها.
func RequestStates() []string { return append([]string(nil), requestStates...) }

// CoverageRequest طلبُ تغطيةٍ كما تراه الإدارة.
//
// # ولا بياناتٍ شخصيّةٍ على الخريطة (البند ١٧)
//
// **لا اسمَ ولا هاتفَ** — **النقطةُ والعنوانُ النصّيُّ يكفيان لقرار
// التوسّع.** ومن أراد أن يعرف من طلب فتح الحسابَ بمعرّفه.
type CoverageRequest struct {
	ID      string  `json:"id"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
	Status  string  `json:"status"`
	Source  string  `json:"source"`

	CityName     *string `json:"city,omitempty"`
	DistrictName *string `json:"district,omitempty"`

	// UserID **معرِّفٌ لا اسم** — ومن أراد الاسمَ فتح الحساب.
	UserID *string `json:"user_id,omitempty"`
	// Note **ملاحظةُ المكتب** — داخليّةٌ لا يراها من طلب.
	Note string `json:"note"`

	CreatedAt time.Time  `json:"created_at"`
	DecidedAt *time.Time `json:"decided_at,omitempty"`
}

// RequestFilter مُرشِّحاتُ طبقة الطلبات.
type RequestFilter struct {
	Status string
	CityID string
	Since  *time.Time
	Until  *time.Time
}

// CoverageRequests طلباتُ المشهد.
func CoverageRequests(ctx context.Context, q Querier, box *BBox,
	f RequestFilter, limit int) ([]CoverageRequest, error) {

	var where []string
	var args []any
	add := func(cond string, vals ...any) {
		where = append(where, cond)
		args = append(args, vals...)
	}
	if box != nil {
		if !box.Valid() {
			return nil, fmt.Errorf("مستطيلُ مشهدٍ غيرُ معقول")
		}
		add(box.SQL("r.at", len(args)+1), box.Args()...)
	}
	if f.Status != "" {
		if !ValidRequestStatus(f.Status) {
			return nil, httpx.NewError(http.StatusBadRequest, "bad_status", "errors.bad_request")
		}
		add(fmt.Sprintf("r.status = $%d", len(args)+1), f.Status)
	}
	if f.CityID != "" {
		add(fmt.Sprintf("r.city_id = $%d", len(args)+1), f.CityID)
	}
	if f.Since != nil {
		add(fmt.Sprintf("r.created_at >= $%d", len(args)+1), *f.Since)
	}
	if f.Until != nil {
		add(fmt.Sprintf("r.created_at < $%d", len(args)+1), *f.Until)
	}
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}

	sql := `
		SELECT r.id::text,
		       ST_Y(r.at::geometry), ST_X(r.at::geometry),
		       r.address_text, r.status, r.source,
		       c.name, d.name, r.user_id::text, r.note,
		       r.created_at, r.decided_at
		FROM coverage_requests r
		LEFT JOIN cities c ON c.id = r.city_id
		LEFT JOIN districts d ON d.id = r.district_id`
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	sql += " ORDER BY r.created_at DESC LIMIT " + fmt.Sprint(limit)

	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CoverageRequest{}
	for rows.Next() {
		var x CoverageRequest
		if err := rows.Scan(&x.ID, &x.Lat, &x.Lng, &x.Address, &x.Status, &x.Source,
			&x.CityName, &x.DistrictName, &x.UserID, &x.Note,
			&x.CreatedAt, &x.DecidedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// NewRequest ما يصل من زرّ «اطلب تغطية منطقتي».
type NewRequest struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
	Source  string  `json:"source"`
}

// CreateRequest يسجّل طلبَ تغطية.
//
// # والمدينةُ والمنطقةُ تُستنتجان لا تُسألان
//
// **من يضغط الزرَّ لا يعرف في أيّ «منطقةٍ إداريّة» هو** — **والمكتبُ
// يريد أن يعرف.** فتُستنتج المدينةُ من دائرتها، **وتُترك فارغةً إن لم
// تقع في واحدة** — وهي إشارةٌ في ذاتها: طلبٌ خارجَ كلّ مدنِنا.
func CreateRequest(ctx context.Context, e Execer, userID string, in NewRequest) (string, error) {
	// ══════════════════════════════════════════════════════════════════
	// **وكاتبٌ واحدٌ للجدول** (`CR`، ٢٠٢٦-٠٩-١٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// **وكان هذا يكتب صفّاً بلا نوعٍ ولا خليّة** — **فصفٌّ لا يُفرَّد
	// ولا يُدمَج فيه جديد**، **وهو الانقسامُ بعينه في جدولٍ واحد.**
	//
	// **فصار يمرّ بـ`Record`** — النوعُ والخليّةُ والمحافظةُ والتفرّد.
	res, err := Record(ctx, e, userID, Signal{
		Kind:    KindCoverage,
		Lat:     in.Lat,
		Lng:     in.Lng,
		Address: in.Address,
		Source:  in.Source,
	})
	if err != nil {
		return "", err
	}
	return res.ID, nil
}

// UpdateRequest يبدّل حالَ طلبٍ أو ملاحظتَه (البند ٣٥).
//
// **ومن بدّل الحالَ يُسجَّل اسمُه ووقتُه** — **وقرارٌ بلا صاحبٍ لا
// يُراجَع.**
func UpdateRequest(ctx context.Context, e Execer, id, status, note, byUser string) error {
	if status != "" && !ValidRequestStatus(status) {
		return httpx.NewError(http.StatusBadRequest, "bad_status", "errors.bad_request")
	}
	tag, err := e.Exec(ctx, `
		UPDATE coverage_requests SET
		    status     = COALESCE(NULLIF($2, ''), status),
		    note       = COALESCE($3, note),
		    decided_by = CASE WHEN $2 <> '' THEN $4::uuid ELSE decided_by END,
		    decided_at = CASE WHEN $2 <> '' THEN now() ELSE decided_at END,
		    updated_at = now()
		WHERE id = $1::uuid`, id, status, note, byUser)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}
