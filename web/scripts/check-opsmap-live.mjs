/**
 * ══════════════════════════════════════════════════════════════════════
 * **حسابُ الخريطة الحيّة يُنادى ويُقاس** (تحسيناتُ خريطة العمليات ٢٠٢٦-١٠-٠٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الدوالُّ في `components/admin/opsmap/live.ts` خالصة** — فتُنادى هنا
 * بحالاتٍ وتُقاس نتائجُها: **الانزلاقُ** بين نبضتين · **البحثُ** في
 * المحمَّل · **«زاد العالقُ»** · **مرشِّحُ التركيز** · **خطُّ الزمن** ·
 * **خطُّ المسار** · **انزلاقُ كلِّ سائقٍ وحدَه** · **ترتيبُ المواضع بالزمن** · **وأحداثُ التيّار** في ناقل
 * البثّ (`packages/ui/src/liveStream.ts`).
 *
 * **وقراءةُ السطر لا تُثبت أنّه يحكم** — فلا فحصَ نصّيّاً هنا.
 */
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { register } from "node:module";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");
register(pathToFileURL(join(here, "_tsload.mjs")).href);

const L = await import(
  pathToFileURL(join(web, "apps/rahalgo/src/components/admin/opsmap/live.ts")).href
);
const S = await import(pathToFileURL(join(web, "packages/ui/src/liveStream.ts")).href);

let bad = 0;
let cases = 0;
const fail = (msg) => {
  console.error(`  ✗ ${msg}`);
  bad++;
};
const check = (ok, msg) => {
  cases++;
  if (!ok) fail(msg);
};
const near = (a, b, eps = 1e-9) => Math.abs(a - b) < eps;
const pt = (id, lng, lat, extra = {}) => ({
  type: "Feature",
  geometry: { type: "Point", coordinates: [lng, lat] },
  properties: { id, ...extra },
});

// ── ١ · الانزلاق ─────────────────────────────────────────────────────
check(L.easeInOut(0) === 0 && L.easeInOut(1) === 1, "١ · التيسيرُ لا يبدأ من صفرٍ أو لا ينتهي بواحد");
check(near(L.easeInOut(0.5), 0.5), "١ · التيسيرُ في المنتصف ليس نصفاً");
check(L.easeInOut(-3) === 0 && L.easeInOut(7) === 1, "١ · التيسيرُ لا يقصّ الحدود");
{
  const from = L.positionsOf([pt("a", 39.0, 35.9), pt("b", 39.1, 35.95)]);
  const to = [pt("a", 39.002, 35.902, { icon: "driver-busy" }), pt("c", 39.2, 36.0), pt("b", 39.5, 35.95)];
  const mid = L.tweenPoints(from, to, 0.5);
  const [ax, ay] = mid[0].geometry.coordinates;
  check(near(ax, 39.001) && near(ay, 35.901), `١ · السائقُ «a» لم ينزلق إلى المنتصف (${ax}, ${ay})`);
  check(mid[0].properties.icon === "driver-busy", "١ · الخصائصُ أثناء الانزلاق ليست من الهدف");
  check(mid[1].geometry.coordinates[0] === 39.2, "١ · الجديدُ لم يظهر في موضعه");
  check(mid[2].geometry.coordinates[0] === 39.5, "١ · قفزةٌ أبعدُ من الحدّ انزلقت عبر المدينة");
  const end = L.tweenPoints(from, to, 1);
  check(near(end[0].geometry.coordinates[0], 39.002), "١ · نهايةُ الانزلاق ليست الهدف");
  check(L.anyMoved(from, to) === true, "١ · حركةٌ لم تُرصد");
  check(L.anyMoved(from, [pt("a", 39.0, 35.9)]) === false, "١ · سكونٌ عُدّ حركة");
  check(L.positionsOf([{ type: "Feature", geometry: null, properties: { id: "x" } }]).size === 0,
    "١ · معلَمٌ بلا هندسةٍ تُتبِّع");
}

