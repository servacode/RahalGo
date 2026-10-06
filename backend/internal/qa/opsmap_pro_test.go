package qa

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **خريطةُ العمليّات الاحترافيّة — قرارُ المالك ٢٠٢٦-١٠-٠٥**
// ══════════════════════════════════════════════════════════════════════
//
// **موظّفُ العمليّات يفتح كلَّ ما تناديه الصفحة**، **والزبائنُ خلايا لا بيوت**
// (حدٌّ أدنى ثلاثة، ولا معرّفَ ولا اسمَ ولا هاتف)، **والمكتبُ اسمٌ ودورٌ لمن
// حضر** بلا هاتف.

// opsMapPagePaths **كلُّ ما تناديه صفحةُ الخريطة** — بقراءاته وحدَها.
var opsMapPagePaths = []string{
	"/ops-map/meta", "/ops-map/summary", "/ops-map/office", "/ops-map/customers",
	"/ops-map/drivers", "/ops-map/merchants", "/ops-map/orders",
	"/ops-map/coverage", "/ops-map/coverage-requests", "/ops-map/branches",
	"/ops-map/areas", "/ops-map/demand?range=30d",
	"/ops-map/opportunities?range=30d", "/ops-map/search?q=zz",
	"/ops-map/coverage-demand/places", "/emergencies/map",
}

// TestOpsMapPro_OperationsCallsEveryPageEndpoint **كلُّ نداءٍ في الصفحة ٢٠٠
// لموظّف العمليّات ولمدير المنصّة.**
func TestOpsMapPro_OperationsCallsEveryPageEndpoint(t *testing.T) {
	hh := New(t)
	_, ops := roleUser(t, hh, "operations")
	_, owner := roleUser(t, hh, "owner_super_admin")
	_, admin := roleUser(t, hh, "admin")
	for _, who := range []struct{ name, tok string }{
		{"operations", ops}, {"owner_super_admin", owner}, {"admin", admin},
	} {
		for _, p := range opsMapPagePaths {
			if r := hh.GET("/api/v1/admin"+p, who.tok); r.Code != http.StatusOK {
				t.Errorf("**%s لا يفتح %s** ← %d · %s", who.name, p, r.Code, string(r.Body))
			}
		}
	}
	// **وطبقةُ المندوبين بقدرتها كما كانت** (`merchants.manage`): المالكُ والمدير
	// يفتحانها، **والصفحةُ لا تناديها لموظّف العمليّات** (لا يملك
	// `VIEW_REP_ACTIVITY` في `meta`) — فلا يرى زرّاً يردّ ٤٠٣.
	for _, tok := range []string{owner, admin} {
		if r := hh.GET("/api/v1/admin/ops-map/reps?range=30d", tok); r.Code != http.StatusOK {
			t.Errorf("المالكُ/المديرُ لا يفتح طبقةَ المندوبين ← %d", r.Code)
		}
	}
	meta := hh.GET("/api/v1/admin/ops-map/meta", ops)
	if strings.Contains(string(meta.Body), "VIEW_REP_ACTIVITY") {
		t.Error("`meta` يمنح العمليّاتِ طبقةَ المندوبين وبابُها يردّها ٤٠٣")
	}
	// **ومن لا يملك الخريطةَ لا يرى زبائنَها ولا مكتبَها.**
	cust := hh.NewUser("customer").Token
	for _, p := range []string{"/ops-map/summary", "/ops-map/customers", "/ops-map/office"} {
		if r := hh.GET("/api/v1/admin"+p, cust); r.Code != http.StatusForbidden && r.Code != http.StatusUnauthorized {
			t.Errorf("زبونٌ فتح %s ← %d", p, r.Code)
		}
	}
}

