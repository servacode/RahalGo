// Command ameen-sync **ساعي رحّال غو على كمبيوتر المول** (قرارُ المالك ٢٠٢٦-١٠-١٩).
//
// يقرأ أسعارَ الموادّ وكميّاتِها من **برنامج الأمين** ويبعثها إلى رحّال غو برمز المتجر:
//
//   - **sql**    يقرأ من قاعدة بيانات الأمين مباشرةً — **قراءةً فقط** — بجملةٍ في الإعدادات.
//   - **folder** يراقب مجلّداً يصدّر إليه المحاسبُ ملفَّ Excel، ويرفع الأحدث.
//
// **ويسأل الخادمَ كلَّ دقيقة** — فتعرف لوحةُ المتجر أنّه حيّ، **وتصله «مزامنة الآن».**
//
// الأوامر:
//
//	ameen-sync            يعمل دائماً (ما يُشغَّل عند الإقلاع)
//	ameen-sync once       مزامنةٌ واحدةٌ ثمّ يخرج
//	ameen-sync test       «جرّب» — يقول ما سيتغيّر ولا يغيّر شيئاً
//	ameen-sync tables     جداولُ قاعدة الأمين وأعمدتُها — لزيارة المول
//	ameen-sync version
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

// Version **نسختُه** — تظهر في لوحة المتجر.
const Version = "1.0.0"

// Config **ملفُّ الإعدادات** — `config.json` بجانب البرنامج.
type Config struct {
	Server          string `json:"server"`
	Token           string `json:"token"`
	Mode            string `json:"mode"`
	IntervalMinutes int    `json:"interval_minutes"`
	SQL             struct {
		Connection string `json:"connection"`
		Query      string `json:"query"`
		Currency   string `json:"currency"`
	} `json:"sql"`
	Folder struct {
		Path    string `json:"path"`
		Pattern string `json:"pattern"`
	} `json:"folder"`
}

// Row **مادّةٌ كما يفهمها الخادم** — `catalog.SyncRow`.
type Row struct {
	Code    string   `json:"code"`
	Barcode string   `json:"barcode,omitempty"`
	Name    string   `json:"name,omitempty"`
	Price   *float64 `json:"price,omitempty"`
	Qty     *float64 `json:"qty,omitempty"`
}

var dir = func() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}()

func loadConfig() (*Config, error) {
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("ما لقيت config.json جنب البرنامج: %w", err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("config.json فيه غلط: %w", err)
	}
	c.Server = strings.TrimRight(strings.TrimSpace(c.Server), "/")
	if c.Server == "" {
		c.Server = "https://api.rahalgo.com"
	}
	if c.IntervalMinutes <= 0 {
		c.IntervalMinutes = 1
	}
	if c.Mode == "" {
		c.Mode = "sql"
	}
	if c.Folder.Pattern == "" {
		c.Folder.Pattern = "*.xlsx"
	}
	if strings.TrimSpace(c.Token) == "" {
		return nil, errors.New("لازم تحط رمز المزامنة (token) بـ config.json — بيطلع من لوحة المتجر")
	}
	return &c, nil
}

func setupLog(console bool) {
	path := filepath.Join(dir, "sync.log")
	if st, err := os.Stat(path); err == nil && st.Size() > 5<<20 {
		_ = os.Rename(path, path+".old")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	if console {
		log.SetOutput(io.MultiWriter(os.Stdout, f))
	} else {
		log.SetOutput(f)
	}
}

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if cmd == "version" {
		fmt.Println("ameen-sync", Version)
		return
	}
	setupLog(cmd != "")
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	switch cmd {
	case "":
		runForever(ctx, cfg)
	case "once":
		if err := syncOnce(ctx, cfg, false, true); err != nil {
			log.Fatal(err)
		}
	case "test":
		if err := syncOnce(ctx, cfg, true, true); err != nil {
			log.Fatal(err)
		}
	case "tables":
		if err := listTables(ctx, cfg); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("أمر غير معروف: %s — الأوامر: once · test · tables · version", cmd)
	}
}