// ── ٢ · البحث في المحمَّل ────────────────────────────────────────────
{
  const src = {
    orders: [
      { id: "o1", number: 1042, drop_lng: 39, drop_lat: 35.9 },
      { id: "o2", number: 21042, drop_lng: 39.01, drop_lat: 35.91 },
      { id: "o3", number: 77, drop_lng: 0, drop_lat: 0 },
    ],
    drivers: [
      { id: "d1", name: "أحمد الخلف", lng: 39, lat: 35.9, freshness: "LIVE" },
      { id: "d2", name: "محمد احمد", lng: 39, lat: 35.9, freshness: "FRESH" },
      { id: "d3", name: "أحمد بلا موقع", lng: 0, lat: 0, freshness: "NO_LOCATION" },
    ],
    merchants: [{ id: "m1", name: "فرن الأمل", lng: 39, lat: 35.9 }],
  };
  const h1 = L.searchLocal("1042", src);
  check(h1[0]?.id === "o1" && h1[0]?.score === 0, "٢ · رقمُ الطلب المطابقُ ليس أوّلاً");
  check(h1.some((h) => h.id === "o2"), "٢ · رقمٌ يحوي المبحوثَ غاب");
  check(L.searchLocal("#١٠٤٢", src)[0]?.id === "o1", "٢ · الأرقامُ الهنديّةُ أو «#» لم تُطبَّع");
  check(L.searchLocal("77", src).length === 0, "٢ · طلبٌ بلا موضعٍ عُرض");
  const h2 = L.searchLocal("احمد", src);
  check(h2[0]?.id === "d1", "٢ · «احمد» لم يطابق «أحمد» أوّلاً (الهمزة)");
  check(h2.some((h) => h.id === "d2"), "٢ · اسمٌ في كلمته الثانية غاب");
  check(!h2.some((h) => h.id === "d3"), "٢ · سائقٌ بلا موقعٍ عُرض");
  check(L.searchLocal("الامل", src)[0]?.kind === "merchant", "٢ · المتجرُ لم يُوجد باسمه");
  check(L.searchLocal("ف", src).length === 0, "٢ · حرفٌ واحدٌ طابق أسماء");
  check(L.searchLocal("   ", src).length === 0, "٢ · الفراغُ طابق شيئاً");
  check(L.normalizeQuery("مَدْرَسَة") === "مدرسه", "٢ · التشكيلُ أو التاءُ المربوطةُ لم تُطبَّع");
}

// ── ٣ · عالقٌ جديد ───────────────────────────────────────────────────
check(L.stuckIncreased(null, 3) === false, "٣ · القراءةُ الأولى عُدّت زيادة — جرسٌ عند فتح الشاشة");
check(L.stuckIncreased(2, 3) === true, "٣ · زيادةٌ لم تُرصد");
check(L.stuckIncreased(3, 3) === false, "٣ · عددٌ ثابتٌ عُدّ زيادة");
check(L.stuckIncreased(3, 1) === false, "٣ · نقصانٌ عُدّ زيادة");
check(L.stuckIncreased(1, undefined) === false, "٣ · غيابُ الرقم عُدّ زيادة");

// ── ٤ · مرشِّحُ التركيز ──────────────────────────────────────────────
{
  const drivers = [
    { id: "d1", tone: "available" },
    { id: "d2", tone: "busy" },
    { id: "d3", tone: "idle" },
  ];
  const orders = [
    { id: "o1", stuck_reason: "no_driver", merchant_id: "m1" },
    { id: "o2", stuck_reason: "too_long", driver_id: "d2", merchant_id: "m2" },
    { id: "o3", driver_id: "d3", merchant_id: "m3" },
  ];
  const merchants = [{ id: "m1" }, { id: "m2" }, { id: "m3" }];
  const all = L.applyFocus("", drivers, orders, merchants);
  check(all.drivers.length === 3 && all.orders.length === 3, "٤ · بلا تركيزٍ أُخفي شيء");
  const st = L.applyFocus("stuck", drivers, orders, merchants);
  check(st.orders.map((o) => o.id).join() === "o1,o2", "٤ · «العالقة فقط» لم تُبقِ العالقَ وحدَه");
  check(st.drivers.map((d) => d.id).join() === "d2", "٤ · «العالقة فقط» لم تُبقِ سائقي العالق وحدَهم");
  check(st.merchants.map((x) => x.id).join() === "m1,m2", "٤ · «العالقة فقط» لم تُبقِ متاجرَ العالق وحدَها");
  const av = L.applyFocus("available", drivers, orders, merchants);
  check(av.drivers.map((d) => d.id).join() === "d1", "٤ · «المتاحون فقط» أبقى غيرَ المتاح");
  check(av.orders.length === 3 && av.merchants.length === 3, "٤ · «المتاحون فقط» أخفى الطلباتِ أو المتاجر");
}