// TestOpsMapPro_CustomersAreAggregatedCellsOnly **خلايا لا بيوت.**
//
// ثلاثةُ زبائنَ في حيٍّ واحدٍ وزبونان في حيٍّ آخر — **في موضعٍ لا يسكنه
// غيرُهم** (وسطُ الصحراء الكبرى). **فيُرى الحيُّ الأوّلُ بعدِّه ثلاثة، ولا
// يُرى الثاني البتّة**، ولا معرّفَ ولا اسمَ ولا هاتفَ في الردّ.
func TestOpsMapPro_CustomersAreAggregatedCellsOnly(t *testing.T) {
	hh := New(t)
	_, ops := roleUser(t, hh, "operations")

	place := func(u *User, lng, lat float64) {
		t.Helper()
		if _, err := hh.Pool.Exec(t.Context(), `
			INSERT INTO user_addresses (user_id, label, address_text, area_building, location, is_default)
			VALUES ($1::uuid, 'البيت', 'عنوانُ اختبار', 'حيُّ التجربة',
			        ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, true)`,
			u.ID, lng, lat); err != nil {
			t.Fatalf("عنوانُ الزبون: %v", err)
		}
	}
	var seeded []*User
	// **الحيُّ الأوّل**: ثلاثةٌ في خليّةٍ واحدة (ضلعُها ٠٫٠٠٤°).
	for _, d := range []float64{0.0001, 0.0005, 0.0009} {
		u := hh.NewUser("customer")
		place(u, 12.0001+d, 21.0001+d)
		seeded = append(seeded, u)
	}
	// **والثاني**: اثنان — دون الحدّ.
	for _, d := range []float64{0.0001, 0.0006} {
		u := hh.NewUser("customer")
		place(u, 12.0201+d, 21.0201+d)
		seeded = append(seeded, u)
	}

	res := hh.GET("/api/v1/admin/ops-map/customers?bbox=11.9,20.9,12.1,21.1", ops)
	if res.Code != http.StatusOK {
		t.Fatalf("طبقةُ الزبائن ← %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Cells    []map[string]any `json:"cells"`
			MinCount int              `json:"min_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.MinCount < 3 {
		t.Fatalf("الحدُّ الأدنى المُعلَن %d — دون ثلاثة", env.Data.MinCount)
	}
	if len(env.Data.Cells) != 1 {
		t.Fatalf("**المتوقَّع خليّةٌ واحدة** (الثانية دون الحدّ) — وجاء %d: %v",
			len(env.Data.Cells), env.Data.Cells)
	}
	cell := env.Data.Cells[0]
	for k := range cell {
		if k != "lat" && k != "lng" && k != "count" {
			t.Errorf("**مفتاحٌ زائدٌ في الخليّة**: %q — الخليّةُ مركزٌ وعددٌ لا غير", k)
		}
	}
	if n, _ := cell["count"].(float64); n != 3 {
		t.Errorf("عدُّ الخليّة %v والمتوقَّع ٣", cell["count"])
	}
	// **والمركزُ مركزُ الخليّة لا موضعُ أحد.**
	lat, _ := cell["lat"].(float64)
	lng, _ := cell["lng"].(float64)
	if math.Abs(lat-21.002) > 1e-9 || math.Abs(lng-12.002) > 1e-9 {
		t.Errorf("مركزُ الخليّة (%v,%v) والمتوقَّع (21.002,12.002)", lat, lng)
	}
	body := string(res.Body)
	for _, u := range seeded {
		for _, leak := range []string{u.ID, u.Phone, u.Name} {
			if leak != "" && strings.Contains(body, leak) {
				t.Errorf("**تسرّبٌ في طبقة الزبائن**: %q", leak)
			}
		}
	}
}

// TestOpsMapPro_OfficeListsStaffNameAndRoleOnly **المكتبُ اسمٌ ودورٌ لمن حضر.**
func TestOpsMapPro_OfficeListsStaffNameAndRoleOnly(t *testing.T) {
	hh := New(t)
	_, ops := roleUser(t, hh, "operations")
	present, _ := roleUser(t, hh, "operations")
	away, _ := roleUser(t, hh, "finance")
	// **واسمٌ فريدٌ للحاضر** (٢٠٢٦-١٠-٠٦) — كان اسماً عامّاً يشاركه موظّفُ اختبارٍ آخرَ حضر قبله
	// في القاعدة المشتركة، **فيُقرأ دورُ ذاك.**
	present.Name = "حاضر-" + present.ID[:8]
	away.Name = "غائب-" + away.ID[:8]
	for _, u := range []struct{ id, name string }{{present.ID, present.Name}, {away.ID, away.Name}} {
		if _, err := hh.Pool.Exec(t.Context(),
			`UPDATE users SET full_name = $2 WHERE id = $1::uuid`, u.id, u.name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := hh.Pool.Exec(t.Context(), `
		UPDATE users SET last_seen_at = CASE WHEN id = $1::uuid THEN now()
		                                    ELSE now() - interval '3 hours' END
		WHERE id IN ($1::uuid, $2::uuid)`, present.ID, away.ID); err != nil {
		t.Fatal(err)
	}
	res := hh.GET("/api/v1/admin/ops-map/office", ops)
	if res.Code != http.StatusOK {
		t.Fatalf("المكتب ← %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Staff []map[string]any `json:"staff"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	var found map[string]any
	for _, s := range env.Data.Staff {
		for k := range s {
			if k != "name" && k != "roles" {
				t.Errorf("**مفتاحٌ زائدٌ في «من في المكتب»**: %q", k)
			}
		}
		if s["name"] == present.Name {
			found = s
		}
		if s["name"] == away.Name && away.Name != present.Name {
			t.Errorf("موظّفٌ غائبٌ منذ ثلاث ساعاتٍ ظهر حاضراً: %v", s)
		}
	}
	if found == nil {
		t.Fatalf("الموظّفُ الحاضرُ غائبٌ عن المكتب: %v", env.Data.Staff)
	}
	roles, _ := found["roles"].([]any)
	if len(roles) != 1 || roles[0] != "operations" {
		t.Errorf("أدوارُ الحاضر %v — المتوقَّع operations وحدَه (لا صفةُ الحساب)", roles)
	}
	body := string(res.Body)
	if present.Phone != "" && strings.Contains(body, present.Phone) {
		t.Error("**هاتفُ الموظّف في ردّ المكتب**")
	}
	if strings.Contains(body, present.ID) {
		t.Error("**معرّفُ الموظّف في ردّ المكتب**")
	}
}

// TestOpsMapPro_DriverToneAndContactGate **السائقُ بحالٍ في لون، والهاتفُ لمن
// يملك قراءةَ الاتّصال.**
func TestOpsMapPro_DriverToneAndContactGate(t *testing.T) {
	hh := New(t)
	_, ops := roleUser(t, hh, "operations")
	drv := hh.NewUser("driver")
	if _, err := hh.Pool.Exec(t.Context(), `
		UPDATE users SET on_shift = true,
		       last_location = ST_SetSRID(ST_MakePoint(39.01, 35.95), 4326)::geography,
		       last_location_at = now()
		WHERE id = $1::uuid`, drv.ID); err != nil {
		t.Fatal(err)
	}
	read := func(tok string) map[string]any {
		t.Helper()
		res := hh.GET("/api/v1/admin/ops-map/drivers", tok)
		if res.Code != http.StatusOK {
			t.Fatalf("طبقةُ السائقين ← %d", res.Code)
		}
		var env struct {
			Data struct {
				Drivers []map[string]any `json:"drivers"`
			} `json:"data"`
		}
		if err := json.Unmarshal(res.Body, &env); err != nil {
			t.Fatal(err)
		}
		for _, d := range env.Data.Drivers {
			if d["id"] == drv.ID {
				return d
			}
		}
		t.Fatalf("السائقُ غائبٌ عن الطبقة")
		return nil
	}
	d := read(ops)
	if d["tone"] != "available" {
		t.Errorf("سائقٌ على الدوام بلا طلبٍ وموضعُه حيّ: tone=%v والمتوقَّع available", d["tone"])
	}
	if d["phone"] == nil || d["phone"] == "" {
		t.Error("موظّفُ العمليّات يملك `users.contact.read` ولا هاتفَ في بطاقة السائق")
	}

	// **ومن يقرأ السائقين بلا قدرةِ الاتّصال لا يرى الرقم.**
	_, noContact := capUserWithCaps(t, hh, "drivers.read", "orders.read")
	if d := read(noContact); d["phone"] != nil {
		t.Errorf("**هاتفُ السائق لمن لا يملك `users.contact.read`**: %v", d["phone"])
	}
}

// capUserWithCaps **حسابٌ بدورٍ مصنوعٍ بقدراتٍ بعينها** — لاختبار الحجب.
func capUserWithCaps(t *testing.T, hh *Harness, caps ...string) (*User, string) {
	t.Helper()
	role := "zz_" + strings.ReplaceAll(uniq("r"), "-", "_")
	if _, err := hh.Pool.Exec(t.Context(),
		`INSERT INTO roles (code, name_key) VALUES ($1, $1)`, role); err != nil {
		t.Fatalf("الدورُ المصنوع: %v", err)
	}
	for _, c := range caps {
		if _, err := hh.Pool.Exec(t.Context(),
			`INSERT INTO role_capabilities (role_code, capability_code) VALUES ($1, $2)`,
			role, c); err != nil {
			t.Fatalf("قدرةُ %s: %v", c, err)
		}
	}
	return capUser(t, hh, role)
}
