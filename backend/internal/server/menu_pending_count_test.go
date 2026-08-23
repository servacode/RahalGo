package server

// **وعدُّ الطابور على الطابور — لا على المعروض.**
//
// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
//
// **كان `count` هو طولَ القائمة نفسِها** — فطابورٌ فيه ثلاثمئةٌ وأربعةَ عشرَ
// يقول «مئتان». **والحقلُ يكذب مرّتين**: يُسمّى عدّاً وهو حدّ.
//
// **وسقفٌ صامتٌ في طابور مراجعةٍ أخطرُ من سواه**: **صنفٌ لا يُوافَق عليه
// لأنّ أحداً لم يره** — ولا يعرف المتجرُ لماذا لم يُقرّ، **ولا يعرف
// المكتبُ أنّ وراء المعروض شيئاً.**

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPendingMenu_CountsWholeQueueNotThePage(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	var menuSectionID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قائمةُ الطابور')
		RETURNING id::text`, f.merchantID).Scan(&menuSectionID); err != nil {
		t.Fatalf("تعذّرت القائمة: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM menu_items WHERE section_id = $1`, menuSectionID)
		_, _ = f.pool.Exec(c, `DELETE FROM menu_sections WHERE id = $1`, menuSectionID)
	})

	// **مئتان وعشرة تنتظر** — أكثرُ من السقف بعشرة، **وهي العشرةُ التي كانت
	// تختفي بلا كلمة.**
	const n = 210
	for i := range n {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO menu_items (merchant_id, section_id, platform_section_id, name, price, merchant_price, approved)
			VALUES ($1, $2, (SELECT id FROM platform_sections ORDER BY sort_order LIMIT 1), $3, 1000, 1000, false)`,
			f.merchantID, menuSectionID, fmt.Sprintf("صنفٌ منتظرٌ %03d", i)); err != nil {
			t.Fatalf("تعذّر الصنف %d: %v", i, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	c := context.WithValue(req.Context(), ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	f.srv.handlePendingMenuItems(w, req.WithContext(c))
	if w.Code != 200 {
		t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			Count int `json:"count"`
			Limit int `json:"limit"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v", err)
	}

	if len(env.Data.Items) != env.Data.Limit {
		t.Fatalf("عُرض %d والسقفُ %d", len(env.Data.Items), env.Data.Limit)
	}
	// **والعدُّ على الطابور** — وهو ما كان يقول «مئتين».
	if env.Data.Count < n {
		t.Fatalf("العدُّ %d وفي الطابور %d — **فيُظنّ نضب وفيه عشرة**",
			env.Data.Count, n)
	}
	// **والسقفُ يُرسَل** — فتقول الشاشةُ «المعروضُ ٢٠٠ من ٢١٠» **برقمٍ من
	// الخادم لا برقمٍ مكتوبٍ فيها يفترق عنه يوما.**
	if env.Data.Limit == 0 {
		t.Fatal("لا سقفَ في الردّ — **ورقمٌ مكتوبٌ في الواجهة يفترق عن رقم الخادم**")
	}
}