// ── ٥ · خطُّ زمن الطلب ───────────────────────────────────────────────
{
  const now = Date.parse("2026-10-06T10:00:00Z");
  const tl = L.orderTimeline(
    {
      created_at: "2026-10-06T09:30:00Z",
      accepted_at: "2026-10-06T09:35:30Z",
      assigned_at: "2026-10-06T09:50:00Z",
    },
    now,
  );
  check(tl.map((s) => s.key).join() === "created,accepted,assigned", "٥ · مراحلُ خُلقت أو سقطت");
  check(tl[0].minutes === 30 && tl[1].minutes === 24 && tl[2].minutes === 10, "٥ · الدقائقُ خطأ");
  const future = L.orderTimeline({ created_at: "2026-10-06T10:05:00Z" }, now);
  check(future[0].minutes === 0, "٥ · ساعةٌ متقدّمةٌ أعطت دقائقَ سالبة");
  check(L.orderTimeline({ created_at: "x", picked_up_at: "" }, now).length === 0, "٥ · وقتٌ تالفٌ عُرض");
}

// ── ٦ · خطُّ المسار ──────────────────────────────────────────────────
{
  const base = { drop_lng: 39.0, drop_lat: 35.9, pick_lng: 39.1, pick_lat: 35.95 };
  const r1 = L.orderRoute({ ...base, driver_lng: 39.2, driver_lat: 36 });
  check(r1.length === 2 && r1[0].leg === "to-store" && r1[0].coords[0][0] === 39.2 && r1[1].leg === "to-customer",
    "٦ · قبل الاستلام: سائقٌ ← متجرٌ ← زبونٌ لم يُرسم");
  const r2 = L.orderRoute({ ...base, driver_lng: 39.2, driver_lat: 36 }, [39.3, 36.1]);
  check(r2[0].coords[0][0] === 39.3, "٦ · موضعُ السائق الحيُّ لم يُقدَّم على المحفوظ");
  const r3 = L.orderRoute({ ...base, driver_lng: 39.2, driver_lat: 36, picked_up_at: "2026-10-06T09:00:00Z" });
  check(r3.length === 1 && r3[0].leg === "to-customer" && r3[0].coords[0][0] === 39.2,
    "٦ · بعد الاستلام: المتجرُ بقي في الطريق");
  const r4 = L.orderRoute(base);
  check(r4.length === 1 && r4[0].coords[0][0] === 39.1, "٦ · بلا سائقٍ: لم يُرسم المتجرُ ← الزبون");
  check(L.orderRoute({ drop_lng: 39, drop_lat: 35.9 }).length === 0, "٦ · بلا متجرٍ ولا سائقٍ رُسم خطّ");
}

