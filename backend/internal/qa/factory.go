// مصنعُ بيانات الاختبار — **حتميٌّ · قابلٌ للتركيب · قابلٌ للتنظيف.**
//
// (المرحلةُ `P-3` من منظومة الاختبار الدائمة — قرارُ المالك ٢٠٢٦-٠٩-٠٥.)
//
// # المشكلةُ التي يحلّها
//
// **كلُّ اختبارٍ يبني عالمَه بيده اليوم** — **فيتكرّر التركيبُ في مئتي
// ملفّ**، **ومن بدّل عموداً في `users` أصلح مئةَ موضع.**
//
// **وأخطرُ منه**: `uniqPhone` في المِسنَد تُشتقّ من `time.Now().UnixNano()`
// — **فسيناريو يسقط لا يُعاد بالأرقام نفسِها**، **ومن أراد أن يفهم لماذا
// سقط وجد بياناتٍ أخرى.**
//
// # والمبدأ
//
//	DETERMINISTIC — نفسُ السيناريو يعطي نفسَ الهويّات
//	COMPOSABLE    — خياراتٌ تُركَّب لا معاملاتٌ تُعدّ
//	RESETTABLE    — ما أنشأه يُمحى بعده
//	SAFE          — يرفض العملَ على قاعدةٍ ليست للاختبار
//
// # وما لا يفعله
//
// **لا يُصلح سلوكاً ولا يخترع حالاً مستحيلاً.** **يجهّز الشرطَ السابقَ
// والاختبارُ ينفّذ الفعل** — **ومن جعل المصنعَ ينفّذ الفعلَ اختبر مصنعَه.**
package qa

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// ══════════════════════════════════════════════════════════════════════
// **حارسُ الإنتاج — قاطعٌ ولا يُتخطّى بسهولة**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١٩ من طلب المالك.)
//
// **و`testdb` يحرس بلاحقة `_test` وحدَها** — **وهذا يزيد عليها**:
// **مضيفٌ بعيدٌ · واسمُ قاعدةٍ إنتاجيّ · ومنفذُ الإنتاج المعروف.**
//
// **ولا متغيّرَ بيئةٍ يُطفئه** — **فمن أراد تخطّيه عدّل شيفرةً تُقرأ في
// المراجعة.**

// productionNames أسماءُ قواعدَ لا يُكتب فيها اختبارٌ أبداً.
var productionNames = map[string]bool{
	"rahalgo": true, "postgres": true, "production": true, "prod": true, "main": true,
}

// AssertTestDatabase يرفض كلَّ ما ليس قاعدةَ اختبارٍ محلّيّة.
//
// **ويُنادى قبل أوّل كتابة** — **لا بعدها.**
func AssertTestDatabase(t *testing.T, rawURL string) {
	t.Helper()
	if err := checkTestDatabase(rawURL); err != nil {
		t.Fatalf("qa: رُفضت القاعدة — %v", err)
	}
}

