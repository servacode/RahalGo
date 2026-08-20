// Package qa **منظومةُ الاختبار الآليّ — الطبقةُ التي تقود النظامَ لا تقرؤه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٩: «المشكلةُ الأساسيّة أنّ المشروع لا يحتوي على
//
//	منظومة اختبارٍ احترافيّةٍ دائمةٍ وقابلةٍ لإعادة التشغيل… لا أريد أن
//	تعتمد على قراءة الكود، أو grep، أو أن أضغط أنا شخصيّاً على كلّ زر».)
//
// # لماذا الموجّهُ الكاملُ لا خادمٌ مركَّبٌ باليد
//
// **اختباراتُ هذا المستودع تبني `Server{}` حقلاً حقلاً** — فتتخطّى
// الوسائطَ كلَّها: **التخويلَ ومنعَ التكرار وترجمةَ الأخطاء.** وهي حيث
// وقع BUG-002 بعينه: البابُ كان مسجَّلاً بلا لفّ، **والاختبارُ الذي
// ينادي المعالجَ رأسا لا يرى اللفّ أصلا.**
//
// **فيُركَّب هنا ما يُركَّب في `cmd/api` حرفاً بحرف** — ويُنادى عبر HTTP
// حقيقيّ. فما مرّ هنا مرّ بكلّ ما يمرّ به نداءُ الزبون.
//
// # ولا تمسّ الإنتاج
//
// **`testdb.Pool` يرفض قاعدةً لا ينتهي اسمُها بـ`_test`** — والحارسُ
// قائمٌ قبلنا. **وRedis في الذاكرة** (`miniredis`) فلا حدَّ إرسالٍ
// يتسرّب بين الاختبارات ولا خادمَ خارجيٌّ يلزم في CI.
//
// # والبياناتُ تُصنع ولا تُلتقط
//
// **اختبارٌ يقرأ «أوّلَ صنفٍ في السوق» يسقط يومَ يُحذف** — فكلُّ اختبارٍ
// يصنع ما يحتاجه بمعرّفاتٍ فريدة، **ويُنظّف بعده.**
package qa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/config"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/notify"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/server"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// jwtSecret **سرٌّ ثابتٌ للاختبار وحدَه** — ولا يُقرأ من البيئة كي لا
// يرث اختبارٌ سرَّ إنتاجٍ من صدفةٍ مفتوحة.
const jwtSecret = "qa-harness-secret-not-for-production-use"

// seq **عدّادٌ يجعل كلَّ صفٍّ فريداً** — والوقتُ وحدَه يتصادم حين تعمل
// حزمتان في الثانية نفسها.
var seq atomic.Int64

func uniq(prefix string) string {
	return fmt.Sprintf("%s%d%04d", prefix, time.Now().UnixNano()%1e10, seq.Add(1))
}

// Harness **النظامُ كلُّه خلف عنوانٍ واحد.**
type Harness struct {
	T      *testing.T
	Pool   *pgxpool.Pool
	Srv    *httptest.Server
	tokens *auth.TokenIssuer
}

// New **يُقلع النظامَ لاختبارٍ واحد** — ويُطفأ بعده كاملاً.
func New(t *testing.T) *Harness {
	t.Helper()
	pool := testdb.Pool(t) // **يتخطّى بهدوءٍ إن لم تُضبط قاعدةُ الاختبار**

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	quiet := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	tokens := auth.NewTokenIssuer(jwtSecret, 15*time.Minute)
	sender := &notify.DevSender{Logger: quiet}

	identitySvc := identity.NewService(identity.NewRepo(pool), rdb, tokens, sender, jwtSecret, quiet)
	settingsStore := settings.NewStore(pool)
	walletSvc := wallet.NewService(pool)
	cashboxSvc := cashbox.NewService(pool, settingsStore)
	hub := realtime.NewHub(quiet)
	ordersSvc := orders.NewService(pool, identitySvc, walletSvc, cashboxSvc, hub, quiet)
	catalogSvc := catalog.NewService(pool, identitySvc)
	supportSvc := support.NewService(pool, identitySvc, walletSvc)
	mediaSvc, err := media.NewService(pool, t.TempDir())
	if err != nil {
		t.Fatalf("qa: تعذّر تركيبُ خدمة الوسائط: %v", err)
	}

	cfg := &config.Config{Env: "test", JWTSecret: jwtSecret}
	s := server.New(cfg, quiet, pool, rdb, tokens, identitySvc, catalogSvc,
		settingsStore, walletSvc, ordersSvc, cashboxSvc, supportSvc, mediaSvc, hub,
		func() map[string]any { return map[string]any{"provider": "test"} },
		func(context.Context) error { return nil },
		func() {})

	ts := httptest.NewServer(s.Router())
	t.Cleanup(ts.Close)

	return &Harness{T: t, Pool: pool, Srv: ts, tokens: tokens}
}

