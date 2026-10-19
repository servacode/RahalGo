// Command ameen-sync **برنامج مزامنة رحّال غو على كمبيوتر المول** (قرارُ المالك ٢٠٢٦-١٠-١٩).
//
// «ملف واحد يشتغل بالخلفية تلقائي، ويشتغل لحاله بعد إعادة التشغيل… ما بدنا المحاسب
// يتلبك». فهو **ملفٌّ واحدٌ يُحمَّل من لوحة المتجر وفي ذيله رمزُ المتجر**:
//
//   - **دبل كليك** ⇒ ينصّب نفسَه في `%LOCALAPPDATA%\RahalGoSync`، ويضيف نفسَه إلى إقلاع
//     ويندوز (بلا صلاحيات مدير)، ويعمل في الخلفية، ويقول «تمّ» مرّةً واحدة.
//   - **ولا إعدادَ على الجهاز**: يسأل الخادمَ كلَّ دقيقة فيأخذ طريقةَ القراءة والمدّة،
//     **ويجد قاعدةَ الأمين وحدَه** على الجهاز، ويبعث حالَه وخطأَه وجداولَ الأمين.
//   - **ويحدّث نفسَه** إن صدرت نسخةٌ أحدث، **ويزيل نفسَه** إن ضُغط «إيقاف» في اللوحة.
//
// **ويقرأ فقط**: جملةُ القراءة `SELECT` خالصة، وتُردّ كلُّ جملةٍ تكتب.
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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

// Version **نسختُه** — يقرؤها الخادمُ عند البناء ليعرف من يحتاج تحديثاً.
var Version = "2.0.0"

const marker = "\n##RAHALGO-SYNC-CONFIG##"

// Local **ما يحفظه على الجهاز** — الخادمُ ورمزُ المتجر وحدهما.
type Local struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

// Remote **ما يأخذه من الخادم كلَّ دقيقة** — `agentConfig` هناك.
type Remote struct {
	SyncNow       bool   `json:"sync_now"`
	Interval      int    `json:"interval_minutes"`
	HasMapping    bool   `json:"has_mapping"`
	Stop          bool   `json:"stop"`
	SQLQuery      string `json:"sql_query"`
	SQLConnection string `json:"sql_connection"`
	ProbeTable    string `json:"probe_table"`
	Folder        string `json:"folder"`
	Currency      string `json:"currency"`
	WantSchema    bool   `json:"want_schema"`
	LatestVersion string `json:"latest_version"`
}

// Row **مادّةٌ كما يفهمها الخادم.**
type Row struct {
	Code    string   `json:"code"`
	Barcode string   `json:"barcode,omitempty"`
	Name    string   `json:"name,omitempty"`
	Price   *float64 `json:"price,omitempty"`
	Qty     *float64 `json:"qty,omitempty"`
}

func installDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "RahalGoSync")
}

func selfPath() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// payload **إعدادُ المتجر من ذيل الملفّ** — كتبه الخادمُ عند التحميل.
func payload(exe string) (*Local, bool) {
	b, err := os.ReadFile(exe)
	if err != nil {
		return nil, false
	}
	i := bytes.LastIndex(b, []byte(marker))
	if i < 0 {
		return nil, false
	}
	var l Local
	if json.Unmarshal(b[i+len(marker):], &l) != nil || l.Token == "" {
		return nil, false
	}
	return &l, true
}

