// Package polyline **خطوط Valhalla المشفّرة (polyline6) والقياس عليها** — حزمة ورقيّة بلا استيراد داخليّ.
//
// النقطة `[2]float64{lng, lat}` (ترتيب GeoJSON)، والمسافة بالمتر بنفس معادلة `routing.MetersBetween`.
// بيستعملها تعلّم الطرق (`maplearn`) ومطابقة مسار السائق (`roadmatch`) والتحليل (`drivermonitor`).
package polyline

import "math"

const earthR = 6371000.0

// Decode6 يفكّ خطّاً مشفَّراً بدقّة ست منازل (صيغة Valhalla) إلى (lng,lat).
func Decode6(s string) [][2]float64 {
	var out [][2]float64
	var lat, lng int
	i := 0
	next := func() (int, bool) {
		var result, shift int
		for {
			if i >= len(s) {
				return 0, false
			}
			c := int(s[i]) - 63
			i++
			result |= (c & 0x1f) << shift
			shift += 5
			if c < 0x20 {
				break
			}
		}
		if result&1 != 0 {
			return ^(result >> 1), true
		}
		return result >> 1, true
	}
	for i < len(s) {
		dLat, ok := next()
		if !ok {
			break
		}
		dLng, ok := next()
		if !ok {
			break
		}
		lat += dLat
		lng += dLng
		out = append(out, [2]float64{float64(lng) / 1e6, float64(lat) / 1e6})
	}
	return out
}

// Encode6 يشفّر (lng,lat) بدقّة ست منازل — عكس `Decode6`.
func Encode6(pts [][2]float64) string {
	out := make([]byte, 0, len(pts)*8)
	enc := func(v int) {
		v <<= 1
		if v < 0 {
			v = ^v
		}
		for v >= 0x20 {
			out = append(out, byte((0x20|(v&0x1f))+63))
			v >>= 5
		}
		out = append(out, byte(v+63))
	}
	var pLat, pLng int
	for _, p := range pts {
		lat, lng := int(math.Round(p[1]*1e6)), int(math.Round(p[0]*1e6))
		enc(lat - pLat)
		enc(lng - pLng)
		pLat, pLng = lat, lng
	}
	return string(out)
}

// Meters المسافة بالمتر بين نقطتين (lng,lat) — haversine.
func Meters(a, b [2]float64) float64 {
	p1 := a[1] * math.Pi / 180
	p2 := b[1] * math.Pi / 180
	dp := (b[1] - a[1]) * math.Pi / 180
	dl := (b[0] - a[0]) * math.Pi / 180
	h := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * earthR * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

// Cumulative المسافة من أوّل الخطّ لكلّ رأس — `cum[0] = 0` و`cum[n-1]` = طول الخطّ.
func Cumulative(line [][2]float64) []float64 {
	cum := make([]float64, len(line))
	for i := 1; i < len(line); i++ {
		cum[i] = cum[i-1] + Meters(line[i-1], line[i])
	}
	return cum
}

// Length طول الخطّ بالمتر.
func Length(line [][2]float64) float64 {
	if len(line) < 2 {
		return 0
	}
	c := Cumulative(line)
	return c[len(c)-1]
}

// Project **يسقط نقطة على قطعة من الخطّ** بين الرأسين `from` و`to` (شاملين): يرجّع موضع الإسقاط
// بالمتر من أوّل الخطّ، وبعد النقطة عنه. `cum` من `Cumulative`. خطّ فاضي أو مدى غلط ⇒ (NaN، +Inf).
func Project(line [][2]float64, cum []float64, p [2]float64, from, to int) (pos, dist float64) {
	if from < 0 {
		from = 0
	}
	if to > len(line)-1 {
		to = len(line) - 1
	}
	if len(line) == 0 || from > to {
		return math.NaN(), math.Inf(1)
	}
	if from == to {
		return cum[from], Meters(line[from], p)
	}
	pos, dist = math.NaN(), math.Inf(1)
	kx := math.Cos(p[1]*math.Pi/180) * earthR * math.Pi / 180
	ky := earthR * math.Pi / 180
	for i := from; i < to; i++ {
		a, b := line[i], line[i+1]
		ax, ay := (a[0]-p[0])*kx, (a[1]-p[1])*ky
		bx, by := (b[0]-p[0])*kx, (b[1]-p[1])*ky
		dx, dy := bx-ax, by-ay
		l2 := dx*dx + dy*dy
		t := 0.0
		if l2 > 0 {
			t = -(ax*dx + ay*dy) / l2
			t = math.Max(0, math.Min(1, t))
		}
		qx, qy := ax+t*dx, ay+t*dy
		d := math.Hypot(qx, qy)
		if d < dist {
			dist = d
			pos = cum[i] + t*(cum[i+1]-cum[i])
		}
	}
	return pos, dist
}

// At النقطة على الخطّ عند الموضع `pos` بالمتر (مقصوص على طرفي الخطّ).
func At(line [][2]float64, cum []float64, pos float64) [2]float64 {
	n := len(line)
	if n == 0 {
		return [2]float64{}
	}
	if pos <= 0 || n == 1 {
		return line[0]
	}
	if pos >= cum[n-1] {
		return line[n-1]
	}
	i := searchSeg(cum, pos)
	seg := cum[i+1] - cum[i]
	t := 0.0
	if seg > 0 {
		t = (pos - cum[i]) / seg
	}
	a, b := line[i], line[i+1]
	return [2]float64{a[0] + t*(b[0]-a[0]), a[1] + t*(b[1]-a[1])}
}

// Cut **الخطّ بين موضعين** بالمتر (a ≤ b): نقطة البداية، الرؤوس بينهما، ونقطة النهاية.
func Cut(line [][2]float64, cum []float64, a, b float64) [][2]float64 {
	if len(line) == 0 {
		return nil
	}
	if b < a {
		a, b = b, a
	}
	out := [][2]float64{At(line, cum, a)}
	for i := range line {
		if cum[i] > a && cum[i] < b {
			out = append(out, line[i])
		}
	}
	out = append(out, At(line, cum, b))
	return out
}

// searchSeg فهرس القطعة [i, i+1] اللي فيها الموضع (0 < pos < cum[n-1]).
func searchSeg(cum []float64, pos float64) int {
	lo, hi := 0, len(cum)-1
	for hi-lo > 1 {
		m := (lo + hi) / 2
		if cum[m] <= pos {
			lo = m
		} else {
			hi = m
		}
	}
	return lo
}