// ── ٧ · كلُّ سائقٍ ينزلق وحدَه (٢٠٢٦-١١-٠٥) ─────────────────────────
//
// **خطّيّاً، بمدّة إيقاعه هو، ومن حيث هو الآن** — «ما بدنا نطّات».
{
  const t0 = Date.parse("2026-11-05T09:00:00Z");
  const north = (m) => [39.0, 35.95 + m / 111195];
  const A = north(0);
  const B = north(400);

  // W1 · خطّيّ: ربعُ الوقت ربعُ الطريق
  const mo = { from: A, to: B, start: 1000, dur: 4000, fade: false };
  for (const f of [0.25, 0.5, 0.75]) {
    const p = L.positionAt(mo, 1000 + 4000 * f);
    check(near(L.metersApart(A, p), 400 * f, 0.5), `٧ · W1 · ${f * 100}٪ من الوقت ليست ${f * 100}٪ من الطريق (${L.metersApart(A, p).toFixed(1)} م)`);
  }
  check(L.positionAt(mo, 9000) === B && L.positionAt(mo, 0) === A, "٧ · W1 · الحركةُ لم تقف عند طرفيها");

  // W2 · نبضةٌ أثناء الانزلاق تبدأ من حيث العلامة، ولا تمسّ حركةَ غيره
  const feat = (id, p, glide) => pt(id, p[0], p[1], { _glide: glide });
  const m0 = L.advanceMotions(undefined, [feat("a", A, 4000), feat("b", B, 4000)], 0);
  check(m0.get("a").dur === 0 && m0.get("a").fade === false, "٧ · W2 · أوّلُ ظهورٍ انزلق أو تلاشى");
  const m1 = L.advanceMotions(m0, [feat("a", north(30), 4000), feat("b", B, 4000)], 1000);
  const m2 = L.advanceMotions(m1, [feat("a", north(60), 4000), feat("b", B, 4000)], 3000);
  const mid = L.positionAt(m1.get("a"), 3000);
  check(near(L.metersApart(m2.get("a").from, mid), 0, 1e-6), "٧ · W2 · نبضةٌ أثناء الانزلاق لم تبدأ من حيث العلامة");
  check(m2.get("b") === m0.get("b") && m1.get("b") === m0.get("b"), "٧ · W2 · نبضةُ سائقٍ أعادت حركةَ غيره");
  check(m2.get("a").dur === 4000, "٧ · W2 · المدّةُ ليست من معلَمه");

  // W3 · المدّةُ وسيطُ إيقاعه بين ثانيةٍ و٤٥ — والواقفةُ والرجفةُ لا تُعدّان
  const run = (gapSec, n, fallback) => {
    let c;
    for (let i = 0; i < n; i++) {
      c = L.noteFix(c, { pos: north(30 * i), atMs: t0 + i * gapSec * 1000, counted: true, fallbackMs: fallback });
    }
    return c;
  };
  check(run(4, 8, 4000).glideMs === 4000, "٧ · W3 · إيقاعُ ٤ ثوانٍ لم يعطِ ٤");
  check(run(20, 8, 60000).glideMs === 20000, "٧ · W3 · إيقاعُ ٢٠ ثانيةً لم يعطِ ٢٠");
  check(run(40, 8, 60000).glideMs === 40000, "٧ · W3 · إيقاعُ ٤٠ ثانيةً لم يعطِ ٤٠");
  check(run(60, 8, 60000).glideMs === 45000, "٧ · W3 · إيقاعُ دقيقةٍ لم يُقصّ إلى ٤٥");
  check(L.noteFix(undefined, { pos: A, atMs: t0, counted: true, fallbackMs: 4000 }).glideMs === 0,
    "٧ · W3 · أوّلُ نبضةٍ لم تُوضع في مكانها");
  {
    let c = run(4, 6, 4000);
    const last = c.at;
    // نبضةٌ واقفةٌ بعد ثانيتين، ثمّ نبضةٌ متحرّكةٌ بعد ثانيتين منها
    c = L.noteFix(c, { pos: c.pos, atMs: last + 2000, counted: true, fallbackMs: 4000 });
    check(c.moving === false && c.glideMs === 4000, "٧ · W3 · النبضةُ الواقفةُ غيّرت الإيقاع");
    c = L.noteFix(c, { pos: north(30 * 6), atMs: last + 4000, counted: true, fallbackMs: 4000 });
    check(c.gaps.every((g) => g === 4000), `٧ · W3 · فاصلٌ بعد نبضةٍ واقفةٍ عُدّ (${c.gaps.join(",")})`);
    // ورجفةٌ دون ٣ م لا حركة
    const j = L.noteFix(c, { pos: [c.pos[0], c.pos[1] + 2 / 111195], atMs: last + 8000, counted: true, fallbackMs: 4000 });
    check(j.moving === false && j.pos === c.pos && j.gaps === c.gaps, "٧ · W3 · رجفةٌ دون ٣ م عُدّت حركة");
    // وصفُّ القائمة لا يُعدّ فاصلاً
    const r = L.noteFix(c, { pos: north(30 * 7), atMs: last + 8000, counted: false, fallbackMs: 4000 });
    check(r.gaps === c.gaps && r.moving === false && r.glideMs > 0, "٧ · W3 · صفُّ القائمة عُدّ فاصلاً");
  }

  // W4 · يُوضع بعد صمتٍ أطولَ بكثيرٍ من عادته أو قفزةٍ لا يقطعها راكب
  {
    const c20 = run(20, 6, 60000);
    const g30 = L.noteFix(c20, { pos: north(30 * 6 + 200), atMs: c20.at + 30000, counted: true, fallbackMs: 60000 });
    check(g30.glideMs > 0, "٧ · W4 · إيقاعُ ٢٠ ثانيةً وفجوةُ ٣٠ وُضع — عادتُه لا انقطاع");
    const c4 = run(4, 6, 4000);
    const g70 = L.noteFix(c4, { pos: north(30 * 6 + 300), atMs: c4.at + 70000, counted: true, fallbackMs: 4000 });
    check(g70.glideMs === 0, "٧ · W4 · إيقاعُ ٤ ثوانٍ وفجوةُ ٧٠ انزلق — انقطاعٌ لا إيقاع");
    check(g70.gaps === c4.gaps, "٧ · W4 · الموضوعُ وضعاً عُدّ فاصلاً");
    const hop = L.noteFix(c20, { pos: north(30 * 6 + 2000), atMs: c20.at + 20000, counted: true, fallbackMs: 60000 });
    check(hop.glideMs === 0, "٧ · W4 · ٢ كم في ٢٠ ثانيةً انزلق فوق البيوت");
    const fast = L.noteFix(c4, { pos: north(30 * 6 + 200), atMs: c4.at + 4000, counted: true, fallbackMs: 4000 });
    check(fast.glideMs === 0, "٧ · W4 · ٢٠٠ م في ٤ ثوانٍ (١٨٠ كم/س) انزلق");
    const placed = L.retarget({ from: A, to: A, start: 0, dur: 0, fade: false }, B, 500, 0);
    check(placed.fade === true && L.positionAt(placed, 500) === B, "٧ · W4 · الموضوعُ لم يظهر في مكانه الجديد");
    check(L.opacityAt(placed, 500) < 1 && L.opacityAt(placed, 500 + L.FADE_MS) === 1, "٧ · W4 · الظهورُ ليس تلاشياً قصيراً");
  }

  // W5 · ٣ أمتار حدُّ الحركة
  {
    const base = { from: A, to: A, start: 0, dur: 0, fade: false };
    check(L.retarget(base, [A[0], A[1] + 2 / 111195], 100, 4000) === base, "٧ · W5 · تبدّلُ مترين حرّك العلامة");
    check(L.retarget(base, [A[0], A[1] + 4 / 111195], 100, 4000) !== base, "٧ · W5 · تبدّلُ أربعة أمتارٍ لم يحرّكها");
  }

  // W6 · «تقليلُ الحركة» لا يجمّد المواضع (والقديمُ كما كان)
  check(L.interpolates("glide", true) === true, "٧ · W6 · تقليلُ الحركة جمّد المواضع — كلُّ نقطةٍ قفزة");
  check(L.interpolates("tween", true) === false && L.interpolates("tween", false) === true,
    "٧ · W6 · الانزلاقُ القديمُ تغيّر مع المفتاح المطفأ");

  // W7 · ما لم يتبدّل لا يُرفع
  {
    const data = { type: "FeatureCollection", features: [] };
    check(L.needsUpload({ data, visible: true }, { data, visible: true }) === false, "٧ · W7 · طبقةٌ لم تتبدّل رُفعت");
    check(L.needsUpload({ data, visible: true }, { data: { ...data }, visible: true }) === true, "٧ · W7 · بياناتٌ جديدةٌ لم تُرفع");
    check(L.needsUpload({ data, visible: true }, { data, visible: false }) === true, "٧ · W7 · إخفاءٌ لم يُرفع");
    check(L.needsUpload(undefined, { data, visible: true }) === true, "٧ · W7 · أوّلُ رفعٍ لم يقع");
  }

  // W9 · إعادةُ مشوارٍ بلا طلب: نبضاتٌ بين ١٠ و٤٠ ثانية، ٣٠ كم/س ⇒ لا وضعَ، ولا خطوةَ فوق ٥ م بين إطارين
  {
    const gaps = [10, 25, 40, 15, 30, 35, 20, 40, 10, 28, 33, 12, 40, 22, 18, 37, 26, 14, 39, 31];
    const v = 30 / 3.6;
    const fixes = [];
    let at = t0;
    let dist = 0;
    fixes.push({ at, pos: north(0) });
    for (const g of gaps) {
      at += g * 1000;
      dist += v * g;
      fixes.push({ at, pos: north(dist) });
    }
    let c;
    let m;
    let k = 0;
    let placedN = 0;
    let maxStep = 0;
    let prev = null;
    for (let t = t0; t <= at + 60000; t += 16) {
      while (k < fixes.length && fixes[k].at <= t) {
        c = L.noteFix(c, { pos: fixes[k].pos, atMs: fixes[k].at, counted: true, fallbackMs: 20000 });
        if (k > 0 && c.glideMs === 0) placedN++;
        m = L.retarget(m, fixes[k].pos, t, c.glideMs);
        k++;
      }
      const p = L.positionAt(m, t);
      if (prev) maxStep = Math.max(maxStep, L.metersApart(prev, p));
      prev = p;
    }
    check(placedN === 0, `٧ · W9 · مشوارٌ عاديٌّ بلا طلبٍ وُضع ${placedN} مرّة`);
    check(maxStep <= 5, `٧ · W9 · خطوةٌ ${maxStep.toFixed(2)} م بين إطارين — قفزة`);
    check(L.metersApart(prev, fixes[fixes.length - 1].pos) < 0.01, "٧ · W9 · العلامةُ لم تبلغ آخرَ نقطة");
  }

  // ولقطةُ الطبقة، وخطُّ الطلب من العلامة، وسهمُ الاتّجاه
  {
    const moves = new Map([["a", { from: A, to: B, start: 0, dur: 4000, fade: false }], ["c", { from: B, to: B, start: 0, dur: 0, fade: true }]]);
    const fr = L.frameFeatures([pt("a", B[0], B[1], { _op: 0.6 }), pt("c", B[0], B[1], { _op: 0.6 }), pt("z", A[0], A[1])], moves, 2000);
    check(near(L.metersApart(A, fr[0].geometry.coordinates), 200, 0.5), "٧ · اللقطةُ ليست في منتصف الحركة");
    check(fr[0].properties._op === 0.6, "٧ · اللقطةُ غيّرت شفافيّةَ الباهت");
    check(near(L.positionAt(moves.get("c"), 100)[1], B[1]) && fr[1].properties._op === 0.6, "٧ · الظاهرُ بعد تلاشيه بغير شفافيّته");
    const fr2 = L.frameFeatures([pt("c", B[0], B[1], { _op: 0.6 })], moves, 100);
    check(fr2[0].properties._op < 0.6, "٧ · الموضوعُ للتوّ لم يتلاشَ");
    check(fr[2].geometry.coordinates[0] === A[0], "٧ · معلَمٌ بلا حركةٍ تحرّك");
    check(L.anyActive(moves, 1000) === true && L.anyActive(moves, 5000) === false, "٧ · «أيتحرّك أحد» خطأ");

    const legs = [
      { type: "Feature", geometry: { type: "LineString", coordinates: [A, B] }, properties: { from_driver: true } },
      { type: "Feature", geometry: { type: "LineString", coordinates: [B, A] }, properties: { from_driver: false } },
    ];
    const anchored = L.anchorLines(legs, north(10));
    check(anchored[0].geometry.coordinates[0][1] === north(10)[1] && anchored[0].geometry.coordinates[1] === B,
      "٧ · خطُّ الطلب لم يخرج من العلامة المنزلقة");
    check(anchored[1] === legs[1], "٧ · ساقُ المتجر ← الزبون تبعت السائق");
    check(L.anchorLines(legs, null) === legs, "٧ · بلا موضعٍ تبدّل الخطّ");

    check(L.headingOf(90, 3) === 90, "٧ · السهمُ غاب وهو ماشٍ");
    check(L.headingOf(90, 1) === null && L.headingOf(undefined, 5) === null && L.headingOf(400, 5) === null,
      "٧ · سهمٌ وهو واقفٌ أو بلا اتّجاهٍ صحيح");
  }
}

