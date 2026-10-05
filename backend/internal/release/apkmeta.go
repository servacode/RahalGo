package release

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ الملفّ تُقرأ منه — لا تُكتب بيد** (طلبُ المالك ٢٠٢٦-١٠-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// «رفعتُ التطبيقات الجديدة وحدّدتُ الإصدارات وما طلب منّي جوّالي التحديث»
// — و«بدون ما ظلّ أبدّل إصدارات وأرقام».
//
// **وكان حقلُ النسخة نصّاً يكتبه المالك** — فكُتب فيه «كابتن رحّال غو»،
// **ورقمُ الإجبار يُرفع بيدٍ ثانيةٍ في لوحٍ آخر.** **والملفُّ يحمل الاثنين
// في `AndroidManifest.xml`**: معرّفَ الحزمة ورقمَها (`versionCode`) واسمَها
// (`versionName`). **فيُقرأ ما فيه ولا يُسأل أحد.**
//
// # ولماذا قارئٌ صغيرٌ هنا لا أداةُ أندرويد
//
// **البيانُ في الملفّ بصيغةٍ ثنائيّة** (`AXML`) لا نصّاً. **وأدواتُ أندرويد
// (`aapt2`) لا تُنصَّب على الخادم لأجل ثلاثةِ حقول** — **والصيغةُ ثابتةٌ
// منذ أندرويد ١ ويكفيها أقلُّ من مئتي سطر**: مجمعُ نصوصٍ ثمّ عناصر.
// **ويُقرأ أوّلُ عنصرٍ (`manifest`) وحدَه ثمّ يُترك الباقي.**

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"unicode/utf16"
)

// Manifest **ما يُقرأ من بيان الحزمة** — ثلاثةُ حقولٍ لا غير.
type Manifest struct {
	Package     string
	VersionCode int64
	VersionName string
}

// ErrNotAPK **الملفُّ ليس حزمةَ أندرويد يُقرأ بيانُها.**
var ErrNotAPK = errors.New("release: not an android package")

// maxManifestBytes **سقفُ البيان** — أكبرُ بيانٍ معقولٍ دون ميغابايت،
// **وأرشيفٌ يَعِد بمئات الميغابايتات في بيانه قنبلةُ ضغط.**
const maxManifestBytes = 8 << 20

// ReadAPKManifest **يقرأ هويّةَ الحزمة من أرشيفها.**
func ReadAPKManifest(r io.ReaderAt, size int64) (Manifest, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return Manifest{}, ErrNotAPK
	}
	for _, f := range zr.File {
		if f.Name != "AndroidManifest.xml" {
			continue
		}
		if f.UncompressedSize64 > maxManifestBytes {
			return Manifest{}, ErrNotAPK
		}
		rc, err := f.Open()
		if err != nil {
			return Manifest{}, ErrNotAPK
		}
		raw, err := io.ReadAll(io.LimitReader(rc, maxManifestBytes+1))
		_ = rc.Close()
		if err != nil || len(raw) > maxManifestBytes {
			return Manifest{}, ErrNotAPK
		}
		return ParseAXMLManifest(raw)
	}
	return Manifest{}, ErrNotAPK
}

// ══════════════════════════════════════════════════════════════════════
// **قارئُ `AXML` — الصيغةُ كما في `ResourceTypes.h`**
// ══════════════════════════════════════════════════════════════════════

const (
	axmlFile        = 0x0003
	axmlStringPool  = 0x0001
	axmlResourceMap = 0x0180
	axmlStartElem   = 0x0102

	poolUTF8 = 1 << 8

	typeReference = 0x01
	typeString    = 0x03
	typeIntDec    = 0x10
	typeIntHex    = 0x11

	// **ومعرّفا الصفتين في `android.R.attr`** — يُطابَق بهما حين تُجرَّد
	// أسماءُ الصفات من مجمع النصوص (بعضُ أدوات التصغير تفعل).
	attrVersionCode = 0x0101021b
	attrVersionName = 0x0101021c
)

// ParseAXMLManifest **يقرأ الحقولَ الثلاثة من بيانٍ ثنائيّ.**
func ParseAXMLManifest(b []byte) (Manifest, error) {
	le := binary.LittleEndian
	if len(b) < 8 || le.Uint16(b) != axmlFile {
		return Manifest{}, ErrNotAPK
	}
	var pool []string
	var resIDs []uint32
	off := int(le.Uint16(b[2:]))
	end := len(b)
	if total := int(le.Uint32(b[4:])); total > 0 && total < end {
		end = total
	}
	for off+8 <= end {
		typ := le.Uint16(b[off:])
		hsz := int(le.Uint16(b[off+2:]))
		size := int(le.Uint32(b[off+4:]))
		if size < 8 || off+size > end || hsz > size {
			return Manifest{}, ErrNotAPK
		}
		chunk := b[off : off+size]
		switch typ {
		case axmlStringPool:
			p, err := parseStringPool(chunk)
			if err != nil {
				return Manifest{}, err
			}
			pool = p
		case axmlResourceMap:
			for i := hsz; i+4 <= size; i += 4 {
				resIDs = append(resIDs, le.Uint32(chunk[i:]))
			}
		case axmlStartElem:
			return parseManifestElement(chunk, hsz, pool, resIDs)
		}
		off += size
	}
	return Manifest{}, ErrNotAPK
}