func main() {
	dir := installDir()
	_ = os.MkdirAll(dir, 0o755)
	setupLog(dir)
	self := selfPath()
	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	switch arg {
	case "version":
		fmt.Println(Version)
		return
	case "uninstall":
		uninstall(dir)
		notify("تمّ إيقاف برنامج مزامنة رحّال غو وإزالته من التشغيل التلقائي.")
		return
	}

	// **من خارج مجلّده ⇒ تنصيب** (دبل كليك على الملفّ المحمَّل).
	if !strings.EqualFold(filepath.Dir(self), dir) {
		l, ok := payload(self)
		if !ok {
			notify("هالملف ناقص رمز المتجر. حمّل البرنامج من جديد من لوحة المتجر برحّال غو (الأصناف ← مزامنة الأسعار من الأمين).")
			return
		}
		if err := install(dir, self, l); err != nil {
			log.Printf("التنصيب فشل: %v", err)
			notify("ما قدرنا ننصّب البرنامج: " + err.Error())
			return
		}
		notify("تمّ ✓ برنامج مزامنة رحّال غو شغّال بالخلفية.\nبيشتغل لحاله كل ما يقلع الكمبيوتر، وما بيحتاج منك شي.")
		return
	}

	// **من مجلّده ⇒ يعمل** — نسخةٌ واحدةٌ فقط.
	if !singleInstance() {
		return
	}
	_ = os.Remove(filepath.Join(dir, "RahalGoSync.old.exe"))
	l, err := loadLocal(dir)
	if err != nil {
		log.Printf("ما في إعداد: %v", err)
		return
	}
	run(context.Background(), dir, l)
}

func loadLocal(dir string) (*Local, error) {
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return nil, err
	}
	var l Local
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	l.Server = strings.TrimRight(l.Server, "/")
	return &l, nil
}

// install **ينسخ نفسَه إلى مجلّده ويضيف نفسَه إلى الإقلاع ويعمل.**
//
// **ونسخةٌ تعمل قبله لا تمنعه**: يُعاد تسمية الملفّ العامل (يسمح به ويندوز) ويوضع الجديد
// مكانَه، **ويُطلب من القديم أن يخرج** فيأخذ الجديدُ مكانَه.
func install(dir, self string, l *Local) error {
	cfg, _ := json.Marshal(l)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), cfg, 0o600); err != nil {
		return err
	}
	dst := filepath.Join(dir, "RahalGoSync.exe")
	old := filepath.Join(dir, "RahalGoSync.old.exe")
	_ = os.Remove(old)
	if _, err := os.Stat(dst); err == nil {
		if err := os.Rename(dst, old); err != nil {
			return err
		}
	}
	if err := copyFile(self, dst); err != nil {
		return err
	}
	if err := setAutostart(dst); err != nil {
		return err
	}
	askOldToExit(dir)
	return start(dst)
}

func uninstall(dir string) {
	removeAutostart()
	askOldToExit(dir)
	_ = os.Remove(filepath.Join(dir, "config.json"))
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func start(exe string) error {
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	detach(cmd)
	return cmd.Start()
}

// askOldToExit **ملفُّ «اخرج»** — النسخةُ العاملةُ تراه في دورتها فتخرج.
func askOldToExit(dir string) {
	_ = os.WriteFile(filepath.Join(dir, "exit.flag"), []byte(time.Now().Format(time.RFC3339)), 0o644)
	time.Sleep(3 * time.Second)
	_ = os.Remove(filepath.Join(dir, "exit.flag"))
}

func setupLog(dir string) {
	path := filepath.Join(dir, "sync.log")
	if st, err := os.Stat(path); err == nil && st.Size() > 5<<20 {
		_ = os.Rename(path, path+".old")
	}
	if f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		log.SetOutput(f)
	}
}

// ══════════════════════════════════════════════════════════════════════
// الدورة
// ══════════════════════════════════════════════════════════════════════

type state struct {
	dir       string
	local     *Local
	lastErr   string
	lastSync  time.Time
	lastSent  string
	lastFile  string
	connFound string
	dbFound   string
	mode      string
}

func run(ctx context.Context, dir string, l *Local) {
	st := &state{dir: dir, local: l}
	log.Printf("بدأ برنامج مزامنة رحّال غو %s", Version)
	for {
		if _, err := os.Stat(filepath.Join(dir, "exit.flag")); err == nil {
			log.Printf("طُلب الخروج — نسخةٌ أحدث تأخذ مكاني")
			return
		}
		if fresh, err := loadLocal(dir); err == nil {
			st.local = fresh
		}
		rc, err := st.poll(ctx)
		if err != nil {
			log.Printf("ما قدرت أتصل بالخادم: %v", err)
			time.Sleep(time.Minute)
			continue
		}
		if rc.Stop {
			log.Printf("أُوقف من لوحة المتجر")
			uninstall(dir)
			return
		}
		if rc.LatestVersion != "" && rc.LatestVersion != Version {
			if err := st.update(ctx); err != nil {
				st.lastErr = "التحديث فشل: " + err.Error()
			} else {
				return
			}
		}
		if rc.WantSchema {
			st.sendSchema(ctx, rc)
		}
		every := rc.Interval
		if every <= 0 {
			every = 1
		}
		if rc.SyncNow || time.Since(st.lastSync) >= time.Duration(every)*time.Minute-5*time.Second {
			if err := st.sync(ctx, rc, rc.SyncNow); err != nil {
				st.lastErr = err.Error()
				log.Printf("المزامنة فشلت: %v", err)
			} else {
				st.lastErr = ""
			}
			st.lastSync = time.Now()
		}
		time.Sleep(time.Minute)
	}
}

