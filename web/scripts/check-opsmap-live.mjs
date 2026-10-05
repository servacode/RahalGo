/**
 * ══════════════════════════════════════════════════════════════════════
 * **حسابُ الخريطة الحيّة يُنادى ويُقاس** (تحسيناتُ خريطة العمليات ٢٠٢٦-١٠-٠٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الدوالُّ في `components/admin/opsmap/live.ts` خالصة** — فتُنادى هنا
 * بحالاتٍ وتُقاس نتائجُها: **الانزلاقُ** بين نبضتين · **البحثُ** في
 * المحمَّل · **«زاد العالقُ»** · **مرشِّحُ التركيز** · **خطُّ الزمن** ·
 * **خطُّ المسار.**
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

if (bad) {
  console.error(`\n**حسابُ الخريطة الحيّة مكسور** — ${bad} من ${cases}.`);
  process.exit(1);
}
console.log(`حسابُ الخريطة الحيّة — ${cases} حالةً قِيست بالنداء: انزلاقٌ · بحثٌ · عالقٌ جديد · تركيزٌ · خطُّ زمن · مسار.`);