func parseStringPool(c []byte) ([]string, error) {
	le := binary.LittleEndian
	if len(c) < 28 {
		return nil, ErrNotAPK
	}
	hsz := int(le.Uint16(c[2:]))
	count := int(le.Uint32(c[8:]))
	flags := le.Uint32(c[16:])
	start := int(le.Uint32(c[20:]))
	if count < 0 || hsz+count*4 > len(c) || start > len(c) {
		return nil, ErrNotAPK
	}
	out := make([]string, count)
	for i := 0; i < count; i++ {
		at := start + int(le.Uint32(c[hsz+i*4:]))
		if at < 0 || at >= len(c) {
			continue
		}
		if flags&poolUTF8 != 0 {
			out[i] = utf8At(c, at)
		} else {
			out[i] = utf16At(c, at)
		}
	}
	return out, nil
}

// utf8At **نصٌّ بترميز UTF-8**: طولُ المحارف ثمّ طولُ البايتات (كلٌّ بايتٌ
// أو اثنان) ثمّ البايتات.
func utf8At(c []byte, at int) string {
	skip := func() bool {
		if at >= len(c) {
			return false
		}
		if c[at]&0x80 != 0 {
			at++
		}
		at++
		return true
	}
	if !skip() || at >= len(c) {
		return ""
	}
	n := int(c[at])
	if n&0x80 != 0 {
		if at+1 >= len(c) {
			return ""
		}
		n = (n&0x7f)<<8 | int(c[at+1])
		at++
	}
	at++
	if at+n > len(c) {
		return ""
	}
	return string(c[at : at+n])
}

// utf16At **نصٌّ بترميز UTF-16**: طولٌ بوحدةٍ أو اثنتين ثمّ الوحدات.
func utf16At(c []byte, at int) string {
	le := binary.LittleEndian
	if at+2 > len(c) {
		return ""
	}
	n := int(le.Uint16(c[at:]))
	at += 2
	if n&0x8000 != 0 {
		if at+2 > len(c) {
			return ""
		}
		n = (n&0x7fff)<<16 | int(le.Uint16(c[at:]))
		at += 2
	}
	if at+n*2 > len(c) {
		return ""
	}
	u := make([]uint16, n)
	for i := range u {
		u[i] = le.Uint16(c[at+i*2:])
	}
	return string(utf16.Decode(u))
}

func parseManifestElement(c []byte, hsz int, pool []string, resIDs []uint32) (Manifest, error) {
	le := binary.LittleEndian
	str := func(i uint32) string {
		if int(i) < len(pool) && i != 0xffffffff {
			return pool[i]
		}
		return ""
	}
	ext := hsz
	if ext+20 > len(c) {
		return Manifest{}, ErrNotAPK
	}
	if str(le.Uint32(c[ext+4:])) != "manifest" {
		return Manifest{}, ErrNotAPK
	}
	aStart := int(le.Uint16(c[ext+8:]))
	aSize := int(le.Uint16(c[ext+10:]))
	aCount := int(le.Uint16(c[ext+12:]))
	if aSize < 20 {
		return Manifest{}, ErrNotAPK
	}
	var m Manifest
	for i := 0; i < aCount; i++ {
		a := ext + aStart + i*aSize
		if a+20 > len(c) {
			return Manifest{}, ErrNotAPK
		}
		nameIdx := le.Uint32(c[a+4:])
		raw := le.Uint32(c[a+8:])
		dataType := c[a+15]
		data := le.Uint32(c[a+16:])

		name := str(nameIdx)
		var id uint32
		if int(nameIdx) < len(resIDs) {
			id = resIDs[nameIdx]
		}
		value := func() string {
			if s := str(raw); s != "" {
				return s
			}
			if dataType == typeString {
				return str(data)
			}
			return ""
		}
		switch {
		case name == "package" && id == 0:
			m.Package = strings.TrimSpace(value())
		case id == attrVersionCode || (id == 0 && name == "versionCode"):
			switch dataType {
			case typeIntDec, typeIntHex:
				m.VersionCode = int64(data)
			default:
				m.VersionCode = parseInt(value())
			}
		case id == attrVersionName || (id == 0 && name == "versionName"):
			// **ومرجعٌ إلى موارد (`@string/…`) لا يُحلّ هنا** — يبقى فارغاً
			// ولا يُخترَع اسم.
			if dataType != typeReference {
				m.VersionName = strings.TrimSpace(value())
			}
		}
	}
	if m.Package == "" {
		return Manifest{}, ErrNotAPK
	}
	return m, nil
}

func parseInt(s string) int64 {
	var n int64
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int64(r-'0')
		if n > 1<<31 {
			return 0
		}
	}
	return n
}

// ExpectedPackage **المعرّفُ الذي يُقبل في خانة التطبيق** — على التجهيز
// نسخةُ التجهيز (`.staging`) وعلى غيره نسخةُ المتجر بعينها.
//
// **ومن رفع تطبيقَ السائق في خانة الزبون وزّع على الزبائن تطبيقاً لا
// يعرفونه**، **ومن رفع نسخةَ التجهيز في الإنتاج وزّع تطبيقاً يكلّم خادماً
// آخر.** فيُردّ الرفعُ ولا يُكتب شيء.
func ExpectedPackage(a App, staging bool) string {
	if staging {
		return a.PackageID + ".staging"
	}
	return a.PackageID
}

// VersionCodeKey **رقمُ النسخة المقروءُ من الملفّ** — يكتبه الخادمُ وحدَه.
func VersionCodeKey(key string) string { return "release." + key + ".version_code" }

// AutoForceKey **أيُفرَض التحديثُ تلقائيّاً على من دون الملفّ المرفوع؟**
func AutoForceKey(key string) string { return "release." + key + ".auto_force" }
