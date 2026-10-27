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
 *
 * # والتصميمُ الاحترافيّ (تدقيقُ ٢٠٢٦-١٠-٢٧ — «بدي اللوحة كلها احترافية»)
 *
 * **رأسٌ موحّد (`PageHero`) · أربعةُ أرقامٍ تُضغط (`Tile`)** — والتركيزُ فيها لا في حبّاتٍ
 * وحدَها · **سطحُ تحكّمٍ واحد**: البحثُ وحبّاتُ الطبقات الإضافيّة بعددها والمدّةُ وساعاتُ اليوم
 * حبّاتٌ لا قوائم · **فلترُ الرابط شارةٌ تُرى وتُشال** · بطاقةُ المُحدَّد برأسٍ ملوّنٍ ورقمٍ كبيرٍ
 * وأزرارٍ بعرضها، **ولوحةٌ سفليّةٌ على الجوّال** · وكلُّ فعلٍ بزرٍّ مقفولٍ وقتَ الإرسال
 * ورسالةِ خطأٍ ونجاح، **وتأكيدٌ قبل الرفض والإيقاف.**
 */

import { useCallback, useEffect, useMemo, useRef, useState, type ComponentType, type ReactNode } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtDateTime, fmtMoney, fmtNum, errorText } from "@rahalgo/i18n";
import {
  PageContainer,
  Button,
  LoadingState,
  ReloadState,
  Alert,
  Badge,
  Chips,
  Input,
  IconPhone,
  IconFullscreen,
  IconFullscreenExit,
  IconSearch,
  IconTarget,
  IconWhatsApp,
  IconLocation,
  IconLock,
  IconMoto,
  IconOrder,
  IconStore,
  IconWarning,
  IconZones,
  IconUser,
  IconRoute,
  IconBlock,
  IconStar,
  IconTrendUp,
  IconClose,
  IconOverview,
  useLiveData,
  useChime,
  useToast,
  alertsSoundOn,
  themeColor,
  useLiveEvent,
} from "@rahalgo/ui";
import { AssignDialog } from "@/components/admin/orders/AssignDialog";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";
import { api, mediaUrl, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall, useCanOpen } from "@/lib/policy";
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
const P = T.pro;

/**
 * **كلُّ قيمةٍ من الخادم تُترجم قبل أن تُعرض** (تدقيقُ ٢٠٢٦-١٠-٢٧: «حالُ الحساب: active») —
 * من القواميس القائمة، **والمجهولُ «—» لا كلمةٌ إنكليزيّة.**
 */
function pick(dict: Record<string, string>, key: string | undefined | null): string {
  return (key && dict[key]) || P.unknown;
}
const ORDER_STATUS = m.admin.ordersPage.board.status as Record<string, string>;
const ACCOUNT_STATUS: Record<string, string> = {
  active: m.terms.active,
  suspended: m.terms.suspended,
  blocked: m.terms.blocked,
};
const MERCHANT_STATUS: Record<string, string> = {
  active: m.admin.merchants.active,
  inactive: m.admin.merchants.inactive,
  suspended: m.admin.merchants.suspended,
};
const PAYMENT = m.orders.payment as Record<string, string>;
const BRANCH_STATUS: Record<string, string> = { active: P.branchActive, inactive: P.branchInactive };
function sourceLabel(s: string): string {
  if (s === "customer_app") return P.sourceCustomerApp;
  if (s === "web" || s === "shop" || s === "pwa") return P.sourceWeb;
  return P.sourceOther;
}

/** **أجزاءُ اليوم بساعاتها** — من الأولى حتّى قبل الثانية، والليلُ يعبر منتصفَ الليل. */
const DAY_PARTS = {
  morning: [6, 12],
  noon: [12, 17],
  evening: [17, 22],
  night: [22, 6],
} as const;

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
  /** **موضعُ الزبون الحيُّ إن شاركه** — الأحدثُ من دقيقتين (٢٠٢٦-١٠-١٩). */
  customer_live?: { lat: number; lng: number; accuracy_m?: number; at: string; pin_gap_m: number };
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

/** **طريقٌ مسكّرٌ بلّغ عنه سائق** (قرارُ المالك ٢٠٢٦-١٠-٢١). */
interface Closure {
  id: number;
  lat: number;
  lng: number;
  reason: string;
  reports: number;
  created_at: string;
  expires_at: string;
  reported_by_name: string;
}

/** **معلمٌ اقترحه سائقٌ وينتظر الموافقة** (قرارُ المالك ٢٠٢٦-١٠-٢١). */
interface LandmarkSugg {
  id: number;
  name: string;
  kind: string;
  lat: number;
  lng: number;
  created_at: string;
  suggested_by_name: string;
}

/** **ما هو المُحدَّد؟** — واللوحةُ الجانبيّةُ واحدةٌ لكلّ الطبقات. */
type Picked =
  | { kind: "driver"; v: Driver }
  | { kind: "merchant"; v: Merchant }
  | { kind: "order"; v: OrderPin }
  | { kind: "zone"; v: Zone }
  | { kind: "request"; v: CovRequest }
  | { kind: "branch"; v: BranchPin }
  | { kind: "closure"; v: Closure }
  | { kind: "landmark"; v: LandmarkSugg }
  | { kind: "opportunity"; v: OpportunityCell }
  | { kind: "customers"; v: DemandCell }
  | { kind: "rep"; v: RepRow }
  | { kind: "office" };

/** **ما يُرى أوّلَ فتحة** — والمستخدمُ يبدّله فيُحفظ له. */
const DEFAULT_VISIBLE: Record<string, boolean> = {
  drivers: true,
  orders: true,
  merchants: true,
  customers: true,
  reps: true,
  office: true,
};

/**
 * **الطبقاتُ الستُّ دائماً على الخريطة** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «لازم تكون خريطة عمليّات بالحقيقة…
 * السائقين والمتاجر والطلبات والزبائن والمناديب والمكتب دايماً موجودين — هالشريط ما بيلزم»).
 * **ولا شريطَ حبّاتٍ يطفئها** — ويبقى لكلٍّ قدرتُه (من لا يملك طبقةً لا تُجلَب له).
 */
const ALWAYS_ON: Record<string, boolean> = {
  drivers: true,
  orders: true,
  merchants: true,
  customers: true,
  reps: true,
  office: true,
};

/**
 * **القادمُ من «طلبات التوسّع» يرى طلباتِ التغطية** (`?focus=`) — وإلّا وصل إلى المكان
 * بلا علامةٍ تقول أين الطلب. **ولا يُحفظ** — فتحةٌ عاديّةٌ بعدها تعود كما اختار.
 */
