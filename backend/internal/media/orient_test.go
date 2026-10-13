package media

// **صورةُ الكاميرا تُحفظ كما صُوّرت** (ملاحظةُ المالك ٢٠٢٦-١٠-١٣): لقطةٌ
// بوسم «أدِرها ربعَ دورة» (6) — كما يكتبها الجوّال — تُفكّ فتُدار.

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// withOrientation **يُدخل وسمَ EXIF بالاتّجاه بعد بداية JPEG** — كما يفعل الجوّال.
func withOrientation(t *testing.T, plain []byte, o uint16) []byte {
	t.Helper()
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1}
	entry := make([]byte, 12)
	binary.BigEndian.PutUint16(entry[0:], 0x0112)
	binary.BigEndian.PutUint16(entry[2:], 3) // SHORT
	binary.BigEndian.PutUint32(entry[4:], 1)
	binary.BigEndian.PutUint16(entry[8:], o)
	tiff = append(tiff, entry...)
	tiff = append(tiff, 0, 0, 0, 0)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	seg = append(seg, payload...)
	out := append([]byte{}, plain[:2]...)
	out = append(out, seg...)
	return append(out, plain[2:]...)
}

func TestOrient_CameraPhotoTurnsUpright(t *testing.T) {
	// عرضٌ ٤٠ وارتفاعٌ ٢٠ — ونصفُها الأيسرُ أحمر.
	src := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			c := color.RGBA{0, 0, 255, 255}
			if x < 20 {
				c = color.RGBA{255, 0, 0, 255}
			}
			src.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	raw := withOrientation(t, buf.Bytes(), 6)

	if o := exifOrientation(raw); o != 6 {
		t.Fatalf("الوسمُ قُرئ %d — والمكتوبُ 6", o)
	}
	if o := exifOrientation(buf.Bytes()); o != 1 {
		t.Fatalf("صورةٌ بلا وسمٍ قُرئت %d — والمنتظَر 1", o)
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	up := orient(img, exifOrientation(raw))
	if b := up.Bounds(); b.Dx() != 20 || b.Dy() != 40 {
		t.Fatalf("الأبعادُ بعد الإدارة %dx%d — والمنتظَر 20x40", b.Dx(), b.Dy())
	}
	// ربعُ دورةٍ مع العقارب: الأيسرُ الأحمرُ يصير أعلى.
	r, _, bl, _ := up.At(10, 5).RGBA()
	if r>>8 < 200 || bl>>8 > 60 {
		t.Fatalf("أعلى الصورة ليس أحمر (r=%d b=%d) — **ما زالت على جنبها**", r>>8, bl>>8)
	}
	r, _, bl, _ = up.At(10, 35).RGBA()
	if bl>>8 < 200 || r>>8 > 60 {
		t.Fatalf("أسفلُ الصورة ليس أزرق (r=%d b=%d)", r>>8, bl>>8)
	}
}