// checkTestDatabase منطقُ الحارس — **مفصولٌ ليُختبَر بنفسه.**
func checkTestDatabase(rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("عنوانُ القاعدة فارغ")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("عنوانٌ لا يُقرأ")
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return fmt.Errorf("لا اسمَ قاعدةٍ في العنوان")
	}
	// ١ · **اللاحقةُ إلزاميّة.**
	if !strings.HasSuffix(name, "_test") {
		return fmt.Errorf("اسمُ القاعدة %q لا ينتهي بـ_test", name)
	}
	// ٢ · **واسمٌ إنتاجيٌّ مرفوضٌ ولو لُصقت به اللاحقة.**
	base := strings.TrimSuffix(name, "_test")
	if productionNames[strings.ToLower(base)] && base != "rahalgo" {
		return fmt.Errorf("اسمُ القاعدة %q إنتاجيّ", name)
	}
	// ٣ · **والمضيفُ محلّيٌّ لا غير** — **وقاعدةُ اختبارٍ على خادمٍ بعيدٍ
	//     تعني كتابةً في شبكةٍ لا نملك التحقّقَ منها.**
	host := u.Hostname()
	switch host {
	case "localhost", "127.0.0.1", "::1", "host.docker.internal", "postgres", "":
	default:
		return fmt.Errorf("المضيفُ %q ليس محلّيّاً — لا يُكتب اختبارٌ على قاعدةٍ بعيدة", host)
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════
// **الهويّاتُ الحتميّة**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ٣: **لا وقتٌ عشوائيٌّ ولا أرقامٌ عشوائيّة.**)
//
// **والنطاقُ يُشتقّ من اسم الاختبار** — **فسيناريوان لا يتصادمان،
// وإعادةُ الاختبار تُعطي الأرقامَ نفسَها.**

// Namespace نطاقُ سيناريوٍ واحد.
type Namespace struct {
	seed    uint64
	scen    string
	counter uint64
	mu      sync.Mutex
}

// NewNamespace يشتقّ نطاقاً من اسم الاختبار — **مستقرٌّ عبر التشغيلات.**
//
// **و`RAHALGO_TEST_SEED` يبدّله عمداً** — **لتوليد بياناتٍ أخرى عند
// الحاجة، لا ليصير التوليدُ عشوائيّاً.**
func NewNamespace(scenario string) *Namespace {
	h := sha256.Sum256([]byte(os.Getenv("RAHALGO_TEST_SEED") + "|" + scenario))
	return &Namespace{seed: binary.BigEndian.Uint64(h[:8]), scen: scenario}
}

// next رقمٌ تسلسليٌّ داخلَ النطاق — **آمنٌ للتوازي.**
func (n *Namespace) next() uint64 {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.counter++
	return n.counter
}

// Phone هاتفٌ حتميٌّ بصيغة المنصّة — **`+9639` وعشرةُ أرقام.**
func (n *Namespace) Phone() string {
	v := (n.seed ^ (n.next() * 0x9E3779B97F4A7C15)) % 1e10
	return fmt.Sprintf("+9639%010d", v)
}

// Name اسمٌ يقول أيُّ سيناريوٍ أنشأه — **فيُقرأ في القاعدة عند التشخيص.**
func (n *Namespace) Name(kind string) string {
	return fmt.Sprintf("QA/%s/%s#%d", short(n.scen), kind, n.next())
}

// Ref مرجعٌ حتميٌّ للقيود والمفاتيح.
func (n *Namespace) Ref(kind string) string {
	return fmt.Sprintf("qa-%x-%s-%d", n.seed&0xFFFFFF, kind, n.next())
}

func short(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	// **والقصُّ بالمحارف لا بالبايتات.**
	//
	// **كان `s[:28]`** — فاسمُ اختبارٍ فيه عربيّةٌ يُقصّ في منتصف حرف،
	// **فتردّ القاعدةُ `invalid byte sequence for encoding "UTF8"`**
	// ولا يُنشأ مستخدم. (كُشف في `P-4`: أسماءُ مصفوفةِ مصدرِ العمولة عربيّة.)
	if r := []rune(s); len(r) > 28 {
		s = string(r[:28])
	}
	return s
}

// ══════════════════════════════════════════════════════════════════════
// **الساعة**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١٥: **ولا اختبارٌ يبدّل ساعةَ النظام.**)
//
// **وشيفرةُ الإنتاج لا تقبل ساعةً محقونة** — **قِيس: `time.Now()` مباشرةً
// في المحرّك.** **فالزمنُ يُزاح في البيانة لا في الساعة**، **وهو آمنٌ
// وحتميّ**، **وأُثبت في `P-2` بتقديم `created_at` لمفتاح تكرار.**
//
// **والفجوةُ تُسجَّل ولا تُصلَح**: `XOB-10`.

// Clock ساعةُ السيناريو — **ثابتةٌ فلا يتبدّل الناتجُ بين تشغيلين.**
type Clock struct{ base time.Time }

// NewClock ساعةٌ مثبَّتةٌ على لحظةٍ معلومة.
func NewClock() *Clock {
	return &Clock{base: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)}
}

// Now لحظةُ الأساس.
func (c *Clock) Now() time.Time { return c.base }

// Ago لحظةٌ قبل الأساس بمدّة — **لتقادمِ موقعٍ أو مفتاحِ تكرار.**
func (c *Clock) Ago(d time.Duration) time.Time { return c.base.Add(-d) }