// runForever **يسأل كلَّ دقيقة، ويزامن كلَّ مهلةٍ أو حين يُطلب.**
func runForever(ctx context.Context, cfg *Config) {
	log.Printf("بدأ ساعي رحّال غو %s — الطريقة %s", Version, cfg.Mode)
	var last time.Time
	for {
		now, every, err := poll(ctx, cfg)
		if err != nil {
			log.Printf("ما قدرت أتصل بالخادم: %v", err)
		}
		// **والمدّةُ من لوحة المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٩) — وملفُّ الإعدادات احتياطٌ إن لم يُجب الخادم.
		if every <= 0 {
			every = cfg.IntervalMinutes
		}
		if now || time.Since(last) >= time.Duration(every)*time.Minute-5*time.Second {
			// **و«مزامنة الآن» ترفع ولو لم يتبدّل الملفّ** — وإلّا بقي الطلبُ معلّقاً في اللوحة.
			if err := syncOnce(ctx, cfg, false, now); err != nil {
				log.Printf("المزامنة فشلت: %v", err)
			}
			last = time.Now()
		}
		time.Sleep(time.Minute)
	}
}

func request(ctx context.Context, cfg *Config, method, path string, body io.Reader, ctype string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, cfg.Server+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Sync-Token", strings.TrimSpace(cfg.Token))
	req.Header.Set("X-Agent-Version", Version)
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("الرمز مرفوض — اعمل رمز جديد من لوحة المتجر وحطه بـ config.json")
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("الخادم ردّ %d: %s", res.StatusCode, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func poll(ctx context.Context, cfg *Config) (bool, int, error) {
	b, err := request(ctx, cfg, http.MethodGet, "/api/v1/public/price-sync/poll", nil, "")
	if err != nil {
		return false, 0, err
	}
	var env struct {
		Data struct {
			SyncNow  bool `json:"sync_now"`
			Interval int  `json:"interval_minutes"`
		} `json:"data"`
	}
	_ = json.Unmarshal(b, &env)
	return env.Data.SyncNow, env.Data.Interval, nil
}

// lastSent **بصمةُ آخر قائمةٍ بُعثت** — فلا يُبعث ما لم يتبدّل فيه سعرٌ ولا كميّة.
var lastSent string

func syncOnce(ctx context.Context, cfg *Config, dry, force bool) error {
	var out []byte
	var err error
	switch cfg.Mode {
	case "sql":
		var rows []Row
		if rows, err = readSQL(ctx, cfg); err != nil {
			return err
		}
		sum := sha256.Sum256(must(json.Marshal(rows)))
		mark := hex.EncodeToString(sum[:])
		if !dry && !force && mark == lastSent {
			return nil // **ما تبدّل شيء** — لا يُتعب الخادمُ كلَّ دقيقة.
		}
		body, _ := json.Marshal(map[string]any{"rows": rows, "currency": cfg.SQL.Currency, "dry_run": dry})
		out, err = request(ctx, cfg, http.MethodPost, "/api/v1/public/price-sync", bytes.NewReader(body), "application/json")
		if err == nil && !dry {
			lastSent = mark
		}
	case "folder":
		out, err = uploadNewest(ctx, cfg, dry, force)
		if out == nil && err == nil {
			return nil
		}
	default:
		return fmt.Errorf("الطريقة «%s» غير معروفة — sql أو folder", cfg.Mode)
	}
	if err != nil {
		return err
	}
	logReport(out, dry)
	return nil
}

func logReport(b []byte, dry bool) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(b, &env)
	r := env.Data
	if r == nil {
		_ = json.Unmarshal(b, &r)
	}
	what := "تمّت المزامنة"
	if dry {
		what = "تجربة (ما تغيّر شي)"
	}
	log.Printf("%s — انطابقت %v، سعر تغيّر %v، صار متوفر %v، صار غير متوفر %v، ما انطابقت %v",
		what, r["matched"], r["price_changed"], r["made_available"], r["made_unavailable"], r["unmatched"])
}

// readSQL **يقرأ الموادّ بجملة الإعدادات** — وأعمدتُها بأسمائها: code · barcode · name · price · qty.
func readSQL(ctx context.Context, cfg *Config) ([]Row, error) {
	if strings.TrimSpace(cfg.SQL.Query) == "" {
		return nil, errors.New("لازم تحط جملة القراءة (sql.query) بـ config.json — شغّل «ameen-sync tables» لتشوف الجداول")
	}
	db, err := sql.Open("sqlserver", cfg.SQL.Connection)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	rs, err := db.QueryContext(ctx, cfg.SQL.Query)
	if err != nil {
		return nil, fmt.Errorf("قراءة الأمين فشلت: %w", err)
	}
	defer rs.Close()
	cols, err := rs.Columns()
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for i, c := range cols {
		idx[strings.ToLower(strings.TrimSpace(c))] = i
	}
	if _, ok := idx["price"]; !ok {
		return nil, errors.New("جملة القراءة لازم ترجّع عمود اسمه price (… AS price)")
	}
	_, hasCode := idx["code"]
	_, hasName := idx["name"]
	if !hasCode && !hasName {
		return nil, errors.New("جملة القراءة لازم ترجّع عمود code أو name")
	}
	var out []Row
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rs.Next() {
		if err := rs.Scan(ptrs...); err != nil {
			return nil, err
		}
		str := func(k string) string {
			i, ok := idx[k]
			if !ok {
				return ""
			}
			return strings.TrimSpace(toString(vals[i]))
		}
		num := func(k string) *float64 {
			i, ok := idx[k]
			if !ok || vals[i] == nil {
				return nil
			}
			v, err := strconv.ParseFloat(strings.TrimSpace(toString(vals[i])), 64)
			if err != nil {
				return nil
			}
			return &v
		}
		out = append(out, Row{Code: str("code"), Barcode: str("barcode"), Name: str("name"), Price: num("price"), Qty: num("qty")})
	}
	log.Printf("قريت %d مادة من الأمين", len(out))
	return out, rs.Err()
}

func toString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case []byte:
		return string(t)
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		return fmt.Sprint(t)
	}
}

