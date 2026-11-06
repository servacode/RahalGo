/**
 * هندسةُ المناطق — دوالُّ صافيةٌ بلا واجهة (قرارُ المالك ٢٠٢٦-١١-٠٦).
 *
 * - `coverageMask`: مضلَّعُ التغميق = مستطيلُ العالم ناقصَ اتّحاد المناطق.
 * - `ringSelfIntersects`: فحصٌ بسيطٌ لتقاطع الشكل مع نفسه قبل الحفظ.
 * - `midpoints`: منتصفاتُ الأضلاع لمقابض الإدراج.
 */

export type Pt = [number, number];
type Ring = Pt[];

const EPS = 1e-12;

/** أضلاعٌ حلقةٌ مغلقة بلا تكرار نقطة الإغلاق. */
export function openRing(r: Pt[]): Ring {
  if (r.length > 1) {
    const a = r[0]!;
    const b = r[r.length - 1]!;
    if (a[0] === b[0] && a[1] === b[1]) return r.slice(0, -1).map((p) => [p[0], p[1]] as Pt);
  }
  return r.map((p) => [p[0], p[1]] as Pt);
}

/** دائرةٌ بالمتر كمضلَّع (تقريبٌ بـ n ضلعاً). */
export function circleRing(lng: number, lat: number, radiusM: number, n = 64): Ring {
  const dLat = radiusM / 111_320;
  const dLng = radiusM / (111_320 * Math.cos((lat * Math.PI) / 180) || 1);
  const out: Ring = [];
  for (let i = 0; i < n; i++) {
    const a = (i / n) * Math.PI * 2;
    out.push([lng + dLng * Math.cos(a), lat + dLat * Math.sin(a)]);
  }
  return out;
}

function cross(ax: number, ay: number, bx: number, by: number) {
  return ax * by - ay * bx;
}

/** نقطةُ تقاطع القطعتين (p→p2) و(q→q2) إن وُجدت، ومعاملاها على كلٍّ منهما. */
function segHit(p: Pt, p2: Pt, q: Pt, q2: Pt): { t: number; u: number } | null {
  const rx = p2[0] - p[0];
  const ry = p2[1] - p[1];
  const sx = q2[0] - q[0];
  const sy = q2[1] - q[1];
  const den = cross(rx, ry, sx, sy);
  if (Math.abs(den) < EPS) return null;
  const t = cross(q[0] - p[0], q[1] - p[1], sx, sy) / den;
  const u = cross(q[0] - p[0], q[1] - p[1], rx, ry) / den;
  if (t < 0 || t > 1 || u < 0 || u > 1) return null;
  return { t, u };
}

/** هل يتقاطع الشكلُ مع نفسه؟ (أضلاعٌ غيرُ متجاورة تتلاقى). */
export function ringSelfIntersects(ringIn: Pt[]): boolean {
  const r = openRing(ringIn);
  const n = r.length;
  if (n < 4) return false;
  for (let i = 0; i < n; i++) {
    const a = r[i]!;
    const b = r[(i + 1) % n]!;
    for (let j = i + 1; j < n; j++) {
      // المتجاورةُ تتشارك رأساً — لا تُحسب.
      if (j === i || (j + 1) % n === i || (i + 1) % n === j) continue;
      const c = r[j]!;
      const d = r[(j + 1) % n]!;
      if (segHit(a, b, c, d)) return true;
    }
  }
  return false;
}

/** منتصفُ كلّ ضلع: i بين الرأس i والرأس i+1. */
export function midpoints(r: Pt[]): Pt[] {
  const n = r.length;
  return r.map((p, i) => {
    const q = r[(i + 1) % n]!;
    return [(p[0] + q[0]) / 2, (p[1] + q[1]) / 2] as Pt;
  });
}

function inside(pt: Pt, ring: Ring): boolean {
  let c = false;
  for (let i = 0, j = ring.length - 1; i < ring.length; j = i++) {
    const [xi, yi] = ring[i]!;
    const [xj, yj] = ring[j]!;
    if (yi > pt[1] !== yj > pt[1] && pt[0] < ((xj - xi) * (pt[1] - yi)) / (yj - yi) + xi) c = !c;
  }
  return c;
}

const key = (p: Pt) => `${p[0].toFixed(10)},${p[1].toFixed(10)}`;

/**
 * حدودُ اتّحاد مضلَّعاتٍ بسيطة: تُقطَّع الأضلاعُ عند التقاطعات، ويبقى منها ما هو خارجَ كلّ
 * المضلَّعات الأخرى، ثمّ تُوصَل حلقات. يعيد null إن تعذّر الوصلُ (حالةٌ شاذّة).
 */
