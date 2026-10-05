/**
 * ══════════════════════════════════════════════════════════════════════
 * **حسابُ الخريطة الحيّة — دوالُّ خالصةٌ بلا React ولا خريطة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كلُّ ما يُقاس هنا يُقاس بلا متصفّح** (`scripts/check-opsmap-live.mjs`):
 * انزلاقُ السائق بين نبضتين · البحثُ في المحمَّل · «زاد العالقُ أم لا» ·
 * مرشِّحُ التركيز · خطُّ زمن الطلب · خطُّ مساره.
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

/** **مدّةُ الانزلاق** — ثانيةٌ واحدة: تُرى الحركةُ ولا تتأخّر الحقيقة. */
export const TWEEN_MS = 1000;

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