func (st *state) request(ctx context.Context, method, path string, body io.Reader, ctype string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, st.local.Server+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Sync-Token", st.local.Token)
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
		return nil, errors.New("رمز المتجر مرفوض — حمّل البرنامج من جديد من لوحة المتجر")
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("الخادم ردّ %d: %s", res.StatusCode, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func (st *state) poll(ctx context.Context) (*Remote, error) {
	body, _ := json.Marshal(map[string]any{
		"version": Version, "error": st.lastErr,
		"status": map[string]any{"mode": st.mode, "connection": redact(st.connFound), "database": st.dbFound,
			"last_sync": st.lastSync},
	})
	b, err := st.request(ctx, http.MethodPost, "/api/v1/public/price-sync/poll", bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}
	var env struct {
		Data Remote `json:"data"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// redact **عنوانُ القاعدة بلا كلمة السرّ** — لا تُبعث كلمةُ سرٍّ إلى أحد.
func redact(conn string) string {
	u, err := url.Parse(conn)
	if err != nil || u.User == nil {
		return conn
	}
	u.User = url.User(u.User.Username())
	return u.String()
}

// update **يحدّث نفسَه** — ينزّل الأحدث، ويضعه مكانَه، ويشغّله، ويخرج.
func (st *state) update(ctx context.Context) error {
	b, err := st.request(ctx, http.MethodGet, "/api/v1/public/price-sync/agent", nil, "")
	if err != nil {
		return err
	}
	if len(b) < 1<<20 || !bytes.HasPrefix(b, []byte("MZ")) {
		return errors.New("الملف المنزّل مو برنامج ويندوز")
	}
	dst := filepath.Join(st.dir, "RahalGoSync.exe")
	old := filepath.Join(st.dir, "RahalGoSync.old.exe")
	tmp := filepath.Join(st.dir, "RahalGoSync.new.exe")
	if err := os.WriteFile(tmp, b, 0o755); err != nil {
		return err
	}
	_ = os.Remove(old)
	if err := os.Rename(dst, old); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Rename(old, dst)
		return err
	}
	log.Printf("تحدّثت إلى النسخة الأحدث")
	releaseInstance()
	return start(dst)
}

// ══════════════════════════════════════════════════════════════════════
// القراءة
// ══════════════════════════════════════════════════════════════════════

var forbidden = regexp.MustCompile(`(?i)\b(insert|update|delete|drop|alter|create|truncate|merge|exec|execute|grant|revoke|deny|backup|restore|shutdown|dbcc|into|openrowset|opendatasource|xp_\w+|sp_\w+)\b`)

// safeQuery **قراءةٌ لا كتابة** — والخادمُ يحرسها كذلك؛ وهنا حارسٌ ثانٍ على كمبيوتر المول.
func safeQuery(q string) bool {
	t := strings.TrimSpace(q)
	low := strings.ToLower(t)
	return t != "" && (strings.HasPrefix(low, "select") || strings.HasPrefix(low, "with")) &&
		!strings.Contains(t, ";") && !strings.Contains(t, "--") && !strings.Contains(t, "/*") &&
		!forbidden.MatchString(t)
}

func (st *state) sync(ctx context.Context, rc *Remote, force bool) error {
	if strings.TrimSpace(rc.Folder) != "" {
		st.mode = "folder"
		return st.syncFolder(ctx, rc.Folder, force)
	}
	st.mode = "sql"
	if strings.TrimSpace(rc.SQLQuery) == "" {
		return errors.New("بانتظار ضبط جملة القراءة من فريق رحّال غو — البرنامج شغّال وبعت جداول الأمين")
	}
	if !safeQuery(rc.SQLQuery) {
		return errors.New("جملة القراءة مرفوضة: لازم تكون قراءة بس (SELECT)")
	}
	db, err := st.open(ctx, rc)
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := readRows(ctx, db, rc.SQLQuery)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(mustJSON(rows))
	mark := hex.EncodeToString(sum[:])
	if !force && mark == st.lastSent {
		return nil
	}
	cur := rc.Currency
	if cur == "" {
		cur = "SYP"
	}
	body, _ := json.Marshal(map[string]any{"rows": rows, "currency": cur})
	out, err := st.request(ctx, http.MethodPost, "/api/v1/public/price-sync", bytes.NewReader(body), "application/json")
	if err != nil {
		return err
	}
	st.lastSent = mark
	logReport(out)
	return nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func logReport(b []byte) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(b, &env)
	r := env.Data
	log.Printf("تمّت المزامنة — انطابقت %v، سعر تغيّر %v، صار متوفر %v، صار غير متوفر %v، ما انطابقت %v",
		r["matched"], r["price_changed"], r["made_available"], r["made_unavailable"], r["unmatched"])
}

// open **قاعدةُ الأمين** — بعنوان الخادم إن ضُبط، وإلّا يبحث عنها على الجهاز وحده.
func (st *state) open(ctx context.Context, rc *Remote) (*sql.DB, error) {
	if strings.TrimSpace(rc.SQLConnection) != "" {
		st.connFound = rc.SQLConnection
		return sql.Open("sqlserver", rc.SQLConnection)
	}
	if st.connFound != "" {
		if db, err := sql.Open("sqlserver", st.connFound); err == nil && db.PingContext(ctx) == nil {
			return db, nil
		}
	}
	conn, dbName, err := findAmeen(ctx, rc.ProbeTable)
	if err != nil {
		return nil, err
	}
	st.connFound, st.dbFound = conn, dbName
	log.Printf("لقيت قاعدة الأمين: %s", dbName)
	return sql.Open("sqlserver", conn)
}

// findAmeen **يبحث عن قاعدة الأمين على الجهاز** — في كلّ نسخةٍ من SQL Server مسجّلة،
// بدخول ويندوز، **عن قاعدةٍ فيها جدولُ المواد** (`probe_table`).
func findAmeen(ctx context.Context, probe string) (string, string, error) {
	if probe == "" {
		probe = "mt000"
	}
	hosts := []string{"localhost"}
	for _, inst := range sqlInstances() {
		if !strings.EqualFold(inst, "MSSQLSERVER") {
			hosts = append(hosts, "localhost/"+inst)
		}
	}
	hosts = append(hosts, "localhost/SQLEXPRESS")
	seen := map[string]bool{}
	for _, h := range hosts {
		if seen[strings.ToLower(h)] {
			continue
		}
		seen[strings.ToLower(h)] = true
		base := "sqlserver://" + h + "?encrypt=disable&connection+timeout=8"
		db, err := sql.Open("sqlserver", base+"&database=master")
		if err != nil {
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		names, err := userDatabases(cctx, db)
		cancel()
		db.Close()
		if err != nil {
			continue
		}
		for _, n := range names {
			conn := base + "&database=" + url.QueryEscape(n)
			d, err := sql.Open("sqlserver", conn)
			if err != nil {
				continue
			}
			var id *int64
			err = d.QueryRowContext(ctx, `SELECT OBJECT_ID(@p1)`, "dbo."+probe).Scan(&id)
			d.Close()
			if err == nil && id != nil {
				return conn, n, nil
			}
		}
	}
	return "", "", fmt.Errorf("ما لقيت قاعدة الأمين على هالكمبيوتر (جدول %s) — تأكد إنو البرنامج منصّب على كمبيوتر الأمين", probe)
}

func userDatabases(ctx context.Context, db *sql.DB) ([]string, error) {
	rs, err := db.QueryContext(ctx, `SELECT name FROM sys.databases WHERE database_id > 4 AND state = 0`)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	var out []string
	for rs.Next() {
		var n string
		if rs.Scan(&n) == nil {
			out = append(out, n)
		}
	}
	return out, rs.Err()
}

// readRows **الموادّ بجملة القراءة** — بلا أقفالٍ على حسابات المول (`READ UNCOMMITTED`).
func readRows(ctx context.Context, db *sql.DB, query string) ([]Row, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("ما قدرت أفتح قاعدة الأمين: %w", err)
	}
	defer conn.Close()
	_, _ = conn.ExecContext(ctx, `SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED`)
	rs, err := conn.QueryContext(ctx, query)
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
		return nil, errors.New("جملة القراءة لازم ترجّع عمود price")
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

// sendSchema **جداولُ الأمين وأعمدتُها وعددُ صفوفها** — ليُضبط سطرُ القراءة من بعيد.
func (st *state) sendSchema(ctx context.Context, rc *Remote) {
	db, err := st.open(ctx, rc)
	if err != nil {
		st.lastErr = err.Error()
		return
	}
	defer db.Close()
	cctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	rs, err := db.QueryContext(cctx, `
		SELECT t.name, c.name, ty.name,
		       (SELECT SUM(p.rows) FROM sys.partitions p WHERE p.object_id = t.object_id AND p.index_id IN (0, 1))
		  FROM sys.tables t
		  JOIN sys.columns c ON c.object_id = t.object_id
		  JOIN sys.types ty ON ty.user_type_id = c.user_type_id
		 ORDER BY t.name, c.column_id`)
	if err != nil {
		st.lastErr = "ما قدرت أقرأ جداول الأمين: " + err.Error()
		return
	}
	defer rs.Close()
	type table struct {
		Name    string   `json:"name"`
		Rows    int64    `json:"rows"`
		Columns []string `json:"columns"`
	}
	byName := map[string]*table{}
	for rs.Next() {
		var t, c, ty string
		var n *int64
		if rs.Scan(&t, &c, &ty, &n) != nil {
			continue
		}
		tb := byName[t]
		if tb == nil {
			tb = &table{Name: t}
			if n != nil {
				tb.Rows = *n
			}
			byName[t] = tb
		}
		tb.Columns = append(tb.Columns, c+" ("+ty+")")
	}
	var tables []*table
	for _, t := range byName {
		tables = append(tables, t)
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i].Rows > tables[j].Rows })
	body, _ := json.Marshal(map[string]any{"database": st.dbFound, "tables": tables, "at": time.Now()})
	if _, err := st.request(ctx, http.MethodPost, "/api/v1/public/price-sync/schema", bytes.NewReader(body), "application/json"); err != nil {
		log.Printf("ما قدرت أبعت الجداول: %v", err)
		return
	}
	log.Printf("بعتت جداول الأمين (%d جدول)", len(tables))
}

// syncFolder **أحدثُ ملفٍّ في المجلّد** — مرّةً لكلّ نسخة، أو الآن إن طُلب.
func (st *state) syncFolder(ctx context.Context, folder string, force bool) error {
	var matches []string
	for _, p := range []string{"*.xlsx", "*.xlsm", "*.csv"} {
		m, _ := filepath.Glob(filepath.Join(folder, p))
		matches = append(matches, m...)
	}
	if len(matches) == 0 {
		return fmt.Errorf("ما في ملفات Excel بالمجلد %s", folder)
	}
	sort.Slice(matches, func(i, j int) bool {
		a, _ := os.Stat(matches[i])
		b, _ := os.Stat(matches[j])
		return a.ModTime().After(b.ModTime())
	})
	newest := matches[0]
	info, err := os.Stat(newest)
	if err != nil {
		return err
	}
	mark := fmt.Sprintf("%s|%d", newest, info.ModTime().Unix())
	if !force && mark == st.lastFile {
		return nil
	}
	data, err := os.ReadFile(newest)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", filepath.Base(newest))
	_, _ = fw.Write(data)
	_ = mw.Close()
	out, err := st.request(ctx, http.MethodPost, "/api/v1/public/price-sync/file", &buf, mw.FormDataContentType())
	if err != nil {
		return err
	}
	st.lastFile = mark
	logReport(out)
	return nil
}
