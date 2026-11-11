package server

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestRepFollowUpReasons(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -10)
	recent := now.AddDate(0, 0, -2)
	cases := []struct {
		name   string
		status string
		goal   string
		hours  bool
		loc    bool
		photos int
		items  int
		last   *time.Time
		joined string
		want   []string
	}{
		{"بلا أصناف", "active", "counted", true, true, 3, 0, &recent, "2026-09-01", []string{FollowUpNoItems}},
		{"شروط ناقصة", "active", "pending", false, true, 3, 5, &recent, "2026-09-01", []string{FollowUpConditions}},
		{"صور أقلّ من الحدّ", "active", "pending", true, true, 1, 5, &recent, "2026-09-01", []string{FollowUpConditions}},
		{"لا طلب من أسبوع", "active", "counted", true, true, 3, 5, &old, "2026-09-01", []string{FollowUpNoOrders}},
		{"لم يُسلَّم له قطّ", "active", "counted", true, true, 3, 5, nil, "2026-09-01", []string{FollowUpNoOrders}},
		{"جديدٌ لم يمرّ أسبوع", "active", "counted", true, true, 3, 5, nil, "2026-11-09", []string{}},
		{"موقوف", "suspended", "pending", false, false, 0, 0, nil, "2026-09-01", []string{}},
		{"سليم", "active", "counted", true, true, 3, 5, &recent, "2026-09-01", []string{}},
		{"فارغ الأصناف لا يُقال له لا طلب", "active", "counted", true, true, 3, 0, &old, "2026-09-01", []string{FollowUpNoItems}},
	}
	for _, c := range cases {
		got := repFollowUpReasons(c.status, c.goal, c.hours, c.loc, c.photos, 3, c.items, c.last, c.joined, now)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: المتوقّع %v والوارد %v", c.name, c.want, got)
		}
	}
}

// القائمةُ ترسل follow_up مصفوفةً دائماً، وبلا أصنافٍ فيها no_items.
func TestRepMerchantsListFollowUp(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	rep := testdb.NewUser(t, f.pool, "sales")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET sales_rep_user_id = $2 WHERE id = $1`, f.merchantID, rep); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `DELETE FROM menu_items WHERE merchant_id = $1`, f.merchantID); err != nil {
		t.Skipf("يتعذّر تفريغ الأصناف في هذا العتاد: %v", err)
	}
	w := callWithID(f, f.srv.handleRepMerchants, "", rep, []string{"sales"}, "{}")
	if w.Code != http.StatusOK {
		t.Fatalf("%d: %s", w.Code, w.Body.String())
	}
	var env struct {
		Data []struct {
			ID       string    `json:"id"`
			FollowUp *[]string `json:"follow_up"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	for _, m := range env.Data {
		if m.ID != f.merchantID {
			continue
		}
		if m.FollowUp == nil {
			t.Fatal("follow_up غائب أو null")
		}
		found := false
		for _, r := range *m.FollowUp {
			found = found || r == FollowUpNoItems
		}
		if !found {
			t.Fatalf("متجرٌ بلا أصناف بلا no_items: %v", *m.FollowUp)
		}
		return
	}
	t.Fatal("المتجر غير موجود في القائمة")
}
