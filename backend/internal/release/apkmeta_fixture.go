package release

import (
	"archive/zip"
	"bytes"
)

// ══════════════════════════════════════════════════════════════════════
// **حزمةٌ مصنوعةٌ للفحص** — بيانٌ ثنائيٌّ حقيقيُّ الصيغة داخل أرشيف.
// ══════════════════════════════════════════════════════════════════════
//
// **وتُصدَّر لأنّ فحوصَ الرفع في `qa` تحتاجها** — ولا يُرفع ملفُّ المالك
// الحقيقيُّ في فحصٍ يجري على كلّ جهاز.

// BuildTestAPK **أرشيفٌ فيه `AndroidManifest.xml` بالحقول الثلاثة.**
func BuildTestAPK(pkg string, code uint32, name string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("AndroidManifest.xml")
	_, _ = w.Write(BuildTestAXML(true, false, pkg, code, name))
	w2, _ := zw.Create("classes.dex")
	_, _ = w2.Write([]byte("dex\n035\x00"))
	_ = zw.Close()
	return buf.Bytes()
}

// BuildTestAXML **بيانٌ ثنائيٌّ صغير** — مجمعُ نصوصٍ وخريطةُ مواردَ وعنصرُ
// `manifest` بثلاث صفات. **و`stripped` يُفرغ اسمَي الصفتين** فلا يُطابَق
// إلّا بمعرّف المورد.
func BuildTestAXML(utf8, stripped bool, pkg string, code uint32, name string) []byte {
	vc, vn := "versionCode", "versionName"
	if stripped {
		vc, vn = "", ""
	}
	strs := []string{vc, vn, "package", "manifest", pkg, name,
		"http://schemas.android.com/apk/res/android"}
	u16 := func(b *bytes.Buffer, x uint16) { b.Write([]byte{byte(x), byte(x >> 8)}) }
	u32 := func(b *bytes.Buffer, x uint32) {
		b.Write([]byte{byte(x), byte(x >> 8), byte(x >> 16), byte(x >> 24)})
	}

	var data bytes.Buffer
	offs := make([]uint32, 0, len(strs))
	for _, s := range strs {
		offs = append(offs, uint32(data.Len()))
		if utf8 {
			data.WriteByte(byte(len([]rune(s))))
			data.WriteByte(byte(len(s)))
			data.WriteString(s)
			data.WriteByte(0)
		} else {
			r := []rune(s)
			u16(&data, uint16(len(r)))
			for _, c := range r {
				u16(&data, uint16(c))
			}
			u16(&data, 0)
		}
	}
	for data.Len()%4 != 0 {
		data.WriteByte(0)
	}
	var pool bytes.Buffer
	start := uint32(28 + 4*len(strs))
	u16(&pool, axmlStringPool)
	u16(&pool, 28)
	u32(&pool, start+uint32(data.Len()))
	u32(&pool, uint32(len(strs)))
	u32(&pool, 0)
	if utf8 {
		u32(&pool, poolUTF8)
	} else {
		u32(&pool, 0)
	}
	u32(&pool, start)
	u32(&pool, 0)
	for _, o := range offs {
		u32(&pool, o)
	}
	pool.Write(data.Bytes())

	var rm bytes.Buffer
	u16(&rm, axmlResourceMap)
	u16(&rm, 8)
	u32(&rm, 16)
	u32(&rm, attrVersionCode)
	u32(&rm, attrVersionName)

	var el bytes.Buffer
	u16(&el, axmlStartElem)
	u16(&el, 16)
	u32(&el, 16+20+3*20)
	u32(&el, 1)
	u32(&el, 0xffffffff)
	u32(&el, 0xffffffff) // ns
	u32(&el, 3)          // manifest
	u16(&el, 20)
	u16(&el, 20)
	u16(&el, 3)
	u16(&el, 0)
	u16(&el, 0)
	u16(&el, 0)
	attr := func(ns, nm, raw uint32, typ byte, d uint32) {
		u32(&el, ns)
		u32(&el, nm)
		u32(&el, raw)
		u16(&el, 8)
		el.WriteByte(0)
		el.WriteByte(typ)
		u32(&el, d)
	}
	attr(6, 0, 0xffffffff, typeIntDec, code)
	attr(6, 1, 5, typeString, 5)
	attr(0xffffffff, 2, 4, typeString, 4)

	var out bytes.Buffer
	u16(&out, axmlFile)
	u16(&out, 8)
	u32(&out, uint32(8+pool.Len()+rm.Len()+el.Len()))
	out.Write(pool.Bytes())
	out.Write(rm.Bytes())
	out.Write(el.Bytes())
	return out.Bytes()
}
