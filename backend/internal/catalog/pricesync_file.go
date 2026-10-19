package catalog

// ══════════════════════════════════════════════════════════════════════
// **ملفُّ الأمين — Excel أو CSV** (قرارُ المالك ٢٠٢٦-١٠-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// المحاسبُ يصدّر «قائمة المواد» ويرفعها. **وأعمدةُ الأمين تختلف بين نسخة ونسخة**،
// فلا تُفترض: **يختار صاحبُ المتجر مرّةً أيُّ عمودٍ رقمُ المادّة والسعرُ والكميّة**
// (`SyncMapping`)، **وتُحفظ بأسماء الأعمدة** فتصلح للملفّ التالي ولو تبدّل ترتيبُها.
//
// **وCSV بترميز ويندوز العربيّ يُقرأ** — كثيرٌ من برامج ويندوز تصدّره كذلك،
// **فإن لم يكن UTF-8 صحيحاً قُرئ بـ windows-1256.**
//
// **وملفُّ .xls القديم لا يُقرأ** — يُقال للمحاسب أن يحفظه xlsx.

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// SyncMapping **أيُّ عمودٍ لأيّ حقل** — بأسماء الأعمدة كما في رأس الملفّ.
type SyncMapping struct {
	Code     string `json:"code"`
	Barcode  string `json:"barcode,omitempty"`
	Name     string `json:"name,omitempty"`
	Price    string `json:"price"`
	Qty      string `json:"qty,omitempty"`
	Currency string `json:"currency,omitempty"`
}

// Sheet **الملفُّ مقروءاً** — الرأسُ ثمّ الصفوف.
type Sheet struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"-"`
}

var (
	// ErrSyncFileType **نوعُ ملفٍّ لا يُقرأ** — والقديمُ .xls منه.
	ErrSyncFileType = httpx.NewError(http.StatusBadRequest, "price_sync_file_type", "errors.price_sync_file_type")
	// ErrSyncMapping **عمودٌ مختارٌ ليس في الملفّ** — أو لا رقمَ مادّةٍ ولا اسم.
	ErrSyncMapping = httpx.NewError(http.StatusBadRequest, "price_sync_mapping", "errors.price_sync_mapping")
)

// ReadSheet **يقرأ الورقةَ الأولى** من xlsx أو CSV — والرأسُ أوّلُ صفٍّ فيه خانتان.
func ReadSheet(filename string, data []byte) (*Sheet, error) {
	var all [][]string
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".xlsx", ".xlsm":
		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			return nil, ErrSyncFileType
		}
		defer func() { _ = f.Close() }()
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, ErrSyncEmpty
		}
		if all, err = f.GetRows(sheets[0]); err != nil {
			return nil, ErrSyncFileType
		}
	case ".csv", ".txt":
		if !utf8.Valid(data) {
			dec, err := charmap.Windows1256.NewDecoder().Bytes(data)
			if err != nil {
				return nil, ErrSyncFileType
			}
			data = dec
		}
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
		r := csv.NewReader(bytes.NewReader(data))
		r.FieldsPerRecord = -1
		r.LazyQuotes = true
		if first, _, ok := bytes.Cut(data, []byte("\n")); ok || len(first) > 0 {
			if bytes.Count(first, []byte(";")) > bytes.Count(first, []byte(",")) {
				r.Comma = ';'
			} else if bytes.Count(first, []byte("\t")) > bytes.Count(first, []byte(",")) {
				r.Comma = '\t'
			}
		}
		var err error
		if all, err = r.ReadAll(); err != nil {
			return nil, ErrSyncFileType
		}
	default:
		return nil, ErrSyncFileType
	}
	head := -1
	for i, row := range all {
		filled := 0
		for _, c := range row {
			if strings.TrimSpace(c) != "" {
				filled++
			}
		}
		if filled >= 2 {
			head = i
			break
		}
	}
	if head < 0 {
		return nil, ErrSyncEmpty
	}
	sh := &Sheet{}
	for _, h := range all[head] {
		sh.Headers = append(sh.Headers, strings.TrimSpace(h))
	}
	for _, row := range all[head+1:] {
		empty := true
		for _, c := range row {
			if strings.TrimSpace(c) != "" {
				empty = false
				break
			}
		}
		if !empty {
			sh.Rows = append(sh.Rows, row)
		}
	}
	return sh, nil
}

// Sample **أوّلُ صفوفٍ للعرض** — ليختار صاحبُ المتجر الأعمدةَ وهو يرى ما فيها.
func (sh *Sheet) Sample(n int) [][]string {
	if len(sh.Rows) < n {
		n = len(sh.Rows)
	}
	return sh.Rows[:n]
}

// SyncRows **الصفوفُ بحسب الاختيار** — وعمودٌ مختارٌ غائبٌ عن الرأس يُردّ.
func (sh *Sheet) SyncRows(m SyncMapping) ([]SyncRow, error) {
	col := func(name string) (int, error) {
		if strings.TrimSpace(name) == "" {
			return -1, nil
		}
		for i, h := range sh.Headers {
			if h == strings.TrimSpace(name) {
				return i, nil
			}
		}
		return -1, ErrSyncMapping
	}
	code, err := col(m.Code)
	if err != nil {
		return nil, err
	}
	barcode, err := col(m.Barcode)
	if err != nil {
		return nil, err
	}
	name, err := col(m.Name)
	if err != nil {
		return nil, err
	}
	price, err := col(m.Price)
	if err != nil {
		return nil, err
	}
	qty, err := col(m.Qty)
	if err != nil {
		return nil, err
	}
	if code < 0 && barcode < 0 && name < 0 {
		return nil, ErrSyncMapping
	}
	cell := func(row []string, i int) string {
		if i < 0 || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	out := make([]SyncRow, 0, len(sh.Rows))
	for _, row := range sh.Rows {
		r := SyncRow{Code: cell(row, code), Barcode: cell(row, barcode), Name: cell(row, name)}
		if v, ok := ParseAmount(cell(row, price)); ok {
			r.Price = &v
		}
		if v, ok := ParseAmount(cell(row, qty)); ok {
			r.Qty = &v
		}
		out = append(out, r)
	}
	return out, nil
}

// ParseAmount **رقمٌ كما يكتبه محاسبٌ عربيّ** — «١٬٢٥٠٫٥» و«1,250.5» و«1250 ل.س».
func ParseAmount(s string) (float64, bool) {
	var b strings.Builder
	rs := []rune(s)
	digit := func(r rune) bool {
		return (r >= '0' && r <= '9') || (r >= '٠' && r <= '٩') || (r >= '۰' && r <= '۹')
	}
	for i, r := range rs {
		// **والنقطةُ فاصلةٌ عشريّةٌ إن تبعها رقمٌ وحدَه** — «ل.س» ليست كسراً.
		if (r == '.' || r == '٫') && (i+1 >= len(rs) || !digit(rs[i+1]) || b.Len() == 0) {
			continue
		}
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r == '.' || r == '٫':
			b.WriteRune('.')
		case r == '-' && b.Len() == 0:
			b.WriteRune('-')
		}
	}
	t := b.String()
	if t == "" || t == "-" || strings.Count(t, ".") > 1 {
		return 0, false
	}
	v, err := strconv.ParseFloat(t, 64)
	return v, err == nil
}