function fromExpansion(): Record<string, boolean> {
  if (typeof window === "undefined") return {};
  return new URLSearchParams(window.location.search).get("focus") ? { requests: true } : {};
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
      setVisible({ ...(raw ? (JSON.parse(raw) as Record<string, boolean>) : DEFAULT_VISIBLE), ...ALWAYS_ON, ...fromExpansion() });
    } catch {
      /* تخزينٌ ممنوعٌ في نافذةٍ خاصّة — والافتراضُ يكفي */
      setVisible((v) => ({ ...v, ...fromExpansion() }));
    }
  }, [storeKey]);
  const toggle = useCallback(
    (id: string) => {
      setVisible((prev) => {
        const next = { ...prev, [id]: !prev[id], ...ALWAYS_ON };
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
  //
  // **وكانت تعمل بالخفاء** (تدقيقُ ٢٠٢٦-١٠-٢٧): لا شارةَ ولا زرَّ يشيلها. **فصارت شارةً فوق
  // الخريطة بـ✕** تمسحها من الحال ومن الرابط معاً.
  const [onShift, setOnShift] = useState<"" | "true" | "false">(() => tri("on_shift"));
  const [hasActive, setHasActive] = useState<"" | "true" | "false">(() => tri("has_active"));
  const [driverStatus, setDriverStatus] = useState(() =>
    typeof window === "undefined" ? "" : (new URLSearchParams(window.location.search).get("status") ?? ""),
  );
  const clearUrlFilter = useCallback(() => {
    setOnShift("");
    setHasActive("");
    setDriverStatus("");
    try {
      const u = new URL(window.location.href);
      for (const k of ["on_shift", "has_active", "status"]) u.searchParams.delete(k);
      window.history.replaceState(null, "", `${u.pathname}${u.search}${u.hash}`);
    } catch {
      /* رابطٌ لا يُكتب — والحالُ مُسحت */
    }
  }, []);
  const [selected, setSelected] = useState<Picked | null>(null);
  const [legendOpen, setLegendOpen] = useState(false);
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

  // ── المدى الزمنيُّ للتحليلات (البند ٢٩) ───────────────────
  //
  // **وثلاثون يوماً افتراضاً** — **ومدىً مفتوحٌ على كلّ التاريخ يقول
  // «هنا عملٌ» عن حيٍّ مات فيه العملُ منذ سنة.**
  const [range, setRange] = useState<"today" | "7d" | "30d">("30d");
  // **وساعاتُ اليوم** (قرارُ المالك ٢٠٢٦-١٠-٢١: «وين الطلبات بين ٨ و١٠ المسا») —
  // بتوقيت سوريا، **والليلُ يعبر منتصفَ الليل** (١٠ ← ٦). للكثافة والفرص وحدهما.
  const [dayPart, setDayPart] = useState<keyof typeof DAY_PARTS | "all">("all");
  const hoursQ =
    dayPart === "all" ? "" : `&hour_from=${DAY_PARTS[dayPart][0]}&hour_to=${DAY_PARTS[dayPart][1]}`;

  // ── البحثُ في الخريطة (البند ٣٦) ──────────────────────────
  //
  // **ابحث واذهب** — رقمُ طلبٍ أو اسمُ سائقٍ أو متجرٍ مما حُمِّل للخريطة،
  // **ثمّ ما يجده الخادمُ مما لم يُحمَّل** (فروعٌ ومناطقُ ومندوبون)، **وهو
  // يُقَصُّ بصلاحيّات الباحث** — ومن وجد اسمَ من لا يملك رؤيتَه عرف أنّه موجود.
  const [hunt, setHunt] = useState("");
  const [huntOpen, setHuntOpen] = useState(false);
  const [huntAt, setHuntAt] = useState(0);

  // ── ما يملكه من يقف أمامها ───────────────────────────────────
  //
  // **و٤٠٣ هنا «لا صلاحية» لا «تعذّر الوصول»** (تدقيقُ ٢٠٢٦-١٠-٢٧) — الخادمُ يردّ
  // `/meta` بـ٤٠٣ لمن لا يملك الخريطة، **فكان يرى خطأَ شبكةٍ وزرَّ إعادةٍ لا يفيد.**
  const meta = useLiveData<Meta>(
    () =>
      api<Meta>("/api/v1/admin/ops-map/meta").catch((e: unknown) => {
        if (e instanceof ApiError && e.status === 403) {
          return { permissions: [], location_ping_sec: 60, assignable_sec: 0 };
        }
        throw e;
      }),
    [],
  );
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
  //
  // **وأرضيّتُها عشرُ ثوانٍ لا ثلاثون** (طلبُ المالك ٢٠٢٦-١٠-١٧: «بدي موقع السائق يكون
  // مباشر») — **فنبضةُ ١٥ ثانيةً من الإعدادات تُرى بعد ١٥ ثانيةً لا بعد نصف دقيقة.**
  const pingSec = Math.max(10, meta.data?.location_ping_sec ?? 60);
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
    if (hasActive) q.set("has_active", hasActive);
    if (driverStatus) q.set("status", driverStatus);
    const s = q.toString();
    return s ? `?${s}` : "";
  }, [onShift, hasActive, driverStatus]);

  const drivers = useLiveData<{ drivers: Driver[]; count: number }>(
    () =>
      can("VIEW_DRIVER_LOCATIONS") && visible.drivers
        ? api<{ drivers: Driver[]; count: number }>(`/api/v1/admin/ops-map/drivers${driverQuery}`)
        : Promise.resolve({ drivers: [], count: 0 }),
    [],
    [driverQuery, visible.drivers, meta.data, tick],
  );

  // ══════════════════════════════════════════════════════════════
  // **والسائقُ يتحرّك حيّاً** (قرارُ المالك ٢٠٢٦-١٠-١٩: «الموقع المباشر»)
  // ══════════════════════════════════════════════════════════════
  //
  // **كلُّ نبضةٍ تُبثّ على `ops` بموضعها** (`driver_live.go`) — فيُحرَّك السائقُ في
  // القائمة نفسِها بلا سؤالٍ للخادم، **وتنزلقه الخريطةُ إلى موضعه الجديد.** والمدّةُ
  // فوق تبقى احتياطاً لمن انقطع بثُّه.
  const setDrivers = drivers.setData;
  useLiveEvent((e) => {
    if (e.type !== "driver_location") return;
    const id = String(e.driver_id ?? "");
    const lat = Number(e.lat);
    const lng = Number(e.lng);
    if (!id || !Number.isFinite(lat) || !Number.isFinite(lng)) return;
    setDrivers((prev) =>
      prev && prev.drivers.some((d) => d.id === id)
        ? {
            ...prev,
            drivers: prev.drivers.map((d) =>
              d.id === id
                ? { ...d, lat, lng, last_location_at: String(e.at ?? d.last_location_at), age_sec: 0, freshness: "LIVE" }
                : d,
            ),
          }
        : prev,
    );
  });

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

  // **وموضعُ الزبون الحيُّ يصل بالبثّ** (٢٠٢٦-١٠-١٩) — فتتحرّك علامتُه بلا سؤال.
  const setOrders = orders.setData;
  useLiveEvent((e) => {
    if (e.type !== "customer_location") return;
    const id = String(e.order_id ?? "");
    setOrders((prev) =>
      prev && prev.orders.some((o) => o.id === id)
        ? {
            ...prev,
            orders: prev.orders.map((o) => {
              if (o.id !== id) return o;
              if (e.stopped) return { ...o, customer_live: undefined };
              const lat = Number(e.lat);
              const lng = Number(e.lng);
              if (!Number.isFinite(lat) || !Number.isFinite(lng)) return o;
              return {
                ...o,
                customer_live: {
                  lat,
                  lng,
                  accuracy_m: e.accuracy_m == null ? undefined : Number(e.accuracy_m),
                  at: String(e.at ?? new Date().toISOString()),
                  pin_gap_m: metersBetween(lat, lng, o.drop_lat, o.drop_lng),
                },
              };
            }),
          }
        : prev,
    );
  });

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
            "/api/v1/admin/ops-map/coverage-requests",
          )
        : Promise.resolve({ requests: [], count: 0 }),
    [],
    [visible.requests, meta.data],
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

  // **بلاغاتُ السائقين للخريطة** (قرارات المالك ٢٠٢٦-١٠-٢١) — المسكَّرُ الآن والمعالمُ المنتظِرة.
  const closures = useLiveData<{ closures: Closure[]; count: number }>(
    () =>
      visible.closures
        ? api<{ closures: Closure[]; count: number }>("/api/v1/admin/ops-map/road-closures")
        : Promise.resolve({ closures: [], count: 0 }),
    ["road_closure"],
    [visible.closures, meta.data],
  );
  const landmarks = useLiveData<{ suggestions: LandmarkSugg[]; count: number }>(
    () =>
      visible.landmarks
        ? api<{ suggestions: LandmarkSugg[]; count: number }>("/api/v1/admin/ops-map/landmark-suggestions")
        : Promise.resolve({ suggestions: [], count: 0 }),
    ["landmark"],
    [visible.landmarks, meta.data],
  );

  // ── أفعالُ البطاقة — زرٌّ مقفولٌ وقتَ الإرسال · خطأٌ يُقال · نجاحٌ يمرّ ──
  //
  // **كانت تفشل بلا كلمة** (تدقيقُ ٢٠٢٦-١٠-٢٧): لا `try` ولا قفلَ زرّ — فيُنبعث الفعلُ
  // مرّتين، **والبطاقةُ تبقى كما هي بلا سبب.** **والرفضُ والإيقافُ يسألان أوّلاً.**
  const { push: toast, Toaster } = useToast();
  const [busy, setBusy] = useState(false);
  const [actErr, setActErr] = useState("");
  const [confirming, setConfirming] = useState<string | null>(null);
  useEffect(() => {
    setActErr("");
    setConfirming(null);
  }, [selected]);
  const run = useCallback(
    async (fn: () => Promise<unknown>, ok: string) => {
      setBusy(true);
      setActErr("");
      try {
        await fn();
        setConfirming(null);
        setSelected(null);
        toast(ok);
      } catch (e) {
        setActErr(errorText(e));
      } finally {
        setBusy(false);
      }
    },
    [toast],
  );

  // **انفتح الطريق · وقبولُ المعلم أو رفضُه** (قرارات المالك ٢٠٢٦-١٠-٢١).
  const clearClosure = useCallback(
    (id: number) =>
      run(async () => {
        await api(`/api/v1/admin/ops-map/road-closures/${id}`, { method: "DELETE" });
        closures.reload();
      }, P.doneClosure),
    [closures, run],
  );
  const decideLandmark = useCallback(
    (id: number, approve: boolean) =>
      run(async () => {
        await api(`/api/v1/admin/ops-map/landmark-suggestions/${id}/decide`, {
          method: "POST",
          body: JSON.stringify({ approve }),
        });
        landmarks.reload();
      }, approve ? P.doneApprove : P.doneReject),
    [landmarks, run],
  );

  // **ولا طبقةَ «مناطق تشغيليّة»** (تدقيقُ ٢٠٢٦-١٠-٢٧): كان مفتاحُها يجلب ويعدّ ولا يرسم
  // شيئاً — **ومفتاحٌ بلا أثرٍ يُقرأ عطلاً.** والمناطقُ تُدار من صفحتها.

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
        ? api<DemandOut>(`/api/v1/admin/ops-map/demand?range=${range}${hoursQ}`)
        : Promise.resolve({
            orders: [], requests: [], merchants: [], drivers: [],
            unserved: [], cell_deg: 0.01,
          }),
    [],
    [visible.demand, visible.opportunities, range, hoursQ, meta.data],
  );

  const opportunities = useLiveData<{ opportunities: OpportunityCell[]; count: number }>(
    () =>
      can("VIEW_DEMAND_ANALYTICS") && visible.opportunities
        ? api<{ opportunities: OpportunityCell[]; count: number }>(
            `/api/v1/admin/ops-map/opportunities?range=${range}${hoursQ}`,
          )
        : Promise.resolve({ opportunities: [], count: 0 }),
    [],
    [visible.opportunities, range, hoursQ, meta.data],
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
  const canCallFn = useCanCall();
  const canProfile = canCallFn("GET", "/users/{id}");
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
  //
  // **ومن هو برّا الدوام لا يُرسم افتراضاً** (تدقيقُ ٢٠٢٦-١٠-٢٧): كان كلُّ سائقٍ يُرسم
  // رماديّاً باسمه — الموقوفُ ومن لم يفتح الدوامَ أصلاً — **فتزدحم الخريطةُ بمن لا يعمل.**
  // **وحبّةٌ صريحةٌ تُظهرهم**، وفلترُ الرابط (`on_shift`) يحكم إن جاء.
  const driverPool = useMemo(() => {
    const all = drivers.data?.drivers ?? [];
    return onShift || visible.offShift ? all : all.filter((d) => d.on_shift);
  }, [drivers.data, onShift, visible.offShift]);
  const focused = useMemo(
    () =>
      applyFocus(
        focusMode,
        driverPool,
        orders.data?.orders ?? [],
        merchants.data?.merchants ?? [],
      ),
    [focusMode, driverPool, orders.data, merchants.data],
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
      // **واسمُه فوق رمزه** (٢٠٢٦-١٠-١٧) — ويتراكب، فلا يختفي سائقٌ لأنّ جاره قريب.
      labelField: "name",
      labelOverlap: true,
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
          .map((d) => pt(d.lng, d.lat, { id: d.id, name: d.name, icon: driverTone(d.tone), live: d.freshness === "LIVE" })),
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
      // **واسمُ المتجر فوق رمزه** (٢٠٢٦-١٠-١٧) — والمتزاحمةُ تُخفى حتّى يُكبَّر عليها.
      labelField: "name",
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
          pt(x.lng, x.lat, { id: x.id, name: x.name, icon: storeTone(x.open_now), open: x.open_now }),
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

    // ── الزبونُ بموضعه الحيّ (طلبُ المالك ٢٠٢٦-١٠-١٩) ─────────────
    //
    // **«لنقدر نعطي احداثيات للسائق»** — علامةٌ حيث هو الآن، **وخطٌّ متقطّعٌ إلى
    // دبّوسه** فيُرى إن كان الدبّوسُ خطأً.
    const liveOrders = orderList.filter((o) => o.customer_live);
    out.push({
      id: "customer-live",
      kind: "symbol",
      cluster: false,
      animate: true,
      order: 62,
      visible: !!visible.orders,
      color: toneColor("customer-live"),
      data: fc(
        liveOrders.map((o) =>
          pt(o.customer_live!.lng, o.customer_live!.lat, { id: o.id, icon: "customer-live" }),
        ),
      ),
    });
    out.push({
      id: "customer-live-gap",
      kind: "line",
      order: 56,
      lineWidth: 1.5,
      visible: !!visible.orders,
      color: toneColor("customer-live"),
      data: fc(
        liveOrders.map((o) => ({
          type: "Feature" as const,
          geometry: {
            type: "LineString",
            coordinates: [[o.customer_live!.lng, o.customer_live!.lat], [o.drop_lng, o.drop_lat]],
          },
          properties: {},
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
      id: "closures",
      kind: "point",
      order: 21,
      visible: !!visible.closures,
      color: themeColor("danger"),
      data: fc((closures.data?.closures ?? []).map((c) => pt(c.lng, c.lat, { id: c.id }))),
    });
    out.push({
      id: "landmarks",
      kind: "point",
      order: 21,
      visible: !!visible.landmarks,
      color: themeColor("accent"),
      data: fc((landmarks.data?.suggestions ?? []).map((l) => pt(l.lng, l.lat, { id: l.id, name: l.name }))),
    });
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
      visible, current, draft, drawing, emergencies.data, closures.data, landmarks.data]);

  // ── المكتبُ علامةٌ بشعار المنصّة ─────────────────────────────
  //
  // **والمكانُ المطلوبُ من «طلبات التوسّع» علامةٌ ظاهرة** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كان «اعرض على
  // الخريطة» يصل إلى المكان بلا ما يدلّ عليه.
  const markers = useMemo<MapMarker[]>(() => {
    const out: MapMarker[] = [];
    if (focus) {
      out.push({ id: "focus", lng: focus.lng, lat: focus.lat, letter: P.focusLetter, label: P.focusPin });
    }
    const o = office.data;
    if (!visible.office || !o || !o.has_location || o.lat == null || o.lng == null) return out;
    out.push({
      id: "office",
      lng: o.lng,
      lat: o.lat,
      imageUrl: mediaUrl(o.logo),
      letter: o.name.trim().charAt(0),
      label: T.office.title,
    });
    return out;
  }, [office.data, visible.office, focus]);

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
      if (layerID === "closures") {
        const c = closures.data?.closures.find((y) => y.id === props.id);
        if (c) setSelected({ kind: "closure", v: c });
      }
      if (layerID === "landmarks") {
        const l = landmarks.data?.suggestions.find((y) => y.id === props.id);
        if (l) setSelected({ kind: "landmark", v: l });
      }
      if (layerID === "opportunities") {
        const o = (opportunities.data?.opportunities ?? []).find(
          (y) => y.lat === props.lat && y.lng === props.lng,
        );
        if (o) setSelected({ kind: "opportunity", v: o });
      }
    },
    [drivers.data, merchants.data, orders.data, customers.data, reps.data, zones.data,
     requests.data, branches.data, opportunities.data, router, drawing,
     closures.data, landmarks.data],
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
          // **ومن هو برّا الدوام تُظهَر طبقتُه** — وإلّا طار إلى مكانٍ لا رمزَ فيه.
          if (!d.on_shift && !onShift && !visible.offShift) toggle("offShift");
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
    [orders.data, drivers.data, merchants.data, focused, onShift, visible.offShift, toggle],
  );

  // ── «رجّع على المكتب» ──────────────────────────────────────────
  const officeAt = useMemo(() => {
    const o = office.data;
    return o?.has_location && o.lat != null && o.lng != null ? { lng: o.lng, lat: o.lat } : null;
  }, [office.data]);
  const recenter = useCallback(() => {
    if (officeAt) setFly({ lng: officeAt.lng, lat: officeAt.lat, zoom: 14, key: Date.now() });
  }, [officeAt]);

  const [saving, setSaving] = useState(false);
  const savePolygon = useCallback(async () => {
    setSaveErr("");
    setSaving(true);
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
      toast(P.drawSaved);
    } catch (e) {
      setSaveErr(errorText(e));
    } finally {
      setSaving(false);
    }
  }, [zoneName, draft, zones, toast]);

  const setZoneActive = useCallback(
    (id: string, active: boolean) =>
      run(async () => {
        await api(`/api/v1/admin/ops-map/coverage/${id}/active`, {
          method: "POST",
          body: JSON.stringify({ active }),
        });
        zones.reload();
      }, active ? P.doneZoneOn : P.doneZoneOff),
    [zones, run],
  );

  const setRequestStatus = useCallback(
    (id: string, status: string) =>
      run(async () => {
        await api(`/api/v1/admin/ops-map/coverage-requests/${id}`, {
          method: "PATCH",
          body: JSON.stringify({ status }),
        });
        requests.reload();
      }, P.doneRequest),
    [requests, run],
  );

  // ── ما يُفتح لصاحب الجلسة — قبل أيّ رجوعٍ مبكّر (قواعدُ الخطّافات) ──
  const canOpen = useCanOpen();
  const canDriversList = canCallFn("GET", "/users");

  // ── الحالاتُ الحدّيّة (البند ٣٩) ─────────────────────────────
  if (meta.loading) return <LoadingState />;
  if (meta.error) {
    return (
      <PageContainer>
        <ReloadState onRetry={meta.reload} label={T.networkError} />
      </PageContainer>
    );
  }
  if (!can("VIEW_OPERATIONS_MAP")) {
    return (
      <PageContainer>
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-field text-ink-muted">
            <IconLock size={28} />
          </span>
          <h2 className="heading-section">{T.noPermission}</h2>
        </section>
      </PageContainer>
    );
  }

  // **ومن لا موضعَ له وهو على الدوام يُقال** — **ومن هو برّا الدوام لا يُسأل عن موضعه**
  // (تدقيقُ ٢٠٢٦-١٠-٢٧: كانت القائمةُ تقول «غالباً النظامُ أوقف التطبيق» عمّن لم يفتح الدوام).
  const noLoc = (drivers.data?.drivers ?? []).filter((d) => d.on_shift && d.freshness === "NO_LOCATION");
  const S = summary.data;
  const stuckN = S?.orders?.stuck ?? 0;

  // **حدودُ التغطية** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «الخريطة تطلع بس منطقة تغطيتنا») — من المناطق الفعّالة:
  // المضلَّعُ بنقاطه، والدائرةُ بمركزها ونصف قطرها. وبلا مناطق لا حدود.
  // **والمكانُ المطلوبُ داخلَها دائماً** (تدقيقُ ٢٠٢٦-١٠-٢٧): طلبُ التوسّع غالباً برّا التغطية،
  // **فكانت الكاميرا ترجع وحدَها إلى الداخل ولا يُرسم حيث يجب.**
  const coverageBounds = (() => {
    let w = Infinity, so = Infinity, e = -Infinity, n = -Infinity;
    const add = (lng: number, lat: number) => {
      if (!Number.isFinite(lng) || !Number.isFinite(lat)) return;
      w = Math.min(w, lng); e = Math.max(e, lng); so = Math.min(so, lat); n = Math.max(n, lat);
    };
    const walk = (c: unknown) => {
      if (Array.isArray(c) && c.length >= 2 && typeof c[0] === "number" && typeof c[1] === "number") {
        add(c[0], c[1]);
      } else if (Array.isArray(c)) c.forEach(walk);
    };
    for (const z of zones.data?.zones ?? []) {
      if (!z.active) continue;
      if (z.shape === "polygon" && z.area) walk(z.area.coordinates);
      else if (z.radius_m > 0) {
        const dLat = z.radius_m / 111_320;
        const dLng = z.radius_m / (111_320 * Math.cos((z.lat * Math.PI) / 180) || 1);
        add(z.lng - dLng, z.lat - dLat);
        add(z.lng + dLng, z.lat + dLat);
      }
    }
    if (focus && Number.isFinite(w)) {
      // **هامشٌ حول المكان المطلوب** — كيلومترٌ تقريباً، فيُرسم حولَه.
      add(focus.lng - 0.01, focus.lat - 0.01);
      add(focus.lng + 0.01, focus.lat + 0.01);
    }
    return Number.isFinite(w) ? ([w, so, e, n] as [number, number, number, number]) : undefined;
  })();

  // ── الطبقةُ الفارغةُ تُقال لا تُسكَت ───────────────────────────────
  //
  // **خريطةٌ فارغةٌ بلا كلمةٍ تُقرأ عطلاً** — «لا سائقين على الدوام الآن»
  // تُقرأ حقيقة. **ولا تُقال قبل أن تصل البيانات** — فلا يُكذَب أثناءَ التحميل.
  const urlFiltered = !!(onShift || hasActive || driverStatus);
  const emptyHints: string[] = [];
  let driversEmpty = false;
  // **وفي «العالقة فقط» بلا عالقٍ تكفي جملةُ الطلبات** — لا جملتان لفراغٍ واحد.
  const noStuckAtAll = focusMode === "stuck" && focused.orders.length === 0;
  if (can("VIEW_DRIVER_LOCATIONS") && visible.drivers && drivers.data && !noStuckAtAll &&
      focused.drivers.filter((d) => d.freshness !== "NO_LOCATION").length === 0) {
    driversEmpty = focusMode === "";
    emptyHints.push(
      focusMode === "available" ? T.emptyHint.available
        : focusMode === "stuck" ? T.emptyHint.stuckDrivers
          : urlFiltered ? T.emptyHint.driversFiltered : T.emptyHint.drivers,
    );
  }
  if (can("VIEW_ACTIVE_ORDERS") && visible.orders && orders.data && focused.orders.length === 0) {
    emptyHints.push(focusMode === "stuck" ? T.emptyHint.stuck : T.emptyHint.orders);
  }
  if (can("VIEW_MERCHANT_LOCATIONS") && visible.merchants && merchants.data &&
      focused.merchants.length === 0 && focusMode !== "stuck") {
    emptyHints.push(T.emptyHint.merchants);
  }

  // ── شارةُ فلتر الرابط — ما جاء من الرئيسيّة يُرى ويُشال ─────────────
  const urlBits: string[] = [];
  if (hasActive === "true") urlBits.push(P.urlBusy);
  else if (hasActive === "false") urlBits.push(P.urlFree);
  else if (onShift === "true") urlBits.push(P.urlOnShift);
  if (onShift === "false") urlBits.push(P.urlOffShift);
  if (driverStatus && driverStatus !== "active") {
    urlBits.push(P.urlStatus.replace("{s}", pick(ACCOUNT_STATUS, driverStatus)));
  }
  if (urlBits.length === 0 && urlFiltered) urlBits.push(P.urlOnShift);

  // ── حبّاتُ الطبقات الإضافيّة — بعددها، والعددُ عددُ ما على الخريطة ────────
  //
  // **وعددُ الكثافة عددُ الطلبات لا المربّعات** (تدقيقُ ٢٠٢٦-١٠-٢٧: «٢» وهي أربعةُ طلبات).
  const demandOrders = (demand.data?.orders ?? []).reduce((sum, c) => sum + c.count, 0);
  const offShiftN = (drivers.data?.drivers ?? []).filter((d) => !d.on_shift).length;
  const layerChips: Array<{ id: string; label: string; count?: number }> = [];
  if (can("VIEW_DRIVER_LOCATIONS") && !onShift) {
    layerChips.push({ id: "offShift", label: P.offShiftLayer, count: drivers.data ? offShiftN : undefined });
  }
  if (canEmergencies) {
    layerChips.push({ id: "emergencies", label: m.admin.emergencyRoom.mapLayer, count: emergencies.data?.count });
  }
  if (can("VIEW_DEMAND_ANALYTICS")) {
    layerChips.push(
      { id: "demand", label: T.layer.demand, count: visible.demand ? demandOrders : undefined },
      { id: "opportunities", label: T.layer.opportunities, count: visible.opportunities ? opportunities.data?.count : undefined },
      { id: "requests", label: T.layer.coverageRequests, count: visible.requests ? requests.data?.count : undefined },
    );
  }
  layerChips.push(
    { id: "closures", label: T.layer.closures, count: visible.closures ? closures.data?.count : undefined },
    { id: "landmarks", label: T.layer.landmarks, count: visible.landmarks ? landmarks.data?.count : undefined },
    { id: "branches", label: T.layer.branches, count: visible.branches ? branches.data?.count : undefined },
  );
  const layerOn = layerChips.filter((c) => visible[c.id]).map((c) => c.id);

  const rangeName = range === "today" ? T.demandWhen.today : range === "7d" ? T.demandWhen.d7 : T.demandWhen.d30;
  const maxScore = Math.max(1, ...(opportunities.data?.opportunities ?? []).map((o) => o.score));

  return (
    <PageContainer>
      <PageHero
        icon={IconLocation}
        title={T.title}
        subtitle={T.subtitle}
        actions={
          <Button variant="secondary" onClick={() => setFull(true)}>
            <IconFullscreen size={16} aria-hidden />
            {T.bar.fullscreen}
          </Button>
        }
      />

      {/* ══ الأرقامُ الأربعة — تُضغط، والتركيزُ فيها ══════════════════ */}
      {(S?.drivers || S?.orders || S?.merchants) && (
        <div className="mb-4 grid grid-cols-2 gap-2 lg:grid-cols-4">
          {S?.drivers && (
            <Tile
              icon={IconMoto}
              label={P.tileAvailable}
              value={fmtNum(S.drivers.available)}
              sub={P.subAvailable}
              tone={S.drivers.available > 0 ? "success" : "muted"}
              onClick={() => setFocusMode(focusMode === "available" ? "" : "available")}
              extra={focusMode === "available" ? <Badge variant="success">{P.focusOn}</Badge> : undefined}
            />
          )}
          {S?.orders && (
            <Tile
              icon={IconOrder}
              label={P.tileMoving}
              value={fmtNum(S.orders.active)}
              sub={focusMode ? P.subMoving : undefined}
              tone={S.orders.active > 0 ? "accent" : "muted"}
              onClick={focusMode ? () => setFocusMode("") : undefined}
            />
          )}
          {S?.orders && (
            <div className={`rounded-card ${flash ? "animate-pulse ring-2 ring-danger" : ""}`}>
              <Tile
                icon={IconWarning}
                label={P.tileStuck}
                value={fmtNum(stuckN)}
                sub={stuckN > 0 ? P.subStuck : P.subStuckNone}
                tone={stuckN > 0 ? "danger" : "muted"}
                onClick={
                  stuckN > 0
                    ? () => {
                        if (focusMode === "stuck") {
                          setFocusMode("");
                        } else {
                          setFocusMode("stuck");
                          goStuck();
                        }
                      }
                    : undefined
                }
                extra={focusMode === "stuck" ? <Badge variant="danger">{P.focusOn}</Badge> : undefined}
              />
            </div>
          )}
          {S?.merchants && (
            <Tile
              icon={IconStore}
              label={P.tileOpen}
              value={fmtNum(S.merchants.open)}
              sub={P.subOpen.replace("{n}", fmtNum(S.merchants.total))}
              tone={S.merchants.open > 0 ? "accent" : "muted"}
            />
          )}
          {/* **ومضةُ العالق الجديد تُقال لقارئ الشاشة أيضاً.** */}
          <span className="sr-only" role="status" aria-live="polite">
            {flash ? T.counter.newStuck : ""}
          </span>
        </div>
      )}

      {/* ══ ملءُ الشاشة — سطحُ التحكّم والخريطةُ معاً ════════════════════ */}
      <div className={full ? "fixed inset-0 z-40" : ""}>
      <div className={full ? "surface-raised flex h-full flex-col gap-2 overflow-hidden rounded-none p-2" : "flex flex-col gap-3"}>
      {/* ══ سطحُ التحكّم — بحثٌ وطبقاتٌ ومدّةٌ ودليل ══════════════════════ */}
      {/* `relative z-30` يرفع نتائجَ البحث فوق لوح الخريطة الذي يليه. */}
      <section className={`surface relative z-30 flex flex-col gap-3 p-3 ${full ? "shrink-0" : ""}`}>
        {/* ── ابحث واذهب ── */}
        <div className="relative w-full">
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

        {/* ── فلترُ الرابط — يُرى ويُشال (تدقيقُ ٢٠٢٦-١٠-٢٧) ── */}
        {urlBits.length > 0 && (
          <div className="flex flex-wrap items-center gap-2 rounded-control bg-warning-tint px-3 py-2 text-sm">
            <span className="text-ink-muted">{P.urlFilter}</span>
            <b>{urlBits.join(" · ")}</b>
            <Button variant="secondary" className="ms-auto" onClick={clearUrlFilter}>
              <IconClose size={14} aria-hidden />
              {P.urlClear}
            </Button>
          </div>
        )}

        {/* ── الطبقاتُ الإضافيّة حبّاتٌ بعددها ── */}
        <div className="flex flex-col gap-1.5">
          <h2 className="heading-card">{P.controlsTitle}</h2>
          <p className="text-xs text-ink-muted">{P.controlsHint}</p>
          <Chips
            items={layerChips.map((c) => ({ id: c.id, label: c.label, count: c.count }))}
            value={layerOn}
            onChange={toggle}
            wrap
          />
        </div>

        {/* ── المدّةُ وساعاتُ اليوم — حين تُفتح الكثافةُ أو الفرصُ وحدَهما ── */}
        {can("VIEW_DEMAND_ANALYTICS") && (visible.demand || visible.opportunities) && (
          <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
            <div className="flex flex-col gap-1">
              <span className="text-xs text-ink-muted">{P.when}</span>
              <Chips
                items={[
                  { id: "today", label: T.demandWhen.today },
                  { id: "7d", label: T.demandWhen.d7 },
                  { id: "30d", label: T.demandWhen.d30 },
                ]}
                value={range}
                onChange={(k) => setRange(k as typeof range)}
                wrap
              />
            </div>
            <div className="flex flex-col gap-1">
              <span className="text-xs text-ink-muted">{P.hours}</span>
              <Chips
                items={[
                  { id: "all", label: T.demandWhen.all },
                  { id: "morning", label: T.demandWhen.morning },
                  { id: "noon", label: T.demandWhen.noon },
                  { id: "evening", label: T.demandWhen.evening },
                  { id: "night", label: T.demandWhen.night },
                ]}
                value={dayPart}
                onChange={(k) => setDayPart(k as typeof dayPart)}
                wrap
              />
            </div>
          </div>
        )}

        {/* ── معنى الألوان · ورسمُ التغطية — زرّان باسميهما ── */}
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <Button variant="secondary" className="w-full" aria-expanded={legendOpen} onClick={() => setLegendOpen((o) => !o)}>
            {legendOpen ? P.legendHide : P.legendShow}
          </Button>
          {/* **ولا يظهر الرسمُ لمن لا يملكه** — ورؤيةُ زرٍّ يردّ `403` أسوأُ من غيابه. */}
          {can("MANAGE_COVERAGE") && !drawing && (
            <Button variant="secondary" className="w-full" onClick={() => { setDrawing(true); setDraft([]); }}>
              <IconZones size={16} aria-hidden />
              {P.draw}
            </Button>
          )}
        </div>
        {legendOpen && <Legend />}

        {visible.office && office.data && !office.data.has_location && (
          <p className="text-xs text-ink-muted">{T.office.noLocation}</p>
        )}
      </section>

      {/* ══ شريطُ الرسم — فوق الخريطة لا تحتها (تدقيقُ ٢٠٢٦-١٠-٢٧) ══════════ */}
      {can("MANAGE_COVERAGE") && drawing && (
        <section className={`surface flex flex-col gap-3 border-2 border-accent p-3 ${full ? "shrink-0" : ""}`}>
          <div className="flex items-center gap-3">
            <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
              <IconZones size={20} />
            </span>
            <div className="min-w-0 flex-1">
              <h2 className="heading-card">{P.drawTitle}</h2>
              <p className="text-xs text-ink-muted">{focus ? P.focusFound : P.drawHint}</p>
            </div>
            <span className="text-end">
              <span className="figure block leading-none">{fmtNum(draft.length)}</span>
              <span className="text-2xs text-ink-muted">{P.drawPoints}</span>
            </span>
          </div>
          <Input
            placeholder={P.drawName}
            aria-label={P.drawName}
            value={zoneName}
            onChange={(e) => setZoneName(e.target.value)}
          />
          {draft.length > 0 && draft.length < 3 && <p className="text-xs text-ink-muted">{T.coverage.needThree}</p>}
          {saveErr && <Alert tone="error">{saveErr}</Alert>}
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
            <Button
              className="w-full"
              disabled={saving || draft.length < 3 || !zoneName.trim()}
              onClick={savePolygon}
            >
              {saving ? P.sending : T.coverage.save}
            </Button>
            <Button
              variant="secondary"
              className="w-full"
              disabled={saving || draft.length === 0}
              onClick={() => setDraft((d) => d.slice(0, -1))}
            >
              {T.coverage.undo}
            </Button>
            <Button
              variant="secondary"
              className="w-full"
              disabled={saving}
              onClick={() => { setDrawing(false); setDraft([]); setSaveErr(""); }}
            >
              {T.coverage.cancel}
            </Button>
          </div>
        </section>
      )}

      {/* ══ الخريطة — والبطاقةُ فوقها (لوحةٌ سفليّةٌ على الجوّال، جانبيّةٌ على الحاسوب) ══ */}
      <div
        className={
          full
            ? "relative min-h-0 w-full flex-1"
            : "relative h-[60vh] w-full lg:h-[calc(100vh-16rem)]"
        }
      >
        <OpsMapCanvas
          layers={layers}
          icons={icons}
          markers={markers}
          flyTo={fly}
          onFeatureClick={onFeature}
          onMarkerClick={(id) => {
            if (id === "office") setSelected({ kind: "office" });
          }}
          onMapClick={onGround}
          unavailableLabel={T.unavailable}
          focus={focus}
          bounds={coverageBounds}
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

        {/* ── الفراغُ يُقال — شارةٌ بدائرةٍ ملوّنةٍ وجملةٍ مفيدة ── */}
        {emptyHints.length > 0 && !current && (
          <div className="pointer-events-none absolute inset-x-0 bottom-8 z-[5] flex justify-center px-4">
            <div className="surface pointer-events-auto flex max-w-md items-center gap-3 px-3 py-2.5 shadow-lift">
              <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-field text-ink-muted">
                <IconLocation size={18} />
              </span>
              <span className="min-w-0 text-sm">
                <b className="block">{P.emptyTitle}</b>
                <span className="block text-xs text-ink-muted">{emptyHints.join(" · ")}</span>
                {driversEmpty && canDriversList && (
                  <Link href="/dashboard/users?role=driver" className="text-xs font-bold text-accent-text underline">
                    {P.driversList}
                  </Link>
                )}
              </span>
            </div>
          </div>
        )}

        {current && (
          <aside
            className="absolute inset-x-0 bottom-0 z-10 flex max-h-[65%] flex-col gap-3 overflow-y-auto rounded-t-card
                       border-t border-line-soft bg-surface p-4 shadow-xl
                       lg:inset-x-auto lg:inset-y-0 lg:end-0 lg:max-h-none lg:w-96 lg:rounded-none lg:border-s lg:border-t-0"
          >
            <span aria-hidden className="mx-auto h-1.5 w-10 shrink-0 rounded-full bg-line lg:hidden" />
            <SelectedCard
              current={current}
              office={office.data}
              orders={orders.data?.orders ?? []}
              customersMin={customers.data?.min_count ?? 3}
              rangeName={rangeName}
              maxScore={maxScore}
              can={can}
              canIntervene={canIntervene}
              canProfile={canProfile}
              canOpenMerchant={canOpen.merchant}
              busy={busy}
              actErr={actErr}
              confirming={confirming}
              setConfirming={setConfirming}
              onClose={() => setSelected(null)}
              onAssign={(o) => setAssigning(o)}
              onZoneActive={setZoneActive}
              onRequestStatus={setRequestStatus}
              onClearClosure={clearClosure}
              onDecideLandmark={decideLandmark}
            />
          </aside>
        )}
      </div>

      {/* ══ على الدوام وما في موقع — تحت الخريطة لا عمودٌ يصغّرها ══════════ */}
      {!full && noLoc.length > 0 && (
        <section className="surface flex flex-col gap-2 p-4">
          <h2 className="heading-card">{P.noLocTitle.replace("{n}", fmtNum(noLoc.length))}</h2>
          <p className="text-xs text-ink-muted">{T.driver.noLocationHint}</p>
          {/* **والاسمُ يفتح بطاقتَه** — فيُتّصل به من هنا. */}
          <Chips
            items={noLoc.slice(0, 24).map((d) => ({ id: d.id, label: d.name }))}
            value={[] as string[]}
            onChange={(id) => {
              const d = noLoc.find((x) => x.id === id);
              if (d) setSelected({ kind: "driver", v: d });
            }}
            wrap
          />
        </section>
      )}
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

      <Toaster />
      {/* **«طلبُ التوسّع بالمكان» انتقل لقسم «طلبات التوسّع»** (قرارُ المالك ٢٠٢٦-١٠-٢٥). */}
    </PageContainer>
  );
}