// uploadNewest **يرفع أحدثَ ملفٍّ في المجلّد** — مرّةً لكلّ نسخة (بتاريخ تعديله).
func uploadNewest(ctx context.Context, cfg *Config, dry, force bool) ([]byte, error) {
	matches, err := filepath.Glob(filepath.Join(cfg.Folder.Path, cfg.Folder.Pattern))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		log.Printf("ما في ملفات بالمجلد %s", cfg.Folder.Path)
		return nil, nil
	}
	sort.Slice(matches, func(i, j int) bool {
		a, _ := os.Stat(matches[i])
		b, _ := os.Stat(matches[j])
		return a.ModTime().After(b.ModTime())
	})
	newest := matches[0]
	st, err := os.Stat(newest)
	if err != nil {
		return nil, err
	}
	stamp := filepath.Join(dir, "last-upload.txt")
	mark := fmt.Sprintf("%s|%d", newest, st.ModTime().Unix())
	if prev, _ := os.ReadFile(stamp); !dry && !force && string(prev) == mark {
		return nil, nil
	}
	data, err := os.ReadFile(newest)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", filepath.Base(newest))
	_, _ = fw.Write(data)
	if dry {
		_ = mw.WriteField("dry_run", "1")
	}
	_ = mw.Close()
	out, err := request(ctx, cfg, http.MethodPost, "/api/v1/public/price-sync/file", &buf, mw.FormDataContentType())
	if err == nil && !dry {
		_ = os.WriteFile(stamp, []byte(mark), 0o644)
	}
	log.Printf("رفعت الملف %s", filepath.Base(newest))
	return out, err
}

// listTables **جداولُ قاعدة الأمين وأعمدتُها** — لنعرف في زيارة المول أين الأسعارُ والكميّات.
func listTables(ctx context.Context, cfg *Config) error {
	db, err := sql.Open("sqlserver", cfg.SQL.Connection)
	if err != nil {
		return err
	}
	defer db.Close()
	rs, err := db.QueryContext(ctx, `
		SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS
		 ORDER BY TABLE_NAME, ORDINAL_POSITION`)
	if err != nil {
		return fmt.Errorf("ما قدرت أفتح قاعدة الأمين: %w", err)
	}
	defer rs.Close()
	cur := ""
	for rs.Next() {
		var t, c, ty string
		if err := rs.Scan(&t, &c, &ty); err != nil {
			return err
		}
		if t != cur {
			fmt.Printf("\n== %s\n", t)
			cur = t
		}
		fmt.Printf("   %s (%s)\n", c, ty)
	}
	return rs.Err()
}

func must(b []byte, _ error) []byte { return b }