// --- النداء ---------------------------------------------------------

// Res **ردٌّ مقروءٌ بلا تكرارِ فكِّ ترميز.**
type Res struct {
	Code int
	Body []byte
}

// JSON **يفكّ غلافَ `{"data": …}`** — وهو غلافُ كلّ ردٍّ في هذا المحرّك.
func (r Res) JSON() map[string]any {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if json.Unmarshal(r.Body, &env) == nil && env.Data != nil {
		return env.Data
	}
	var raw map[string]any
	_ = json.Unmarshal(r.Body, &raw)
	return raw
}

// Err **رمزُ الخطأ كما يقرؤه التطبيق** — لا نصُّه.
func (r Res) Err() string {
	var e struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(r.Body, &e)
	return e.Error.Code
}

func (r Res) String() string { return fmt.Sprintf("%d %s", r.Code, strings.TrimSpace(string(r.Body))) }

// Call **نداءٌ خامٌ بترويسات** — وبه تُبنى كلُّ المساعدات.
func (h *Harness) Call(method, path, token string, body any, headers map[string]string) Res {
	h.T.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			h.T.Fatalf("qa: تعذّر ترميزُ الجسم: %v", err)
		}
		rdr = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, h.Srv.URL+path, rdr)
	if err != nil {
		h.T.Fatalf("qa: نداءٌ معطوب: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := h.Srv.Client().Do(req)
	if err != nil {
		h.T.Fatalf("qa: تعذّر النداء %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return Res{Code: resp.StatusCode, Body: b}
}

func (h *Harness) GET(path, token string) Res { return h.Call("GET", path, token, nil, nil) }
func (h *Harness) DEL(path, token string) Res { return h.Call("DELETE", path, token, nil, nil) }
func (h *Harness) POST(path, token string, body any) Res {
	return h.Call("POST", path, token, body, nil)
}
func (h *Harness) PATCH(path, token string, body any) Res {
	return h.Call("PATCH", path, token, body, nil)
}

// POSTKey **نداءُ إنشاءٍ بمفتاحِ منعِ التكرار** — وهو نصفُ العقد الذي
// نُسي في التطبيق (BUG-001).
func (h *Harness) POSTKey(path, token, key string, body any) Res {
	return h.Call("POST", path, token, body, map[string]string{"Idempotency-Key": key})
}

// --- المصانع --------------------------------------------------------

// User **حسابٌ جاهزٌ بتوكنِه** — ولا يمرّ برمزِ واتساب.
//
// **ورمزُ التوثيق ليس موضوعَ كلّ اختبار**: من أراد اختبارَ IDOR لا يريد
// أن يسقط اختبارُه لأنّ حدَّ الإرسال بلغ حدَّه.
type User struct {
	ID    string
	Phone string
	Name  string
	Roles []string
	Token string
}

// NewUser **يصنع حساباً بدورٍ ويُصدر توكنَه.**
//
// # وموثَّقٌ على واتساب
//
// **قاعدةُ عملٍ حقيقيّةٌ اكتشفها الهيكلُ ٢٠٢٦-٠٨-١٩**: الطلبُ يُردّ
// بـwhatsapp_required ما لم يكن الحسابُ موثَّقا. **ولم تكن مكتوبةً في
// وثيقة** — وهذا أوّلُ ما كشفته المنظومة.
func (h *Harness) NewUser(role string) *User {
	h.T.Helper()
	phone := uniq("+96390")
	name := "QA " + role
	var id string
	err := h.Pool.QueryRow(context.Background(), `
		INSERT INTO users (phone, full_name, status, whatsapp_phone, whatsapp_verified_at)
		VALUES ($1, $2, 'active', $1, now()) RETURNING id::text`, phone, name).Scan(&id)
	if err != nil {
		h.T.Fatalf("qa: تعذّر إنشاءُ مستخدم: %v", err)
	}
	roles := []string{role}
	if role != "" {
		_, err = h.Pool.Exec(context.Background(), `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, $2)
			ON CONFLICT DO NOTHING`, id, role)
		if err != nil {
			h.T.Fatalf("qa: تعذّر إسنادُ الدور %q: %v", role, err)
		}
	}
	h.T.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1::uuid`, id)
	})
	tok, _, err := h.tokens.IssueAccess(id, roles, "")
	if err != nil {
		h.T.Fatalf("qa: تعذّر إصدارُ التوكن: %v", err)
	}
	return &User{ID: id, Phone: phone, Name: name, Roles: roles, Token: tok}
}

// Customer **زبونٌ جاهز** — أشيعُ ما تحتاجه الاختبارات.
func (h *Harness) Customer() *User { return h.NewUser("customer") }

// TokenFor **توكنٌ بأدوارٍ مصنوعة** — لاختبار حراسة الأدوار.
func (h *Harness) TokenFor(userID string, roles ...string) string {
	h.T.Helper()
	tok, _, err := h.tokens.IssueAccess(userID, roles, "")
	if err != nil {
		h.T.Fatalf("qa: تعذّر إصدارُ التوكن: %v", err)
	}
	return tok
}

// ExpiredToken **توكنٌ منتهٍ** — يُصدَر بمُصدِرٍ عمرُه سالب.
func (h *Harness) ExpiredToken(userID string, roles ...string) string {
	h.T.Helper()
	past := auth.NewTokenIssuer(jwtSecret, -time.Minute)
	tok, _, err := past.IssueAccess(userID, roles, "")
	if err != nil {
		h.T.Fatalf("qa: تعذّر إصدارُ توكنٍ منتهٍ: %v", err)
	}
	return tok
}

// Item **صنفٌ في قسمٍ لمتجرٍ فعّال** — الحدُّ الأدنى لطلبٍ صحيح.
type Item struct {
	ID         string
	SectionID  string
	MerchantID string
	Name       string
	Cost       int64
}

// NewItem **يصنع متجراً وقسماً وصنفاً** بسعرٍ معلوم.
//
// **والسعرُ يُمرَّر لأنّ اختبارَ الحسابِ يحتاج رقماً يعرفه** — واختبارٌ
// يقرأ السعرَ من الصفّ الذي أنشأه لا يكشف خطأً في الحساب.
func (h *Harness) NewItem(cost int64) *Item {
	h.T.Helper()
	ctx := context.Background()
	owner := h.NewUser("merchant")

	// **وكلُّ متجرٍ في تصنيف** — قيدٌ في القاعدة، فيُصنع معه.
	var categoryID string
	if err := h.Pool.QueryRow(ctx, `
		INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("تصنيف QA ")).Scan(&categoryID); err != nil {
		h.T.Fatalf("qa: تعذّر إنشاءُ تصنيف: %v", err)
	}

	var merchantID string
	err := h.Pool.QueryRow(ctx, `
		INSERT INTO merchants (name, owner_user_id, category_id, status)
		VALUES ($1, $2::uuid, $3::uuid, 'active') RETURNING id::text`,
		uniq("متجر QA "), owner.ID, categoryID).Scan(&merchantID)
	if err != nil {
		h.T.Fatalf("qa: تعذّر إنشاءُ متجر: %v", err)
	}

	var sectionID string
	err = h.Pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, active)
		VALUES ($1, true) RETURNING id::text`, uniq("قسم QA ")).Scan(&sectionID)
	if err != nil {
		h.T.Fatalf("qa: تعذّر إنشاءُ قسم: %v", err)
	}

	var itemID string
	itemName := uniq("صنف QA ")
	err = h.Pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, platform_section_id, name,
		                        price, merchant_price, available, approved)
		VALUES ($1::uuid, $2::uuid, $3, $4, $4, true, true) RETURNING id::text`,
		merchantID, sectionID, itemName, cost).Scan(&itemID)
	if err != nil {
		h.T.Fatalf("qa: تعذّر إنشاءُ صنف: %v", err)
	}

	h.T.Cleanup(func() {
		_, _ = h.Pool.Exec(ctx, `DELETE FROM menu_items WHERE id = $1::uuid`, itemID)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM platform_sections WHERE id = $1::uuid`, sectionID)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM merchants WHERE id = $1::uuid`, merchantID)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM categories WHERE id = $1::uuid`, categoryID)
	})
	return &Item{ID: itemID, SectionID: sectionID, MerchantID: merchantID,
		Name: itemName, Cost: cost}
}

// Setting **يضبط إعداداً ويُعيده بعد الاختبار** — والإعدادُ المتروكُ
// يسمّم كلَّ اختبارٍ بعده.
//
// # وجدولٌ خطأٌ بقي حتّى ناداه أوّلُ اختبار
//
// **كُتبت هذه الدالّةُ على جدولٍ اسمه `settings` ولا وجودَ له** — واسمُه
// `app_settings`. **وبقيت شهراً بلا منادٍ**، فلم يكشفها بناءٌ ولا فحص:
// **الشيفرةُ الميتةُ تُترجَم صحيحةً وهي كاذبة.** كشفها أوّلُ نداءٍ لها
// (CITY-011، ٢٠٢٦-٠٨-٢٠).
//
// **والقيمةُ `jsonb`** — فنصٌّ خامٌّ يُرفض، والرقمُ يُكتب رقماً.
func (h *Harness) Setting(key, value string) {
	h.T.Helper()
	ctx := context.Background()
	var old *string
	_ = h.Pool.QueryRow(ctx,
		`SELECT value::text FROM app_settings WHERE key = $1`, key).Scan(&old)
	if _, err := h.Pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, value); err != nil {
		h.T.Fatalf("qa: تعذّر ضبطُ الإعداد %q: %v", key, err)
	}
	h.T.Cleanup(func() {
		if old == nil {
			_, _ = h.Pool.Exec(ctx, `DELETE FROM app_settings WHERE key = $1`, key)
			return
		}
		_, _ = h.Pool.Exec(ctx,
			`UPDATE app_settings SET value = $2::jsonb WHERE key = $1`, key, *old)
	})
}

// CountOrders **كم طلباً لهذا الزبون** — مقياسُ كلّ اختبارِ تكرار.
func (h *Harness) CountOrders(customerID string) int {
	h.T.Helper()
	var n int
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, customerID).Scan(&n); err != nil {
		h.T.Fatalf("qa: تعذّر عدُّ الطلبات: %v", err)
	}
	return n
}

// Enabled **أتعمل هذه الحزمة؟** — تُستعمل في التقارير لا في التخطّي.
func Enabled() bool { return os.Getenv("TEST_DATABASE_URL") != "" }

// signWith **يوقّع توكناً بسرٍّ يُملى** — لاختبار التزوير وحدَه.
//
// **ولا يمرّ عبر `TokenIssuer`** عمداً: المُصدِرُ يحمل سرَّ الخادم،
// **واختبارُ التزوير يحتاج سرّاً غيرَه.**
func signWith(secret, userID string, roles []string) string {
	claims := jwt.MapClaims{
		"sub":   userID,
		"roles": roles,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"iss":   "rahalgo",
	}
	s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return s
}