// ══════════════════════════════════════════════════════════════════════
// **بطاقةُ المُحدَّد** — رأسٌ ملوّنٌ ورقمٌ كبيرٌ وصفوفٌ قليلةٌ وأزرارٌ بعرضها
// ══════════════════════════════════════════════════════════════════════

type IconT = ComponentType<{ size?: number; className?: string }>;

function SelectedCard({
  current,
  office,
  orders,
  customersMin,
  rangeName,
  maxScore,
  can,
  canIntervene,
  canProfile,
  canOpenMerchant,
  busy,
  actErr,
  confirming,
  setConfirming,
  onClose,
  onAssign,
  onZoneActive,
  onRequestStatus,
  onClearClosure,
  onDecideLandmark,
}: {
  current: Picked;
  office: Office | null;
  orders: OrderPin[];
  customersMin: number;
  rangeName: string;
  maxScore: number;
  can: (p: string) => boolean;
  canIntervene: boolean;
  canProfile: boolean;
  canOpenMerchant: boolean;
  busy: boolean;
  actErr: string;
  confirming: string | null;
  setConfirming: (k: string | null) => void;
  onClose: () => void;
  onAssign: (o: OrderPin) => void;
  onZoneActive: (id: string, active: boolean) => void;
  onRequestStatus: (id: string, status: string) => void;
  onClearClosure: (id: number) => void;
  onDecideLandmark: (id: number, approve: boolean) => void;
}) {
  const head = (p: {
    tone?: MapTone;
    icon?: IconT;
    tint?: string;
    title: string;
    badge?: ReactNode;
  }) => (
    <div className="flex items-start gap-3">
      {p.tone ? (
        <ToneIcon tone={p.tone} big />
      ) : (
        <span className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${p.tint ?? "bg-field text-ink"}`}>
          {p.icon && <p.icon size={20} />}
        </span>
      )}
      <div className="min-w-0 flex-1">
        <h3 className="heading-card truncate">{p.title}</h3>
        {p.badge && <div className="mt-1 flex flex-wrap gap-1">{p.badge}</div>}
      </div>
      <button
        type="button"
        aria-label={T.close}
        title={T.close}
        onClick={onClose}
        className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-line bg-surface text-ink-muted hover:text-ink"
      >
        <IconClose size={16} aria-hidden />
      </button>
    </div>
  );
  const errBox = actErr ? <Alert tone="error">{actErr}</Alert> : null;

  /** **فعلٌ يسأل أوّلاً** — نقرةٌ تفتح السؤالَ تحتَه، والثانيةُ تنفّذ. */
  const confirmBox = (key: string, text: string, go: () => void) =>
    confirming === key ? (
      <div className="flex flex-col gap-2 rounded-control bg-danger-tint p-3">
        <p className="text-sm">
          <b>{P.sure}</b> {text}
        </p>
        <div className="grid grid-cols-2 gap-2">
          <Button variant="danger" className="w-full" disabled={busy} onClick={go}>
            {busy ? P.sending : P.yesGo}
          </Button>
          <Button variant="secondary" className="w-full" disabled={busy} onClick={() => setConfirming(null)}>
            {P.noBack}
          </Button>
        </div>
      </div>
    ) : null;

  switch (current.kind) {
    case "driver": {
      const d = current.v;
      return (
        <>
          {head({
            tone: driverTone(d.tone),
            title: d.name,
            badge: (
              <>
                <Badge variant={d.tone === "available" ? "success" : d.tone === "busy" ? "warning" : "neutral"}>
                  {T.tone[d.tone] ?? T.tone.idle}
                </Badge>
                <Badge variant="neutral">{d.on_shift ? T.driver.onShift : T.driver.offShift}</Badge>
              </>
            ),
          })}
          <Big
            value={d.active_orders > 0 ? P.inHand.replace("{n}", fmtNum(d.active_orders)) : P.freeNow}
            sub={
              // **والمالُ لمن يملك صلاحيّتَه وحدَه** (البند ٦).
              d.cash_held !== undefined ? P.cashInHand.replace("{amount}", fmtMoney(d.cash_held)) : undefined
            }
          />
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.driver.status} v={pick(ACCOUNT_STATUS, d.status)} />
            <Row k={T.driver.lastSeen} v={d.last_location_at ? fmtDateTime(d.last_location_at) : P.unknown} />
            <Row k={T.live} v={T.freshness[d.freshness]} />
          </dl>
          <div className="grid grid-cols-1 gap-2">
            {/* **والهاتفُ لا يصل إلّا لمن يملك `users.contact.read`** — يحذفه الخادم. */}
            {d.phone && <ContactButtons phone={d.phone} />}
            {d.current_order && (
              <Link href={`/dashboard/orders?id=${d.current_order}`}>
                <Button variant="secondary" className="w-full">
                  <IconOrder size={16} aria-hidden />
                  {T.order.openPage}
                </Button>
              </Link>
            )}
            {/* **إلى ملفّ السائق نفسِه** (تدقيقُ ٢٠٢٦-١٠-٢٧): كان `/dashboard/users?id=` — وقائمةُ
                الحسابات لا تقرأ `id`، فيصل إلى الناس كلِّهم. */}
            {canProfile && (
              <Link href={`/dashboard/users/${d.id}`}>
                <Button variant="secondary" className="w-full">
                  <IconUser size={16} aria-hidden />
                  {T.driver.openProfile}
                </Button>
              </Link>
            )}
          </div>
        </>
      );
    }

    case "merchant": {
      const x = current.v;
      return (
        <>
          {head({
            tone: storeTone(x.open_now),
            title: x.name,
            badge: (
              <Badge variant={x.open_now ? "success" : "neutral"}>
                {x.open_now ? T.merchant.open : T.merchant.closed}
              </Badge>
            ),
          })}
          <Big value={P.activeAtStore.replace("{n}", fmtNum(x.active_orders))} />
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.merchant.status} v={pick(MERCHANT_STATUS, x.status)} />
            {x.city && <Row k={T.merchant.city} v={x.city} />}
            {/* **والمندوبُ لمن يراقب نشاطَهم وحدَه** (البند ٩). */}
            {x.rep && <Row k={T.merchant.rep} v={x.rep} />}
          </dl>
          <div className="grid grid-cols-1 gap-2">
            {x.phone && (
              <a href={`tel:${x.phone}`}>
                <Button variant="secondary" className="w-full">
                  <IconPhone size={16} aria-hidden />
                  {T.contact.call}
                </Button>
              </a>
            )}
            {/* **إلى صفحة المتجر نفسِه، ولمن تُفتح له** (تدقيقُ ٢٠٢٦-١٠-٢٧): كان
                `/dashboard/sections?merchant=` — والسوقُ لا يقرأ `merchant`. */}
            {canOpenMerchant && (
              <Link href={`/dashboard/merchants/${x.id}`}>
                <Button variant="secondary" className="w-full">
                  <IconStore size={16} aria-hidden />
                  {T.merchant.openPage}
                </Button>
              </Link>
            )}
          </div>
        </>
      );
    }

    case "order": {
      const o = current.v;
      const stuck = !!o.stuck_reason;
      return (
        <>
          {head({
            tone: orderTone(o.stuck_reason),
            title: `#${fmtNum(o.number)}`,
            badge: (
              <>
                <Badge variant={stuck ? "danger" : "info"}>{pick(ORDER_STATUS, o.status)}</Badge>
                {stuck && (
                  <Badge variant="danger">
                    {T.orderCard.stuck} ·{" "}
                    {pick(m.admin.ordersPage.alertReasons as Record<string, string>, o.stuck_reason)}
                  </Badge>
                )}
              </>
            ),
          })}
          <Big
            label={P.orderAgeLabel}
            value={P.orderAge.replace("{n}", fmtNum(minutesSince(o.created_at)))}
            // **والمالُ بصلاحيّته** (البند ١٠).
            sub={o.total !== undefined ? `${T.order.total}: ${fmtMoney(o.total)}` : undefined}
            danger={stuck}
          />
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.order.merchant} v={o.merchant ?? P.unknown} />
            {/* **واسمُ الزبون لمن يملك تفاصيلَه وحدَه** — يحذفه الخادم. */}
            {o.customer_name && <Row k={T.orderCard.customer} v={o.customer_name} />}
            <Row k={T.order.driver} v={o.driver ?? T.order.noDriver} />
            {o.address && <Row k={T.order.area} v={o.address} />}
            {o.payment_method && <Row k={T.order.payment} v={pick(PAYMENT, o.payment_method)} />}
            {o.delivery_fee !== undefined && <Row k={T.order.delivery} v={fmtMoney(o.delivery_fee)} />}
            <Row k={T.order.createdAt} v={fmtDateTime(o.created_at)} />
          </dl>
          <CustomerLiveRow live={orders.find((y) => y.id === o.id)?.customer_live ?? o.customer_live} />
          {/* **كم مضى على كلّ مرحلة** — المراحلُ الواقعةُ وحدَها. */}
          <OrderSteps order={o} />
          <div className="grid grid-cols-1 gap-2">
            {/* **والإسنادُ بمساره القائم** (`AssignDialog`) — بالقرب وبتأكيدٍ وسبب، **ولمن يملك التدخّل وحدَه.** */}
            {canIntervene && !o.driver_id && (o.status === "preparing" || o.status === "dispatching") && (
              <Button className="w-full" onClick={() => onAssign(o)}>
                <IconMoto size={16} aria-hidden />
                {m.admin.ordersPage.assignHere}
              </Button>
            )}
            <Link href={`/dashboard/orders?id=${o.id}`}>
              <Button variant="secondary" className="w-full">
                <IconOrder size={16} aria-hidden />
                {T.order.openPage}
              </Button>
            </Link>
            {canProfile && o.driver_id && (
              <Link href={`/dashboard/users/${o.driver_id}`}>
                <Button variant="secondary" className="w-full">
                  <IconUser size={16} aria-hidden />
                  {T.driver.openProfile}
                </Button>
              </Link>
            )}
          </div>
        </>
      );
    }

    case "customers": {
      const c = current.v;
      return (
        <>
          {head({ tone: "customer", title: T.customerCard.title })}
          <Big value={P.customersN.replace("{n}", fmtNum(c.count))} />
          <p className="text-xs leading-5 text-ink-muted">
            {T.customerCard.privacy.replace("{n}", fmtNum(customersMin))}
          </p>
        </>
      );
    }

    case "rep": {
      const r = current.v;
      // **ورصيدُ المحفظة يُقال باتّجاهه** (تدقيقُ ٢٠٢٦-١٠-٢٧): «مستحقّاته: -٤٫٩٩» كانت تُقرأ
      // مالاً له وهو عليه. **عرضٌ فقط — لا يُمسّ كيف يُحسب.**
      const wallet =
        r.earnings === undefined
          ? undefined
          : r.earnings < 0
            ? P.walletOwes.replace("{amount}", fmtMoney(Math.abs(r.earnings)))
            : P.walletHas.replace("{amount}", fmtMoney(r.earnings));
      return (
        <>
          {head({ tone: "rep", title: r.name })}
          <Big label={P.storesAll} value={fmtNum(r.merchants)} />
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={P.rangeOf.replace("{label}", T.rep.converted).replace("{range}", rangeName)} v={fmtNum(r.converted)} />
            {r.last_activity && <Row k={T.rep.lastActivity} v={fmtDateTime(r.last_activity)} />}
            {wallet && <Row k={T.rep.earnings} v={wallet} danger={(r.earnings ?? 0) < 0} />}
          </dl>
          <p className="text-xs text-ink-muted">{T.repCard.where}</p>
        </>
      );
    }

    case "office": {
      const staff = office?.staff ?? [];
      return (
        <>
          {head({ icon: IconOverview, tint: "bg-paper text-on-bright border border-line", title: office?.name || T.office.title })}
          <Big value={P.staffN.replace("{n}", fmtNum(staff.length))} sub={T.office.window.replace("{n}", fmtNum(office?.presence_min ?? 5))} />
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.office.address} v={office?.address || T.office.noAddress} />
          </dl>
          {staff.length === 0 ? (
            <p className="text-sm text-ink-muted">{T.office.noneOnline}</p>
          ) : (
            <ul className="flex flex-col gap-1.5 text-sm">
              {staff.map((s, i) => (
                <li key={`${s.name}-${i}`} className="flex items-baseline gap-2">
                  <Dot tone="driver-available" />
                  <span className="font-medium">{s.name}</span>
                  <span className="ms-auto text-xs text-ink-muted">
                    {s.roles.map((rc) => roleLabelByCode(rc)).join(" · ")}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </>
      );
    }

    case "zone": {
      const z = current.v;
      const key = `zone-${z.id}`;
      return (
        <>
          {head({
            icon: IconZones,
            tint: z.active ? "bg-accent text-on-bright" : "bg-field text-ink-muted",
            title: z.name,
            badge: <Badge variant={z.active ? "success" : "neutral"}>{z.active ? P.zoneOn : P.zoneOff}</Badge>,
          })}
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.coverage.shape} v={z.shape === "polygon" ? T.coverage.polygon : T.coverage.radius} />
            {z.city && <Row k={T.coverage.city} v={z.city} />}
            {z.shape === "radius" && <Row k={T.coverage.radius} v={P.metres.replace("{n}", fmtNum(z.radius_m))} />}
          </dl>
          {errBox}
          {/* **والمنطقةُ تُوقَف ولا تُحذف** — منطقةٌ حُذفت تترك طلباتٍ تشير إلى عدم. */}
          {can("MANAGE_COVERAGE") &&
            (confirmBox(key, P.confirmDisable, () => onZoneActive(z.id, false)) ?? (
              <Button
                variant={z.active ? "danger" : "primary"}
                className="w-full"
                disabled={busy}
                onClick={() => (z.active ? setConfirming(key) : onZoneActive(z.id, true))}
              >
                {busy ? P.sending : z.active ? T.coverage.disable : T.coverage.enable}
              </Button>
            ))}
          <p className="text-xs text-ink-muted">{T.coverage.deleteHint}</p>
        </>
      );
    }

    case "request": {
      const q = current.v;
      const key = `request-${q.id}`;
      const states = ["new", "reviewing", "planned", "covered", "rejected"] as const;
      return (
        <>
          {head({
            icon: IconLocation,
            tint: "bg-violet-tint text-violet",
            title: T.request.title,
            badge: (
              <Badge variant={q.status === "covered" ? "success" : q.status === "rejected" ? "neutral" : "violet"}>
                {pick(T.request.state as Record<string, string>, q.status)}
              </Badge>
            ),
          })}
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.request.createdAt} v={fmtDateTime(q.created_at)} />
            {q.address && <Row k={T.request.address} v={q.address} />}
            {q.city && <Row k={T.request.city} v={q.city} />}
            <Row k={T.request.source} v={sourceLabel(q.source)} />
            {q.note && <Row k={T.request.note} v={q.note} />}
          </dl>
          {errBox}
          {can("MANAGE_COVERAGE") && (
            <div className="flex flex-col gap-1.5">
              <span className="text-xs text-ink-muted">{P.requestStatusPick}</span>
              <Chips
                items={states.map((s) => ({ id: s, label: T.request.state[s], disabled: busy }))}
                value={q.status}
                onChange={(s) => {
                  if (s === q.status || busy) return;
                  if (s === "rejected") setConfirming(key);
                  else onRequestStatus(q.id, s);
                }}
                wrap
              />
              {confirmBox(key, P.confirmRequestReject, () => onRequestStatus(q.id, "rejected"))}
            </div>
          )}
        </>
      );
    }

    case "branch": {
      const b = current.v;
      return (
        <>
          {head({
            icon: IconStore,
            tint: b.type === "primary" ? "bg-accent text-on-bright" : "bg-field text-ink",
            title: b.name,
            badge: <Badge variant={b.status === "active" ? "success" : "neutral"}>{pick(BRANCH_STATUS, b.status)}</Badge>,
          })}
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.branch.type} v={b.type === "primary" ? T.branch.primary : T.branch.sub} />
            {b.city && <Row k={T.branch.city} v={b.city} />}
            {b.parent && <Row k={T.branch.parent} v={b.parent} />}
            <Row k={T.layer.areas} v={P.areaUnit.replace("{n}", fmtNum(b.areas))} />
          </dl>
          <p className="text-xs text-ink-muted">{T.branch.onePrimary}</p>
        </>
      );
    }

    case "closure": {
      const c = current.v;
      const key = `closure-${c.id}`;
      return (
        <>
          {head({
            icon: IconBlock,
            tint: "bg-danger-solid text-on-solid",
            title: T.closure.title,
            badge: (
              <Badge variant="danger">
                {pick(T.closure.reasons as Record<string, string>, c.reason)}
              </Badge>
            ),
          })}
          <Big value={P.reportsN.replace("{n}", fmtNum(c.reports))} label={T.closure.reports} />
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.closure.by} v={c.reported_by_name || P.unknown} />
            <Row k={T.closure.until} v={fmtDateTime(c.expires_at)} />
          </dl>
          {errBox}
          {can("MANAGE_COVERAGE") &&
            (confirmBox(key, P.confirmClear, () => onClearClosure(c.id)) ?? (
              <Button className="w-full" disabled={busy} onClick={() => setConfirming(key)}>
                <IconRoute size={16} aria-hidden />
                {T.closure.clear}
              </Button>
            ))}
        </>
      );
    }

    case "landmark": {
      const l = current.v;
      const key = `landmark-${l.id}`;
      return (
        <>
          {head({
            icon: IconStar,
            tint: "bg-accent text-on-bright",
            title: l.name,
            badge: <Badge variant="accent">{pick(T.landmark.kinds as Record<string, string>, l.kind)}</Badge>,
          })}
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.landmark.by} v={l.suggested_by_name || P.unknown} />
            <Row k={T.landmark.at} v={fmtDateTime(l.created_at)} />
          </dl>
          {errBox}
          {can("MANAGE_COVERAGE") &&
            (confirmBox(key, P.confirmReject, () => onDecideLandmark(l.id, false)) ?? (
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                <Button className="w-full" disabled={busy} onClick={() => onDecideLandmark(l.id, true)}>
                  {busy ? P.sending : T.landmark.approve}
                </Button>
                <Button variant="danger" className="w-full" disabled={busy} onClick={() => setConfirming(key)}>
                  {T.landmark.reject}
                </Button>
              </div>
            ))}
        </>
      );
    }

    case "opportunity": {
      const o = current.v;
      // **قوّةُ الفرصة كلمةٌ لا رقمٌ بلا معنى** — نسبةً إلى أقوى فرصةٍ معروضة.
      const ratio = o.score / maxScore;
      const strength = ratio >= 0.66 ? P.strong : ratio >= 0.33 ? P.medium : P.weak;
      return (
        <>
          {head({ icon: IconTrendUp, tint: "bg-danger-solid text-on-solid", title: T.opportunity.title })}
          <Big label={T.opportunity.score} value={strength} sub={fmtNum(o.score)} />
          <dl className="flex flex-col gap-2 text-sm">
            <Row k={T.layer.orders} v={fmtNum(o.orders)} />
            <Row k={T.layer.coverageRequests} v={fmtNum(o.requests)} />
            <Row k={T.layer.merchants} v={fmtNum(o.merchants)} />
            <Row k={T.layer.drivers} v={fmtNum(o.drivers)} />
          </dl>
          {/* **ولا درجةَ بلا سببٍ مسمّى** — والأرقامُ الخامُّ فوقها، فمن لم يقبل الوزنَ حسب بنفسه. */}
          {o.reasons.length > 0 && (
            <div className="flex flex-col gap-1">
              <span className="text-xs font-bold text-ink-muted">{P.opportunityWhy}</span>
              <ul className="flex flex-col gap-1 text-sm">
                {o.reasons.map((why) => (
                  <li key={why} className="border-s-2 border-line ps-2">
                    {why === "high_demand_low_coverage" && T.opportunity.highDemandLowCoverage}
                    {why === "high_demand_low_drivers" && T.opportunity.highDemandLowDrivers}
                    {why === "many_requests" && T.opportunity.manyRequests}
                    {why === "merchants_no_drivers" && T.opportunity.merchantsNoDrivers}
                  </li>
                ))}
              </ul>
            </div>
          )}
          <p className="text-xs text-ink-muted">{T.opportunity.explain}</p>
        </>
      );
    }
  }
}

