package media

import (
	"encoding/binary"
	"image"
	"image/draw"
)

// ══════════════════════════════════════════════════════════════════════
// **صورةُ الكاميرا تُعدَّل قبل حفظها** (ملاحظةُ المالك ٢٠٢٦-١٠-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// «المتجر وقت يصوّر صنف من الكاميرا عم تطلع الصورة مقلوبة.» **الجوّالُ يحفظ
// اللقطةَ كما خرجت من الحسّاس ويكتب في ترويستها «أدِرها»** (وسمُ EXIF
// `Orientation`) — **و`image.Decode` لا يقرأ الوسم**، فتُصغَّر الصورةُ
// وتُحفظ بلا وسمٍ وعلى جنبها. **والإصلاحُ في المحرّك يصل كلَّ تطبيقٍ بلا
// تحديث**، من الكاميرا ومن المعرض.

// exifOrientation **وسمُ الاتّجاه في JPEG** — من ١ إلى ٨، وواحدٌ لكلّ ما سواه.
func exifOrientation(raw []byte) int {
	if len(raw) < 4 || raw[0] != 0xFF || raw[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(raw) {
		if raw[i] != 0xFF {
			return 1
		}
		marker := raw[i+1]
		if marker == 0xD9 || marker == 0xDA { // نهايةُ الصورة أو بدءُ بياناتها
			return 1
		}
		size := int(binary.BigEndian.Uint16(raw[i+2:]))
		if size < 2 || i+2+size > len(raw) {
			return 1
		}
		seg := raw[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	ifd := int(bo.Uint32(t[4:]))
	if ifd+2 > len(t) {
		return 1
	}
	n := int(bo.Uint16(t[ifd:]))
	for k := 0; k < n; k++ {
		e := ifd + 2 + k*12
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 {
			o := int(bo.Uint16(t[e+8:]))
			if o >= 1 && o <= 8 {
				return o
			}
			return 1
		}
	}
	return 1
}

// orient **يُدير الصورةَ بحسب وسمها** — فتُحفظ كما رآها من صوّرها.
func orient(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	in := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(in, in.Bounds(), src, b.Min, draw.Src)
	ow, oh := w, h
	if o >= 5 { // الخمسةُ وما بعدها تبادل العرضَ والارتفاع
		ow, oh = h, w
	}
	out := image.NewRGBA(image.Rect(0, 0, ow, oh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2: // مرآةٌ أفقيّة
				nx, ny = w-1-x, y
			case 3: // نصفُ دورة
				nx, ny = w-1-x, h-1-y
			case 4: // مرآةٌ عموديّة
				nx, ny = x, h-1-y
			case 5: // تبديلُ المحورين
				nx, ny = y, x
			case 6: // ربعُ دورةٍ مع عقارب الساعة
				nx, ny = h-1-y, x
			case 7: // تبديلٌ معاكس
				nx, ny = h-1-y, w-1-x
			case 8: // ربعُ دورةٍ عكس عقارب الساعة
				nx, ny = y, w-1-x
			}
			si := in.PixOffset(x, y)
			di := out.PixOffset(nx, ny)
			copy(out.Pix[di:di+4], in.Pix[si:si+4])
		}
	}
	return out
}