// ── ٨ · ترتيبُ المواضع بالزمن، وأحداثُ التيّار (٢٠٢٦-١١-٠٥) ──────────
//
// **جوابُ قائمةٍ قُرئ قبل نبضةٍ ووصل بعدها لا يُرجع السائقَ إلى الوراء** — «ما بدنا نطّات».
{
  const at = (s) => new Date(Date.UTC(2026, 10, 5, 9, 0, s)).toISOString();
  check(L.newerFix(undefined, at(6)) === true, "٨ · بلا سابقٍ لم يُطبَّق الموضع");
  check(L.newerFix(at(10), at(6)) === false, "٨ · موضعٌ أقدمُ من آخرِ ما طُبّق طُبّق");
  check(L.newerFix(at(10), at(12)) === true, "٨ · موضعٌ أحدثُ لم يُطبَّق");
  check(L.newerFix(at(10), at(10)) === true, "٨ · الموضعُ نفسُه (صفٌّ بعد نبضته) لم يُطبَّق");
  check(L.newerFix(at(10), "x") === false, "٨ · وقتٌ تالفٌ بعد سابقٍ طُبّق");
  check(L.newerFix("x", at(6)) === true, "٨ · سابقٌ تالفٌ منع موضعاً");

  // **نبضةٌ عند ١٠ ثمّ صفٌّ عند ٦ ⇒ العلامةُ لا تتحرّك؛ وصفٌّ عند ١٢ ⇒ يُطبَّق.**
  const book = new Map();
  const push = { lat: 35.95, lng: 39.01, last_location_at: at(10), age_sec: 0, freshness: "LIVE" };
  check(L.acceptPush(book, "d1", push) === true, "٨ · أوّلُ نبضةٍ رُدّت");
  const row = (s, lat, extra = {}) => ({
    id: "d1", lat, lng: 39.0, last_location_at: at(s), age_sec: 4, freshness: "LIVE", status: "active", ...extra,
  });
  const [r6] = L.mergeRows(book, [row(6, 35.9, { status: "busy" })]);
  check(r6.lat === 35.95 && r6.lng === 39.01 && r6.last_location_at === at(10),
    "٨ · صفٌّ قُرئ قبل النبضة أرجع السائقَ إلى الوراء");
  check(r6.age_sec === 0 && r6.freshness === "LIVE", "٨ · عمرُ الموضع لم يبقَ مع موضعه");
  check(r6.status === "busy", "٨ · ما ليس موضعاً في الصفّ لم يُطبَّق");
  const [r12] = L.mergeRows(book, [row(12, 35.96)]);
  check(r12.lat === 35.96 && book.get("d1").last_location_at === at(12), "٨ · صفٌّ أحدثُ من النبضة لم يُطبَّق");
  // **ونبضةٌ تأخّرت في الطريق بعد صفٍّ أحدث لا تُرجعه**، **وبلا وقتٍ لا تُرتَّب فلا تُطبَّق.**
  check(L.acceptPush(book, "d1", { ...push, lat: 35.0, last_location_at: at(11) }) === false,
    "٨ · نبضةٌ أقدمُ من الصفّ المطبَّق قُبلت");
  check(book.get("d1").lat === 35.96, "٨ · نبضةٌ مرفوضةٌ غيّرت الدفتر");
  check(L.acceptPush(book, "d1", { ...push, last_location_at: "" }) === false, "٨ · نبضةٌ بلا وقتٍ قُبلت");
  check(L.acceptPush(book, "d1", { ...push, lat: 35.97, last_location_at: at(16) }) === true,
    "٨ · نبضةٌ أحدثُ من الصفّ رُدّت");
  // **وصفُّ الموضع نفسِه يُحدّث عمرَه**، **وسائقٌ بلا سابقٍ يُطبَّق صفُّه كما هو.**
  const [same] = L.mergeRows(book, [row(16, 35.97, { age_sec: 40, freshness: "FRESH" })]);
  check(same.age_sec === 40 && same.freshness === "FRESH", "٨ · صفُّ الموضع نفسِه لم يُحدّث عمرَه");
  const [other] = L.mergeRows(book, [{ ...row(3, 35.5), id: "d2" }]);
  check(other.lat === 35.5 && book.get("d2")?.lat === 35.5, "٨ · سائقٌ بلا سابقٍ لم يُطبَّق صفُّه");

  // **وحدثُ التيّار لا يصير نوعاً يُعيد الجلب** (`liveStream.ts`).
  check(S.isStreamEvent("driver_location") === true, "٨ · موضعُ السائق صار نوعاً يُعيد جلبَ الصفحات");
  check(S.isStreamEvent("customer_location") === true, "٨ · موضعُ الزبون صار نوعاً يُعيد جلبَ الصفحات");
  check(S.isStreamEvent("order") === false && S.isStreamEvent("catalog") === false,
    "٨ · حدثُ طلبٍ أو لوحةٍ عُدّ تيّاراً فلا يُعيد جلبَ شيء");
}