/** **الرقمُ الأهمّ في البطاقة** — كبيرٌ في صندوقٍ هادئ، وسطرٌ صغيرٌ تحته. */
function Big({ label, value, sub, danger }: { label?: string; value: string; sub?: string; danger?: boolean }) {
  return (
    <div className={`rounded-control px-3 py-2.5 ${danger ? "bg-danger-tint" : "bg-field"}`}>
      {label && <span className="block text-2xs text-ink-muted">{label}</span>}
      <span className={`figure block ${danger ? "text-danger" : ""}`}>{value}</span>
      {sub && <span className="block text-xs text-ink-muted">{sub}</span>}
    </div>
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
    <div>
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

/** **رمزُ حالٍ كما يُرسم على الخريطة** — دائرةٌ ورمزٌ أبيض. */
function ToneIcon({ tone, big = false }: { tone: MapTone; big?: boolean }) {
  const Glyph = MAP_TONES[tone].glyph;
  return (
    <span
      aria-hidden
      className={`grid shrink-0 place-items-center rounded-full border-2 border-paper ${big ? "h-11 w-11" : "h-6 w-6"}`}
      style={{ background: toneColor(tone) }}
    >
      <Glyph size={big ? 20 : 13} color={themeColor("on-solid")} strokeWidth={2.25} />
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
    <div className="flex flex-col gap-2 rounded-control bg-field p-3">
      <h3 className="heading-card">{T.bar.legendTitle}</h3>
      <ul className="grid grid-cols-1 gap-1.5 text-xs sm:grid-cols-2">
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
    <div className="grid grid-cols-2 gap-2">
      <a href={`tel:${phone}`}>
        <Button variant="secondary" className="w-full">
          <IconPhone size={16} aria-hidden />
          {T.contact.call}
        </Button>
      </a>
      <a href={waLink(phone)} target="_blank" rel="noopener noreferrer">
        <Button variant="secondary" className="w-full">
          <IconWhatsApp size={16} aria-hidden />
          {T.contact.whatsapp}
        </Button>
      </a>
    </div>
  );
}

/** **مسافةٌ بالمتر بين نقطتين** — لبعد الزبون عن دبّوسه حين يصل بثُّه. */
function metersBetween(lat1: number, lng1: number, lat2: number, lng2: number): number {
  const r = 6371000;
  const toRad = (d: number) => (d * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a = Math.sin(dLat / 2) ** 2 + Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return 2 * r * Math.asin(Math.sqrt(a));
}

/**
 * **موضعُ الزبون الحيُّ في بطاقة الطلب** (طلبُ المالك ٢٠٢٦-١٠-١٩: «لنقدر نعطي احداثيات
 * للسائق») — كم مضى ودقّتُه، **ونسخُ الإحداثيّات ورابطُ خرائط غوغل**، وتنبيهٌ إن بعُد عن دبّوسه.
 */
function CustomerLiveRow({ live }: { live?: OrderPin["customer_live"] }) {
  const [copied, setCopied] = useState(false);
  if (!live) return null;
  const C = T.orderCard;
  const coords = `${live.lat.toFixed(6)},${live.lng.toFixed(6)}`;
  const ago = Math.max(0, Math.round((Date.now() - new Date(live.at).getTime()) / 1000));
  const bits = [C.liveAgo.replace("{n}", fmtNum(ago))];
  if (live.accuracy_m != null) bits.push(C.liveAccuracy.replace("{n}", fmtNum(Math.round(live.accuracy_m))));
  return (
    <div className="flex flex-col gap-2 rounded-control bg-accent-tint p-3">
      <Row k={C.live} v={bits.join(" · ")} />
      {live.pin_gap_m > 50 && (
        <p className="text-xs font-bold text-danger">
          {C.liveFar.replace("{n}", fmtNum(Math.round(live.pin_gap_m)))}
        </p>
      )}
      <span dir="ltr" className="select-all text-center font-mono text-xs">
        {coords}
      </span>
      <div className="grid grid-cols-2 gap-2">
        <Button
          variant="secondary"
          className="w-full"
          onClick={() => {
            void navigator.clipboard?.writeText(coords).then(() => setCopied(true));
          }}
        >
          {copied ? C.copied : C.copyCoords}
        </Button>
        <a href={`https://www.google.com/maps?q=${coords}`} target="_blank" rel="noreferrer">
          <Button variant="secondary" className="w-full">
            {C.openGoogle}
          </Button>
        </a>
      </div>
    </div>
  );
}

function Row({ k, v, danger }: { k: string; v: string; danger?: boolean }) {
  return (
    <div className="flex items-baseline gap-2">
      <dt className="text-ink-muted">{k}</dt>
      <dd className={`ms-auto text-end font-medium ${danger ? "text-danger" : ""}`}>{v}</dd>
    </div>
  );
}
