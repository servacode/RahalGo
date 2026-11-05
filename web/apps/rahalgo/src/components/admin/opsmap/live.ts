/**
 * ══════════════════════════════════════════════════════════════════════
 * **حسابُ الخريطة الحيّة — دوالُّ خالصةٌ بلا React ولا خريطة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كلُّ ما يُقاس هنا يُقاس بلا متصفّح** (`scripts/check-opsmap-live.mjs`):
 * انزلاقُ السائق بين نبضتين · البحثُ في المحمَّل · «زاد العالقُ أم لا» ·
 * مرشِّحُ التركيز · خطُّ زمن الطلب · خطُّ مساره · انزلاقُ كلِّ سائقٍ وحدَه ·
 * ترتيبُ المواضع بالزمن.
 *
 * **ولا شيءَ هنا يقرأ الثيمَ أو النافذة** — الألوانُ في `palette.ts`
 * والرسمُ في `canvas.tsx`.
 */

export type LngLat = [number, number];

/** **معلَمٌ** — بالشكل الذي يأخذه MapLibre. */
export interface PointFeature {
  type: "Feature";
  geometry: unknown;
  properties: Record<string, unknown>;
}

// ══════════════════════════════════════════════════════════════════════
// **١ · الانزلاق** — السائقُ يمشي بين نبضتين ولا يقفز
// ══════════════════════════════════════════════════════════════════════

/**
 * **مدّةُ الانزلاق** — ثلاثُ ثوانٍ (٢٠٢٦-١٠-١٩): **نبضةُ التوصيل كلَّ أربع**، فينزلق السائقُ
 * أغلبَ ما بين نبضتين فيُرى يمشي لا يقفز. وكانت ثانيةً مع نبضةٍ كلَّ عشر.
 */
export const TWEEN_MS = 3000;

/**
 * **أبعدُ ما ينزلق** (بالدرجة، نحو خمسة كيلومترات) — **ومن قفز أبعدَ قفز**:
 * سائقٌ فتح التطبيقَ في مدينةٍ أخرى لا يُرسَم عابراً للبيوت.
 */
export const MAX_TWEEN_DEG = 0.05;

/** **تسارعٌ فتباطؤ** — والحدودُ تُقصّ. */
export function easeInOut(t: number): number {
  const x = t <= 0 ? 0 : t >= 1 ? 1 : t;
  return x < 0.5 ? 2 * x * x : 1 - Math.pow(-2 * x + 2, 2) / 2;
}

export function lerpLngLat(a: LngLat, b: LngLat, t: number): LngLat {
  return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t];
}

function pointOf(f: PointFeature): LngLat | null {
  const g = f.geometry as { type?: string; coordinates?: unknown } | null;
  if (!g || g.type !== "Point" || !Array.isArray(g.coordinates)) return null;
  const [lng, lat] = g.coordinates as number[];
  return Number.isFinite(lng) && Number.isFinite(lat) ? [lng as number, lat as number] : null;
}

/** **مواضعُ النقاط بمعرّفها** (خاصّةُ `id`) — وما لا معرّفَ له لا يُتتبَّع. */
export function positionsOf(features: PointFeature[]): Map<string, LngLat> {
  const out = new Map<string, LngLat>();
  for (const f of features) {
    const id = f.properties?.id;
    const p = pointOf(f);
    if (typeof id === "string" && p) out.set(id, p);
  }
  return out;
}

/**
 * **لقطةٌ من الانزلاق** عند الكسر `t` (٠ ← ١، مُيسَّراً من قبل).
 *
 * **والجديدُ يظهر في موضعه** · **والبعيدُ يقفز** · **والخصائصُ من الهدف**
 * — فالنقرُ أثناء الانزلاق يفتح الحالَ الجديد.
 */
export function tweenPoints<F extends PointFeature>(
  from: Map<string, LngLat>,
  to: F[],
  t: number,
  maxJumpDeg = MAX_TWEEN_DEG,
): F[] {
  return to.map((f) => {
    const id = f.properties?.id;
    const target = pointOf(f);
    const start = typeof id === "string" ? from.get(id) : undefined;
    if (!target || !start) return f;
    if (Math.abs(target[0] - start[0]) > maxJumpDeg || Math.abs(target[1] - start[1]) > maxJumpDeg) {
      return f;
    }
    return { ...f, geometry: { type: "Point", coordinates: lerpLngLat(start, target, t) } };
  });
}

