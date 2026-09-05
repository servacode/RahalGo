"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خريطةُ العمليات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أين الناسُ والطلباتُ والتغطيةُ الآن** — في شاشةٍ واحدة.
 *
 * # ولماذا شاشةٌ وقد صارت الجداولُ كلُّها موجودة
 *
 * **الجدولُ يجيب «كم»، والخريطةُ تجيب «أين»** — **وسؤالُ «أين» لا
 * يُجاب بجدول.** «سائقٌ على الدوام في الطرف الآخر من المدينة وطلبٌ
 * ينتظر هنا» **حقيقةٌ في القاعدة منذ شهور ولا شاشةَ تقولها.**
 *
 * # واللوحةُ الجانبيّةُ لا نافذةٌ منبثقة (البند ٣٧)
 *
 * **المنبثقةُ فوق الخريطة تحجب ما جاء ينظر إليه** — **واللوحةُ تُزيح
 * ولا تحجب**، ولا تصنع تمريراً أفقيّاً.
 *
 * # ولا تُعدَّل الأشياءُ كلُّها من هنا (البند ١٠)
 *
 * **الخريطةُ تقول وتوصّل** — ومن أراد أن يعدّل طلباً فتح صفحتَه.
 */

import { useCallback, useMemo, useState } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { getMessages, defaultLocale, fmtDateTime, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  Card,
  Button,
  EmptyState,
  LoadingState,
  Alert,
  Checkbox,
  Input,
  Select,
  useLiveData,
  themeColor,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import type { FeatureCollection, LayerSpec } from "@/components/admin/opsmap/canvas";

const OpsMapCanvas = dynamic(
  () => import("@/components/admin/opsmap/canvas").then((m) => m.OpsMapCanvas),
  { ssr: false },
);

const m = getMessages(defaultLocale);
const T = m.admin.opsMap;

// ══════════════════════════════════════════════════════════════════════
// **الأنواع**
// ══════════════════════════════════════════════════════════════════════

interface Meta {
  permissions: string[];
  location_ping_sec: number;
  assignable_sec: number;
}

interface Driver {
  id: string;
  name: string;
  lat: number;
  lng: number;
  status: string;
  on_shift: boolean;
  active_orders: number;
  current_order?: string;
  last_location_at: string;
  age_sec: number;
  freshness: "LIVE" | "FRESH" | "STALE" | "NO_LOCATION";
  cash_held?: number;
}

interface Merchant {
  id: string;
  name: string;
  lat: number;
  lng: number;
  status: string;
  open_now: boolean;
  city?: string;
  area?: string;
  active_orders: number;
  rep?: string;
}

interface OrderPin {
  id: string;
  number: number;
  status: string;
  kind: string;
  drop_lat: number;
  drop_lng: number;
  address?: string;
  merchant_id?: string;
  merchant?: string;
  pick_lat?: number;
  pick_lng?: number;
  driver_id?: string;
  driver?: string;
  driver_lat?: number;
  driver_lng?: number;
  created_at: string;
  accepted_at?: string;
  picked_up_at?: string;
  total?: number;
  payment_method?: string;
  delivery_fee?: number;
}

interface Zone {
  id: string;
  name: string;
  shape: "radius" | "polygon";
  active: boolean;
  lat: number;
  lng: number;
  radius_m: number;
  area?: { type: string; coordinates: unknown };
  delivery_fee: number;
  min_order: number;
  city?: string;
}

interface CovRequest {
  id: string;
  lat: number;
  lng: number;
  address: string;
  status: string;
  source: string;
  city?: string;
  note: string;
  created_at: string;
}

interface BranchPin {
  id: string;
  name: string;
  type: "primary" | "sub";
  status: string;
  city_id: string;
  city?: string;
  parent_id?: string;
  parent?: string;
  lat?: number;
  lng?: number;
  areas: number;
}

interface AreaRow {
  id: string;
  name: string;
  active: boolean;
  city_id: string;
  city?: string;
  branch_id?: string;
  branch?: string;
  zone_id?: string;
  zone?: string;
}

interface RepRow {
  id: string;
  name: string;
  merchants: number;
  converted: number;
  lat?: number;
  lng?: number;
  last_activity?: string;
  earnings?: number;
}

interface RepPoint {
  merchant_id: string;
  rep_id: string;
  lat: number;
  lng: number;
}

interface DemandCell {
  lat: number;
  lng: number;
  count: number;
}

interface DemandOut {
  orders: DemandCell[];
  requests: DemandCell[];
  merchants: DemandCell[];
  drivers: DemandCell[];
  unserved: DemandCell[];
  cell_deg: number;
}

interface OpportunityCell {
  lat: number;
  lng: number;
  orders: number;
  requests: number;
  merchants: number;
  drivers: number;
  score: number;
  reasons: string[];
}

interface SearchHit {
  kind: string;
  id: string;
  label: string;
  lat?: number;
  lng?: number;
}

/** **ما هو المُحدَّد؟** — واللوحةُ الجانبيّةُ واحدةٌ لكلّ الطبقات. */
type Picked =
  | { kind: "driver"; v: Driver }
  | { kind: "merchant"; v: Merchant }
  | { kind: "order"; v: OrderPin }
  | { kind: "zone"; v: Zone }
  | { kind: "request"; v: CovRequest }
  | { kind: "branch"; v: BranchPin }
  | { kind: "opportunity"; v: OpportunityCell };

/**
 * **ألوانُ الطزاجة — من الثيم لا من هنا.**
 *
 * **ولا لوحةَ ثانيةٌ تُولد** (وهي عينُ ما حُذف من `tokens.ts`): يُبدَّل
 * اللونُ في `theme.css` **فتتبعه الخريطةُ في التحديث نفسِه.**
 *
 * **والصفحةُ تُحمَّل `ssr: false`** — فالمتصفّحُ موجودٌ حين تُنادى.
 */
function freshColors(): Record<string, string> {
  return {
    LIVE: themeColor("success"),
    FRESH: themeColor("info"),
    STALE: themeColor("danger-solid"),
    NO_LOCATION: themeColor("disabled"),
  };
}

function fc(features: FeatureCollection["features"]): FeatureCollection {
  return { type: "FeatureCollection", features };
}

/** **معلَمُ نقطةٍ** — والخصائصُ نصوصٌ وأرقامٌ لا كائنات (MapLibre لا يمرّرها). */
function pt(lng: number, lat: number, props: Record<string, unknown>) {
  return {
    type: "Feature" as const,
    geometry: { type: "Point", coordinates: [lng, lat] },
    properties: props,
  };
}

export default function OpsMapPage() {
  // ── حالُ الواجهة (البند ٣٨) ──────────────────────────────────
  //
  // **ويُحفَظ ما يختاره المستخدم** — الطبقاتُ والمرشِّحات.
  // **ولا يُحفَظ ما يشيخ**: البياناتُ تُجلَب دائماً، **فلا تُعرض قديمةٌ
  // على أنّها حيّة.**
  const [visible, setVisible] = useState<Record<string, boolean>>(() => {
    if (typeof window === "undefined") return { drivers: true };
    try {
      const raw = window.localStorage.getItem("opsmap.layers");
      if (raw) return JSON.parse(raw) as Record<string, boolean>;
    } catch {
      /* تخزينٌ ممنوعٌ في نافذةٍ خاصّة — والافتراضُ يكفي */
    }
    return { drivers: true };
  });
  const toggle = useCallback((id: string) => {
    setVisible((prev) => {
      const next = { ...prev, [id]: !prev[id] };
      try {
        window.localStorage.setItem("opsmap.layers", JSON.stringify(next));
      } catch {
        /* لا يضرّ */
      }
      return next;
    });
  }, []);

  const [onShift, setOnShift] = useState<"" | "true" | "false">("");
  const [stale, setStale] = useState<"" | "true" | "false">("");
  const [hasActive, setHasActive] = useState<"" | "true" | "false">("");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Picked | null>(null);

  // ── محرِّرُ المضلَّعات (البند ١٣) ──────────────────────────
  //
  // **والرسمُ حالٌ صريحةٌ لا وضعٌ خفيّ** — **ومن نقر الأرضَ وهو لا
  // يرسم فتح معلَماً، ومن نقرها وهو يرسم أضاف نقطة.**
  const [drawing, setDrawing] = useState(false);
  const [draft, setDraft] = useState<[number, number][]>([]);
  const [zoneName, setZoneName] = useState("");
  const [saveErr, setSaveErr] = useState("");
  const [reqStatus, setReqStatus] = useState("");

  // ── المدى الزمنيُّ للتحليلات (البند ٢٩) ───────────────────
  //
  // **وثلاثون يوماً افتراضاً** — **ومدىً مفتوحٌ على كلّ التاريخ يقول
  // «هنا عملٌ» عن حيٍّ مات فيه العملُ منذ سنة.**
  const [range, setRange] = useState<"today" | "7d" | "30d">("30d");

  // ── البحثُ في الخريطة (البند ٣٦) ──────────────────────────
  //
  // **ويُقَصُّ في الخادم بصلاحيّات الباحث** — **ومن وجد اسمَ من لا
  // يملك رؤيتَه عرف أنّه موجود.**
  const [hunt, setHunt] = useState("");

  // ── ما يملكه من يقف أمامها ───────────────────────────────────
  const meta = useLiveData<Meta>(() => api<Meta>("/api/v1/admin/ops-map/meta"), []);
  const can = useCallback(
    (p: string) => !!meta.data?.permissions.includes(p),
    [meta.data],
  );

  // ── السائقون ─────────────────────────────────────────────────
  //
  // # ولماذا مدّةٌ لا بثّ (البندان ٨ و٤٤)
  //
  // **كاتبُ الموضع لا يبثّ** — ولا يُمسّ (البند ٤٥). **والمدّةُ ليست
  // اعتباطاً**: هي نبضةُ السائق نفسُها من الإعدادات، **فلا تُسأل
  // القاعدةُ عمّا لم يتغيّر بعد.**
  //
  // **والطلباتُ تُحدَّث ببثّ `ops` القائم** — لا بمدّة.
  const driverQuery = useMemo(() => {
    const q = new URLSearchParams();
    if (onShift) q.set("on_shift", onShift);
    if (stale) q.set("stale", stale);
    if (hasActive) q.set("has_active", hasActive);
    if (search.trim()) q.set("q", search.trim());
    const s = q.toString();
    return s ? `?${s}` : "";
  }, [onShift, stale, hasActive, search]);

  const drivers = useLiveData<{ drivers: Driver[]; count: number }>(
    () =>
      can("VIEW_DRIVER_LOCATIONS") && visible.drivers
        ? api<{ drivers: Driver[]; count: number }>(`/api/v1/admin/ops-map/drivers${driverQuery}`)
        : Promise.resolve({ drivers: [], count: 0 }),
    [],
    [driverQuery, visible.drivers, meta.data],
  );

  // ── المتاجر ──────────────────────────────────────────────────
  const merchants = useLiveData<{ merchants: Merchant[]; count: number }>(
    () =>
      can("VIEW_MERCHANT_LOCATIONS") && visible.merchants
        ? api<{ merchants: Merchant[]; count: number }>("/api/v1/admin/ops-map/merchants")
        : Promise.resolve({ merchants: [], count: 0 }),
    // **والمتجرُ يتبدّل بكتابةِ اللوحة** — `catalog` إشارتُها القائمة.
    ["catalog"],
    [visible.merchants, meta.data],
  );

  // ── الطلباتُ النشطة ─────────────────────────────────────────
  //
  // **وهذه وحدَها تُحدَّث ببثٍّ حقيقيّ** (البند ٤٤): غرفةُ `ops` تبثّ
  // `{"type":"order"}` عند كلّ تبدّل — **ولا مدّةَ ولا استطلاع.**
  const orders = useLiveData<{ orders: OrderPin[]; count: number }>(
    () =>
      can("VIEW_ACTIVE_ORDERS") && visible.orders
        ? api<{ orders: OrderPin[]; count: number }>("/api/v1/admin/ops-map/orders")
        : Promise.resolve({ orders: [], count: 0 }),
    ["order"],
    [visible.orders, meta.data],
  );

  // ── التغطية ─────────────────────────────────────────────────
  const zones = useLiveData<{ zones: Zone[]; count: number }>(
    () =>
      visible.coverage
        ? api<{ zones: Zone[]; count: number }>("/api/v1/admin/ops-map/coverage")
        : Promise.resolve({ zones: [], count: 0 }),
    ["catalog"],
    [visible.coverage, meta.data],
  );

  // ── طلباتُ التغطية ──────────────────────────────────────────
  const requests = useLiveData<{ requests: CovRequest[]; count: number }>(
    () =>
      can("VIEW_DEMAND_ANALYTICS") && visible.requests
        ? api<{ requests: CovRequest[]; count: number }>(
            `/api/v1/admin/ops-map/coverage-requests${reqStatus ? `?status=${reqStatus}` : ""}`,
          )
        : Promise.resolve({ requests: [], count: 0 }),
    [],
    [visible.requests, reqStatus, meta.data],
  );

  // ── الفروعُ والمناطقُ التشغيليّة ────────────────────────────
  //
  // **وهما جدولان لا يتبدّلان في الدقيقة** — فتُجلبان مع كتابةِ
  // اللوحة، ولا مدّةَ لهما.
  const branches = useLiveData<{ branches: BranchPin[]; count: number }>(
    () =>
      visible.branches
        ? api<{ branches: BranchPin[]; count: number }>("/api/v1/admin/ops-map/branches")
        : Promise.resolve({ branches: [], count: 0 }),
    ["catalog"],
    [visible.branches, meta.data],
  );

  const areas = useLiveData<{ areas: AreaRow[]; count: number }>(
    () =>
      visible.areas
        ? api<{ areas: AreaRow[]; count: number }>("/api/v1/admin/ops-map/areas")
        : Promise.resolve({ areas: [], count: 0 }),
    ["catalog"],
    [visible.areas, meta.data],
  );

  // ── المندوبون ───────────────────────────────────────────────
  const reps = useLiveData<{ reps: RepRow[]; points: RepPoint[] }>(
    () =>
      can("VIEW_REP_ACTIVITY") && visible.reps
        ? api<{ reps: RepRow[]; points: RepPoint[] }>(
            `/api/v1/admin/ops-map/reps?range=${range}`,
          )
        : Promise.resolve({ reps: [], points: [] }),
    [],
    [visible.reps, range, meta.data],
  );

  // ── الكثافةُ والفرص ─────────────────────────────────────────
  //
  // **ولقطةٌ لا بثّ** (البند ٤٤): **خريطةُ كثافةٍ كاملةٌ كلَّ ثانيةٍ
  // عبر القناة تُغرقها بلا فائدة** — والتحليلُ لا يتبدّل في الثانية.
  const demand = useLiveData<DemandOut>(
    () =>
      can("VIEW_DEMAND_ANALYTICS") && (visible.demand || visible.opportunities)
        ? api<DemandOut>(`/api/v1/admin/ops-map/demand?range=${range}`)
        : Promise.resolve({
            orders: [], requests: [], merchants: [], drivers: [],
            unserved: [], cell_deg: 0.01,
          }),
    [],
    [visible.demand, visible.opportunities, range, meta.data],
  );

  const opportunities = useLiveData<{ opportunities: OpportunityCell[]; count: number }>(
    () =>
      can("VIEW_DEMAND_ANALYTICS") && visible.opportunities
        ? api<{ opportunities: OpportunityCell[]; count: number }>(
            `/api/v1/admin/ops-map/opportunities?range=${range}`,
          )
        : Promise.resolve({ opportunities: [], count: 0 }),
    [],
    [visible.opportunities, range, meta.data],
  );

  const hits = useLiveData<{ hits: SearchHit[]; count: number }>(
    () =>
      hunt.trim().length >= 2
        ? api<{ hits: SearchHit[]; count: number }>(
            `/api/v1/admin/ops-map/search?q=${encodeURIComponent(hunt.trim())}`,
          )
        : Promise.resolve({ hits: [], count: 0 }),
    [],
    [hunt],
  );

  const layers = useMemo<LayerSpec[]>(() => {
    const out: LayerSpec[] = [];
    const C = freshColors();
    const list = drivers.data?.drivers ?? [];
    out.push({
      id: "drivers",
      kind: "point",
      cluster: true,
      order: 50,
      visible: !!visible.drivers,
      color: [
        "match",
        ["get", "freshness"],
        "LIVE", C.LIVE,
        "FRESH", C.FRESH,
        "STALE", C.STALE,
        C.NO_LOCATION,
      ],
      // **ومن لا موضعَ له لا يُرسَم** (البند ٤) — ولا يُخترَع له دبّوس.
      data: fc(
        list
          .filter((d) => d.freshness !== "NO_LOCATION")
          .map((d) =>
            pt(d.lng, d.lat, {
              id: d.id,
              name: d.name,
              freshness: d.freshness,
            }),
          ),
      ),
    });

    // ── المتاجر ──────────────────────────────────────────────
    //
    // **والمغلقُ يُرى مغلقاً** — لونٌ باهتٌ لا اختفاء: «متجرٌ مغلقٌ
    // وطلبٌ ينتظره» حقيقةٌ يريد المكتبُ أن يراها.
    out.push({
      id: "merchants",
      kind: "point",
      cluster: true,
      order: 40,
      visible: !!visible.merchants,
      color: ["case", ["get", "open"], themeColor("accent"), themeColor("disabled")],
      data: fc(
        (merchants.data?.merchants ?? []).map((x) =>
          pt(x.lng, x.lat, { id: x.id, name: x.name, open: x.open_now }),
        ),
      ),
    });

    // ── الطلباتُ النشطة ──────────────────────────────────────
    //
    // **وتُميَّز بصريّاً بحالها** (البند ١٠): ما لا سائقَ له أحمرُ —
    // **وهو ما يحتاج يداً الآن.**
    const orderList = orders.data?.orders ?? [];
    out.push({
      id: "orders",
      kind: "point",
      cluster: true,
      order: 60,
      visible: !!visible.orders,
      color: [
        "match",
        ["get", "state"],
        "unassigned", themeColor("danger-solid"),
        "picked", themeColor("success"),
        "accepted", themeColor("warning"),
        themeColor("violet"),
      ],
      data: fc(
        orderList.map((o) =>
          pt(o.drop_lng, o.drop_lat, {
            id: o.id,
            number: o.number,
            state: !o.driver_id
              ? "unassigned"
              : o.picked_up_at
                ? "picked"
                : o.accepted_at
                  ? "accepted"
                  : "other",
          }),
        ),
      ),
    });

    // ── التغطية ─────────────────────────────────────────────
    //
    // **والدائرةُ والمضلَّعُ طبقتان لا واحدة**: MapLibre ترسم الدائرةَ
    // بنصفِ قطرٍ بالبكسل والمضلَّعَ بحشوٍ — **ولا شكلَ واحدٌ يسعهما.**
    const zoneList = zones.data?.zones ?? [];
    out.push({
      id: "coverage-radius",
      kind: "circle-m",
      radiusField: "radius_m",
      order: 10,
      visible: !!visible.coverage,
      color: themeColor("primary"),
      data: fc(
        zoneList
          .filter((z) => z.shape === "radius" && z.active)
          .map((z) => pt(z.lng, z.lat, { id: z.id, name: z.name, radius_m: z.radius_m })),
      ),
    });
    out.push({
      id: "coverage-polygon",
      kind: "fill",
      order: 11,
      visible: !!visible.coverage,
      color: themeColor("accent"),
      data: fc(
        zoneList
          .filter((z) => z.shape === "polygon" && z.area && z.active)
          .map((z) => ({
            type: "Feature" as const,
            geometry: z.area as unknown,
            properties: { id: z.id, name: z.name },
          })),
      ),
    });

    // ── مسوَّدةُ الرسم ───────────────────────────────────────
    //
    // **وتُرى وهي تُرسَم** — **ومن رسم على العمياء رسم مرّتين.**
    out.push({
      id: "coverage-draft",
      kind: "line",
      order: 12,
      visible: draft.length > 1,
      color: themeColor("warning"),
      data: fc(
        draft.length > 1
          ? [
              {
                type: "Feature" as const,
                geometry: {
                  type: "LineString",
                  coordinates: [...draft, draft[0]],
                },
                properties: {},
              },
            ]
          : [],
      ),
    });

    // ── طلباتُ التغطية (البند ١٧) ───────────────────────────
    //
    // **ولا بياناتٍ شخصيّةٍ على الأرض** — نقطةٌ وحالُها.
    out.push({
      id: "requests",
      kind: "point",
      cluster: true,
      order: 30,
      visible: !!visible.requests,
      color: [
        "match",
        ["get", "state"],
        "new", themeColor("violet"),
        "planned", themeColor("info"),
        "covered", themeColor("success"),
        "rejected", themeColor("disabled"),
        themeColor("warning"),
      ],
      data: fc(
        (requests.data?.requests ?? []).map((r) =>
          pt(r.lng, r.lat, { id: r.id, state: r.status }),
        ),
      ),
    });

    // ── الفروع (البند ٢٣) ───────────────────────────────────
    //
    // **والرئيسيُّ يُميَّز عن الفرعيّ** — **ومن رأى عشرَ نقاطٍ متشابهةٍ
    // لم يعرف أيُّها مركزُ المدينة.**
    out.push({
      id: "branches",
      kind: "point",
      order: 20,
      visible: !!visible.branches,
      color: [
        "case",
        ["==", ["get", "kind"], "primary"],
        themeColor("primary"),
        themeColor("accent"),
      ],
      // **وفرعٌ بلا موقعٍ لا يُرسَم** — ويُقرأ في القائمة.
      data: fc(
        (branches.data?.branches ?? [])
          .filter((b) => b.lat != null && b.lng != null)
          .map((b) => pt(b.lng as number, b.lat as number, {
            id: b.id, name: b.name, kind: b.type,
          })),
      ),
    });

    // ── نشاطُ المندوبين (البند ٢٦) ──────────────────────────
    //
    // **ونقاطُ متاجرهم لا مساراتُ هواتفهم** — **ولا تتبّعَ لخطوات
    // إنسانٍ لم يطلبه أحد.**
    out.push({
      id: "reps",
      kind: "heat",
      order: 5,
      visible: !!visible.reps,
      color: themeColor("violet"),
      data: fc(
        (reps.data?.points ?? []).map((p) =>
          pt(p.lng, p.lat, { rep: p.rep_id }),
        ),
      ),
    });

    // ── الكثافتان — منفصلتان (البند ١٨) ─────────────────────
    //
    // **طلبٌ نُفِّذ غيرُ طلبِ تغطيةٍ لم يُغطَّ** — **الأوّلُ يقول «هنا
    // عملٌ نأخذه»، والثاني «هنا عملٌ لا نأخذه».**
    out.push({
      id: "demand-orders",
      kind: "heat",
      order: 3,
      weightField: "count",
      visible: !!visible.demand,
      color: themeColor("warning"),
      data: fc((demand.data?.orders ?? []).map((c) => pt(c.lng, c.lat, { count: c.count }))),
    });
    out.push({
      id: "demand-requests",
      kind: "heat",
      order: 4,
      weightField: "count",
      visible: !!visible.demand,
      color: themeColor("violet"),
      data: fc((demand.data?.requests ?? []).map((c) => pt(c.lng, c.lat, { count: c.count }))),
    });

    // ── فرصُ التوسّع (البند ٢٨) ─────────────────────────────
    //
    // **ونقطةٌ تُنقَر فتقول لماذا** — **ودرجةٌ بلا سببٍ رأيٌ يُباع
    // على أنّه حساب.**
    out.push({
      id: "opportunities",
      kind: "point",
      order: 80,
      visible: !!visible.opportunities,
      color: themeColor("danger-solid"),
      data: fc(
        (opportunities.data?.opportunities ?? []).map((o) =>
          pt(o.lng, o.lat, { lat: o.lat, lng: o.lng, score: o.score }),
        ),
      ),
    });

    // ── الربطُ الجغرافيُّ للطلب المُحدَّد (البند ١١) ─────────
    //
    // **وخطٌّ مستقيمٌ لا مسار** — **الخريطةُ ليست محرّكَ ملاحة**، وخطٌّ
    // يقول «من أين إلى أين ومن يحمله» يكفي المكتب.
    const link: FeatureCollection["features"] = [];
    if (selected?.kind === "order") {
      const o = selected.v;
      const line = (a: [number, number], b: [number, number]) => ({
        type: "Feature" as const,
        geometry: { type: "LineString", coordinates: [a, b] },
        properties: {},
      });
      if (o.pick_lng != null && o.pick_lat != null) {
        link.push(line([o.pick_lng, o.pick_lat], [o.drop_lng, o.drop_lat]));
        if (o.driver_lng != null && o.driver_lat != null) {
          link.push(line([o.driver_lng, o.driver_lat], [o.pick_lng, o.pick_lat]));
        }
      }
    }
    out.push({
      id: "order-link",
      kind: "line",
      order: 70,
      visible: link.length > 0,
      color: themeColor("primary"),
      data: fc(link),
    });

    return out;
  }, [drivers.data, merchants.data, orders.data, zones.data, requests.data,
      branches.data, reps.data, demand.data, opportunities.data,
      visible, selected, draft]);

  const onFeature = useCallback(
    (layerID: string, props: Record<string, unknown>) => {
      if (layerID === "drivers") {
        const d = drivers.data?.drivers.find((x) => x.id === props.id);
        if (d) setSelected({ kind: "driver", v: d });
      }
      if (layerID === "merchants") {
        const x = merchants.data?.merchants.find((y) => y.id === props.id);
        if (x) setSelected({ kind: "merchant", v: x });
      }
      if (layerID === "orders") {
        const o = orders.data?.orders.find((y) => y.id === props.id);
        if (o) setSelected({ kind: "order", v: o });
      }
      if (layerID === "coverage-radius" || layerID === "coverage-polygon") {
        const z = zones.data?.zones.find((y) => y.id === props.id);
        if (z) setSelected({ kind: "zone", v: z });
      }
      if (layerID === "requests") {
        const q = requests.data?.requests.find((y) => y.id === props.id);
        if (q) setSelected({ kind: "request", v: q });
      }
      if (layerID === "branches") {
        const b = branches.data?.branches.find((y) => y.id === props.id);
        if (b) setSelected({ kind: "branch", v: b });
      }
      if (layerID === "opportunities") {
        const o = (opportunities.data?.opportunities ?? []).find(
          (y) => y.lat === props.lat && y.lng === props.lng,
        );
        if (o) setSelected({ kind: "opportunity", v: o });
      }
    },
    [drivers.data, merchants.data, orders.data, zones.data, requests.data,
     branches.data, opportunities.data],
  );

  // ── نقرُ الأرض ───────────────────────────────────────────────
  const onGround = useCallback(
    (lng: number, lat: number) => {
      if (drawing) setDraft((prev) => [...prev, [lng, lat]]);
    },
    [drawing],
  );

  const savePolygon = useCallback(async () => {
    setSaveErr("");
    try {
      await api("/api/v1/admin/ops-map/coverage", {
        method: "POST",
        body: JSON.stringify({
          name: zoneName.trim(), ring: draft, delivery_fee: 0, min_order: 0,
        }),
      });
      setDrawing(false);
      setDraft([]);
      setZoneName("");
      zones.reload();
    } catch (e) {
      setSaveErr(errorText(e));
    }
  }, [zoneName, draft, zones]);

  const setZoneActive = useCallback(
    async (id: string, active: boolean) => {
      await api(`/api/v1/admin/ops-map/coverage/${id}/active`, {
        method: "POST",
        body: JSON.stringify({ active }),
      });
      zones.reload();
      setSelected(null);
    },
    [zones],
  );

  const setRequestStatus = useCallback(
    async (id: string, status: string) => {
      await api(`/api/v1/admin/ops-map/coverage-requests/${id}`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      });
      requests.reload();
      setSelected(null);
    },
    [requests],
  );

  // ── الحالاتُ الحدّيّة (البند ٣٩) ─────────────────────────────
  if (meta.loading) return <LoadingState />;
  if (meta.error) {
    return (
      <PageContainer>
        <Alert tone="error">{T.networkError}</Alert>
        <Button onClick={meta.reload}>{T.retry}</Button>
      </PageContainer>
    );
  }
  if (!can("VIEW_OPERATIONS_MAP")) {
    return (
      <PageContainer>
        <EmptyState title={T.noPermission} />
      </PageContainer>
    );
  }

  const noLoc = (drivers.data?.drivers ?? []).filter((d) => d.freshness === "NO_LOCATION");

  return (
    <PageContainer>
      <PageHeader title={T.title} subtitle={T.subtitle} />

      <div className="grid gap-4 lg:grid-cols-[18rem_1fr]">
        {/* ── الطبقاتُ والمرشِّحات ────────────────────────────── */}
        <div className="flex flex-col gap-4">
          {/* ── البحث (البند ٣٦) ───────────────────────────────── */}
          <Card>
            <h3 className="mb-2 text-sm font-bold">{T.search}</h3>
            <Input
              placeholder={T.search}
              value={hunt}
              onChange={(e) => setHunt(e.target.value)}
            />
            {hunt.trim().length >= 2 && (
              <ul className="mt-2 flex flex-col gap-1 text-sm">
                {(hits.data?.hits ?? []).length === 0 ? (
                  <li className="text-xs text-ink-muted">{T.searchNone}</li>
                ) : (
                  (hits.data?.hits ?? []).slice(0, 12).map((x) => (
                    <li key={`${x.kind}-${x.id}`} className="flex items-center gap-2">
                      <span>{x.label}</span>
                      <span className="ms-auto text-xs text-ink-muted">
                        {T.layer[
                          (x.kind === "driver"
                            ? "drivers"
                            : x.kind === "merchant"
                              ? "merchants"
                              : x.kind === "order"
                                ? "orders"
                                : x.kind === "branch"
                                  ? "branches"
                                  : x.kind === "area"
                                    ? "areas"
                                    : "reps") as keyof typeof T.layer
                        ]}
                      </span>
                    </li>
                  ))
                )}
              </ul>
            )}
          </Card>

          <Card>
            <h3 className="mb-2 text-sm font-bold">{T.layers}</h3>
            <div className="flex flex-col gap-1">
              {can("VIEW_DRIVER_LOCATIONS") && (
                <LayerToggle
                  id="drivers"
                  label={T.layer.drivers}
                  on={!!visible.drivers}
                  count={drivers.data?.count}
                  onToggle={toggle}
                />
              )}
              {can("VIEW_MERCHANT_LOCATIONS") && (
                <LayerToggle
                  id="merchants"
                  label={T.layer.merchants}
                  on={!!visible.merchants}
                  count={merchants.data?.count}
                  onToggle={toggle}
                />
              )}
              {can("VIEW_ACTIVE_ORDERS") && (
                <LayerToggle
                  id="orders"
                  label={T.layer.orders}
                  on={!!visible.orders}
                  count={orders.data?.count}
                  onToggle={toggle}
                />
              )}
              <LayerToggle
                id="coverage"
                label={T.layer.coverage}
                on={!!visible.coverage}
                count={zones.data?.count}
                onToggle={toggle}
              />
              {can("VIEW_DEMAND_ANALYTICS") && (
                <LayerToggle
                  id="requests"
                  label={T.layer.coverageRequests}
                  on={!!visible.requests}
                  count={requests.data?.count}
                  onToggle={toggle}
                />
              )}
              <LayerToggle
                id="branches"
                label={T.layer.branches}
                on={!!visible.branches}
                count={branches.data?.count}
                onToggle={toggle}
              />
              <LayerToggle
                id="areas"
                label={T.layer.areas}
                on={!!visible.areas}
                count={areas.data?.count}
                onToggle={toggle}
              />
              {can("VIEW_REP_ACTIVITY") && (
                <LayerToggle
                  id="reps"
                  label={T.layer.reps}
                  on={!!visible.reps}
                  count={reps.data?.reps.length}
                  onToggle={toggle}
                />
              )}
              {can("VIEW_DEMAND_ANALYTICS") && (
                <>
                  <LayerToggle
                    id="demand"
                    label={T.layer.demand}
                    on={!!visible.demand}
                    count={demand.data?.orders.length}
                    onToggle={toggle}
                  />
                  <LayerToggle
                    id="opportunities"
                    label={T.layer.opportunities}
                    on={!!visible.opportunities}
                    count={opportunities.data?.count}
                    onToggle={toggle}
                  />
                </>
              )}
            </div>
          </Card>

          {can("VIEW_DRIVER_LOCATIONS") && visible.drivers && (
            <Card>
              <h3 className="mb-2 text-sm font-bold">{T.filters}</h3>
              <div className="flex flex-col gap-2 text-sm">
                <Input
                  placeholder={T.search}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
                <Tri label={T.filter.onShift} value={onShift} onChange={setOnShift} />
                <Tri label={T.filter.hasActive} value={hasActive} onChange={setHasActive} />
                <Tri label={T.filter.stale} value={stale} onChange={setStale} />
              </div>
              <p className="mt-3 text-xs leading-5 text-ink-muted">
                {T.freshness.explain}
              </p>
            </Card>
          )}

          {/* ── محرِّرُ التغطية (البند ١٣) ────────────────────────
              **ولا يظهر لمن لا يملكه** — ورؤيةُ زرٍّ يردّ `403` أسوأُ
              من غيابه. */}
          {can("MANAGE_COVERAGE") && visible.coverage && (
            <Card>
              <h3 className="mb-2 text-sm font-bold">{T.coverage.title}</h3>
              {!drawing ? (
                <Button variant="secondary" onClick={() => { setDrawing(true); setDraft([]); }}>
                  {T.coverage.draw}
                </Button>
              ) : (
                <div className="flex flex-col gap-2 text-sm">
                  <Input
                    placeholder={T.coverage.name}
                    value={zoneName}
                    onChange={(e) => setZoneName(e.target.value)}
                  />
                  <p className="text-xs text-ink-muted">
                    {draft.length} · {T.coverage.needThree}
                  </p>
                  {saveErr && <Alert tone="error">{saveErr}</Alert>}
                  <div className="flex flex-wrap gap-2">
                    <Button
                      disabled={draft.length < 3 || !zoneName.trim()}
                      onClick={savePolygon}
                    >
                      {T.coverage.save}
                    </Button>
                    <Button
                      variant="secondary"
                      disabled={draft.length === 0}
                      onClick={() => setDraft((d) => d.slice(0, -1))}
                    >
                      {T.coverage.undo}
                    </Button>
                    <Button
                      variant="secondary"
                      onClick={() => { setDrawing(false); setDraft([]); setSaveErr(""); }}
                    >
                      {T.coverage.cancel}
                    </Button>
                  </div>
                </div>
              )}
              <p className="mt-3 text-xs leading-5 text-ink-muted">
                {T.coverage.legacyHint}
              </p>
            </Card>
          )}

          {/* ── مرشِّحُ طلبات التغطية ──────────────────────────── */}
          {can("VIEW_DEMAND_ANALYTICS") && visible.requests && (
            <Card>
              <h3 className="mb-2 text-sm font-bold">{T.layer.coverageRequests}</h3>
              <Select
                label={T.filter.requestStatus}
                value={reqStatus}
                onChange={(e) => setReqStatus(e.target.value)}
              >
                <option value="">{T.all}</option>
                <option value="new">{T.request.state.new}</option>
                <option value="reviewing">{T.request.state.reviewing}</option>
                <option value="planned">{T.request.state.planned}</option>
                <option value="covered">{T.request.state.covered}</option>
                <option value="rejected">{T.request.state.rejected}</option>
              </Select>
            </Card>
          )}

          {/* ── المدى الزمنيّ (البند ٢٩) ─────────────────────────
              **ويخصُّ التحليلاتِ ونشاطَ المندوبين وحدَها** — **والسائقون
              والطلباتُ «الآن» لا مدّةَ لها.** */}
          {can("VIEW_DEMAND_ANALYTICS") &&
            (visible.demand || visible.opportunities || visible.reps) && (
            <Card>
              <h3 className="mb-2 text-sm font-bold">{T.filter.range}</h3>
              <Select
                label={T.filter.range}
                value={range}
                onChange={(e) => setRange(e.target.value as "today" | "7d" | "30d")}
              >
                <option value="today">{T.filter.today}</option>
                <option value="7d">{T.filter.d7}</option>
                <option value="30d">{T.filter.d30}</option>
              </Select>
              <p className="mt-3 text-xs leading-5 text-ink-muted">{T.demand.separate}</p>
            </Card>
          )}

          {/* ── المندوبون (البند ٢٦) ─────────────────────────── */}
          {can("VIEW_REP_ACTIVITY") && visible.reps && (
            <Card>
              <h3 className="mb-1 text-sm font-bold">{T.layer.reps}</h3>
              <p className="mb-2 text-xs text-ink-muted">{T.rep.noTerritory}</p>
              <ul className="flex flex-col gap-1 text-sm">
                {(reps.data?.reps ?? []).slice(0, 15).map((r) => (
                  <li key={r.id} className="flex items-center gap-2">
                    <span>{r.name}</span>
                    <span className="ms-auto text-xs text-ink-muted">
                      {r.merchants} · {r.converted}
                    </span>
                  </li>
                ))}
              </ul>
            </Card>
          )}

          {/* ── المناطقُ التشغيليّة ──────────────────────────────
              **ولا هندسةَ لها بعد** — **فتُقرأ قائمةً ولا تُرسَم على
              الأرض.** ومن ربطها بمنطقةِ تغطيةٍ رأى شكلَها في طبقة
              التغطية، **ولا يُخترَع لها مضلَّعٌ لتبدو مرسومة.** */}
          {visible.areas && (
            <Card>
              <h3 className="mb-1 text-sm font-bold">{T.layer.areas}</h3>
              <p className="mb-2 text-xs text-ink-muted">{T.area.notDistrict}</p>
              {(areas.data?.areas ?? []).length === 0 ? (
                <p className="text-xs text-ink-muted">{T.empty}</p>
              ) : (
                <ul className="flex flex-col gap-1 text-sm">
                  {(areas.data?.areas ?? []).slice(0, 20).map((a) => (
                    <li key={a.id} className="flex items-center gap-2">
                      <span>{a.name}</span>
                      <span className="ms-auto text-xs text-ink-muted">
                        {a.branch ?? "—"}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </Card>
          )}

          {/* **ومن لا موضعَ له يُقال ولا يُرسَم** — وهو أهمُّ ما تقوله
              الخريطة: تطبيقٌ أوقفه النظامُ وورديّةٌ مفتوحة. */}
          {noLoc.length > 0 && (
            <Card>
              <h3 className="mb-1 text-sm font-bold">{T.freshness.NO_LOCATION}</h3>
              <p className="mb-2 text-xs text-ink-muted">{T.driver.noLocationHint}</p>
              <ul className="flex flex-col gap-1 text-sm">
                {noLoc.slice(0, 12).map((d) => (
                  <li key={d.id} className="flex items-center gap-2">
                    <span
                      className="inline-block h-2 w-2 rounded-full"
                      style={{ background: themeColor("disabled") }}
                    />
                    <span>{d.name}</span>
                    {d.on_shift && (
                      <span className="ms-auto text-xs text-ink-muted">
                        {T.driver.onShift}
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            </Card>
          )}
        </div>

        {/* ── الخريطةُ واللوحةُ الجانبيّة ─────────────────────── */}
        <div className="relative min-h-[28rem] lg:min-h-[calc(100vh-14rem)]">
          <OpsMapCanvas
            layers={layers}
            onFeatureClick={onFeature}
            onMapClick={onGround}
            unavailableLabel={T.unavailable}
          />
          {selected && (
            <aside
              className="absolute inset-y-0 end-0 z-10 w-full max-w-sm overflow-y-auto
                         border-s border-line-soft bg-surface p-4 shadow-xl"
            >
              <div className="mb-3 flex items-center gap-2">
                <h3 className="text-base font-bold">
                  {selected.kind === "driver" && selected.v.name}
                  {selected.kind === "merchant" && selected.v.name}
                  {selected.kind === "order" && `#${selected.v.number}`}
                  {selected.kind === "zone" && selected.v.name}
                  {selected.kind === "request" && T.request.title}
                  {selected.kind === "branch" && selected.v.name}
                  {selected.kind === "opportunity" && T.opportunity.title}
                </h3>
                <button
                  className="ms-auto text-sm text-ink-muted"
                  onClick={() => setSelected(null)}
                >
                  {T.close}
                </button>
              </div>

              {selected.kind === "driver" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.driver.status} v={selected.v.status} />
                    <Row k={T.driver.onShift} v={selected.v.on_shift ? T.yes : T.no} />
                    <Row k={T.driver.activeOrders} v={String(selected.v.active_orders)} />
                    <Row
                      k={T.driver.lastSeen}
                      v={selected.v.last_location_at ? fmtDateTime(selected.v.last_location_at) : "—"}
                    />
                    <Row k={T.live} v={T.freshness[selected.v.freshness]} />
                    {/* **والمالُ لمن يملك صلاحيّتَه وحدَه** (البند ٦). */}
                    {selected.v.cash_held !== undefined && (
                      <Row k={T.driver.cashHeld} v={fmtMoney(selected.v.cash_held)} />
                    )}
                  </dl>
                  <div className="mt-4 flex flex-wrap gap-2">
                    <Link href={`/dashboard/users?id=${selected.v.id}`}>
                      <Button variant="secondary">{T.driver.openProfile}</Button>
                    </Link>
                    {selected.v.current_order && (
                      <Link href={`/dashboard/orders?id=${selected.v.current_order}`}>
                        <Button variant="secondary">{T.order.openPage}</Button>
                      </Link>
                    )}
                  </div>
                </>
              )}

              {selected.kind === "merchant" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.merchant.status} v={selected.v.status} />
                    <Row
                      k={T.merchant.open}
                      v={selected.v.open_now ? T.merchant.open : T.merchant.closed}
                    />
                    {selected.v.city && <Row k={T.merchant.city} v={selected.v.city} />}
                    <Row k={T.merchant.activeOrders} v={String(selected.v.active_orders)} />
                    {/* **والمندوبُ لمن يراقب نشاطَهم وحدَه** (البند ٩). */}
                    {selected.v.rep && <Row k={T.merchant.rep} v={selected.v.rep} />}
                  </dl>
                  <div className="mt-4">
                    <Link href={`/dashboard/sections?merchant=${selected.v.id}`}>
                      <Button variant="secondary">{T.merchant.openPage}</Button>
                    </Link>
                  </div>
                </>
              )}

              {selected.kind === "order" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.order.state} v={selected.v.status} />
                    <Row k={T.order.merchant} v={selected.v.merchant ?? "—"} />
                    <Row k={T.order.driver} v={selected.v.driver ?? T.order.noDriver} />
                    {selected.v.address && <Row k={T.order.area} v={selected.v.address} />}
                    <Row k={T.order.createdAt} v={fmtDateTime(selected.v.created_at)} />
                    {/* **والمالُ بصلاحيّته** (البند ١٠). */}
                    {selected.v.total !== undefined && (
                      <Row k={T.order.total} v={fmtMoney(selected.v.total)} />
                    )}
                    {selected.v.delivery_fee !== undefined && (
                      <Row k={T.order.delivery} v={fmtMoney(selected.v.delivery_fee)} />
                    )}
                    {selected.v.payment_method && (
                      <Row k={T.order.payment} v={selected.v.payment_method} />
                    )}
                  </dl>
                  {/* **ولا يُعدّل الطلبُ من الخريطة** (البند ١٠) —
                      تُوصّل إلى صفحته. */}
                  <div className="mt-4 flex flex-wrap gap-2">
                    <Link href={`/dashboard/orders?id=${selected.v.id}`}>
                      <Button variant="secondary">{T.order.openPage}</Button>
                    </Link>
                    {selected.v.driver_id && (
                      <Link href={`/dashboard/users?id=${selected.v.driver_id}`}>
                        <Button variant="secondary">{T.driver.openProfile}</Button>
                      </Link>
                    )}
                  </div>
                </>
              )}
              {selected.kind === "zone" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row
                      k={T.coverage.shape}
                      v={selected.v.shape === "polygon" ? T.coverage.polygon : T.coverage.radius}
                    />
                    <Row k={T.coverage.active} v={selected.v.active ? T.yes : T.no} />
                    {selected.v.city && <Row k={T.coverage.city} v={selected.v.city} />}
                    {selected.v.shape === "radius" && (
                      <Row k={T.coverage.radius} v={`${selected.v.radius_m} ${T.coverage.metres}`} />
                    )}
                  </dl>
                  {/* **والمنطقةُ تُوقَف ولا تُحذف** — منطقةٌ حُذفت تترك
                      طلباتٍ تشير إلى عدم. */}
                  {can("MANAGE_COVERAGE") && (
                    <div className="mt-4 flex flex-wrap gap-2">
                      <Button
                        variant="secondary"
                        onClick={() => setZoneActive(selected.v.id, !selected.v.active)}
                      >
                        {selected.v.active ? T.coverage.disable : T.coverage.enable}
                      </Button>
                    </div>
                  )}
                  <p className="mt-3 text-xs text-ink-muted">{T.coverage.deleteHint}</p>
                </>
              )}

              {selected.kind === "request" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row
                      k={T.request.status}
                      v={T.request.state[selected.v.status as keyof typeof T.request.state]}
                    />
                    <Row k={T.request.createdAt} v={fmtDateTime(selected.v.created_at)} />
                    {selected.v.address && <Row k={T.request.address} v={selected.v.address} />}
                    {selected.v.city && <Row k={T.request.city} v={selected.v.city} />}
                    <Row k={T.request.source} v={selected.v.source} />
                    {selected.v.note && <Row k={T.request.note} v={selected.v.note} />}
                  </dl>
                  {can("MANAGE_COVERAGE") && (
                    <div className="mt-4">
                      <Select
                        label={T.request.changeStatus}
                        value={selected.v.status}
                        onChange={(e) => setRequestStatus(selected.v.id, e.target.value)}
                      >
                        <option value="new">{T.request.state.new}</option>
                        <option value="reviewing">{T.request.state.reviewing}</option>
                        <option value="planned">{T.request.state.planned}</option>
                        <option value="covered">{T.request.state.covered}</option>
                        <option value="rejected">{T.request.state.rejected}</option>
                      </Select>
                    </div>
                  )}
                </>
              )}
              {selected.kind === "branch" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row
                      k={T.branch.type}
                      v={selected.v.type === "primary" ? T.branch.primary : T.branch.sub}
                    />
                    <Row k={T.branch.status} v={selected.v.status} />
                    {selected.v.city && <Row k={T.branch.city} v={selected.v.city} />}
                    {selected.v.parent && <Row k={T.branch.parent} v={selected.v.parent} />}
                    <Row k={T.layer.areas} v={String(selected.v.areas)} />
                  </dl>
                  <p className="mt-3 text-xs text-ink-muted">{T.branch.onePrimary}</p>
                </>
              )}
              {selected.kind === "opportunity" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.opportunity.score} v={String(selected.v.score)} />
                    <Row k={T.layer.orders} v={String(selected.v.orders)} />
                    <Row k={T.request.count} v={String(selected.v.requests)} />
                    <Row k={T.layer.merchants} v={String(selected.v.merchants)} />
                    <Row k={T.layer.drivers} v={String(selected.v.drivers)} />
                  </dl>
                  {/* **ولا درجةَ بلا سببٍ مسمّى** — والأرقامُ الخامُّ
                      فوقها، فمن لم يقبل الوزنَ حسب بنفسه. */}
                  <ul className="mt-3 flex flex-col gap-1 text-xs text-ink-muted">
                    {selected.v.reasons.map((why) => (
                      <li key={why}>
                        {why === "high_demand_low_coverage" && T.opportunity.highDemandLowCoverage}
                        {why === "high_demand_low_drivers" && T.opportunity.highDemandLowDrivers}
                        {why === "many_requests" && T.opportunity.manyRequests}
                        {why === "merchants_no_drivers" && T.opportunity.merchantsNoDrivers}
                      </li>
                    ))}
                  </ul>
                  <p className="mt-3 text-xs text-ink-muted">{T.opportunity.explain}</p>
                </>
              )}
            </aside>
          )}
        </div>
      </div>
    </PageContainer>
  );
}

/** **مفتاحُ طبقةٍ وعدّادُها** — والعددُ يقول «هل ثمّة شيءٌ أصلاً». */
function LayerToggle({
  id,
  label,
  on,
  count,
  onToggle,
}: {
  id: string;
  label: string;
  on: boolean;
  count?: number;
  onToggle: (id: string) => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <Checkbox id={`layer-${id}`} label={label} checked={on} onChange={() => onToggle(id)} />
      <span className="ms-auto text-xs text-ink-muted">{count ?? 0}</span>
    </div>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-baseline gap-2">
      <dt className="text-ink-muted">{k}</dt>
      <dd className="ms-auto font-medium">{v}</dd>
    </div>
  );
}

/** **مرشِّحٌ ثلاثيّ** — والفراغُ «الكلّ» لا «لا». */
function Tri({
  label,
  value,
  onChange,
}: {
  label: string;
  value: "" | "true" | "false";
  onChange: (v: "" | "true" | "false") => void;
}) {
  return (
    <Select
      label={label}
      value={value}
      onChange={(e) => onChange(e.target.value as "" | "true" | "false")}
    >
      <option value="">{T.all}</option>
      <option value="true">{T.yes}</option>
      <option value="false">{T.no}</option>
    </Select>
  );
}