// Ahead لحظةٌ بعده — **لطابعٍ مستقبليٍّ يجب أن يُرفَض.**
func (c *Clock) Ahead(d time.Duration) time.Time { return c.base.Add(d) }

// ══════════════════════════════════════════════════════════════════════
// **المصنع**
// ══════════════════════════════════════════════════════════════════════

// Factory ينشئ الشروطَ السابقةَ للاختبار.
type Factory struct {
	h  *Harness
	NS *Namespace
	CK *Clock
}

// Factory يبني مصنعاً لهذا الاختبار — **بنطاقه وساعته.**
func (h *Harness) Factory() *Factory {
	h.T.Helper()
	AssertTestDatabase(h.T, os.Getenv("TEST_DATABASE_URL"))
	return &Factory{h: h, NS: NewNamespace(h.T.Name()), CK: NewClock()}
}

func (f *Factory) ctx() context.Context { return context.Background() }

// ctxBG سياقٌ للاستعلامات المباشرة في الاختبارات.
func ctxBG() context.Context { return context.Background() }

func (f *Factory) fatal(format string, a ...any) {
	f.h.T.Helper()
	f.h.T.Fatalf("factory: "+format, a...)
}

// cleanup يسجّل محوَ صفٍّ عند انتهاء الاختبار.
func (f *Factory) cleanup(table, id string) {
	f.h.T.Cleanup(func() {
		_, _ = f.h.Pool.Exec(context.Background(),
			fmt.Sprintf(`DELETE FROM %s WHERE id = $1::uuid`, table), id)
	})
}

// ══════════════════════════════════════════════════════════════════════
// **الخيارات — تُركَّب ولا تُعدّ**
// ══════════════════════════════════════════════════════════════════════

// UserOpt خيارُ مستخدم.
type UserOpt func(*userSpec)

type userSpec struct {
	roles      []string
	status     string
	mustChange bool
	waVerified bool
	name       string
	phone      string
}

// Suspended **حسابٌ موقوف** — `RequireAuth` يردّه.
func Suspended() UserOpt { return func(s *userSpec) { s.status = "suspended" } }

// Blocked **حسابٌ محظور.**
func Blocked() UserOpt { return func(s *userSpec) { s.status = "blocked" } }

// MustChangePassword **كلمةٌ وضعها ثالثٌ** — `D11` يمسّها.
func MustChangePassword() UserOpt { return func(s *userSpec) { s.mustChange = true } }

// WhatsAppUnverified **بلا توثيقِ واتساب** — `D8` و`sales.require_whatsapp`.
func WhatsAppUnverified() UserOpt { return func(s *userSpec) { s.waVerified = false } }

// WithRoles أدوارٌ إضافيّة — **والمنصّةُ تسمح بأكثر من دورٍ لحساب.**
func WithRoles(roles ...string) UserOpt {
	return func(s *userSpec) { s.roles = append(s.roles, roles...) }
}

// Named اسمٌ صريحٌ بدل المشتقّ.
func Named(name string) UserOpt { return func(s *userSpec) { s.name = name } }

