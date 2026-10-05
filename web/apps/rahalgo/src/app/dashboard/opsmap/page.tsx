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
 * # وشريطٌ علويٌّ لا صندوقُ مفاتيح (قرارُ المالك ٢٠٢٦-١٠-٠٥)
 *
 * **عدّاداتٌ حيّةٌ** («N سائق متاح · N طلب ماشي · N عالق · N متجر مفتوح»)،
 * **وحبّاتٌ للطبقات الستّ الرئيسيّة** بعددها ولونها، **وما سواها في «طبقات
 * إضافية»**. **ولونٌ ورمزٌ لكلّ نوعٍ وحالٍ من موضعٍ واحد** (`palette.ts`) —
 * الخريطةُ والحبّةُ والدليلُ والبطاقةُ تقرؤه نفسَه.
 *
 * # واللوحةُ الجانبيّةُ لا نافذةٌ منبثقة (البند ٣٧)
 *
 * **المنبثقةُ فوق الخريطة تحجب ما جاء ينظر إليه** — **واللوحةُ تُزيح
 * ولا تحجب**، ولا تصنع تمريراً أفقيّاً.
 *
 * # ولا تتحرّك الكاميرا وحدَها
 *
 * **إلّا بنقرةٍ صريحة** — رقمُ «عالق» يطير إلى أوّل عالقٍ ويفتح بطاقتَه،
 * واختيارُ نتيجة بحثٍ يطير إليها، و«رجّع على المكتب» يطير إليه.
 *
 * # والحيُّ يُرى حيّاً (تحسيناتُ المالك ٢٠٢٦-١٠-٠٦)
 *
 * **السائقُ ينزلق بين نبضتين ولا يقفز**، **وحلقةٌ نابضةٌ حولَ من موقعُه حيّ**،
 * **وعالقٌ جديدٌ يُومض عدّادُه ويرنّ جرسُ اللوحة** (بمفتاح الصوت نفسِه).
 * **والحسابُ كلُّه في `opsmap/live.ts`** — دوالُّ خالصةٌ يقيسها حارس.
 */

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtDateTime, fmtMoney, fmtNum, errorText } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  Card,
  Button,
  EmptyState,
  LoadingState,
  Alert,
  Checkbox,
  Chips,
  CountBadge,
  Input,
  Select,
  IconPhone,
  IconFullscreen,
  IconFullscreenExit,
  IconSearch,
  IconTarget,
  IconWhatsApp,
  useLiveData,
  useChime,
  alertsSoundOn,
  themeColor,
} from "@rahalgo/ui";
import PlaceDemandPanel from "@/components/admin/PlaceDemandPanel";
import { AssignDialog } from "@/components/admin/orders/AssignDialog";
import { api, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";
import { roleLabelByCode } from "@/lib/rolemeta";
import type {
  FeatureCollection,
  FlyRequest,
  LayerSpec,
  MapMarker,
} from "@/components/admin/opsmap/canvas";
import {
  MAP_TONES,
  type MapTone,
  driverTone,
  mapIcons,
  orderTone,
  storeTone,
  toneColor,
} from "@/components/admin/opsmap/palette";
import {
  type FocusMode,
  type LngLat,
  applyFocus,
  orderRoute,
  orderTimeline,
  searchLocal,
  stuckIncreased,
} from "@/components/admin/opsmap/live";

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
  /** **حالُه في لون** — من الخادم (`opsmap.DriverTone`). */
  tone: "available" | "busy" | "idle";
  /** **لمن يملك `users.contact.read` وحدَه.** */
  phone?: string;
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
  phone?: string;
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
  /** **آخرُ إسنادٍ إلى سائق** — من سجلّ الحالات. */
  assigned_at?: string;
  picked_up_at?: string;
  total?: number;
  payment_method?: string;
  delivery_fee?: number;
  /** **سببُ العلوق** — بشرط لوحة الطلبات. */
  stuck_reason?: string;
  /** **لمن يملك `orders.customer_details.read` وحدَه.** */
  customer_name?: string;
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

/** **طارئٌ مفتوحٌ على الخريطة** — غرفةُ الطوارئ (٢٠٢٦-١٠-٠٤). */
interface EmergencyPin {
  id: string;
  kind: string;
  lat: number;
  lng: number;
  label: string;
  order_number: number | null;
  stale: boolean;
}

/** **عدّاداتُ الشريط** — وكلُّ قسمٍ يغيب لمن لا يملك طبقتَه. */
interface Summary {
  drivers?: { total: number; available: number; busy: number; idle: number };
  orders?: { active: number; stuck: number; first_stuck: OrderPin | null };
  merchants?: { total: number; open: number };
  customers?: { total: number; cells: number };
  reps?: number;
  staff_online: number;
}

/** **المكتب** — شعارُ المنصّة في موضعها ومن في اللوحة الآن. */
interface Office {
  name: string;
  address: string;
  logo: string | null;
  has_location: boolean;
  lat?: number;
  lng?: number;
  presence_min: number;
  staff: Array<{ name: string; roles: string[] }>;
}

/** **ما هو المُحدَّد؟** — واللوحةُ الجانبيّةُ واحدةٌ لكلّ الطبقات. */
type Picked =
  | { kind: "driver"; v: Driver }
  | { kind: "merchant"; v: Merchant }
  | { kind: "order"; v: OrderPin }
  | { kind: "zone"; v: Zone }
  | { kind: "request"; v: CovRequest }
  | { kind: "branch"; v: BranchPin }
  | { kind: "opportunity"; v: OpportunityCell }
  | { kind: "customers"; v: DemandCell }
  | { kind: "rep"; v: RepRow }
  | { kind: "office" };

/** **الطبقاتُ الستُّ في الشريط — بترتيب المالك.** */
type MainLayer = "drivers" | "orders" | "merchants" | "customers" | "reps" | "office";

/** **ما يُرى أوّلَ فتحة** — والمستخدمُ يبدّله فيُحفظ له. */
const DEFAULT_VISIBLE: Record<string, boolean> = {
  drivers: true,
  orders: true,
  merchants: true,
  office: true,
};

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

/** **رابطُ واتساب** — أرقامٌ فقط، والمحلّيُّ (`09…`) يُكمَّل برمز سوريا. */
function waLink(phone: string): string {
  const digits = phone.replace(/[^\d]/g, "").replace(/^0/, "963");
  return `https://wa.me/${digits}`;
}

/** **كم دقيقةً مضت** — والسالبُ (ساعةُ جهازٍ متقدّمة) صفر. */
function minutesSince(iso: string): number {
  const ms = Date.now() - Date.parse(iso);
  return Number.isFinite(ms) && ms > 0 ? Math.floor(ms / 60000) : 0;
}

export default function OpsMapPage() {
  const router = useRouter();
  const { capabilities, user } = useAuth();

  // ── حالُ الواجهة (البند ٣٨) ──────────────────────────────────
  //
  // **ويُحفَظ ما يختاره المستخدم لكلّ مستخدم** — موظّفان على جهازٍ واحدٍ
  // لا يتقاسمان طبقاتهما. **ولا يُحفَظ ما يشيخ**: البياناتُ تُجلَب دائماً،
  // **فلا تُعرض قديمةٌ على أنّها حيّة.**
  const storeKey = `opsmap.layers.${user?.id ?? "anon"}`;
  const [visible, setVisible] = useState<Record<string, boolean>>(DEFAULT_VISIBLE);
  useEffect(() => {
    try {
      const raw =
        window.localStorage.getItem(storeKey) ?? window.localStorage.getItem("opsmap.layers");
      setVisible(raw ? (JSON.parse(raw) as Record<string, boolean>) : DEFAULT_VISIBLE);
    } catch {
      /* تخزينٌ ممنوعٌ في نافذةٍ خاصّة — والافتراضُ يكفي */
    }
  }, [storeKey]);
  const toggle = useCallback(
    (id: string) => {
      setVisible((prev) => {
        const next = { ...prev, [id]: !prev[id] };
        try {
          window.localStorage.setItem(storeKey, JSON.stringify(next));
        } catch {
          /* لا يضرّ */
        }
        return next;
      });
    },
    [storeKey],
  );

  // **والمرشِّحاتُ من الرابط** — بطاقاتُ السائقين في رئيسيّة المدير تفتح هنا
  // على «على الدوام · فعّال · متفرّغ أو معه طلب» (قرارُ المالك ٢٠٢٦-١٠-٠٤).
  const tri = (k: string): "" | "true" | "false" => {
    if (typeof window === "undefined") return "";
    const v = new URLSearchParams(window.location.search).get(k);
    return v === "true" || v === "false" ? v : "";
  };
  const [onShift, setOnShift] = useState<"" | "true" | "false">(() => tri("on_shift"));
  const [stale, setStale] = useState<"" | "true" | "false">("");
  const [hasActive, setHasActive] = useState<"" | "true" | "false">(() => tri("has_active"));
  const [driverStatus] = useState(() =>
    typeof window === "undefined" ? "" : (new URLSearchParams(window.location.search).get("status") ?? ""),
  );
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Picked | null>(null);
  const [menu, setMenu] = useState<"" | "extra" | "legend">("");
  const [fly, setFly] = useState<FlyRequest | undefined>(undefined);
  const [assigning, setAssigning] = useState<OrderPin | null>(null);

  // ── التركيزُ وملءُ الشاشة ───────────────────────────────────────
  //
  // **تركيزٌ واحدٌ في كلّ مرّة** — «العالقة فقط» و«المتاحون فقط» معاً
  // لا يُبقيان شيئاً تقريباً، **ومفتاحان يُطفئ أحدُهما الآخرَ أوضحُ من
  // خريطةٍ فارغةٍ بلا سبب.**
  const [focusMode, setFocusMode] = useState<FocusMode>("");
  const [full, setFull] = useState(false);
  useEffect(() => {
    if (!full) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setFull(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [full]);

  // ── محرِّرُ المضلَّعات (البند ١٣) ──────────────────────────
  //
  // **والرسمُ حالٌ صريحةٌ لا وضعٌ خفيّ** — **ومن نقر الأرضَ وهو لا
  // يرسم فتح معلَماً، ومن نقرها وهو يرسم أضاف نقطة.**
  //
  // **و«ارسم تغطيةً هنا» من «طلبات التوسّع»** يفتح الخريطةَ على الموضع
  // والرسمُ قائم (`?focus=lat,lng&draw=1`).
  const [focus] = useState<{ lng: number; lat: number; zoom: number } | undefined>(() => {
    if (typeof window === "undefined") return undefined;
    const [lat = NaN, lng = NaN] = (new URLSearchParams(window.location.search).get("focus") ?? "")
      .split(",")
      .map(Number);
    return Number.isFinite(lat) && Number.isFinite(lng) && (lat !== 0 || lng !== 0)
      ? { lat, lng, zoom: 15 }
      : undefined;
  });
  const [drawing, setDrawing] = useState(
    () => typeof window !== "undefined" && new URLSearchParams(window.location.search).get("draw") === "1",
  );
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
  // **ابحث واذهب** — رقمُ طلبٍ أو اسمُ سائقٍ أو متجرٍ مما حُمِّل للخريطة،
  // **ثمّ ما يجده الخادمُ مما لم يُحمَّل** (فروعٌ ومناطقُ ومندوبون)، **وهو
  // يُقَصُّ بصلاحيّات الباحث** — ومن وجد اسمَ من لا يملك رؤيتَه عرف أنّه موجود.
  const [hunt, setHunt] = useState("");
  const [huntOpen, setHuntOpen] = useState(false);
  const [huntAt, setHuntAt] = useState(0);

  // ── ما يملكه من يقف أمامها ───────────────────────────────────
  const meta = useLiveData<Meta>(() => api<Meta>("/api/v1/admin/ops-map/meta"), []);
  const can = useCallback(
    (p: string) => !!meta.data?.permissions.includes(p),
    [meta.data],
  );

  // ── نبضةُ الموضع ─────────────────────────────────────────────
  //
  // # ولماذا مدّةٌ لا بثّ (البندان ٨ و٤٤)
  //
  // **كاتبُ الموضع لا يبثّ** — ولا يُمسّ (البند ٤٥). **والمدّةُ ليست
  // اعتباطاً**: هي نبضةُ السائق نفسُها من الإعدادات (ولا أقلَّ من نصف
  // دقيقة)، **فلا تُسأل القاعدةُ عمّا لم يتغيّر بعد.**
  //
  // **والطلباتُ تُحدَّث ببثّ `ops` القائم** — لا بمدّة.
  const pingSec = Math.max(30, meta.data?.location_ping_sec ?? 60);
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const h = window.setInterval(() => setTick((t) => t + 1), pingSec * 1000);
    return () => window.clearInterval(h);
  }, [pingSec]);

  // ── عدّاداتُ الشريط ─────────────────────────────────────────
  const summary = useLiveData<Summary>(
    () =>
      can("VIEW_OPERATIONS_MAP")
        ? api<Summary>("/api/v1/admin/ops-map/summary")
        : Promise.resolve({ staff_online: 0 }),
    ["order", "catalog"],
    [meta.data, tick],
  );

  // ── السائقون ─────────────────────────────────────────────────
  const driverQuery = useMemo(() => {
    const q = new URLSearchParams();
    if (onShift) q.set("on_shift", onShift);
    if (stale) q.set("stale", stale);
    if (hasActive) q.set("has_active", hasActive);
    if (driverStatus) q.set("status", driverStatus);
    if (search.trim()) q.set("q", search.trim());
    const s = q.toString();
    return s ? `?${s}` : "";
  }, [onShift, stale, hasActive, driverStatus, search]);

  const drivers = useLiveData<{ drivers: Driver[]; count: number }>(
    () =>
      can("VIEW_DRIVER_LOCATIONS") && visible.drivers
        ? api<{ drivers: Driver[]; count: number }>(`/api/v1/admin/ops-map/drivers${driverQuery}`)
        : Promise.resolve({ drivers: [], count: 0 }),
    [],
    [driverQuery, visible.drivers, meta.data, tick],
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

  // ── الزبائن — خلايا مجمَّعةٌ لا بيوت ────────────────────────
  const customers = useLiveData<{ cells: DemandCell[]; min_count: number }>(
    () =>
      can("VIEW_ACTIVE_ORDERS") && visible.customers
        ? api<{ cells: DemandCell[]; min_count: number }>("/api/v1/admin/ops-map/customers")
        : Promise.resolve({ cells: [], min_count: 3 }),
    [],
    [visible.customers, meta.data],
  );

  // ── المكتب ───────────────────────────────────────────────────
  const office = useLiveData<Office | null>(
    () =>
      can("VIEW_OPERATIONS_MAP") && visible.office
        ? api<Office>("/api/v1/admin/ops-map/office")
        : Promise.resolve(null),
    [],
    [visible.office, meta.data, tick],
  );

  // ── التغطية — حدودٌ رفيعةٌ دائماً لا مفتاح ───────────────────
  const zones = useLiveData<{ zones: Zone[]; count: number }>(
    () =>
      can("VIEW_OPERATIONS_MAP")
        ? api<{ zones: Zone[]; count: number }>("/api/v1/admin/ops-map/coverage")
        : Promise.resolve({ zones: [], count: 0 }),
    ["catalog"],
    [meta.data],
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

  // ── المندوبون — «أين يعمل»: مركزُ متاجره ─────────────────────
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

  // ── الطوارئ طبقةٌ على الخريطة (قراراتُ المالك ٢٠٢٦-١٠-٠٤ — غرفةُ الطوارئ) ──
  //
  // **لمن يملك الطوارئ** — والنقرةُ تفتح صفحةَ الطارئ.
  const canEmergencies = capabilities.includes("emergencies.manage");
  const canIntervene = capabilities.includes("orders.intervene");
  // **وملفُّ الحساب لمن يقرأ الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٥): موظّفُ العمليّات
  // لا يملك `users.read` — **فلا يُرسَم له رابطٌ يُردّ ٤٠٣.**
  const canProfile = useCanCall()("GET", "/users/{id}");
  const emergencies = useLiveData<{ emergencies: EmergencyPin[]; count: number }>(
    () =>
      canEmergencies && visible.emergencies
        ? api<{ emergencies: EmergencyPin[]; count: number }>("/api/v1/admin/emergencies/map")
        : Promise.resolve({ emergencies: [], count: 0 }),
    ["emergency"],
    [canEmergencies, visible.emergencies],
  );

  // **والرموزُ تُرسم مرّةً** — لونُ كلّ حالٍ من الثيم ورمزُه من مجموعتنا.
  const icons = useMemo(() => mapIcons(), []);

  // ── التركيز — ما يُرسَم بعد «العالقة فقط» أو «المتاحون فقط» ─────
  const focused = useMemo(
    () =>
      applyFocus(
        focusMode,
        drivers.data?.drivers ?? [],
        orders.data?.orders ?? [],
        merchants.data?.merchants ?? [],
      ),
    [focusMode, drivers.data, orders.data, merchants.data],
  );

  // ── المُحدَّدُ حيٌّ لا لقطة ─────────────────────────────────────
  //
  // **البطاقةُ تقرأ آخرَ ما جاء من الخادم** — طلبٌ أُسند وهي مفتوحةٌ يظهر
  // سائقُه فيها، **ولا تبقى على ما كان لحظةَ النقر.**
  const current = useMemo<Picked | null>(() => {
    if (!selected) return null;
    if (selected.kind === "order") {
      const v = orders.data?.orders.find((x) => x.id === selected.v.id);
      return v ? { kind: "order", v } : selected;
    }
    if (selected.kind === "driver") {
      const v = drivers.data?.drivers.find((x) => x.id === selected.v.id);
      return v ? { kind: "driver", v } : selected;
    }
    if (selected.kind === "merchant") {
      const v = merchants.data?.merchants.find((x) => x.id === selected.v.id);
      return v ? { kind: "merchant", v } : selected;
    }
    return selected;
  }, [selected, orders.data, drivers.data, merchants.data]);

  // ── عالقٌ جديد — ومضةٌ وجرس ─────────────────────────────────────
  //
  // **حين يزيد العددُ والشاشةُ مفتوحة** — لا حين تُفتح. **والجرسُ جرسُ
  // اللوحة نفسُه** (`useChime`)، **ويصمت بمفتاح الصوت نفسِه**
  // (`rahalgo_alerts_sound`) — ولا مفتاحَ ثانٍ يُنسى.
  const chime = useChime();
  const lastStuck = useRef<number | null>(null);
  const [flash, setFlash] = useState(0);
  const stuckNow = summary.data?.orders?.stuck;
  useEffect(() => {
    if (stuckNow == null) return;
    if (stuckIncreased(lastStuck.current, stuckNow)) {
      setFlash(Date.now());
      if (alertsSoundOn()) chime();
    }
    lastStuck.current = stuckNow;
  }, [stuckNow, chime]);
  useEffect(() => {
    if (!flash) return;
    const h = window.setTimeout(() => setFlash(0), 4000);
    return () => window.clearTimeout(h);
  }, [flash]);

  // ── نتائجُ «ابحث واذهب» ─────────────────────────────────────────
  const goItems = useMemo(() => {
    const local = searchLocal(hunt, {
      orders: orders.data?.orders ?? [],
      drivers: drivers.data?.drivers ?? [],
      merchants: merchants.data?.merchants ?? [],
    });
    const seen = new Set(local.map((x) => `${x.kind}-${x.id}`));
    const remote = (hunt.trim().length >= 2 ? (hits.data?.hits ?? []) : [])
      .filter((x) => x.lat != null && x.lng != null && !seen.has(`${x.kind}-${x.id}`))
      .slice(0, 5)
      .map((x) => ({ kind: x.kind, id: x.id, label: x.label, lng: x.lng as number, lat: x.lat as number }));
    return [...local.map(({ kind, id, label, lng, lat }) => ({ kind: kind as string, id, label, lng, lat })), ...remote];
  }, [hunt, orders.data, drivers.data, merchants.data, hits.data]);

  const layers = useMemo<LayerSpec[]>(() => {
    const out: LayerSpec[] = [];
    out.push({
      id: "emergencies",
      kind: "point",
      cluster: false,
      order: 90,
      visible: !!visible.emergencies,
      color: themeColor("danger-solid"),
      data: fc(
        (emergencies.data?.emergencies ?? []).map((x) =>
          pt(x.lng, x.lat, { id: x.id, kind: x.kind }),
        ),
      ),
    });

    // ── السائقون — أخضرُ متاح · برتقاليٌّ معه طلب · رماديٌّ خامل ──
    //
    // **ولا يُجمَّعون**: فقّاعةٌ خضراءُ تخفي برتقاليّاً بداخلها، **والحالُ هو
    // ما جاء المكتبُ يراه.** ومن لا موضعَ له لا يُرسَم (البند ٤).
    out.push({
      id: "drivers",
      kind: "symbol",
      cluster: false,
      order: 50,
      visible: !!visible.drivers,
      color: toneColor("driver-available"),
      // **ينزلق بين نبضتين، وحلقةٌ نابضةٌ حولَ من موقعُه حيّ** — بلون حاله.
      animate: true,
      pulse: {
        field: "live",
        color: [
          "match", ["get", "icon"],
          "driver-available", toneColor("driver-available"),
          "driver-busy", toneColor("driver-busy"),
          toneColor("driver-idle"),
        ],
      },
      data: fc(
        focused.drivers
          .filter((d) => d.freshness !== "NO_LOCATION")
          .map((d) => pt(d.lng, d.lat, { id: d.id, icon: driverTone(d.tone), live: d.freshness === "LIVE" })),
      ),
    });

    // ── المتاجر — بنفسجيٌّ مفتوح · رماديٌّ مغلق ─────────────────
    //
    // **والمغلقُ يُرى مغلقاً** — لونٌ باهتٌ لا اختفاء: «متجرٌ مغلقٌ
    // وطلبٌ ينتظره» حقيقةٌ يريد المكتبُ أن يراها.
    //
    // **والفقّاعةُ تقول القسمةَ** (تحسيناتُ ٢٠٢٦-١٠-٠٦): **لونُها لونُ
    // الأكثر** (مفتوحٌ أو مسكّر)، **وحلقةٌ عريضةٌ بلون الأقلّ إن وُجد** —
    // «عشرون متجراً ستّةٌ منها مسكّرة» بنظرةٍ لا بنقرة.
    const openC = toneColor("store-open");
    const closedC = toneColor("store-closed");
    const opened = ["get", "open_n"];
    const total = ["get", "point_count"];
    const openMajority = [">=", ["*", opened, 2], total];
    out.push({
      id: "merchants",
      kind: "symbol",
      cluster: true,
      order: 40,
      visible: !!visible.merchants,
      color: openC,
      clusterProperties: { open_n: ["+", ["case", ["==", ["get", "open"], true], 1, 0]] },
      clusterColor: ["case", openMajority, openC, closedC],
      clusterStroke: {
        color: [
          "case",
          ["any", ["==", opened, 0], ["==", opened, total]], themeColor("paper"),
          openMajority, closedC,
          openC,
        ],
        width: ["case", ["any", ["==", opened, 0], ["==", opened, total]], 2, 5],
      },
      data: fc(
        focused.merchants.map((x) =>
          pt(x.lng, x.lat, { id: x.id, icon: storeTone(x.open_now), open: x.open_now }),
        ),
      ),
    });

    // ── الطلباتُ النشطة — أزرقُ ماشٍ · أحمرُ عالق ────────────────
    //
    // **والعالقُ بشرط لوحة الطلبات نفسِه** (`stuck_reason` من الخادم) — فلا
    // تقول الخريطةُ «عالق» عن طلبٍ لا تعدّه اللوحة. **وخطٌّ من المتجر إلى
    // نقطة التسليم لكلّ طلب** — «من أين إلى أين» بنظرة.
    const orderList = focused.orders;
    out.push({
      id: "orders",
      kind: "symbol",
      cluster: false,
      order: 60,
      visible: !!visible.orders,
      color: toneColor("order"),
      data: fc(
        orderList.map((o) =>
          pt(o.drop_lng, o.drop_lat, { id: o.id, icon: orderTone(o.stuck_reason) }),
        ),
      ),
    });
    out.push({
      id: "order-routes",
      kind: "line",
      order: 55,
      lineWidth: 1.5,
      visible: !!visible.orders,
      color: ["case", ["get", "stuck"], toneColor("order-stuck"), toneColor("order")],
      data: fc(
        orderList
          .filter((o) => o.pick_lng != null && o.pick_lat != null)
          .map((o) => ({
            type: "Feature" as const,
            geometry: {
              type: "LineString",
              coordinates: [[o.pick_lng, o.pick_lat], [o.drop_lng, o.drop_lat]],
            },
            properties: { stuck: !!o.stuck_reason },
          })),
      ),
    });

    // ── الزبائن — خلايا ورديّةٌ بعددها ───────────────────────────
    out.push({
      id: "customers",
      kind: "bubble",
      order: 35,
      visible: !!visible.customers,
      color: toneColor("customer"),
      data: fc(
        (customers.data?.cells ?? []).map((c) =>
          pt(c.lng, c.lat, { lat: c.lat, lng: c.lng, count: c.count }),
        ),
      ),
    });

    // ── المندوبون — أصفرُ في مركز متاجرهم (البند ٢٦) ─────────────
    //
    // **ونقاطُ متاجرهم لا مساراتُ هواتفهم** — **ولا تتبّعَ لخطوات
    // إنسانٍ لم يطلبه أحد.**
    out.push({
      id: "reps",
      kind: "symbol",
      order: 45,
      visible: !!visible.reps,
      color: toneColor("rep"),
      data: fc(
        (reps.data?.reps ?? [])
          .filter((r) => r.lat != null && r.lng != null)
          .map((r) => pt(r.lng as number, r.lat as number, { id: r.id, icon: "rep" })),
      ),
    });

    // ── التغطية — حدودٌ رفيعةٌ دائماً ────────────────────────────
    //
    // **والدائرةُ والمضلَّعُ طبقتان لا واحدة**: MapLibre ترسم الدائرةَ
    // بنصفِ قطرٍ بالبكسل والمضلَّعَ بحشوٍ — **ولا شكلَ واحدٌ يسعهما.**
    // **ولا مفتاحَ لها** (قرارُ المالك ٢٠٢٦-١٠-٠٥): من لا يرى أين نصل لا
    // يقرأ الخريطةَ أصلاً، **والخطُّ الرفيعُ لا يحجب شيئاً تحته.**
    const zoneList = zones.data?.zones ?? [];
    out.push({
      id: "coverage-radius",
      kind: "circle-m",
      radiusField: "radius_m",
      order: 10,
      visible: true,
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
      visible: true,
      color: themeColor("cta-end"),
      lineWidth: drawing ? 2.5 : 1.25,
      fillOpacity: 0.04,
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
      color: themeColor("cta-end"),
      lineWidth: 4,
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

    // ── مسارُ الطلب المُحدَّد (البند ١١ · تحسيناتُ ٢٠٢٦-١٠-٠٦) ──────
    //
    // **سائقٌ ← متجرٌ ← زبون، خطوطٌ مستقيمةٌ متقطّعة** — **الخريطةُ ليست
    // محرّكَ ملاحة**، ولا بابَ توجيهٍ للعمليّات. **والسائقُ من موضعه الحيّ**
    // إن كان في الطبقة، **وبعد الاستلام لا متجرَ في الطريق.**
    const link: FeatureCollection["features"] = [];
    if (current?.kind === "order") {
      const o = current.v;
      const d = o.driver_id ? drivers.data?.drivers.find((x) => x.id === o.driver_id) : undefined;
      const live: LngLat | null = d && d.freshness !== "NO_LOCATION" ? [d.lng, d.lat] : null;
      for (const leg of orderRoute(o, live)) {
        link.push({
          type: "Feature" as const,
          geometry: { type: "LineString", coordinates: leg.coords },
          properties: { leg: leg.leg },
        });
      }
    }
    out.push({
      id: "order-link",
      kind: "line",
      order: 70,
      lineWidth: 3,
      visible: link.length > 0,
      color: ["match", ["get", "leg"], "to-store", toneColor("driver-busy"), themeColor("primary")],
      data: fc(link),
    });

    return out;
  }, [drivers.data, zones.data, requests.data, focused,
      branches.data, reps.data, demand.data, opportunities.data, customers.data,
      visible, current, draft, drawing, emergencies.data]);

  // ── المكتبُ علامةٌ بشعار المنصّة ─────────────────────────────
  const markers = useMemo<MapMarker[]>(() => {
    const o = office.data;
    if (!visible.office || !o || !o.has_location || o.lat == null || o.lng == null) return [];
    return [{
      id: "office",
      lng: o.lng,
      lat: o.lat,
      imageUrl: mediaUrl(o.logo),
      letter: o.name.trim().charAt(0),
      label: T.office.title,
    }];
  }, [office.data, visible.office]);

  const onFeature = useCallback(
    (layerID: string, props: Record<string, unknown>) => {
      if (layerID === "emergencies" && typeof props.id === "string") {
        router.push(`/dashboard/emergencies/${props.id}`);
        return;
      }
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
      if (layerID === "customers") {
        const c = customers.data?.cells.find((y) => y.lat === props.lat && y.lng === props.lng);
        if (c) setSelected({ kind: "customers", v: c });
      }
      if (layerID === "reps") {
        const r = reps.data?.reps.find((y) => y.id === props.id);
        if (r) setSelected({ kind: "rep", v: r });
      }
      if (layerID === "coverage-radius" || layerID === "coverage-polygon") {
        // **والرسمُ أولى من فتح البطاقة** — نقطةٌ داخلَ منطقةٍ قائمةٍ نقطةٌ.
        if (drawing) return;
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
    [drivers.data, merchants.data, orders.data, customers.data, reps.data, zones.data,
     requests.data, branches.data, opportunities.data, router, drawing],
  );

  // ── نقرُ الأرض ───────────────────────────────────────────────
  const onGround = useCallback(
    (lng: number, lat: number) => {
      if (drawing) setDraft((prev) => [...prev, [lng, lat]]);
    },
    [drawing],
  );

  // ── «عالق» — نقرةٌ صريحةٌ تطير إلى أوّل عالقٍ وتفتح بطاقتَه ──
  const goStuck = useCallback(() => {
    const o = summary.data?.orders?.first_stuck;
    if (!o) return;
    setSelected({ kind: "order", v: o });
    setFly({ lng: o.drop_lng, lat: o.drop_lat, zoom: 15, key: Date.now() });
  }, [summary.data]);

  // ── «ابحث واذهب» — يفتح البطاقةَ ويطير ─────────────────────────
  //
  // **ومن اختار ما يخفيه التركيزُ رُفع التركيز** — وإلّا طار إلى فراغ.
  const goTo = useCallback(
    (it: { kind: string; id: string; lng: number; lat: number }) => {
      setHunt("");
      setHuntOpen(false);
      setHuntAt(0);
      if (it.kind === "order") {
        const o = orders.data?.orders.find((x) => x.id === it.id);
        if (o) {
          if (!focused.orders.some((x) => x.id === it.id)) setFocusMode("");
          setSelected({ kind: "order", v: o });
        }
      } else if (it.kind === "driver") {
        const d = drivers.data?.drivers.find((x) => x.id === it.id);
        if (d) {
          if (!focused.drivers.some((x) => x.id === it.id)) setFocusMode("");
          setSelected({ kind: "driver", v: d });
        }
      } else if (it.kind === "merchant") {
        const x = merchants.data?.merchants.find((y) => y.id === it.id);
        if (x) {
          if (!focused.merchants.some((y) => y.id === it.id)) setFocusMode("");
          setSelected({ kind: "merchant", v: x });
        }
      }
      setFly({ lng: it.lng, lat: it.lat, zoom: 16, key: Date.now() });
    },
    [orders.data, drivers.data, merchants.data, focused],
  );

  // ── «رجّع على المكتب» ──────────────────────────────────────────
  const officeAt = useMemo(() => {
    const o = office.data;
    return o?.has_location && o.lat != null && o.lng != null ? { lng: o.lng, lat: o.lat } : null;
  }, [office.data]);
  const recenter = useCallback(() => {
    if (officeAt) setFly({ lng: officeAt.lng, lat: officeAt.lat, zoom: 14, key: Date.now() });
  }, [officeAt]);

  const savePolygon = useCallback(async () => {
    setSaveErr("");
    try {
      await api("/api/v1/admin/ops-map/coverage", {
        method: "POST",
        body: JSON.stringify({
          name: zoneName.trim(), ring: draft, min_order: 0,
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
  const S = summary.data;

  // ── حبّاتُ الطبقات الستّ — بترتيب المالك ─────────────────────
  const chips: Array<{ id: MainLayer; label: string; tone: MapTone | null; count?: number; show: boolean }> = [
    { id: "drivers", label: T.chip.drivers, tone: "driver-available", count: S?.drivers?.total, show: can("VIEW_DRIVER_LOCATIONS") },
    { id: "orders", label: T.chip.orders, tone: "order", count: S?.orders?.active, show: can("VIEW_ACTIVE_ORDERS") },
    { id: "merchants", label: T.chip.merchants, tone: "store-open", count: S?.merchants?.total, show: can("VIEW_MERCHANT_LOCATIONS") },
    { id: "customers", label: T.chip.customers, tone: "customer", count: S?.customers?.total, show: can("VIEW_ACTIVE_ORDERS") },
    { id: "reps", label: T.chip.reps, tone: "rep", count: S?.reps, show: can("VIEW_REP_ACTIVITY") },
    { id: "office", label: T.chip.office, tone: null, count: S?.staff_online, show: true },
  ];

  const stuckN = S?.orders?.stuck ?? 0;

  // ── الطبقةُ الفارغةُ تُقال لا تُسكَت ───────────────────────────────
  //
  // **خريطةٌ فارغةٌ بلا كلمةٍ تُقرأ عطلاً** — «لا سائقين على الدوام الآن»
  // تُقرأ حقيقة. **ولا تُقال قبل أن تصل البيانات** — فلا يُكذَب أثناءَ التحميل.
  const driversFiltered = !!(onShift || stale || hasActive || driverStatus || search.trim());
  const emptyHints: string[] = [];
  // **وفي «العالقة فقط» بلا عالقٍ تكفي جملةُ الطلبات** — لا جملتان لفراغٍ واحد.
  const noStuckAtAll = focusMode === "stuck" && focused.orders.length === 0;
  if (can("VIEW_DRIVER_LOCATIONS") && visible.drivers && drivers.data && !noStuckAtAll &&
      focused.drivers.filter((d) => d.freshness !== "NO_LOCATION").length === 0) {
    emptyHints.push(
      focusMode === "available" ? T.emptyHint.available
        : focusMode === "stuck" ? T.emptyHint.stuckDrivers
          : driversFiltered ? T.emptyHint.driversFiltered : T.emptyHint.drivers,
    );
  }
  if (can("VIEW_ACTIVE_ORDERS") && visible.orders && orders.data && focused.orders.length === 0) {
    emptyHints.push(focusMode === "stuck" ? T.emptyHint.stuck : T.emptyHint.orders);
  }
  if (can("VIEW_MERCHANT_LOCATIONS") && visible.merchants && merchants.data &&
      focused.merchants.length === 0 && focusMode !== "stuck") {
    emptyHints.push(T.emptyHint.merchants);
  }

  return (
    <PageContainer>
      <PageHeader title={T.title} subtitle={T.subtitle} />

      {/* ══ ملءُ الشاشة — الشريطُ والخريطةُ معاً، والقوائمُ تغيب ════════ */}
      <div className={full ? "fixed inset-0 z-40" : ""}>
      <div className={full ? "surface-raised flex h-full flex-col gap-2 overflow-hidden rounded-none p-2" : ""}>
      {/* ══ الشريطُ العلويّ — عدّاداتٌ حيّة وحبّاتُ الطبقات ══════════ */}
      <div className={`surface flex flex-col gap-2 p-2 ${full ? "shrink-0" : "mb-3"}`}>
        {/* ── العدّادات — «N سائق متاح · N طلب ماشي · N عالق · N متجر مفتوح» ── */}
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 px-1 text-sm">
          {S?.drivers && (
            <Counter n={S.drivers.available} label={T.counter.available} tone="driver-available" />
          )}
          {S?.orders && (
            <>
              <Counter n={S.orders.active} label={T.counter.moving} tone="order" />
              {stuckN > 0 ? (
                <button
                  type="button"
                  onClick={goStuck}
                  title={T.counter.stuckHint}
                  className={`inline-flex items-center gap-1.5 rounded-full font-bold text-danger underline-offset-4 hover:underline ${
                    flash ? "animate-pulse bg-danger-tint px-2 ring-2 ring-danger" : ""
                  }`}
                >
                  <Dot tone="order-stuck" />
                  <span className="tabular-nums">{fmtNum(stuckN)}</span>
                  <span>{T.counter.stuck}</span>
                </button>
              ) : (
                <Counter n={0} label={T.counter.stuck} tone="order-stuck" />
              )}
            </>
          )}
          {S?.merchants && (
            <Counter n={S.merchants.open} label={T.counter.open} tone="store-open" />
          )}
          {/* **ومضةُ العالق الجديد تُقال لقارئ الشاشة أيضاً.** */}
          <span className="sr-only" role="status" aria-live="polite">
            {flash ? T.counter.newStuck : ""}
          </span>

          {/* ── مفاتيحُ التركيز — بجوار العدّادات ── */}
          {can("VIEW_ACTIVE_ORDERS") && (
            <FocusToggle
              on={focusMode === "stuck"}
              label={T.focus.stuckOnly}
              hint={T.focus.stuckHint}
              tone="order-stuck"
              onClick={() => {
                // **والعالقُ يُقرأ من طبقة الطلبات** — فتُفتح معه إن كانت مطفأة.
                if (focusMode !== "stuck" && !visible.orders) toggle("orders");
                setFocusMode(focusMode === "stuck" ? "" : "stuck");
              }}
            />
          )}
          {can("VIEW_DRIVER_LOCATIONS") && (
            <FocusToggle
              on={focusMode === "available"}
              label={T.focus.availableOnly}
              hint={T.focus.availableHint}
              tone="driver-available"
              onClick={() => {
                if (focusMode !== "available" && !visible.drivers) toggle("drivers");
                setFocusMode(focusMode === "available" ? "" : "available");
              }}
            />
          )}

          {/* ── ابحث واذهب ── */}
          <div className="relative w-full sm:ms-auto sm:w-72">
            <Input
              icon={<IconSearch size={16} aria-hidden />}
              placeholder={T.searchGo}
              aria-label={T.searchGo}
              value={hunt}
              onChange={(e) => {
                setHunt(e.target.value);
                setHuntOpen(true);
                setHuntAt(0);
              }}
              onFocus={() => setHuntOpen(true)}
              onKeyDown={(e) => {
                if (e.key === "Escape") {
                  setHuntOpen(false);
                } else if (e.key === "ArrowDown") {
                  e.preventDefault();
                  setHuntAt((i) => Math.min(i + 1, Math.max(goItems.length - 1, 0)));
                } else if (e.key === "ArrowUp") {
                  e.preventDefault();
                  setHuntAt((i) => Math.max(i - 1, 0));
                } else if (e.key === "Enter" && goItems[huntAt]) {
                  e.preventDefault();
                  goTo(goItems[huntAt]);
                }
              }}
            />
            {huntOpen && hunt.trim() !== "" && (
              <>
                <button
                  type="button"
                  aria-label={T.close}
                  className="fixed inset-0 z-20 cursor-default"
                  onClick={() => setHuntOpen(false)}
                />
                <ul
                  role="listbox"
                  className="surface-raised absolute inset-x-0 top-full z-30 mt-1 max-h-80 overflow-y-auto p-1 text-sm"
                >
                  {goItems.length === 0 ? (
                    <li className="px-2 py-1.5 text-xs text-ink-muted">{T.searchNone}</li>
                  ) : (
                    goItems.map((x, i) => (
                      <li key={`${x.kind}-${x.id}`} role="option" aria-selected={i === huntAt}>
                        <button
                          type="button"
                          onMouseEnter={() => setHuntAt(i)}
                          onClick={() => goTo(x)}
                          className={`flex w-full items-center gap-2 rounded-control px-2 py-1.5 text-start ${
                            i === huntAt ? "bg-row-hover" : ""
                          }`}
                        >
                          <span className="truncate">{x.label}</span>
                          <span className="ms-auto shrink-0 text-xs text-ink-muted">{kindLabel(x.kind)}</span>
                        </button>
                      </li>
                    ))
                  )}
                </ul>
              </>
            )}
          </div>
        </div>

        {/* ── الحبّات — تلتفّ على الحاسوب وتنزلق على الجوّال ── */}
        <div className="flex items-center gap-2">
          <Chips
            className="min-w-0 flex-1 sm:flex-wrap"
            items={chips
              .filter((c) => c.show)
              .map((c) => ({
                id: c.id,
                label: (
                  <span className="inline-flex items-center gap-1.5">
                    {c.tone ? <Dot tone={c.tone} /> : <OfficeDot />}
                    {c.label}
                    {c.count != null && (
                      <CountBadge count={c.count} on={!!visible[c.id]} max={99999} />
                    )}
                  </span>
                ),
              }))}
            value={chips.filter((c) => c.show && visible[c.id]).map((c) => c.id)}
            onChange={(id) => toggle(id)}
          />

          {/* ── طبقاتٌ إضافيّةٌ ودليلُ الألوان ── */}
          <div className="relative flex shrink-0 items-center gap-1.5">
            <Button
              variant="secondary"
              aria-expanded={menu === "extra"}
              onClick={() => setMenu(menu === "extra" ? "" : "extra")}
            >
              {T.bar.extra}
            </Button>
            <button
              type="button"
              aria-label={T.bar.legend}
              title={T.bar.legend}
              aria-expanded={menu === "legend"}
              onClick={() => setMenu(menu === "legend" ? "" : "legend")}
              className="grid h-9 w-9 place-items-center rounded-full border border-line bg-surface text-sm font-bold text-ink-muted hover:text-ink"
            >
              {T.bar.legendMark}
            </button>
            {menu !== "" && (
              <>
                {/* **ونقرةٌ خارجَها تغلقها** — ستارٌ شفّافٌ تحت اللوحة. */}
                <button
                  type="button"
                  aria-label={T.close}
                  className="fixed inset-0 z-20 cursor-default"
                  onClick={() => setMenu("")}
                />
                <div className="surface-raised absolute end-0 top-full z-30 mt-1.5 w-72 max-w-[calc(100vw-2rem)] p-3 text-sm">
                  {menu === "extra" ? (
                    <div className="flex flex-col gap-1">
                      {canEmergencies && (
                        <LayerToggle
                          id="emergencies"
                          label={m.admin.emergencyRoom.mapLayer}
                          on={!!visible.emergencies}
                          count={emergencies.data?.count}
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
                          <LayerToggle
                            id="requests"
                            label={T.layer.coverageRequests}
                            on={!!visible.requests}
                            count={requests.data?.count}
                            onToggle={toggle}
                          />
                        </>
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
                      <p className="mt-2 text-xs leading-5 text-ink-muted">{T.bar.coverageAlways}</p>
                      {/* **ولا يظهر الرسمُ لمن لا يملكه** — ورؤيةُ زرٍّ يردّ
                          `403` أسوأُ من غيابه. */}
                      {can("MANAGE_COVERAGE") && !drawing && (
                        <Button
                          variant="secondary"
                          className="mt-2"
                          onClick={() => { setDrawing(true); setDraft([]); setMenu(""); }}
                        >
                          {T.bar.drawCoverage}
                        </Button>
                      )}
                    </div>
                  ) : (
                    <Legend />
                  )}
                </div>
              </>
            )}
          </div>
        </div>

        {visible.office && office.data && !office.data.has_location && (
          <p className="px-1 text-xs text-ink-muted">{T.office.noLocation}</p>
        )}
      </div>

      <div className={full ? "flex min-h-0 flex-1" : "grid gap-4 lg:grid-cols-[18rem_1fr]"}>
        {/* ── الخريطةُ واللوحةُ الجانبيّة — أوّلاً على الجوّال ───────── */}
        <div
          className={
            full
              ? "relative h-full min-h-0 w-full"
              : "relative order-1 h-[28rem] lg:order-2 lg:h-[calc(100vh-16rem)]"
          }
        >
          <OpsMapCanvas
            layers={layers}
            icons={icons}
            markers={markers}
            flyTo={fly}
            onFeatureClick={onFeature}
            onMarkerClick={() => setSelected({ kind: "office" })}
            onMapClick={onGround}
            unavailableLabel={T.unavailable}
            focus={focus}
          />

          {/* ── أدواتُ الخريطة — ملءُ الشاشة والرجوعُ إلى المكتب ── */}
          <div className="absolute start-2 top-2 z-[5] flex flex-col gap-1.5">
            <MapTool
              label={full ? T.bar.exitFullscreen : T.bar.fullscreen}
              onClick={() => setFull((f) => !f)}
            >
              {full ? <IconFullscreenExit size={16} aria-hidden /> : <IconFullscreen size={16} aria-hidden />}
            </MapTool>
            {officeAt && (
              <MapTool label={T.bar.recenter} onClick={recenter}>
                <IconTarget size={16} aria-hidden />
              </MapTool>
            )}
          </div>

          {/* ── الفراغُ يُقال — نصٌّ خافتٌ لا خريطةٌ صامتة ── */}
          {emptyHints.length > 0 && (
            <div className="pointer-events-none absolute inset-x-0 bottom-8 z-[5] flex justify-center px-4">
              <p className="surface rounded-full px-3 py-1.5 text-center text-xs text-ink-muted shadow-lift">
                {emptyHints.join(" · ")}
              </p>
            </div>
          )}

          {current && (
            <aside
              className="absolute inset-y-0 end-0 z-10 w-full max-w-sm overflow-y-auto
                         border-s border-line-soft bg-surface p-4 shadow-xl"
            >
              <div className="mb-3 flex items-center gap-2">
                <h3 className="text-base font-bold">
                  {current.kind === "driver" && current.v.name}
                  {current.kind === "merchant" && current.v.name}
                  {current.kind === "order" && `#${current.v.number}`}
                  {current.kind === "zone" && current.v.name}
                  {current.kind === "request" && T.request.title}
                  {current.kind === "branch" && current.v.name}
                  {current.kind === "opportunity" && T.opportunity.title}
                  {current.kind === "customers" && T.customerCard.title}
                  {current.kind === "rep" && current.v.name}
                  {current.kind === "office" && (office.data?.name || T.office.title)}
                </h3>
                <button
                  className="ms-auto text-sm text-ink-muted"
                  onClick={() => setSelected(null)}
                >
                  {T.close}
                </button>
              </div>

              {current.kind === "driver" && (
                <>
                  <div className="mb-3 flex items-center gap-2 text-sm font-bold">
                    <Dot tone={driverTone(current.v.tone)} />
                    {T.tone[current.v.tone] ?? T.tone.idle}
                  </div>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.driver.status} v={current.v.status} />
                    <Row k={T.driver.onShift} v={current.v.on_shift ? T.yes : T.no} />
                    <Row k={T.driver.activeOrders} v={String(current.v.active_orders)} />
                    <Row
                      k={T.driver.lastSeen}
                      v={current.v.last_location_at ? fmtDateTime(current.v.last_location_at) : "—"}
                    />
                    <Row k={T.live} v={T.freshness[current.v.freshness]} />
                    {/* **والمالُ لمن يملك صلاحيّتَه وحدَه** (البند ٦). */}
                    {current.v.cash_held !== undefined && (
                      <Row k={T.driver.cashHeld} v={fmtMoney(current.v.cash_held)} />
                    )}
                  </dl>
                  <div className="mt-4 flex flex-wrap gap-2">
                    {/* **والهاتفُ لا يصل إلّا لمن يملك `users.contact.read`** — يحذفه الخادم. */}
                    {current.v.phone && <ContactButtons phone={current.v.phone} />}
                    {current.v.current_order && (
                      <Link href={`/dashboard/orders?id=${current.v.current_order}`}>
                        <Button variant="secondary">{T.order.openPage}</Button>
                      </Link>
                    )}
                    {canProfile && (
                      <Link href={`/dashboard/users?id=${current.v.id}`}>
                        <Button variant="secondary">{T.driver.openProfile}</Button>
                      </Link>
                    )}
                  </div>
                </>
              )}

              {current.kind === "merchant" && (
                <>
                  <div className="mb-3 flex items-center gap-2 text-sm font-bold">
                    <Dot tone={storeTone(current.v.open_now)} />
                    {current.v.open_now ? T.merchant.open : T.merchant.closed}
                  </div>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.merchant.activeOrders} v={String(current.v.active_orders)} />
                    <Row k={T.merchant.status} v={current.v.status} />
                    {current.v.city && <Row k={T.merchant.city} v={current.v.city} />}
                    {/* **والمندوبُ لمن يراقب نشاطَهم وحدَه** (البند ٩). */}
                    {current.v.rep && <Row k={T.merchant.rep} v={current.v.rep} />}
                  </dl>
                  <div className="mt-4 flex flex-wrap gap-2">
                    {current.v.phone && (
                      <a href={`tel:${current.v.phone}`}>
                        <Button variant="secondary">
                          <IconPhone size={16} aria-hidden />
                          {T.contact.call}
                        </Button>
                      </a>
                    )}
                    <Link href={`/dashboard/sections?merchant=${current.v.id}`}>
                      <Button variant="secondary">{T.merchant.openPage}</Button>
                    </Link>
                  </div>
                </>
              )}

              {current.kind === "order" && (
                <>
                  {current.v.stuck_reason && (
                    <div className="mb-3 flex items-center gap-2 text-sm font-bold text-danger">
                      <Dot tone="order-stuck" />
                      {T.orderCard.stuck} ·{" "}
                      {m.admin.ordersPage.alertReasons[
                        current.v.stuck_reason as keyof typeof m.admin.ordersPage.alertReasons
                      ] ?? current.v.stuck_reason}
                    </div>
                  )}
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.order.state} v={current.v.status} />
                    <Row k={T.order.merchant} v={current.v.merchant ?? "—"} />
                    {/* **واسمُ الزبون لمن يملك تفاصيلَه وحدَه** — يحذفه الخادم. */}
                    {current.v.customer_name && (
                      <Row k={T.orderCard.customer} v={current.v.customer_name} />
                    )}
                    <Row k={T.order.driver} v={current.v.driver ?? T.order.noDriver} />
                    {current.v.address && <Row k={T.order.area} v={current.v.address} />}
                    <Row
                      k={T.orderCard.age}
                      v={T.orderCard.minutes.replace("{n}", fmtNum(minutesSince(current.v.created_at)))}
                    />
                    <Row k={T.order.createdAt} v={fmtDateTime(current.v.created_at)} />
                  </dl>
                  {/* **كم مضى على كلّ مرحلة** — المراحلُ الواقعةُ وحدَها. */}
                  <OrderSteps order={current.v} />
                  <dl className="mt-2 flex flex-col gap-2 text-sm">
                    {/* **والمالُ بصلاحيّته** (البند ١٠). */}
                    {current.v.total !== undefined && (
                      <Row k={T.order.total} v={fmtMoney(current.v.total)} />
                    )}
                    {current.v.delivery_fee !== undefined && (
                      <Row k={T.order.delivery} v={fmtMoney(current.v.delivery_fee)} />
                    )}
                    {current.v.payment_method && (
                      <Row k={T.order.payment} v={current.v.payment_method} />
                    )}
                  </dl>
                  <div className="mt-4 flex flex-wrap gap-2">
                    {/* **والإسنادُ بمساره القائم** (`AssignDialog`) — بالقرب
                        وبتأكيدٍ وسبب، **ولمن يملك التدخّل وحدَه.** */}
                    {canIntervene &&
                      !current.v.driver_id &&
                      (current.v.status === "preparing" || current.v.status === "dispatching") && (
                        <Button onClick={() => setAssigning(current.v)}>
                          {m.admin.ordersPage.assignHere}
                        </Button>
                      )}
                    <Link href={`/dashboard/orders?id=${current.v.id}`}>
                      <Button variant="secondary">{T.order.openPage}</Button>
                    </Link>
                    {canProfile && current.v.driver_id && (
                      <Link href={`/dashboard/users?id=${current.v.driver_id}`}>
                        <Button variant="secondary">{T.driver.openProfile}</Button>
                      </Link>
                    )}
                  </div>
                </>
              )}

              {current.kind === "customers" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.customerCard.count} v={fmtNum(current.v.count)} />
                  </dl>
                  <p className="mt-3 text-xs leading-5 text-ink-muted">
                    {T.customerCard.privacy.replace("{n}", fmtNum(customers.data?.min_count ?? 3))}
                  </p>
                </>
              )}

              {current.kind === "rep" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.rep.merchants} v={fmtNum(current.v.merchants)} />
                    <Row k={T.rep.converted} v={fmtNum(current.v.converted)} />
                    {current.v.last_activity && (
                      <Row k={T.rep.lastActivity} v={fmtDateTime(current.v.last_activity)} />
                    )}
                    {current.v.earnings !== undefined && (
                      <Row k={T.rep.earnings} v={fmtMoney(current.v.earnings)} />
                    )}
                  </dl>
                  <p className="mt-3 text-xs text-ink-muted">{T.repCard.where}</p>
                </>
              )}

              {current.kind === "office" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.office.address} v={office.data?.address || T.office.noAddress} />
                  </dl>
                  <h4 className="mb-1 mt-4 text-sm font-bold">{T.office.online}</h4>
                  <p className="mb-2 text-xs text-ink-muted">
                    {T.office.window.replace("{n}", fmtNum(office.data?.presence_min ?? 5))}
                  </p>
                  {(office.data?.staff ?? []).length === 0 ? (
                    <p className="text-sm text-ink-muted">{T.office.noneOnline}</p>
                  ) : (
                    <ul className="flex flex-col gap-1.5 text-sm">
                      {(office.data?.staff ?? []).map((s, i) => (
                        <li key={`${s.name}-${i}`} className="flex items-baseline gap-2">
                          <span
                            className="inline-block h-2 w-2 shrink-0 rounded-full"
                            style={{ background: toneColor("driver-available") }}
                          />
                          <span className="font-medium">{s.name}</span>
                          <span className="ms-auto text-xs text-ink-muted">
                            {s.roles.map((r) => roleLabelByCode(r)).join(" · ")}
                          </span>
                        </li>
                      ))}
                    </ul>
                  )}
                </>
              )}

              {current.kind === "zone" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row
                      k={T.coverage.shape}
                      v={current.v.shape === "polygon" ? T.coverage.polygon : T.coverage.radius}
                    />
                    <Row k={T.coverage.active} v={current.v.active ? T.yes : T.no} />
                    {current.v.city && <Row k={T.coverage.city} v={current.v.city} />}
                    {current.v.shape === "radius" && (
                      <Row k={T.coverage.radius} v={`${current.v.radius_m} ${T.coverage.metres}`} />
                    )}
                  </dl>
                  {/* **والمنطقةُ تُوقَف ولا تُحذف** — منطقةٌ حُذفت تترك
                      طلباتٍ تشير إلى عدم. */}
                  {can("MANAGE_COVERAGE") && (
                    <div className="mt-4 flex flex-wrap gap-2">
                      <Button
                        variant="secondary"
                        onClick={() => setZoneActive(current.v.id, !current.v.active)}
                      >
                        {current.v.active ? T.coverage.disable : T.coverage.enable}
                      </Button>
                    </div>
                  )}
                  <p className="mt-3 text-xs text-ink-muted">{T.coverage.deleteHint}</p>
                </>
              )}

              {current.kind === "request" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row
                      k={T.request.status}
                      v={T.request.state[current.v.status as keyof typeof T.request.state]}
                    />
                    <Row k={T.request.createdAt} v={fmtDateTime(current.v.created_at)} />
                    {current.v.address && <Row k={T.request.address} v={current.v.address} />}
                    {current.v.city && <Row k={T.request.city} v={current.v.city} />}
                    <Row k={T.request.source} v={current.v.source} />
                    {current.v.note && <Row k={T.request.note} v={current.v.note} />}
                  </dl>
                  {can("MANAGE_COVERAGE") && (
                    <div className="mt-4">
                      <Select
                        label={T.request.changeStatus}
                        value={current.v.status}
                        onChange={(e) => setRequestStatus(current.v.id, e.target.value)}
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
              {current.kind === "branch" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row
                      k={T.branch.type}
                      v={current.v.type === "primary" ? T.branch.primary : T.branch.sub}
                    />
                    <Row k={T.branch.status} v={current.v.status} />
                    {current.v.city && <Row k={T.branch.city} v={current.v.city} />}
                    {current.v.parent && <Row k={T.branch.parent} v={current.v.parent} />}
                    <Row k={T.layer.areas} v={String(current.v.areas)} />
                  </dl>
                  <p className="mt-3 text-xs text-ink-muted">{T.branch.onePrimary}</p>
                </>
              )}
              {current.kind === "opportunity" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.opportunity.score} v={String(current.v.score)} />
                    <Row k={T.layer.orders} v={String(current.v.orders)} />
                    <Row k={T.request.count} v={String(current.v.requests)} />
                    <Row k={T.layer.merchants} v={String(current.v.merchants)} />
                    <Row k={T.layer.drivers} v={String(current.v.drivers)} />
                  </dl>
                  {/* **ولا درجةَ بلا سببٍ مسمّى** — والأرقامُ الخامُّ
                      فوقها، فمن لم يقبل الوزنَ حسب بنفسه. */}
                  <ul className="mt-3 flex flex-col gap-1 text-xs text-ink-muted">
                    {current.v.reasons.map((why) => (
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

        {/* ── المرشِّحاتُ والقوائم — تغيب في ملء الشاشة ──────────────── */}
        <div className={full ? "hidden" : "order-2 flex flex-col gap-4 lg:order-1"}>
          {/* ── محرِّرُ التغطية (البند ١٣) — حين يُرسَم وحدَه ──────── */}
          {can("MANAGE_COVERAGE") && drawing && (
            <Card>
              <h3 className="mb-2 text-sm font-bold">{T.coverage.title}</h3>
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
              <p className="mt-3 text-xs leading-5 text-ink-muted">
                {T.coverage.legacyHint}
              </p>
            </Card>
          )}

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
          {(can("VIEW_DEMAND_ANALYTICS") || can("VIEW_REP_ACTIVITY")) &&
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
                      style={{ background: toneColor("driver-idle") }}
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
      </div>
      </div>
      </div>

      {assigning && (
        <AssignDialog
          orderId={assigning.id}
          orderNumber={assigning.number}
          onClose={() => setAssigning(null)}
          onDone={() => {
            setAssigning(null);
            setSelected(null);
            orders.reload();
            summary.reload();
          }}
        />
      )}

      {/* ══════════════════════════════════════════════════════════════
          **وطلبُ التوسّع بالمكان الإداريّ** (`CR`، ٢٠٢٦-٠٩-١٤)
          ══════════════════════════════════════════════════════════════

          **والخريطةُ فوقُ تقول «أين» بالخلايا** — **وهذا يقول «في أيّ
          مدينةٍ ومحافظة»**، وهو سؤالُ التوسّع الأوّل.

          **وبالقدرة عينِها التي تحرس طلباتِ التغطية** — ولا قدرةَ
          جديدة. */}
      {can("VIEW_DEMAND_ANALYTICS") && (
        <div className="mt-4">
          <PlaceDemandPanel />
        </div>
      )}
    </PageContainer>
  );
}

/** **نقطةُ لون حالٍ** — من الموضع الواحد (`palette.ts`). */
function Dot({ tone }: { tone: MapTone }) {
  return (
    <span
      aria-hidden
      className="inline-block h-2.5 w-2.5 shrink-0 rounded-full"
      style={{ background: toneColor(tone) }}
    />
  );
}

/** **اسمُ نوع نتيجة البحث** — من أسماء الطبقات نفسِها. */
function kindLabel(kind: string): string {
  const key =
    kind === "driver" ? "drivers"
      : kind === "merchant" ? "merchants"
        : kind === "order" ? "orders"
          : kind === "branch" ? "branches"
            : kind === "area" ? "areas"
              : "reps";
  return T.layer[key as keyof typeof T.layer];
}

/** **مفتاحُ تركيزٍ** — حبّةٌ صغيرةٌ بجوار العدّادات. */
function FocusToggle({
  on,
  label,
  hint,
  tone,
  onClick,
}: {
  on: boolean;
  label: string;
  hint: string;
  tone: MapTone;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={on}
      title={hint}
      onClick={onClick}
      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-bold ${
        on ? "border-primary bg-primary-tint text-ink" : "border-line bg-surface text-ink-muted hover:text-ink"
      }`}
    >
      <Dot tone={tone} />
      {label}
    </button>
  );
}

/** **زرُّ أداةٍ فوق الخريطة** — مربّعٌ صغيرٌ باسمه في التلميح. */
function MapTool({ label, onClick, children }: { label: string; onClick: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={onClick}
      className="grid h-9 w-9 place-items-center rounded-control border border-line bg-paper text-on-bright shadow-lift"
    >
      {children}
    </button>
  );
}

/** **خطُّ زمن الطلب** — كم دقيقةً مضت على كلّ مرحلةٍ وقعت. */
function OrderSteps({ order }: { order: OrderPin }) {
  const steps = orderTimeline(order, Date.now());
  if (steps.length === 0) return null;
  return (
    <div className="mt-3">
      <h4 className="mb-1.5 text-xs font-bold text-ink-muted">{T.timeline.title}</h4>
      <ol className="flex flex-col gap-1.5 border-s-2 border-line ps-3 text-sm">
        {steps.map((st) => (
          <li key={st.key} className="flex items-baseline gap-2">
            <span>{T.timeline[st.key]}</span>
            <span className="ms-auto text-xs tabular-nums text-ink-muted">
              {T.timeline.ago.replace("{n}", fmtNum(st.minutes))}
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}

/** **نقطةُ المكتب** — دائرةٌ بيضاءُ كعلامته على الخريطة. */
function OfficeDot() {
  return (
    <span
      aria-hidden
      className="inline-block h-2.5 w-2.5 shrink-0 rounded-full border border-line bg-paper"
    />
  );
}

/** **عدّادٌ في الشريط** — رقمٌ ولونُه واسمُه. */
function Counter({ n, label, tone }: { n: number; label: string; tone: MapTone }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <Dot tone={tone} />
      <span className="font-bold tabular-nums">{fmtNum(n)}</span>
      <span className="text-ink-muted">{label}</span>
    </span>
  );
}

/** **رمزُ حالٍ كما يُرسم على الخريطة** — دائرةٌ ورمزٌ أبيض. */
function ToneIcon({ tone }: { tone: MapTone }) {
  const Glyph = MAP_TONES[tone].glyph;
  return (
    <span
      aria-hidden
      className="grid h-6 w-6 shrink-0 place-items-center rounded-full border-2 border-paper"
      style={{ background: toneColor(tone) }}
    >
      <Glyph size={13} color={themeColor("on-solid")} strokeWidth={2.25} />
    </span>
  );
}

/** **دليلُ الألوان** — يقرأ الموضعَ الواحدَ الذي ترسم منه الخريطة. */
function Legend() {
  const L = T.legendItem;
  const rows: Array<[MapTone, string]> = [
    ["driver-available", L.driverAvailable],
    ["driver-busy", L.driverBusy],
    ["driver-idle", L.driverIdle],
    ["order", L.order],
    ["order-stuck", L.orderStuck],
    ["store-open", L.storeOpen],
    ["store-closed", L.storeClosed],
    ["customer", L.customer],
    ["rep", L.rep],
  ];
  return (
    <div className="flex flex-col gap-2">
      <h4 className="text-sm font-bold">{T.bar.legendTitle}</h4>
      <ul className="flex flex-col gap-1.5 text-xs">
        {rows.map(([tone, label]) => (
          <li key={tone} className="flex items-center gap-2">
            <ToneIcon tone={tone} />
            <span>{label}</span>
          </li>
        ))}
        <li className="flex items-center gap-2">
          <span aria-hidden className="grid h-6 w-6 shrink-0 place-items-center">
            <span className="block w-5 border-t-2 border-dashed" style={{ borderColor: toneColor("order") }} />
          </span>
          <span>{L.route}</span>
        </li>
        <li className="flex items-center gap-2">
          <span
            aria-hidden
            className="h-6 w-6 shrink-0 rounded-full border-2"
            style={{ borderColor: toneColor("driver-available") }}
          />
          <span>{L.live}</span>
        </li>
        <li className="flex items-center gap-2">
          <span
            aria-hidden
            className="h-6 w-6 shrink-0 rounded-full border-4"
            style={{ background: toneColor("store-open"), borderColor: toneColor("store-closed") }}
          />
          <span>{L.storeCluster}</span>
        </li>
        <li className="flex items-center gap-2">
          <span aria-hidden className="grid h-6 w-6 shrink-0 place-items-center">
            <span className="block w-5 border-t-2 border-dashed" style={{ borderColor: toneColor("driver-busy") }} />
          </span>
          <span>{L.driverLeg}</span>
        </li>
        <li className="flex items-center gap-2">
          <span aria-hidden className="h-6 w-6 shrink-0 rounded-full border-2 border-line bg-paper" />
          <span>{L.office}</span>
        </li>
      </ul>
      <p className="text-xs leading-5 text-ink-muted">{T.bar.coverageAlways}</p>
    </div>
  );
}

/** **زرّا الاتّصال وواتساب** — والهاتفُ لا يصل إلّا لمن يملكه. */
function ContactButtons({ phone }: { phone: string }) {
  return (
    <>
      <a href={`tel:${phone}`}>
        <Button variant="secondary">
          <IconPhone size={16} aria-hidden />
          {T.contact.call}
        </Button>
      </a>
      <a href={waLink(phone)} target="_blank" rel="noopener noreferrer">
        <Button variant="secondary">
          <IconWhatsApp size={16} aria-hidden />
          {T.contact.whatsapp}
        </Button>
      </a>
    </>
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