/** **أتحرّك شيءٌ أصلاً؟** — وإلّا لا انزلاقَ ولا رسمَ ستّين مرّة. */
export function anyMoved(from: Map<string, LngLat>, to: PointFeature[]): boolean {
  for (const f of to) {
    const id = f.properties?.id;
    const p = pointOf(f);
    const s = typeof id === "string" ? from.get(id) : undefined;
    if (p && s && (p[0] !== s[0] || p[1] !== s[1])) return true;
  }
  return false;
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · البحث في المحمَّل** — رقمُ طلبٍ أو اسمُ سائقٍ أو متجر
// ══════════════════════════════════════════════════════════════════════

/**
 * **تطبيعٌ عربيّ** — التشكيلُ والتطويلُ يُحذفان، والألفاتُ ألفٌ، والياءُ
 * ياء، والتاءُ المربوطةُ هاء، والأرقامُ الهنديّةُ لاتينيّة. **«أحمد»
 * و«احمد» اسمٌ واحدٌ في البحث.**
 */
export function normalizeQuery(s: string): string {
  return s
    .toLowerCase()
    .replace(/[ً-ْٰـ]/g, "")
    .replace(/[آأإٱ]/g, "ا")
    .replace(/ى/g, "ي")
    .replace(/ة/g, "ه")
    .replace(/[٠-٩]/g, (d) => String(d.charCodeAt(0) - 0x0660))
    .replace(/[۰-۹]/g, (d) => String(d.charCodeAt(0) - 0x06f0))
    .replace(/\s+/g, " ")
    .trim();
}

export type LocalHitKind = "order" | "driver" | "merchant";

export interface LocalHit {
  kind: LocalHitKind;
  id: string;
  label: string;
  lng: number;
  lat: number;
  /** **الأصغرُ أوّلاً** — ٠ مطابقٌ · ١ بادئ · ٢ بدايةُ كلمة · ٣ في الوسط. */
  score: number;
}

/** **درجةُ اسمٍ** — أو `null` إن لم يطابق. */
function nameScore(name: string, q: string): number | null {
  const n = normalizeQuery(name);
  if (!q || !n) return null;
  if (n === q) return 0;
  if (n.startsWith(q)) return 1;
  if (n.split(" ").some((w) => w.startsWith(q))) return 2;
  if (n.includes(q)) return 3;
  return null;
}

export interface SearchSources {
  orders: Array<{ id: string; number: number; drop_lng: number; drop_lat: number; merchant?: string }>;
  drivers: Array<{ id: string; name: string; lng: number; lat: number; freshness?: string }>;
  merchants: Array<{ id: string; name: string; lng: number; lat: number }>;
}

/**
 * **يبحث فيما حُمِّل للخريطة** — ولا نداءَ للخادم: ما يُرى يُوجد.
 *
 * **والرقمُ يطابق رقمَ الطلب** (و`#` قبله تُهمَل)؛ **والنصُّ يطابق الأسماء.**
 * **ومن لا موضعَ له لا يُعرض** — اختيارُه لا يطير إلى شيء.
 */
export function searchLocal(raw: string, src: SearchSources, limit = 8): LocalHit[] {
  const q = normalizeQuery(raw).replace(/^#/, "").trim();
  if (q.length === 0) return [];
  const hits: LocalHit[] = [];
  const digits = /^\d+$/.test(q);

  if (digits) {
    for (const o of src.orders) {
      const n = String(o.number);
      const score = n === q ? 0 : n.startsWith(q) ? 1 : n.includes(q) ? 3 : null;
      if (score != null) {
        hits.push({ kind: "order", id: o.id, label: `#${n}`, lng: o.drop_lng, lat: o.drop_lat, score });
      }
    }
  }
  // **والاسمُ يحتاج حرفين** — حرفٌ واحدٌ يطابق نصفَ المدينة.
  if (q.length >= 2) {
    for (const d of src.drivers) {
      if (d.freshness === "NO_LOCATION") continue;
      const score = nameScore(d.name, q);
      if (score != null) hits.push({ kind: "driver", id: d.id, label: d.name, lng: d.lng, lat: d.lat, score });
    }
    for (const x of src.merchants) {
      const score = nameScore(x.name, q);
      if (score != null) hits.push({ kind: "merchant", id: x.id, label: x.name, lng: x.lng, lat: x.lat, score });
    }
  }
  return hits
    .filter((h) => Number.isFinite(h.lng) && Number.isFinite(h.lat) && (h.lng !== 0 || h.lat !== 0))
    .sort((a, b) => a.score - b.score || a.label.localeCompare(b.label, "ar"))
    .slice(0, limit);
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · عالقٌ جديد** — يُنبَّه حين يزيد العدد لا حين يُفتح الشاشة
// ══════════════════════════════════════════════════════════════════════

/**
 * **أزاد العالقُ؟** — والقراءةُ الأولى ليست زيادة: **من فتح الخريطةَ على
 * ثلاثة عالقين لا يُقرَع له جرسٌ عن ثلاثةٍ لم تتبدّل.**
 */
export function stuckIncreased(prev: number | null | undefined, next: number | null | undefined): boolean {
  if (prev == null || next == null) return false;
  return next > prev;
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · مرشِّحُ التركيز** — «العالقة فقط» · «المتاحون فقط»
// ══════════════════════════════════════════════════════════════════════

export type FocusMode = "" | "stuck" | "available";

/**
 * **«العالقة فقط»**: الطلباتُ العالقة وسائقوها ومتاجرُها — **وما سواها
 * يُخفى** فيُرى العطبُ وحدَه. **«المتاحون فقط»**: السائقون المتاحون،
 * والطبقاتُ الأخرى كما هي.
 */
export function applyFocus<
  D extends { id: string; tone: string },
  O extends { stuck_reason?: string | null; driver_id?: string; merchant_id?: string },
  M extends { id: string },
>(mode: FocusMode, drivers: D[], orders: O[], merchants: M[]): { drivers: D[]; orders: O[]; merchants: M[] } {
  if (mode === "available") {
    return { drivers: drivers.filter((d) => d.tone === "available"), orders, merchants };
  }
  if (mode === "stuck") {
    const stuck = orders.filter((o) => !!o.stuck_reason);
    const dIDs = new Set(stuck.map((o) => o.driver_id).filter(Boolean));
    const mIDs = new Set(stuck.map((o) => o.merchant_id).filter(Boolean));
    return {
      drivers: drivers.filter((d) => dIDs.has(d.id)),
      orders: stuck,
      merchants: merchants.filter((x) => mIDs.has(x.id)),
    };
  }
  return { drivers, orders, merchants };
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · خطُّ زمن الطلب** — كم دقيقةً مضت على كلّ مرحلة
// ══════════════════════════════════════════════════════════════════════

export type StepKey = "created" | "accepted" | "assigned" | "picked";

export interface TimelineStep {
  key: StepKey;
  at: string;
  /** **دقائقُ منذ المرحلة** — والسالبُ (ساعةٌ متقدّمة) صفر. */
  minutes: number;
}

export function minutesBetween(iso: string, nowMs: number): number {
  const ms = nowMs - Date.parse(iso);
  return Number.isFinite(ms) && ms > 0 ? Math.floor(ms / 60000) : 0;
}

/** **المراحلُ الواقعةُ وحدَها، بترتيبها** — ولا مرحلةَ تُخترَع. */
export function orderTimeline(
  o: { created_at: string; accepted_at?: string; assigned_at?: string; picked_up_at?: string },
  nowMs: number,
): TimelineStep[] {
  const raw: Array<[StepKey, string | undefined]> = [
    ["created", o.created_at],
    ["accepted", o.accepted_at],
    ["assigned", o.assigned_at],
    ["picked", o.picked_up_at],
  ];
  return raw
    .filter((x): x is [StepKey, string] => typeof x[1] === "string" && x[1] !== "" && Number.isFinite(Date.parse(x[1])))
    .map(([key, at]) => ({ key, at, minutes: minutesBetween(at, nowMs) }));
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · مسارُ الطلب المُحدَّد** — سائقٌ ← متجرٌ ← زبون، خطوطٌ مستقيمة
// ══════════════════════════════════════════════════════════════════════

export interface RouteLeg {
  leg: "to-store" | "to-customer";
  coords: [LngLat, LngLat];
}

/**
 * **ولا محرّكَ توجيهٍ هنا** — لا بابَ للعمليّات إليه، **والخطُّ المستقيمُ
 * يقول «من أين إلى أين» ولا يدّعي طريقاً.**
 *
 * **وبعد الاستلام لا متجرَ في الطريق**: السائقُ ← الزبونُ مباشرة.
 * **وموضعُ السائق الحيُّ أولى** من المحفوظ في الطلب.
 */
export function orderRoute(
  o: {
    drop_lng: number;
    drop_lat: number;
    pick_lng?: number;
    pick_lat?: number;
    driver_lng?: number;
    driver_lat?: number;
    picked_up_at?: string;
  },
  liveDriver?: LngLat | null,
): RouteLeg[] {
  const drop: LngLat = [o.drop_lng, o.drop_lat];
  const pick: LngLat | null = o.pick_lng != null && o.pick_lat != null ? [o.pick_lng, o.pick_lat] : null;
  const driver: LngLat | null =
    liveDriver ?? (o.driver_lng != null && o.driver_lat != null ? [o.driver_lng, o.driver_lat] : null);
  const legs: RouteLeg[] = [];
  if (o.picked_up_at) {
    if (driver) legs.push({ leg: "to-customer", coords: [driver, drop] });
    else if (pick) legs.push({ leg: "to-customer", coords: [pick, drop] });
    return legs;
  }
  if (pick) {
    if (driver) legs.push({ leg: "to-store", coords: [driver, pick] });
    legs.push({ leg: "to-customer", coords: [pick, drop] });
  } else if (driver) {
    legs.push({ leg: "to-customer", coords: [driver, drop] });
  }
  return legs;
}

// ══════════════════════════════════════════════════════════════════════
// **٧ · كلُّ سائقٍ ينزلق وحدَه** — بخطٍّ مستقيمٍ وسرعةٍ ثابتة، على قدّ ما يبعت (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// (طلبُ المالك: «بدنا حركة السائق على الخريطة تكون مباشرة ما بدنا نطّات».)
//
// **كانت الطبقةُ كلُّها تنزلق ثلاثَ ثوانٍ بتسارعٍ وتباطؤ** (`١` فوق) — ونبضةُ التوصيل كلَّ أربع:
// يمشي ثمّ يقف ثانيةً كلَّ أربع. **وكلُّ نبضةٍ لأيّ سائقٍ كانت تُعيد انزلاقَ الجميع من الصفر.**
//
// **والآن لكلّ سائقٍ حركتُه**: من حيث هو الآن إلى نقطته الجديدة بخطٍّ مستقيم، **مدّتُها وسيطُ ما
// بين نبضاته** (آخرُ خمسة فواصل بين نبضتين متحرّكتين متتاليتين من البثّ — لا النبضةُ الواقفة ولا
// الرجفة ولا صفُّ القائمة). فيصل كلَّ نقطةٍ قرابةَ وصول التالية: **حركةٌ متّصلةٌ بسرعةٍ ثابتة،
// متأخّرةٌ عن أحدث نقطةٍ نبضةً واحدة** — ٤ ثوانٍ مع طلب، و٢٠–٤٠ بلا طلب.
//
// **ويُوضع وضعاً** (يظهر في مكانه الجديد خلال ٣٠٠ م ث، لا ينزلق فوق البيوت) بعد صمتٍ أطولَ بكثيرٍ
// من عادته — max(٢٫٥ × وسيطه، دقيقة) — أو قفزةٍ فوق كيلو ونصف أو أسرعَ من ١٢٠ كم/س.

/** **أقلُّ تبدّلٍ يُعدّ حركة** (بالمتر) — ما دونه رجفةٌ دون البكسل. */
export const DEAD_BAND_M = 3;
/** **أقصرُ انزلاق** (م ث). */
export const GLIDE_MIN_MS = 1000;
/** **أطولُ انزلاق** (م ث) — من يبعت كلَّ دقيقةٍ ينزلق ٤٥ ثانيةً منها. */
export const GLIDE_MAX_MS = 45000;
/** **صمتٌ يُوضع بعده** — أضعافُ وسيطه… */
export const PLACE_GAP_FACTOR = 2.5;
/** **…ولا أقلَّ من دقيقة.** */
export const PLACE_GAP_MIN_MS = 60000;
/** **قفزةٌ تُوضع** (بالمتر). */
export const PLACE_HOP_M = 1500;
/** **أسرعُ من هذا يُوضع** — ١٢٠ كم/س، عتبةُ القفزة المستحيلة في الخادم. */
export const PLACE_SPEED_MPS = 120 / 3.6;
/** **ظهورٌ في المكان الجديد** (م ث). */
export const FADE_MS = 300;
/** **كم فاصلاً يُحفظ للوسيط.** */
export const CADENCE_GAPS = 5;
/** **أقلُّ ما بين رفعين للطبقة المتحرّكة** (م ث) — نحو ثلاثين في الثانية. */
export const FRAME_MS = 33;
/** **سهمُ الاتّجاه يظهر فوق هذه السرعة** (م/ث). */
export const HEADING_MIN_SPEED_MPS = 2;

/** **المسافةُ بالمتر** بين موضعين `[lng, lat]`. */
export function metersApart(a: LngLat, b: LngLat): number {
  const R = 6371000;
  const rad = Math.PI / 180;
  const dLat = (b[1] - a[1]) * rad;
  const dLng = (b[0] - a[0]) * rad;
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(a[1] * rad) * Math.cos(b[1] * rad) * Math.sin(dLng / 2) ** 2;
  return 2 * R * Math.asin(Math.min(1, Math.sqrt(h)));
}

/** **الوسيط** — أو `null` لما لا شيءَ فيه. */
export function median(xs: number[]): number | null {
  if (xs.length === 0) return null;
  const s = [...xs].sort((a, b) => a - b);
  const m = Math.floor(s.length / 2);
  const hi = s[m] as number;
  return s.length % 2 ? hi : ((s[m - 1] as number) + hi) / 2;
}

/** **مدّةُ انزلاقٍ ضمن حدّيها.** */
export function clampGlide(ms: number): number {
  if (!Number.isFinite(ms)) return GLIDE_MIN_MS;
  return Math.min(GLIDE_MAX_MS, Math.max(GLIDE_MIN_MS, ms));
}

/** **إيقاعُ سائق** — ما يُحفظ له بين نبضةٍ وأخرى. */
export interface Cadence {
  /** **وقتُ آخرِ ما سُمع منه** (م ث) — أيّاً كان. */
  at: number;
  /** **هدفُه** — موضعُ آخرِ نبضةٍ تحرّكت. */
  pos: LngLat;
  /** **أكانت الأخيرةُ نبضةً متحرّكةً من البثّ؟** — الفاصلُ يُعدّ بين متحرّكتين متتاليتين وحدهما. */
  moving: boolean;
  /** **آخرُ الفواصل** (م ث). */
  gaps: number[];
  /** **مدّةُ الانزلاق إلى الهدف** (م ث) — **وصفرٌ: يُوضع.** */
  glideMs: number;
}

/** **نبضةٌ تُسجَّل في الإيقاع.** */
export interface FixNote {
  pos: LngLat;
  atMs: number;
  /** **أتُعدّ فاصلاً؟** — نبضةُ البثّ نعم، **وصفُّ القائمة لا** (نبضةٌ فاتت البثّ). */
  counted: boolean;
  /** **المدّةُ قبل أن يُعرف إيقاعُه** — نبضةُ الإعدادات (مع طلبٍ أو بلا طلب). */
  fallbackMs: number;
}

/**
 * **يسجّل نبضةً ويقرّر: انزلاقٌ بكم، أم وضع.**
 *
 * **ودون `DEAD_BAND_M` لا حركة** — يتحدّث الوقتُ وحدَه (النبضةُ الواقفة والرجفة)، **وتنقطع
 * السلسلة** فلا يُعدّ الفاصلُ التالي. **والموضوعُ وضعاً لا يُعدّ فاصلاً** — هو انقطاعٌ لا إيقاع.
 */
export function noteFix(prev: Cadence | undefined, n: FixNote): Cadence {
  if (!prev || !Number.isFinite(prev.at)) {
    return { at: n.atMs, pos: n.pos, moving: false, gaps: [], glideMs: 0 };
  }
  const hop = metersApart(prev.pos, n.pos);
  if (hop < DEAD_BAND_M) return { ...prev, at: Math.max(prev.at, n.atMs), moving: false };
  const gap = n.atMs - prev.at;
  const med = median(prev.gaps) ?? n.fallbackMs;
  const place =
    !(gap > 0) ||
    gap > Math.max(PLACE_GAP_FACTOR * med, PLACE_GAP_MIN_MS) ||
    hop > PLACE_HOP_M ||
    hop / (gap / 1000) > PLACE_SPEED_MPS;
  const count = n.counted && prev.moving && !place;
  const gaps = count ? [...prev.gaps, gap].slice(-CADENCE_GAPS) : prev.gaps;
  return {
    at: n.atMs,
    pos: n.pos,
    moving: n.counted && !place,
    gaps,
    glideMs: place ? 0 : clampGlide(median(gaps) ?? n.fallbackMs),
  };
}

/** **حركةُ علامةٍ واحدة** — من `from` إلى `to` خطّيّاً خلال `dur` (والصفرُ: وُضعت). */
export interface Motion {
  from: LngLat;
  to: LngLat;
  start: number;
  dur: number;
  /** **وُضعت وضعاً** — تظهر خلال `FADE_MS`. */
  fade: boolean;
}

/** **أين العلامةُ الآن** — خطّيّاً: ربعُ الوقت ربعُ الطريق. */
export function positionAt(m: Motion, now: number): LngLat {
  if (m.dur <= 0) return m.to;
  const t = (now - m.start) / m.dur;
  if (t >= 1) return m.to;
  if (t <= 0) return m.from;
  return lerpLngLat(m.from, m.to, t);
}

/** **شفافيّتُها الآن** — واحدٌ إلّا أثناءَ ظهورها في مكانٍ وُضعت فيه. */
export function opacityAt(m: Motion, now: number): number {
  if (!m.fade) return 1;
  const t = (now - m.start) / FADE_MS;
  return t >= 1 ? 1 : 0.15 + 0.85 * Math.max(0, t);
}

/** **أما زالت تتحرّك (أو تظهر)؟** */
export function activeAt(m: Motion, now: number): boolean {
  return now < m.start + Math.max(m.dur, m.fade ? FADE_MS : 0);
}

/**
 * **هدفٌ جديدٌ لعلامة** — من حيث هي الآن لا من حيث كانت، **ودون `DEAD_BAND_M` لا تُمسّ.**
 * **وأوّلُ ظهورٍ في مكانه** بلا انزلاقٍ ولا تلاشٍ؛ **و`glideMs` صفرٌ: وضعٌ مع ظهور.**
 */
export function retarget(prev: Motion | undefined, target: LngLat, now: number, glideMs: number): Motion {
  if (!prev) return { from: target, to: target, start: now, dur: 0, fade: false };
  if (metersApart(prev.to, target) < DEAD_BAND_M) return prev;
  if (!(glideMs > 0)) return { from: target, to: target, start: now, dur: 0, fade: true };
  return { from: positionAt(prev, now), to: target, start: now, dur: glideMs, fade: false };
}

/**
 * **حركاتُ طبقةٍ بعد بياناتٍ جديدة** — **ومن لم يتبدّل هدفُه تبقى حركتُه هي نفسُها** (لا تُعاد من
 * الصفر). مدّةُ كلِّ واحدٍ من خاصّة `glideField` في معلَمه (يضعها من يعرف إيقاعه).
 */
export function advanceMotions(
  prev: Map<string, Motion> | undefined,
  features: PointFeature[],
  now: number,
  glideField = "_glide",
): Map<string, Motion> {
  const out = new Map<string, Motion>();
  for (const f of features) {
    const id = f.properties?.id;
    const p = pointOf(f);
    if (typeof id !== "string" || !p) continue;
    const g = Number(f.properties?.[glideField]);
    out.set(id, retarget(prev?.get(id), p, now, Number.isFinite(g) ? g : 0));
  }
  return out;
}

/** **أيتحرّك أحدٌ في الطبقة؟** — وإلّا لا إطار. */
export function anyActive(motions: Map<string, Motion> | undefined, now: number): boolean {
  if (!motions) return false;
  for (const m of motions.values()) if (activeAt(m, now)) return true;
  return false;
}

/**
 * **لقطةُ الطبقة الآن** — كلُّ علامةٍ في موضعها من حركتها، **والخصائصُ من الهدف** (فالنقرُ يفتح
 * الحالَ الجديد)، وشفافيّةُ الظاهر للتوّ تُضرب في `opField`.
 */
export function frameFeatures<F extends PointFeature>(
  features: F[],
  motions: Map<string, Motion> | undefined,
  now: number,
  opField = "_op",
): F[] {
  if (!motions) return features;
  return features.map((f) => {
    const id = f.properties?.id;
    const m = typeof id === "string" ? motions.get(id) : undefined;
    if (!m) return f;
    const fade = opacityAt(m, now);
    const base = typeof f.properties?.[opField] === "number" ? (f.properties[opField] as number) : 1;
    return {
      ...f,
      geometry: { type: "Point", coordinates: positionAt(m, now) },
      properties: fade < 1 ? { ...f.properties, [opField]: base * fade } : f.properties,
    };
  });
}

/**
 * **خطٌّ يبدأ من العلامة المنزلقة** — المعالمُ التي خاصّتُها `field` صحيحةٌ يصير أوّلُ رأسِها
 * `pos`، **فيخرج خطُّ الطلب المُحدَّد من السائق حيث يُرى لا حيث كان.**
 */
export function anchorLines<F extends PointFeature>(
  features: F[],
  pos: LngLat | null | undefined,
  field = "from_driver",
): F[] {
  if (!pos) return features;
  return features.map((f) => {
    if (f.properties?.[field] !== true) return f;
    const g = f.geometry as { type?: string; coordinates?: unknown } | null;
    if (!g || g.type !== "LineString" || !Array.isArray(g.coordinates) || g.coordinates.length < 1) return f;
    return { ...f, geometry: { type: "LineString", coordinates: [pos, ...(g.coordinates as LngLat[]).slice(1)] } };
  });
}

/** **ما رُفع للطبقة آخرَ مرّة.** */
export interface Uploaded {
  data: unknown;
  visible: boolean;
}

/** **أيُرفع؟** — **بياناتٌ هي نفسُها لا تُرفع ثانيةً**: نبضةُ سائقٍ لا تُعيد رفعَ المتاجر والمناطق. */
export function needsUpload(prev: Uploaded | undefined, next: Uploaded): boolean {
  return !prev || prev.data !== next.data || prev.visible !== next.visible;
}

/**
 * **أتُحرَّك المواضع؟** — **الموضعُ معلومةٌ لا زينة**: «تقليلُ الحركة» في النظام يوقف النبضَ
 * لا الانزلاق (وكانت كلُّ نقطةٍ عنده قفزة). **والانزلاقُ القديمُ (المفتاحُ مطفأ) كما كان.**
 */
export function interpolates(mode: "glide" | "tween", reducedMotion: boolean): boolean {
  return mode === "glide" || !reducedMotion;
}

/** **سهمُ الاتّجاه** — الاتّجاهُ وهو ماشٍ وحدَه، وإلّا لا سهم. */
export function headingOf(bearing: number | null | undefined, speedMps: number | null | undefined): number | null {
  if (bearing == null || !Number.isFinite(bearing) || bearing < 0 || bearing >= 360) return null;
  if (speedMps == null || !(speedMps >= HEADING_MIN_SPEED_MPS)) return null;
  return bearing;
}

// ══════════════════════════════════════════════════════════════════════
// **٨ · ترتيبُ المواضع بالزمن** — ما هو أقدمُ من آخرِ ما طُبّق لا يحرّك العلامة
// ══════════════════════════════════════════════════════════════════════
//
// (طلبُ المالك ٢٠٢٦-١١-٠٥: «بدنا حركة السائق على الخريطة تكون مباشرة ما بدنا نطّات».)
//
// **موضعُ السائق يصل من بابين**: نبضةٌ تُدفَع بالبثّ، وصفٌّ في قائمةٍ تُجلَب بالمدّة.
// **وجوابُ القائمة قد يُقرأ قبل نبضةٍ ويصل بعدها** — فكان يضع السائقَ حيث كان قبل
// أربع ثوانٍ، ثمّ تقدّمه النبضةُ التالية: رجوعٌ وتقدّمٌ كلَّ بضع ثوانٍ.
//
// **فيُحفظ لكلّ سائقٍ أحدثُ موضعٍ طُبّق** (من البابين معاً)، **وما هو أقدمُ منه لا يحرّك
// العلامة.** والأحدثُ منه يُطبَّق ولو جاء من القائمة — **هو نبضةٌ فاتت البثّ** (ساعةُ
// هاتفٍ متأخّرة، أو رسالةٌ أُسقطت).

/** **وقتُ الالتقاط بالميلّي ثانية** — والتالفُ `NaN`. */
function fixTime(at: string | null | undefined): number {
  return typeof at === "string" && at !== "" ? Date.parse(at) : NaN;
}

/**
 * **أليس هذا الموضعُ أقدمَ من آخرِ ما طُبّق؟**
 *
 * **والمساوي يُطبَّق** — هو الموضعُ نفسُه (صفُّ قائمةٍ قُرئ بعد نبضته): لا يحرّك شيئاً
 * **ويُحدّث عمرَه من الخادم.** **ولا سابقَ يُقارَن به ⇒ يُطبَّق**؛ **ووقتٌ تالفٌ وقبله
 * سابقٌ ⇒ لا يُطبَّق** — ما لا يُرتَّب لا يُرجع أحداً.
 */
export function newerFix(prevAt: string | null | undefined, at: string | null | undefined): boolean {
  const p = fixTime(prevAt);
  if (!Number.isFinite(p)) return true;
  const a = fixTime(at);
  return Number.isFinite(a) && a >= p;
}

/** **ما يقوله البابان عن موضع السائق** — ويُطبَّق كلُّه معاً أو لا يُطبَّق: الموضعُ وعمرُه لا يفترقان. */
export interface DriverFix {
  lat: number;
  lng: number;
  last_location_at: string;
  age_sec: number;
  freshness: string;
  /** **من نقطة العرض** (٢٠٢٦-١١-٠٥) — اتّجاهُه وسرعتُه وباهتٌ أم لا، تمشي مع موضعها. */
  bearing?: number;
  speed_mps?: number;
  rough?: boolean;
}

/** **حقولُ الموضع من صفٍّ بعينه** — بأنواعها هي. */
export type FixOf<R extends DriverFix> = Pick<R, keyof DriverFix>;

function fixOf<R extends DriverFix>(r: R): FixOf<R> {
  return {
    lat: r.lat,
    lng: r.lng,
    last_location_at: r.last_location_at,
    age_sec: r.age_sec,
    freshness: r.freshness,
    bearing: r.bearing,
    speed_mps: r.speed_mps,
    rough: r.rough,
  };
}

/**
 * **نبضةٌ من البثّ** — تُحفظ وتُطبَّق إن لم تكن أقدمَ من آخرِ ما طُبّق. **ونبضةٌ بلا وقتٍ
 * مقروءٍ لا تُطبَّق**: الخادمُ يختمها دائماً، وما لا يُرتَّب لا يحرّك أحداً.
 */
export function acceptPush<R extends DriverFix>(book: Map<string, FixOf<R>>, id: string, fix: FixOf<R>): boolean {
  if (!Number.isFinite(fixTime(fix.last_location_at))) return false;
  const prev = book.get(id);
  if (prev && !newerFix(prev.last_location_at, fix.last_location_at)) return false;
  book.set(id, fix);
  return true;
}

/**
 * **جوابُ القائمة يمرّ بالدفتر** — صفٌّ أقدمُ من آخرِ ما طُبّق يبقى على ذاك الموضع (ومعه
 * وقتُه وعمرُه)، **وما سوى الموضع يُؤخذ من الصفّ** (الحالُ والطلباتُ واللون). وكلُّ صفٍّ
 * آخرَ يُطبَّق كما هو ويصير الأحدث.
 */
export function mergeRows<R extends DriverFix & { id: string }>(book: Map<string, FixOf<R>>, rows: R[]): R[] {
  return rows.map((r) => {
    const prev = book.get(r.id);
    if (prev && !newerFix(prev.last_location_at, r.last_location_at)) return { ...r, ...prev };
    book.set(r.id, fixOf(r));
    return r;
  });
}