// ── ٩ · معدّلُ الرفع (مراجعةُ ٢٠٢٦-١١-٠٦) ──────────────────────────
//
// **انزلاقٌ طويلٌ لا يُرفع ٣٠ مرّةً في الثانية** — سائقٌ بلا طلبٍ ينزلق حتّى ٤٥ ثانية، فتبقى
// الحلقةُ شبهَ دائمة على جهاز الموظف. **والقصيرُ (مع طلب) والظهورُ يبقيان ناعمَين.**
{
  const mo = (dur, fade = false, start = 0) => ({ from: [39, 35.9], to: [39.001, 35.9], start, dur, fade });
  const layer = (...ms) => new Map(ms.map((m, i) => [String(i), m]));
  check(typeof L.frameGap === "function" && L.frameGap([layer(mo(45000))], 1000) === L.SLOW_FRAME_MS,
    "٩ · انزلاقُ ٤٥ ثانيةً يُرفع بالمعدّل السريع");
  check(L.SLOW_FRAME_MS >= 100, "٩ · المعدّلُ البطيءُ أكثرُ من عشرٍ في الثانية");
  check(typeof L.frameGap === "function" && L.frameGap([layer(mo(45000), mo(4000))], 1000) === L.FRAME_MS,
    "٩ · انزلاقٌ قصيرٌ نزل إلى المعدّل البطيء");
  check(typeof L.frameGap === "function" && L.frameGap([layer(mo(45000)), layer(mo(0, true, 900))], 1000) === L.FRAME_MS,
    "٩ · الظهورُ نزل إلى المعدّل البطيء");
  check(typeof L.frameGap === "function" &&
    L.frameGap([layer(mo(45000)), layer(mo(4000, false, -9000))], 1000) === L.SLOW_FRAME_MS,
    "٩ · انزلاقٌ قصيرٌ انتهى أبقى المعدّلَ السريع");
  check(typeof L.frameGap === "function" && L.frameGap([], 1000) === L.SLOW_FRAME_MS, "٩ · بلا حركةٍ معدّلٌ سريع");
}

if (bad) {
  console.error(`\n**حسابُ الخريطة الحيّة مكسور** — ${bad} من ${cases}.`);
  process.exit(1);
}
console.log(`حسابُ الخريطة الحيّة — ${cases} حالةً قِيست بالنداء: انزلاقٌ · بحثٌ · عالقٌ جديد · تركيزٌ · خطُّ زمن · مسار · كلُّ سائقٍ وحدَه · ترتيبٌ بالزمن · تيّار.`);