function unionBoundary(polys: Ring[]): Ring[] | null {
  const kept: Array<[Pt, Pt]> = [];
  for (let a = 0; a < polys.length; a++) {
    const A = polys[a]!;
    for (let i = 0; i < A.length; i++) {
      const p = A[i]!;
      const p2 = A[(i + 1) % A.length]!;
      const ts = [0, 1];
      for (let b = 0; b < polys.length; b++) {
        if (b === a) continue;
        const B = polys[b]!;
        for (let j = 0; j < B.length; j++) {
          const h = segHit(p, p2, B[j]!, B[(j + 1) % B.length]!);
          if (h) ts.push(h.t);
        }
      }
      ts.sort((x, y) => x - y);
      for (let k = 0; k + 1 < ts.length; k++) {
        if (ts[k + 1]! - ts[k]! < 1e-9) continue;
        const s: Pt = [p[0] + (p2[0] - p[0]) * ts[k]!, p[1] + (p2[1] - p[1]) * ts[k]!];
        const e: Pt = [p[0] + (p2[0] - p[0]) * ts[k + 1]!, p[1] + (p2[1] - p[1]) * ts[k + 1]!];
        const mid: Pt = [(s[0] + e[0]) / 2, (s[1] + e[1]) / 2];
        let covered = false;
        for (let b = 0; b < polys.length && !covered; b++) {
          if (b !== a && inside(mid, polys[b]!)) covered = true;
        }
        if (!covered) kept.push([s, e]);
      }
    }
  }
  // وصلُ القطع حلقاتٍ.
  const byStart = new Map<string, Array<[Pt, Pt]>>();
  for (const s of kept) {
    const k = key(s[0]);
    const l = byStart.get(k);
    if (l) l.push(s);
    else byStart.set(k, [s]);
  }
  const used = new Set<[Pt, Pt]>();
  const rings: Ring[] = [];
  for (const seg of kept) {
    if (used.has(seg)) continue;
    const ring: Ring = [seg[0]];
    let cur = seg;
    used.add(cur);
    const startKey = key(seg[0]);
    let guard = kept.length + 2;
    while (guard-- > 0) {
      const endKey = key(cur[1]);
      if (endKey === startKey) break;
      ring.push(cur[1]);
      const next = (byStart.get(endKey) ?? []).find((s) => !used.has(s));
      if (!next) return null;
      used.add(next);
      cur = next;
    }
    if (guard < 0 || ring.length < 3) return null;
    rings.push(ring);
  }
  return rings;
}

const WORLD: Ring = [
  [-180, -85],
  [180, -85],
  [180, 85],
  [-180, 85],
];

/** حلقةٌ مغلقة (نقطةُ الإغلاق مكرَّرة) كما يطلبها GeoJSON. */
const closed = (r: Ring): Ring => [...r, r[0]!];

/**
 * جيومتريّةُ التغميق: العالمُ ناقصَ اتّحاد المناطق. بلا مناطق ← فارغ (لا تغميق).
 * العمقُ المتداخل: حدودُ الاتّحاد الخارجيّة ثقوبٌ في التغميق، وثقوبُ الاتّحاد جُزرٌ مغمَّقة.
 */
export function coverageMask(zones: Pt[][]): unknown | null {
  const polys = zones.map(openRing).filter((r) => r.length >= 3);
  if (polys.length === 0) return null;
  const rings = polys.length === 1 ? polys : (unionBoundary(polys) ?? polys);
  const depth = rings.map((r, i) => rings.reduce((n, o, j) => (i !== j && inside(r[0]!, o) ? n + 1 : n), 0));
  const holes: Ring[] = [];
  const islands: Array<{ outer: Ring; holes: Ring[] }> = [];
  rings.forEach((r, i) => {
    if (depth[i]! === 0) holes.push(r);
    else if (depth[i]! % 2 === 1) islands.push({ outer: r, holes: [] });
  });
  rings.forEach((r, i) => {
    if (depth[i]! > 0 && depth[i]! % 2 === 0) {
      // داخلَ جزيرةٍ: ثقبٌ فيها.
      const isl = islands.find((x) => inside(r[0]!, x.outer));
      isl?.holes.push(r);
    }
  });
  const polygons: Ring[][] = [[closed(WORLD), ...holes.map(closed)]];
  for (const isl of islands) polygons.push([closed(isl.outer), ...isl.holes.map(closed)]);
  return { type: "MultiPolygon", coordinates: polygons };
}

/** حلقاتُ مضلَّع GeoJSON (Polygon أو MultiPolygon) الخارجيّة. */
export function outerRings(geom: unknown): Pt[][] {
  const g = geom as { type?: string; coordinates?: unknown } | null | undefined;
  if (!g || !g.coordinates) return [];
  if (g.type === "Polygon") return [(g.coordinates as Pt[][])[0] ?? []];
  if (g.type === "MultiPolygon") return (g.coordinates as Pt[][][]).map((p) => p[0] ?? []);
  return [];
}