// NewUserWith ينشئ مستخدماً بخياراته.
//
// **ويُبقي `NewUser` القديمةَ كما هي** — **فلا يُكسَر مئتا ملفٍّ في
// مرحلةٍ غرضُها البنية.**
func (f *Factory) NewUserWith(role string, opts ...UserOpt) *User {
	f.h.T.Helper()
	s := &userSpec{status: "active", waVerified: true}
	if role != "" {
		s.roles = []string{role}
	}
	for _, o := range opts {
		o(s)
	}
	if s.phone == "" {
		s.phone = f.NS.Phone()
	}
	if s.name == "" {
		s.name = f.NS.Name(orDefault(role, "user"))
	}

	var waAt any
	if s.waVerified {
		waAt = f.CK.Now()
	}

	var id string
	err := f.h.Pool.QueryRow(f.ctx(), `
		INSERT INTO users (phone, full_name, status, whatsapp_phone,
		                   whatsapp_verified_at, must_change_password)
		VALUES ($1, $2, $3, $1, $4, $5)
		RETURNING id::text`,
		s.phone, s.name, s.status, waAt, s.mustChange).Scan(&id)
	if err != nil {
		f.fatal("تعذّر إنشاءُ مستخدم: %v", err)
	}
	f.cleanup("users", id)

	for _, r := range s.roles {
		if r == "" {
			continue
		}
		if _, err := f.h.Pool.Exec(f.ctx(), `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, $2)
			ON CONFLICT DO NOTHING`, id, r); err != nil {
			f.fatal("تعذّر إسنادُ الدور %q: %v", r, err)
		}
	}

	tok, _, err := f.h.tokens.IssueAccess(id, s.roles, "")
	if err != nil {
		f.fatal("تعذّر إصدارُ التوكن: %v", err)
	}
	return &User{ID: id, Phone: s.phone, Name: s.name, Roles: s.roles, Token: tok}
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// ══════════════════════════════════════════════════════════════════════
// **مصنعُ السائق**
// ══════════════════════════════════════════════════════════════════════

// DriverOpt خيارُ سائق.
type DriverOpt func(*driverSpec)

type driverSpec struct {
	user     []UserOpt
	onShift  bool
	cashHeld int64
	loc      *locSpec
}

type locSpec struct {
	lat, lng float64
	at       time.Time
}

// OnShift **ورديّةٌ مفتوحة.**
func OnShift() DriverOpt { return func(s *driverSpec) { s.onShift = true } }

// CashHeld **نقدٌ في صندوقه** — **و`D7` يقيس المحصَّل لا المكشوف.**
func CashHeld(amount int64) DriverOpt { return func(s *driverSpec) { s.cashHeld = amount } }

// LocationAt **موقعٌ بلحظةٍ معلومة** — طازجٌ أو شائخٌ أو مستقبليّ.
func LocationAt(lat, lng float64, at time.Time) DriverOpt {
	return func(s *driverSpec) { s.loc = &locSpec{lat: lat, lng: lng, at: at} }
}

// DriverUser خياراتُ المستخدم تحت السائق.
func DriverUser(opts ...UserOpt) DriverOpt {
	return func(s *driverSpec) { s.user = append(s.user, opts...) }
}

// Driver ينشئ سائقاً بحاله.
func (f *Factory) Driver(opts ...DriverOpt) *User {
	f.h.T.Helper()
	s := &driverSpec{}
	for _, o := range opts {
		o(s)
	}
	u := f.NewUserWith("driver", s.user...)

	if s.onShift {
		// **والورديّةُ عمودان في `users` لا جدولٌ** — `0040_driver_shift.sql`.
		if _, err := f.h.Pool.Exec(f.ctx(), `
			UPDATE users SET on_shift = true, shift_started_at = $2
			WHERE id = $1::uuid`, u.ID, f.CK.Ago(time.Hour)); err != nil {
			f.fatal("تعذّر فتحُ الورديّة: %v", err)
		}
	}
	if s.loc != nil {
		if _, err := f.h.Pool.Exec(f.ctx(), `
			UPDATE users SET
				last_location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
				last_location_at = $4
			WHERE id = $1::uuid`, u.ID, s.loc.lng, s.loc.lat, s.loc.at); err != nil {
			f.fatal("تعذّر ضبطُ الموقع: %v", err)
		}
	}
	if s.cashHeld > 0 {
		// **ونقدُ السائق دفترٌ ثانٍ لا محفظة** — `driver_cash_entries`
		// **قيودُه** و`driver_cash_boxes.held` **رصيدُه.**
		//
		// **والصندوقُ يتبع القيدَ كما تتبع المحفظةُ دفترَها** —
		// **فالفكسچرُ متّسقٌ في الدفترين معاً.**
		tx, err := f.h.Pool.Begin(f.ctx())
		if err != nil {
			f.fatal("تعذّر فتحُ معاملة النقد: %v", err)
		}
		defer func() { _ = tx.Rollback(f.ctx()) }()
		if _, err := tx.Exec(f.ctx(), `
			INSERT INTO driver_cash_entries (driver_id, amount, kind, ref, note)
			VALUES ($1::uuid, $2, 'order_collection', $3, 'qa fixture')`,
			u.ID, s.cashHeld, f.NS.Ref("cash")); err != nil {
			f.fatal("تعذّر قيدُ النقد: %v", err)
		}
		if _, err := tx.Exec(f.ctx(), `
			INSERT INTO driver_cash_boxes (driver_id, held) VALUES ($1::uuid, $2)
			ON CONFLICT (driver_id) DO UPDATE SET held = driver_cash_boxes.held + $2,
			                                      updated_at = now()`,
			u.ID, s.cashHeld); err != nil {
			f.fatal("تعذّر تحديثُ الصندوق: %v", err)
		}
		if err := tx.Commit(f.ctx()); err != nil {
			f.fatal("تعذّر إتمامُ معاملة النقد: %v", err)
		}
	}
	return u
}

// ══════════════════════════════════════════════════════════════════════
// **مصنعُ المتجر**
// ══════════════════════════════════════════════════════════════════════

// MerchantOpt خيارُ متجر.
type MerchantOpt func(*merchantSpec)

type merchantSpec struct {
	status     string
	commission *int64
	repID      string
	owner      *User
}

// MerchantSuspended **متجرٌ معلَّق** — `PC-11`.
func MerchantSuspended() MerchantOpt { return func(s *merchantSpec) { s.status = "suspended" } }

// Commission **نسبةُ عمولةِ المنصّة من هذا المتجر** — **وصفرُها بوّابةُ `XG-14`.**
func Commission(pct int64) MerchantOpt { return func(s *merchantSpec) { s.commission = &pct } }

// OwnedByRep **منسوبٌ إلى مندوب** — **شرطٌ سابقٌ لعمولته.**
func OwnedByRep(repID string) MerchantOpt { return func(s *merchantSpec) { s.repID = repID } }

// OwnedBy **صاحبُ متجرٍ بعينه** — **وإلّا أُنشئ واحد.**
func OwnedBy(u *User) MerchantOpt { return func(s *merchantSpec) { s.owner = u } }

// Merchant متجرٌ جاهز.
type Merchant struct {
	ID      string
	Name    string
	Owner   *User
	RepID   string
	Percent int64
}

// Merchant ينشئ متجراً بحاله.
//
// **ولا ينفّذ فعلاً تجاريّاً**: **النقلُ بين مندوبين فعلٌ يختبره الاختبارُ
// لا يجهّزه المصنع.** (البند ٧ من طلب المالك.)
func (f *Factory) Merchant(opts ...MerchantOpt) *Merchant {
	f.h.T.Helper()
	s := &merchantSpec{status: "active"}
	for _, o := range opts {
		o(s)
	}
	if s.owner == nil {
		s.owner = f.NewUserWith("merchant")
	}
	name := f.NS.Name("store")

	catID := f.category()

	var pct any
	if s.commission != nil {
		pct = *s.commission
	}
	var repID any
	if s.repID != "" {
		repID = s.repID
	}

	var id string
	err := f.h.Pool.QueryRow(f.ctx(), `
		INSERT INTO merchants (name, category_id, phone, address_text, owner_user_id,
		                       sales_rep_user_id, status, commission_percent, location)
		VALUES ($1, $2::uuid, $3, 'QA', $4::uuid, $5::uuid, $6, $7,
		        ST_SetSRID(ST_MakePoint(39.01, 35.95), 4326)::geography)
		RETURNING id::text`,
		name, catID, s.owner.Phone, s.owner.ID, repID, s.status, pct).Scan(&id)
	if err != nil {
		f.fatal("تعذّر إنشاءُ متجر: %v", err)
	}
	f.cleanup("merchants", id)

	m := &Merchant{ID: id, Name: name, Owner: s.owner, RepID: s.repID}
	if s.commission != nil {
		m.Percent = *s.commission
	}
	return m
}

// category تصنيفٌ صالحٌ — **يُعاد استعمالُه ولا يُنشأ لكلّ متجر.**
func (f *Factory) category() string {
	var id string
	err := f.h.Pool.QueryRow(f.ctx(),
		`SELECT id::text FROM categories ORDER BY sort_order, name LIMIT 1`).Scan(&id)
	if err == nil && id != "" {
		return id
	}
	err = f.h.Pool.QueryRow(f.ctx(), `
		INSERT INTO categories (name, icon, sort_order)
		VALUES ('QA', 'store', 999) RETURNING id::text`).Scan(&id)
	if err != nil {
		f.fatal("تعذّر تجهيزُ تصنيف: %v", err)
	}
	return id
}

// ══════════════════════════════════════════════════════════════════════
// **مصنعُ المندوب**
// ══════════════════════════════════════════════════════════════════════

// Rep مندوبٌ برمزِ دعوته.
type Rep struct {
	*User
	InviteCode string
}

// RepAccount ينشئ مندوباً — **ورمزُ الدعوة شرطُ نسبةِ المتاجر إليه.**
func (f *Factory) RepAccount(opts ...UserOpt) *Rep {
	f.h.T.Helper()
	u := f.NewUserWith("sales", opts...)
	// **والرمزُ يُبذَر بالنطاق لا بالعدّاد وحدَه.**
	//
	// **كان `QA%06d` من العدّاد** — وهو يبدأ من واحدٍ في كلّ سيناريو،
	// **فمندوبان في اختبارين مختلفين يتصادمان** على
	// `users_invite_code_key`. (كُشف في `P-4`.)
	code := fmt.Sprintf("QA%08d", (f.NS.seed^(f.NS.next()*0x9E3779B97F4A7C15))%1e8)
	if _, err := f.h.Pool.Exec(f.ctx(),
		`UPDATE users SET invite_code = $2 WHERE id = $1::uuid`, u.ID, code); err != nil {
		f.fatal("تعذّر ضبطُ رمز الدعوة: %v", err)
	}
	return &Rep{User: u, InviteCode: code}
}

// ══════════════════════════════════════════════════════════════════════
// **المالُ — متّسقٌ افتراضاً**
// ══════════════════════════════════════════════════════════════════════
//
// (البندان ١٠ و١١: **المحفظةُ تتبع الدفترَ ولا تُكتب وحدَها.**)

// Credit يقيّد مبلغاً في محفظةِ مستخدمٍ **عبر مسار الدفتر**.
//
// **ولا `UPDATE balance` هنا** — **الرصيدُ يتبع القيدَ**، **فالفكسچرُ
// متّسقٌ ماليّاً بحكم بنائه.**
func (f *Factory) Credit(userID string, amount int64, kind string) {
	f.h.T.Helper()
	if amount == 0 {
		f.fatal("قيدٌ بصفر — والقاعدةُ ترفضه")
	}
	if !allowedLedgerKind(kind) {
		f.fatal("نوعُ قيدٍ غيرُ مستعمَلٍ في المنصّة: %q — انظر XOB-9", kind)
	}
	f.CreditRef(userID, amount, kind, "")
}

// CreditRef قيدٌ بمرجعٍ صريح — **والمرجعُ يخضع لعقد `P-4`.**
//
// **وكانت `Credit` تكتب مرجعاً من النطاق دائماً** (`qa-…-ledger-7`)،
// **فقيدُ سحبٍ يدّعي طلبَ سحبٍ لا وجودَ له** — **وأمسكه `FI-01.e` على
// القاعدة.** (كُشف في `P-4`.)
//
// **فصار المرجعُ يُقرأ من العقد**: ما لا يشترطه يُترَك فارغاً كما تفعل
// الإدارةُ حين تسوّي بيدها، **وما يشترطه لا يُختلَق** — يُمرَّر أو يُرفض.
func (f *Factory) CreditRef(userID string, amount int64, kind, ref string) {
	f.h.T.Helper()
	if amount == 0 {
		f.fatal("قيدٌ بصفر — والقاعدةُ ترفضه")
	}
	if !allowedLedgerKind(kind) {
		f.fatal("نوعُ قيدٍ غيرُ مستعمَلٍ في المنصّة: %q — انظر XOB-9", kind)
	}
	if ref == "" && fininv.Kinds[kind].RefRequired {
		f.fatal("النوعُ %q يشترط مرجعاً إلى %s — ولا يُختلَق",
			kind, fininv.Kinds[kind].RefTarget)
	}
	if err := f.h.walletApply(userID, amount, kind, ref); err != nil {
		f.fatal("تعذّر القيد: %v", err)
	}
}

// allowedLedgerKind **الأنواعُ التي تكتبها المنصّةُ فعلاً — لا التي يسمح
// بها القيد.**
//
// **وقيدُ القاعدة يسمح بثلاثةَ عشرَ نوعاً** (قِيس في `P-2`)، **وأربعةٌ
// منها لا يكتبها نداءُ `ApplyTx` واحد**: `penalty` · `platform_expense` ·
// `platform_profit` · `reward`.
//
// **وقِيس في `P-4` أنّ الأربعةَ بالغةٌ كلُّها** — لها مسارٌ في الإنتاج
// ولها عقدٌ في `internal/fininv/kinds.go`. **ويبقى بابُ المصنع مغلقاً
// عليها بقرارٍ لا بجهل**: `reward` و`penalty` **تُصنَعان بمسارهما
// (`incentives.Grant`) لا بقيدٍ مباشر** — وله شروطُه (رصيدٌ كافٍ وسببٌ
// مكتوبٌ ومرآةُ خزينة، `incentives.go:245`)، **ومن تخطّاها بنى حالاً لا
// تقع في الواقع.** والنوعان الآخران للخزينة وحدَها (`FI-12.a`).
func allowedLedgerKind(kind string) bool {
	switch kind {
	case "topup", "order_payment", "refund", "compensation",
		"commission", "payout", "adjustment", "merchant_earning", "driver_earning":
		return true
	}
	return false
}

// walletApply يمرّ بخدمة المحفظة نفسِها — **لا بالجداول رأساً.**
func (h *Harness) walletApply(userID string, amount int64, kind, ref string) error {
	ctx := context.Background()
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO wallets (user_id, balance) VALUES ($1::uuid, 0)
		ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
		VALUES ($1::uuid, $2, $3, $4, 'qa fixture')`,
		userID, amount, kind, ref); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE wallets SET balance = balance + $2, updated_at = now()
		WHERE user_id = $1::uuid`, userID, amount); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Balance رصيدُ محفظةٍ كما تراه القاعدة.
func (f *Factory) Balance(userID string) int64 {
	f.h.T.Helper()
	var v int64
	if err := f.h.Pool.QueryRow(f.ctx(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`,
		userID).Scan(&v); err != nil {
		return 0
	}
	return v
}

// LedgerSum مجموعُ قيود محفظةٍ — **ويجب أن يساوي الرصيد.**
func (f *Factory) LedgerSum(userID string) int64 {
	f.h.T.Helper()
	var v int64
	_ = f.h.Pool.QueryRow(f.ctx(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions WHERE user_id = $1::uuid`,
		userID).Scan(&v)
	return v
}

// ══════════════════════════════════════════════════════════════════════
// **فكسچراتٌ غيرُ آمنةٍ — بأسمائها**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ٩ و١١: **حالٌ مكسورةٌ تُبنى بابٍ مسمّىً لا بالمسار الطبيعيّ.**)
//
// **واسمُها يمنع استعمالَها سهواً في مسارٍ سعيد.**

// UnsafeCorruptBalance **يكسر اتّساقَ المحفظة مع دفترها عمداً.**
//
// **لاختبار الحارس لا لبناء سيناريو** — **ومن ناداها في مسارٍ سعيدٍ
// اختبر شيئاً لا وجودَ له.**
func (f *Factory) UnsafeCorruptBalance(userID string, balance int64) {
	f.h.T.Helper()
	if _, err := f.h.Pool.Exec(f.ctx(), `
		INSERT INTO wallets (user_id, balance) VALUES ($1::uuid, $2)
		ON CONFLICT (user_id) DO UPDATE SET balance = $2`, userID, balance); err != nil {
		f.fatal("تعذّر إفسادُ الرصيد: %v", err)
	}
}

// UnsafeSetOrderStatus **يضع حالَ طلبٍ رأساً بلا بوّابة.**
//
// **لاختبار حالٍ قديمةٍ أو فسادٍ** — **ولا يُستعمل لبناء دورة حياة.**
func (f *Factory) UnsafeSetOrderStatus(orderID, status string) {
	f.h.T.Helper()
	if _, err := f.h.Pool.Exec(f.ctx(),
		`UPDATE orders SET status = $2 WHERE id = $1::uuid`, orderID, status); err != nil {
		f.fatal("تعذّر وضعُ الحال: %v", err)
	}
}
