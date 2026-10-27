"use client";

/**
 * الإعدادات — أخطر شاشة في المنصة.
 *
 * **والتعريفُ كلُّه من الخادم**: نوعُ الحقل ومداه وخياراته، **وخطورتُه**
 * (`risk` من قائمة الخطورة الواحدة) **وبيتُه في المواضيع** (`topic`).
 * ولو كُتب شيءٌ منها هنا لانحرف عن المحرّك يوماً.
 *
 * ══════════════════════════════════════════════════════════════════════
 * **قراراتُ المالك ٢٠٢٦-١٠-٠٤ (قسم الإعدادات)**
 * ══════════════════════════════════════════════════════════════════════
 *
 *  - **المواضيعُ بدل ستّة عشر تبويباً** (البند ١٧)، **وكلُّ إعدادٍ في مكانٍ واحد**.
 *  - **مثالٌ ماليٌّ حيٌّ يحسبه المحرّك** فوق مفاتيح المال.
 *  - **«كان ← يصير» قبل حفظ أيّ مفتاحٍ ماليّ** (البند ٨) — وكلمةُ السرّ شِيلت ٢٠٢٦-١٠-٢٢.
 *  - **رابطُ «السجل» على كلّ بطاقة** (البند ٩).
 *  - **كلُّ بابِ إطلاقٍ يُبدَّل بتأكيد** (البند ١٠).
 *  - **رسالةٌ لكلّ حالة فشل، ورجوعُ الحقل إلى المحفوظ، ولا حفظَ بلا تغيير.**
 *  - **بحثٌ لا يفهم التشكيل يدوّر في الألواح أيضاً.**
 *  - **ومن لا يملك التعديل يُقال له أيُّ صلاحيّةٍ تلزم.**
 *
 * **والتصميمُ الاحترافيّ** (تدقيقُ ٢٠٢٦-١٠-٢٧): رأسٌ موحّدٌ بأرقامٍ تُكبس (حالُ التطبيق ·
 * الدوام · البوت · آخرُ تعديل)، والبحثُ والمواضيعُ حبوباً في كرتٍ واحد، والموضوعُ في الرابط،
 * ومفاتيحُ التشغيل تقول حالها بكلمة، وحصّةُ المنصّة بوحدتها الصحيحة حسب طريقتها،
 * وساعةُ بداية اليوم جنب الدوام وبتأكيد.
 */

import { useCallback, useEffect, useMemo, useState, type ComponentType, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtMoney,
  fmtDate,
  fmtTime,
  errorText,
} from "@rahalgo/i18n";
import {
  Tabs,
  Alert,
  Confirm,
  Button,
  Input,
  Textarea,
  Switch,
  Badge,
  Card,
  IconSettings,
  IconWarning,
  IconCheck,
  IconSearch,
  IconWallet,
  IconOrder,
  IconDriver,
  IconStore,
  IconHandshake,
  IconUsers,
  IconSupport,
  IconLock,
  IconChat,
  IconStatus,
  IconSmartphone,
  IconLink,
  IconZones,
  IconRoles,
  IconDate,
  IconUser,
  IconTrendUp,
  LoadingState,
  ReloadState,
  PageContainer,
} from "@rahalgo/ui";
import ImageUpload from "@/components/admin/ImageUpload";
import { FileUpload, useLiveRefresh } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import dynamic from "next/dynamic";
import { PageHero } from "@/components/admin/PageHero";
import { Tile, type TileTone } from "@/components/admin/DashParts";
import ZonesPanel from "@/components/admin/settings/zones";
import HoursPanel from "@/components/admin/settings/hours";
import CitiesPanel from "@/components/admin/settings/cities";
import DivisionsPanel from "@/components/admin/settings/divisions";
import WhatsAppPanel from "@/components/admin/settings/whatsapp";
import AppStatusPanel from "@/components/admin/settings/app-status";
import ReleasePanel from "@/components/admin/settings/release";
import BroadcastPanel from "@/components/admin/BroadcastPanel";
import CampaignsPanel from "@/components/admin/CampaignsPanel";
import BannersPanel from "@/components/admin/settings/banners";
import RolesScreen from "@/components/admin/roles/RolesScreen";

/** **«عرض,طول» ← رقمان** — وفارغٌ أو مشوَّهٌ يعني «لا موقع». */
function geoOf(v: string): [number, number] | null {
  const p = v.split(",");
  if (p.length !== 2) return null;
  const la = Number(p[0]);
  const ln = Number(p[1]);
  return Number.isFinite(la) && Number.isFinite(ln) ? [la, ln] : null;
}

const m = getMessages(defaultLocale);
/** **الفاصلةُ العربيّة «٫» والفاصلةُ تُقرآن نقطةً** — من يكتب ١٫٥ بلوحته. */
const DECIMAL_SEP = new RegExp("[," + String.fromCharCode(0x066b) + "]", "g");

/* **والخريطةُ لا تُصيَّر في الخادم** — `leaflet` يقرأ `window` عند التحميل. */
const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), {
  ssr: false,
});
const S = m.admin.settings;
const U = S.ui;
const T = U.tiles;

/** **يطابق أنواعَ الكتالوج في المحرّك** — ونوعٌ يُضاف هناك ولا يُضاف هنا يسقط إلى الحقل النصّيّ. */
type Kind =
  | "int"
  | "money"
  | "bool"
  | "choice"
  | "text"
  | "media"
  | "percent"
  /** **رقمٌ بمنزلتين** (١٫٥) — عمولةُ المنصّة (قرارُ المالك ٢٠٢٦-١٠-١٨). */
  | "decimal"
  | "file"
  | "geo"
  | "longtext";

interface Setting {
  key: string;
  media_url?: string | null;
  group: string;
  kind: Kind;
  /** **صندوقٌ قديمٌ داخل المجموعة** — صفحاتُ الموقع (`page.*`). */
  section?: string;
  min?: number;
  max?: number;
  options?: string[];
  unit?: string;
  default: unknown;
  sensitive?: boolean;
  /** **مستوى الخطورة من قائمة الخادم الواحدة** — «money» أو «security» أو فراغ. */
  risk?: "money" | "security" | "";
  /** **بيتُه في المواضيع** — بقواعد البادئة في المحرّك. */
  topic: string;
  topic_section?: string;
  /** **لا بطاقةَ له** — يُضبط في لوحٍ مخصَّص (البند ١٢). */
  panel?: string;
  /** **مخفيٌّ** — إعداداتُ الموقع العامّ بعد أن صار الويبُ للموظّفين (البند ١٤). */
  hidden?: boolean;
  value: unknown;
  updated_at: string | null;
  updated_by: string | null;
  /** **أيُحرَّر هذا المفتاحُ لمن يسأل؟** — يقوله المحرّكُ لا اللوحة. */
  editable?: boolean;
  /** **يكتبه الخادمُ وحدَه** — رقمُ النسخة واسمُها من الملفّ المرفوع. */
  read_only?: boolean;
  /** شرطُ الظهور — و`equals` قد تصل `null`. */
  show_when?: { key: string; equals: string[] | null; not_empty?: boolean };
}

const label = (k: string) =>
  (S.keys as Record<string, { label?: string; hint?: string }>)[k]?.label ?? k;
const hint = (k: string) =>
  (S.keys as Record<string, { label?: string; hint?: string }>)[k]?.hint ?? "";
const unitText = (u?: string) => (u ? (S.units as Record<string, string>)[u] ?? "" : "");
const choiceText = (c: string) => (S.choices as Record<string, string>)[c] ?? c;
const zeroNote = (k: string) =>
  (S.keys as Record<string, { zero?: string }>)[k]?.zero ?? "";
/** **حالُ المفتاح بكلمة** — نصُّه الخاصّ إن وُجد، وإلّا «شغّال/طافي». */
const boolText = (k: string, side: "on" | "off") =>
  (S.boolStates as Record<string, { on: string; off: string }>)[k]?.[side] ??
  (side === "on" ? U.boolOnWord : U.boolOffWord);
const topicLabel = (t: string) => (S.topics as Record<string, string>)[t] ?? t;
const topicSectionLabel = (t: string) => (S.topicSections as Record<string, string>)[t] ?? "";
const sectionLabel = (name: string) => (S.sections as Record<string, string>)?.[name] ?? name;

type IconT = ComponentType<{ size?: number; className?: string }>;
const TOPIC_ICON: Record<string, IconT> = {
  money: IconWallet,
  orders: IconOrder,
  drivers: IconDriver,
  merchants: IconStore,
  reps: IconHandshake,
  customers: IconUsers,
  support: IconSupport,
  security: IconLock,
  messages: IconChat,
  launch: IconStatus,
  apps: IconSmartphone,
  site: IconLink,
  coverage: IconZones,
  whatsapp: IconChat,
  roles: IconRoles,
};

/**
 * **بحثٌ لا يفهم التشكيل** (البند ٣٤): «عمولة المندوب» تجد «مصدرُ احتساب
 * عمولةِ المندوب». **ويُسوّى الهمزُ والتاءُ المربوطةُ والألفُ المقصورة.**
 */
function norm(s: string): string {
  return s
    .toLowerCase()
    .replace(/[\u064B-\u065F\u0670\u0640]/g, "")
    .replace(/[\u0622\u0623\u0625]/g, "\u0627")
    .replace(/\u0629/g, "\u0647")
    .replace(/\u0649/g, "\u064A");
}

/**
 * **اسمُ الصلاحيّة اللازمة لتحرير المفتاح** — بالمنطق نفسِه في المحرّك (`settingCapability`):
 * وسقفُ المصروف الذي يحتاج موافقةً ثانيةً بإعدادات الأمان لا الماليّة.
 */
function capOf(s: Setting): string {
  if (s.key.startsWith("security.")) return "settings.security.manage";
  if (s.key === "finance.expense_approval_threshold") return "settings.security.manage";
  if (s.risk) return "settings.financial.manage";
  return "settings.general.manage";
}
const capName = (c: string) => (U.capNames as Record<string, string>)[c] ?? c;

/** **`min()` داخل العمود** — فلا يفيض عشرون ريم عن جوّالٍ ضيّق. */
const GRID = "grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(min(20rem,100%),1fr))]";
const WIDE: ReadonlySet<Kind> = new Set<Kind>(["geo", "media", "longtext", "file"]);

/** **مفاتيحُ تُسأل قبلها وليست ماليّة** — ساعةُ بداية اليوم تقلب يومَ الحساب بالمنصّة كلّها. */
const DAY_START = "platform.day_start_hour";

const MEDIA_KIND: Record<string, "platform_logo" | "auth_background" | "site_background"> = {
  "platform.logo": "platform_logo",
  "auth.background": "auth_background",
  "auth.background_mobile": "auth_background",
  "platform.background": "site_background",
  "platform.background_mobile": "site_background",
};

/**
 * **حصّةُ المنصّة من الأجرة: طريقتُها ورقمُها في بطاقةٍ واحدة** (تدقيقُ ٢٠٢٦-١٠-٢٧).
 * المفتاحُ `delivery.share_X` وطريقتُه `delivery.share_X_mode`.
 */
const SHARE_VALUE = /^delivery\.share_[a-z]+$/;
const modeKeyOf = (k: string) => (SHARE_VALUE.test(k) ? `${k}_mode` : "");
/** **النسبةُ من صفر لتسعين** — المحرّكُ يقف عند تسعين، فما فوقها لا يُكتب. */
const SHARE_PERCENT_MAX = 90;

/** **الوحدةُ والحدُّ الفعليّان** — حصّةُ المنصّة بالنسبة «٪» وبالمبلغ «ل.س». */
function effective(s: Setting, all: Setting[]): { unit?: string; max?: number; percentMode: boolean } {
  const mk = modeKeyOf(s.key);
  if (mk) {
    const mode = all.find((x) => x.key === mk);
    if (String(mode?.value ?? "") === "percent") {
      return { unit: "percent", max: SHARE_PERCENT_MAX, percentMode: true };
    }
  }
  return { unit: s.unit, max: s.max, percentMode: false };
}

/** **الألواحُ المخصَّصةُ في المواضيع** — والبحثُ يجدها بأسمائها. */
type PanelGate = "general" | "security" | "content";
const PANELS: { key: string; topic: string; label: string; gate?: PanelGate }[] = [
  { key: "appStatus", topic: "launch", label: m.admin.appStatus.title },
  { key: "hours", topic: "launch", label: m.admin.platformHours.switch.title, gate: "general" },
  { key: "release", topic: "apps", label: m.admin.release.title },
  { key: "slider", topic: "site", label: m.admin.sliderPanel.title, gate: "content" },
  { key: "divisions", topic: "coverage", label: m.admin.divisions.title, gate: "general" },
  { key: "cities", topic: "coverage", label: m.admin.cities.title, gate: "general" },
  { key: "zones", topic: "coverage", label: m.terms.zones, gate: "general" },
  { key: "whatsapp", topic: "whatsapp", label: m.admin.nav.whatsapp, gate: "security" },
  { key: "broadcast", topic: "whatsapp", label: m.admin.broadcast.title, gate: "content" },
];

function visibleIn(s: Setting, all: Setting[]): boolean {
  if (!s.show_when) return true;
  const on = all.find((x) => x.key === s.show_when!.key);
  if (!on) return false;
  if (s.show_when.not_empty) return on.value !== "" && on.value != null;
  return (s.show_when.equals ?? []).includes(String(on.value));
}

/** **أقسامٌ بترتيب أوّل ظهور** — فترتيبُ الفهرس هو ما يُرى. */
function groupBy(items: Setting[], f: (s: Setting) => string): { name: string; items: Setting[] }[] {
  const out: { name: string; items: Setting[] }[] = [];
  const at = new Map<string, number>();
  for (const it of items) {
    const name = f(it);
    const i = at.get(name);
    if (i === undefined) {
      at.set(name, out.length);
      out.push({ name, items: [it] });
    } else {
      out[i]!.items.push(it);
    }
  }
  return out;
}

/** **حالةٌ فاضية بدائرةٍ ملوّنة وعنوانٍ وجملةٍ تساعد.** */
function EmptyBlock({ icon: Icon, title, text }: { icon: IconT; title: string; text: string }) {
  return (
    <section className="surface flex flex-col items-center gap-3 p-8 text-center">
      <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-accent text-on-bright">
        <Icon size={28} />
      </span>
      <h2 className="heading-section">{title}</h2>
      <p className="max-w-md text-sm text-ink-muted">{text}</p>
    </section>
  );
}

interface WAState {
  provider: string;
  logged_in?: boolean;
}

/** **الموضوعُ واللوحُ الفرعيُّ في الرابط** — التحديثُ وزرُّ الرجوع يفتحان المكانَ نفسَه. */
function readUrl(): { topic: string; sub: string } {
  if (typeof window === "undefined") return { topic: "", sub: "" };
  const p = new URLSearchParams(window.location.search);
  return { topic: p.get("topic") ?? "", sub: p.get("sub") ?? "" };
}

export default function SettingsPage() {
  const router = useRouter();
  const { can } = useAuth();
  const isAdmin = can("settings.general.manage");
  // **والأدوارُ والصلاحيّاتُ موضوعٌ هنا** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — لمن يملك إدارتَها وحدَه.
  const canRoles = can("roles.manage");
  // **وقراءةُ حال البوت بإعدادات الأمان، والإعلانُ والحملاتُ بإدارة المحتوى** (`authz/policy.go`) —
  // كان الموضوعُ يُفتح لكلّ من يملك العامّةَ فيُردّ ٤٠٣.
  const canWa = can("settings.security.manage");
  const canContent = can("content.manage");
  const gateOk = (g?: PanelGate) =>
    !g || (g === "general" ? isAdmin : g === "security" ? canWa : canContent);

  const [list, setList] = useState<Setting[] | null>(null);
  const [topics, setTopics] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [topic, setTopic] = useState("");
  const [sub, setSub] = useState("");
  const [ready, setReady] = useState(false);
  const [q, setQ] = useState("");
  const [launchActive, setLaunchActive] = useState<string | null>(null);
  const [closed, setClosed] = useState(false);
  const [wa, setWa] = useState<WAState | "err" | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await api<{ settings: Setting[]; topics?: string[] }>("/api/v1/admin/settings");
      setList(res.settings ?? []);
      setTopics(res.topics ?? []);
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, []);

  /** **أرقامُ الرأس** — كلٌّ من بابه ولمن يملكه، وفشلُ واحدٍ لا يُسقط الصفحة. */
  const loadHero = useCallback(() => {
    api<{ active: string }>("/api/v1/admin/launch")
      .then((r) => setLaunchActive(r.active ?? ""))
      .catch(() => setLaunchActive(null));
    if (isAdmin) {
      api<{ active: boolean }>("/api/v1/admin/platform/closure")
        .then((c) => setClosed(Boolean(c.active)))
        .catch(() => setClosed(false));
    }
    if (canWa) {
      api<WAState>("/api/v1/admin/whatsapp")
        .then(setWa)
        .catch(() => setWa("err"));
    }
  }, [isAdmin, canWa]);

  // **وتتحدّث لحالها** (طلبُ المالك ٢٠٢٦-١٠-٠٩): مفتاحٌ قُلب من صفحة الطلبات أو من
  // لوحٍ آخر **كانت تبقى تعرض قيمتَه القديمة** فيُظَنّ أنّه رجع لحاله.
  useLiveRefresh(["settings"], () => {
    void load();
    loadHero();
  });

  useEffect(() => {
    void load();
    // **ورابطٌ من الرئيسيّة أو من صفحةٍ أخرى يفتح موضوعَه** (`?topic=drivers`).
    const u = readUrl();
    setTopic(u.topic);
    setSub(u.sub);
    setReady(true);
  }, [load]);

  useEffect(loadHero, [loadHero]);

  useEffect(() => {
    if (!ready || typeof window === "undefined") return;
    const p = new URLSearchParams(window.location.search);
    if (topic) p.set("topic", topic);
    else p.delete("topic");
    if (sub) p.set("sub", sub);
    else p.delete("sub");
    const s = p.toString();
    window.history.replaceState(null, "", s ? `?${s}` : window.location.pathname);
  }, [topic, sub, ready]);

  // **وموضوعا اللوحات لمن يملك بابَها** (فحصُ المتصفّح ٢٠٢٦-١٠-٠٥).
  const allTopics = useMemo(
    () => [
      ...topics,
      ...(isAdmin ? ["coverage"] : []),
      ...(canWa || canContent ? ["whatsapp"] : []),
      ...(canRoles ? ["roles"] : []),
    ],
    [topics, isAdmin, canWa, canContent, canRoles],
  );

  /** **عددُ بطاقات كلّ موضوع** — يُكتب على حبّته. */
  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const s of list ?? []) {
      if (s.hidden || s.panel || !visibleIn(s, list!)) continue;
      c[s.topic] = (c[s.topic] ?? 0) + 1;
    }
    return c;
  }, [list]);

  const found = useMemo(() => {
    const needle = norm(q.trim());
    if (!needle || !list) return null;
    const keys = list.filter(
      (s) =>
        !s.hidden &&
        visibleIn(s, list) &&
        (norm(s.key).includes(needle) ||
          norm(label(s.key)).includes(needle) ||
          norm(hint(s.key)).includes(needle)),
    );
    const panels = PANELS.filter((p) => gateOk(p.gate) && norm(p.label).includes(needle));
    return { keys, panels };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, list, isAdmin, canWa, canContent]);

  if (!list && error) {
    return (
      <PageContainer>
        <PageHero icon={IconSettings} title={m.admin.settingsPage.title} subtitle={U.heroSubtitle} />
        <ReloadState label={`${U.loadFailed}: ${error}`} onRetry={() => void load()} />
      </PageContainer>
    );
  }
  if (!list) return <LoadingState variant="text" />;

  const active = topic && allTopics.includes(topic) ? topic : (allTopics[0] ?? "");
  const readOnlyAll = list.length > 0 && list.every((s) => !(s.editable ?? false));
  const goTopic = (t: string, panel = "") => {
    setTopic(t);
    setSub(panel);
    setQ("");
  };

  const row = (s: Setting) => (
    <SettingRow key={s.key} s={s} all={list} editable={s.editable ?? false} onSaved={load} />
  );

  // ── أرقامُ الرأس ──
  const sw = String(list.find((s) => s.key === "hours.platform_switch")?.value ?? "");
  const hoursText = closed
    ? T.closure
    : sw === "open"
      ? T.hoursOpen
      : sw === "closed"
        ? T.hoursClosed
        : T.hoursSchedule;
  const hoursTone: TileTone = closed || sw === "closed" ? "danger" : sw === "open" ? "success" : "default";
  const launchText =
    launchActive === null
      ? "—"
      : launchActive === ""
        ? m.admin.appStatus.custom
        : ((m.admin.appStatus.presets as Record<string, string>)[launchActive] ?? launchActive);
  const launchTone: TileTone =
    launchActive === "open" ? "success" : launchActive === null ? "muted" : "warning";
  const last = list
    .filter((s) => s.updated_at)
    .reduce<Setting | null>((a, s) => (!a || (s.updated_at ?? "") > (a.updated_at ?? "") ? s : a), null);
  const waText =
    wa === null ? "—" : wa === "err" ? T.waUnknown : wa.provider === "dev" ? T.waDev : wa.logged_in ? T.waPaired : T.waNotPaired;
  const waTone: TileTone =
    wa === null || wa === "err" ? "muted" : wa.provider === "dev" ? "default" : wa.logged_in ? "success" : "danger";

  return (
    <PageContainer>
      <PageHero icon={IconSettings} title={m.admin.settingsPage.title} subtitle={U.heroSubtitle}>
        <div className={`grid grid-cols-2 gap-2 ${canWa ? "lg:grid-cols-4" : "lg:grid-cols-3"}`}>
          <Tile
            icon={IconStatus}
            label={T.appStatus}
            value={launchText}
            sub={T.appStatusSub}
            tone={launchTone}
            onClick={() => goTopic("launch")}
          />
          <Tile
            icon={IconDate}
            label={T.hours}
            value={hoursText}
            tone={hoursTone}
            onClick={() => goTopic("launch")}
          />
          {canWa && (
            <Tile
              icon={IconChat}
              label={T.whatsapp}
              value={waText}
              tone={waTone}
              onClick={() => goTopic("whatsapp")}
            />
          )}
          <Tile
            icon={IconUser}
            label={T.lastChange}
            value={last?.updated_at ? fmtDate(last.updated_at) : T.noChange}
            sub={
              last?.updated_at
                ? T.lastChangeSub.replace("{what}", label(last.key)).replace("{when}", fmtTime(last.updated_at))
                : undefined
            }
            tone={last ? "default" : "muted"}
            onClick={() =>
              router.push(
                last
                  ? `/dashboard/audit?entity=setting&entity_id=${encodeURIComponent(last.key)}`
                  : "/dashboard/audit?entity=setting",
              )
            }
          />
        </div>
      </PageHero>

      {error && <Alert className="mb-4">{error}</Alert>}
      {readOnlyAll && (
        <Alert tone="info" className="mb-4">
          {U.readOnly.replace("{cap}", capName("settings.general.manage"))}
        </Alert>
      )}

      {/* ── البحثُ والمواضيعُ في كرتٍ واحد ── */}
      <section className="surface mb-4 space-y-3 p-4">
        <Input
          id="settings-search"
          aria-label={U.searchLabel}
          icon={<IconSearch size={15} />}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={S.searchPlaceholder}
        />
        {found === null && (
          <div className="space-y-1.5">
            <span className="block text-xs font-bold text-ink-muted">{U.topicsLabel}</span>
            <nav className="flex flex-wrap gap-2">
              {allTopics.map((t) => {
                const on = active === t;
                const Icon = TOPIC_ICON[t] ?? IconSettings;
                const n = counts[t];
                return (
                  <button
                    key={t}
                    type="button"
                    aria-pressed={on}
                    onClick={() => goTopic(t)}
                    className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm transition-colors ${
                      on
                        ? "border-accent bg-accent font-bold text-on-bright"
                        : "border-line-soft text-ink-muted hover:bg-row-hover hover:text-ink"
                    }`}
                  >
                    <Icon size={15} />
                    {topicLabel(t)}
                    {n ? (
                      <span
                        className={`min-w-[1.25rem] rounded-full px-1.5 text-center text-2xs tabular-nums ${
                          on ? "bg-surface text-ink" : "bg-field text-ink-muted"
                        }`}
                        title={U.topicCount.replace("{n}", fmtNum(n))}
                      >
                        {fmtNum(n)}
                      </span>
                    ) : null}
                  </button>
                );
              })}
            </nav>
          </div>
        )}
      </section>

      {found !== null ? (
        found.keys.length === 0 && found.panels.length === 0 ? (
          <EmptyBlock icon={IconSearch} title={S.searchNoResults} text={U.emptySearchHint} />
        ) : (
          <div className="space-y-3">
            <p className="text-xs text-ink-muted">
              {S.searchCount.replace("{n}", fmtNum(found.keys.length + found.panels.length))}
            </p>
            {found.panels.length > 0 && (
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                {found.panels.map((p) => (
                  <Button key={p.key} variant="secondary" className="w-full" onClick={() => goTopic(p.topic, p.key)}>
                    {p.label} · {U.searchIn.replace("{topic}", topicLabel(p.topic))}
                  </Button>
                ))}
              </div>
            )}
            <div className={GRID}>
              {found.keys.map((s) =>
                s.panel ? (
                  <Card key={s.key}>
                    <p className="font-bold">{label(s.key)}</p>
                    <p className="mt-1 text-xs text-ink-muted">
                      {U.searchIn.replace("{topic}", topicLabel(s.topic))}
                    </p>
                    <Button variant="secondary" className="mt-3 w-full" onClick={() => goTopic(s.topic, s.panel!)}>
                      {U.openPanel.replace(
                        "{panel}",
                        PANELS.find((p) => p.key === s.panel)?.label ?? s.panel!,
                      )}
                    </Button>
                  </Card>
                ) : (
                  <div key={s.key} className={WIDE.has(s.kind) ? "col-span-full" : ""}>
                    <p className="mb-1 text-xs text-ink-muted">
                      {U.searchIn.replace("{topic}", topicLabel(s.topic))}
                    </p>
                    {row(s)}
                  </div>
                ),
              )}
            </div>
          </div>
        )
      ) : (
        <div className="min-w-0 space-y-4">
          <TopicBody
            topic={active}
            list={list}
            sub={sub}
            setSub={setSub}
            isAdmin={isAdmin}
            canWa={canWa}
            canContent={canContent}
            row={row}
            onSaved={load}
          />
        </div>
      )}
    </PageContainer>
  );
}

function TopicBody({
  topic,
  list,
  sub,
  setSub,
  isAdmin,
  canWa,
  canContent,
  row,
  onSaved,
}: {
  topic: string;
  list: Setting[];
  sub: string;
  setSub: (v: string) => void;
  isAdmin: boolean;
  canWa: boolean;
  canContent: boolean;
  row: (s: Setting) => ReactNode;
  onSaved: () => void;
}) {
  // **طريقةُ حصّة المنصّة تُعرض داخل بطاقة رقمها** — فلا بطاقتان لشيءٍ واحد.
  const companions = new Set(
    list.filter((s) => SHARE_VALUE.test(s.key)).map((s) => modeKeyOf(s.key)),
  );
  const items = list.filter(
    (s) => s.topic === topic && !s.hidden && !s.panel && visibleIn(s, list) && !companions.has(s.key),
  );

  if (topic === "roles") return <RolesScreen />;

  if (topic === "coverage") {
    const tabs = PANELS.filter((p) => p.topic === "coverage");
    const at = tabs.some((p) => p.key === sub) ? sub : tabs[0]!.key;
    return (
      <>
        <Tabs items={tabs.map((p) => ({ key: p.key, label: p.label }))} value={at} onChange={setSub} />
        {at === "divisions" && <DivisionsPanel />}
        {at === "cities" && <CitiesPanel />}
        {at === "zones" && <ZonesPanel />}
      </>
    );
  }
  if (topic === "whatsapp") {
    return (
      <>
        {canWa && <WhatsAppPanel />}
        {canContent && (
          <section className="surface space-y-3 p-4">
            <div>
              <h2 className="heading-card">{m.admin.broadcast.title}</h2>
              <p className="text-sm text-ink-muted">{m.admin.broadcast.hint}</p>
            </div>
            <BroadcastPanel />
            <CampaignsPanel />
          </section>
        )}
        {!canWa && !canContent && (
          <EmptyBlock icon={IconLock} title={topicLabel(topic)} text={U.noAccessTopic} />
        )}
      </>
    );
  }

  const dayStart = list.find((s) => s.key === DAY_START);
  const sections = groupBy(items, (s) => s.topic_section ?? "");
  const shareRow = (s: Setting) => {
    const mk = modeKeyOf(s.key);
    const mode = mk ? list.find((x) => x.key === mk) : undefined;
    return mode ? (
      <SettingRow
        key={s.key}
        s={s}
        all={list}
        editable={s.editable ?? false}
        onSaved={onSaved}
        companion={mode}
      />
    ) : (
      row(s)
    );
  };
  return (
    <>
      {topic === "money" && <MoneyExample version={list} />}

      {/* **الدوامُ والإيقافُ أوّلُ الإطلاق** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كان زرُّ «أوقف المنصّة» بعد ٢٣ بطاقة،
          **وساعةُ بداية اليوم جنبه** لأنّها لازم تمشي مع ساعة فتح الدوام. */}
      {topic === "launch" &&
        (isAdmin ? (
          <HoursPanel dayStart={dayStart ? row(dayStart) : null} />
        ) : dayStart ? (
          <div className={GRID}>{row(dayStart)}</div>
        ) : null)}
      {topic === "launch" && <AppStatusPanel />}
      {topic === "apps" && (
        <ReleasePanel
          list={list}
          onSaved={onSaved}
          renderKey={(k) => {
            const s = list.find((x) => x.key === k);
            return s ? (
              <SettingRow key={s.key} s={s} all={list} editable={s.editable ?? false} onSaved={onSaved} bare />
            ) : null;
          }}
        />
      )}
      {/* **وسلايدرُ التطبيق رجع إلى بيته** (قرارُ المالك ٢٠٢٦-١٠-٠٥). */}
      {topic === "site" && canContent && (
        <section className="surface space-y-3 p-4">
          <div>
            <h2 className="heading-card">{m.admin.sliderPanel.title}</h2>
            <p className="text-sm text-ink-muted">{m.admin.sliderPanel.hint}</p>
          </div>
          {/* **وتقليبُه ومدّتُه معه** — التطبيقُ يقرؤهما (`banner_auto` · `banner_every_ms`). */}
          <div className={GRID}>
            {list.filter((s) => s.panel === "slider" && visibleIn(s, list)).map(row)}
          </div>
          <BannersPanel isAdmin placement="home" />
        </section>
      )}

      {sections.map((sec) => {
        if (sec.name === "site.pages") {
          const pages = groupBy(sec.items, (s) => s.section ?? "");
          const at = pages.some((p) => p.name === sub) ? sub : (pages[0]?.name ?? "");
          return (
            <section key={sec.name} className="space-y-3">
              <h2 className="heading-card">{topicSectionLabel(sec.name)}</h2>
              <Tabs
                items={pages.map((p) => ({ key: p.name, label: sectionLabel(p.name) }))}
                value={at}
                onChange={setSub}
              />
              <div className={GRID}>{pages.find((p) => p.name === at)?.items.map(row)}</div>
            </section>
          );
        }
        if (sec.name === "site.office") {
          return (
            <section key={sec.name} className="space-y-2">
              <h2 className="heading-card">{U.office.title}</h2>
              <Alert tone="info">{U.office.hint}</Alert>
              <div className={GRID}>{sec.items.map(row)}</div>
            </section>
          );
        }
        return (
          <section key={sec.name || "_"} className="space-y-2">
            {sec.name && topicSectionLabel(sec.name) && (
              <h2 className="heading-card">{topicSectionLabel(sec.name)}</h2>
            )}
            {sec.name === "launch.gates" && (
              <p className="text-sm text-ink-muted">{m.admin.appStatus.gatesNote}</p>
            )}
            <div className={GRID}>{sec.items.map(shareRow)}</div>
          </section>
        );
      })}

      {sections.length === 0 && topic !== "launch" && topic !== "apps" && (
        <EmptyBlock icon={IconSettings} title={S.emptyGroup} text={U.emptyGroupHint} />
      )}
    </>
  );
}

interface Example {
  amount: number;
  delivery_fee: number;
  customer_pays: number;
  merchant_gets: number;
  driver_gets: number;
  rep_gets: number;
  platform_gets: number;
}

/**
 * **مثالٌ ماليٌّ حيٌّ** — يحسبه المحرّكُ بدوالّ الطلب نفسِها
 * (`/admin/settings/money-example`)، ويُعاد بعد كلّ حفظ. **ومربّعاتٌ ملوّنة** لا سطورٌ.
 */
function MoneyExample({ version }: { version: unknown }) {
  const [ex, setEx] = useState<Example | null>(null);
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    api<Example>("/api/v1/admin/settings/money-example?amount=500")
      .then((r) => {
        setEx(r);
        setFailed(false);
      })
      .catch(() => setFailed(true));
  }, [version]);
  if (failed) return <Alert tone="warning">{U.example.failed}</Alert>;
  if (!ex) return <LoadingState variant="text" />;
  const tiles: { label: string; v: number; icon: IconT; tone: TileTone }[] = [
    { label: U.example.customer, v: ex.customer_pays, icon: IconUsers, tone: "default" },
    { label: U.example.merchant, v: ex.merchant_gets, icon: IconStore, tone: "accent" },
    { label: U.example.driver, v: ex.driver_gets, icon: IconDriver, tone: "accent" },
    { label: U.example.rep, v: ex.rep_gets, icon: IconHandshake, tone: "accent" },
    { label: U.example.platform, v: ex.platform_gets, icon: IconTrendUp, tone: "success" },
  ];
  return (
    <section className="surface space-y-3 p-4">
      <div>
        <h2 className="heading-card">{U.example.title}</h2>
        <p className="text-sm text-ink-muted">
          {U.example.intro.replace("{amount}", fmtMoney(ex.amount)).replace("{fee}", fmtMoney(ex.delivery_fee))}
        </p>
      </div>
      <div className="grid grid-cols-2 gap-2 lg:grid-cols-5">
        {tiles.map((t) => (
          <Tile key={t.label} icon={t.icon} label={t.label} value={fmtMoney(t.v)} tone={t.tone} />
        ))}
      </div>
      <p className="text-xs text-ink-muted">{U.exampleNote}</p>
    </section>
  );
}

/** **قيمةٌ بكلماتها** — للتأكيد «كان ← يصير»، وبوحدتها الفعليّة. */
function valueText(s: Setting, v: unknown, unit?: string): string {
  if (s.kind === "bool") return boolText(s.key, v === true ? "on" : "off");
  if (s.kind === "choice") return choiceText(String(v ?? ""));
  if (s.kind === "percent") return `${fmtNum(Number(v ?? 0))}${unitText("percent")}`;
  if (s.kind === "int" || s.kind === "money" || s.kind === "decimal") {
    const uKey = unit ?? s.unit;
    const u = unitText(uKey);
    if (uKey === "percent") return `${fmtNum(Number(v ?? 0))}${u}`;
    return u ? `${fmtNum(Number(v ?? 0))} ${u}` : fmtNum(Number(v ?? 0));
  }
  return String(v ?? "");
}

/** **أهي القيمةُ المحفوظةُ نفسُها؟** — فلا يُحفظ ما لم يتغيّر. */
function same(s: Setting, raw: unknown): boolean {
  if (s.kind === "int" || s.kind === "money" || s.kind === "percent" || s.kind === "decimal") {
    return Number(raw) === Number(s.value);
  }
  return raw === s.value || String(raw) === String(s.value ?? "");
}

/** **رسالةٌ لكلّ حالة فشل** (البند ١١) — لا «المدى المسموح» لكلّ رفض. */
function saveError(s: Setting, err: unknown, max?: number): string {
  if (!(err instanceof ApiError)) return U.errors.network;
  const code = err.body?.code ?? "";
  const details = (err.body?.details ?? {}) as Record<string, unknown>;
  if (code === "step_up_required") return U.errors.cancelled;
  if (code.startsWith("step_up") || code === "invalid_password" || code === "wrong_password") {
    return U.errors.wrongPassword;
  }
  if (err.status === 403) return U.errors.forbidden;
  if (code === "setting_conflict") {
    return U.errors.conflict.replace("{other}", label(String(details.with ?? "")));
  }
  if (code === "setting_placeholder_missing") {
    return U.errors.placeholder.replace("{p}", String(details.placeholder ?? ""));
  }
  if (code === "validation") {
    if ((s.kind === "int" || s.kind === "money" || s.kind === "percent" || s.kind === "decimal") && max !== undefined) {
      return U.errors.range.replace("{min}", fmtNum(s.min ?? 0)).replace("{max}", fmtNum(max));
    }
    return U.errors.invalid;
  }
  return errorText(err);
}

/**
 * بطاقةُ إعدادٍ واحد — تحرّر نفسها في مكانها.
 *
 * `bare`: بلا إطار البطاقة — داخل بطاقةٍ أكبر (بطاقةُ التطبيق في «التحديث»).
 * `companion`: مفتاحٌ مرافقٌ يُعرض فوق الحقل — طريقةُ حصّة المنصّة.
 */
function SettingRow({
  s,
  all,
  editable,
  onSaved,
  bare = false,
  companion,
}: {
  s: Setting;
  all: Setting[];
  editable: boolean;
  onSaved: () => void;
  bare?: boolean;
  companion?: Setting;
}) {
  const [draft, setDraft] = useState<string>(() => String(s.value ?? ""));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [note, setNote] = useState("");
  const [saved, setSaved] = useState(false);
  /** **ما ينتظر تأكيداً** — قيمةٌ ماليّةٌ أو أمنيّةٌ أو بابُ إطلاق أو بدايةُ اليوم. */
  const [pending, setPending] = useState<{ raw: unknown } | null>(null);

  useEffect(() => {
    setDraft(String(s.value ?? ""));
  }, [s.value]);

  const eff = effective(s, all);
  const unit = eff.unit;
  const max = eff.max;
  /** **الحدُّ الأدنى صفرٌ إن لم يُرسَل** — الخادمُ يُسقطه حين يكون صفراً. */
  const min = s.min ?? 0;
  const decimal = s.kind === "decimal";
  const numeric = s.kind === "int" || s.kind === "money" || decimal;
  const gate = s.topic_section === "launch.gates" && s.kind === "bool";
  const dayStart = s.key === DAY_START;
  const dirty =
    s.kind === "bool" || s.kind === "choice" ? false : draft !== String(s.value ?? "");

  async function doSave(raw: unknown) {
    setBusy(true);
    setError("");
    setNote("");
    try {
      await api(`/api/v1/admin/settings/${s.key}`, {
        method: "PUT",
        body: JSON.stringify({ value: raw }),
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
      onSaved();
    } catch (err) {
      setError(saveError(s, err, max));
      // **والحقلُ يرجع إلى المحفوظ** (البند ١٥) — لا يبقى عارضاً ما لم يُحفظ.
      setDraft(String(s.value ?? ""));
    } finally {
      setBusy(false);
      setPending(null);
    }
  }

  /** **الحفظُ يمرّ من هنا** — بلا تغييرٍ لا نداء، والماليُّ والأمنيُّ والبابُ وبدايةُ اليوم بتأكيد. */
  function requestSave(raw: unknown) {
    if (same(s, raw)) {
      setDraft(String(s.value ?? ""));
      setNote(U.unchanged);
      setTimeout(() => setNote(""), 1500);
      return;
    }
    if (s.risk || gate || dayStart) {
      setPending({ raw });
      return;
    }
    void doSave(raw);
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (numeric) {
      const t = draft.trim().replace(DECIMAL_SEP, ".");
      const n = Number(t);
      // **والعشريُّ برقمين بعد الفاصلة بالأكثر** — مثل الخادم؛ والصحيحُ صحيحٌ كما كان.
      if (decimal ? !/^\d+(\.\d{1,2})?$/.test(t) : !Number.isInteger(n)) return setError(U.errors.invalid);
      if (n < min || (max !== undefined && n > max)) {
        return setError(
          U.errors.range.replace("{min}", fmtNum(min)).replace("{max}", fmtNum(max ?? 0)),
        );
      }
      return requestSave(n);
    }
    requestSave(draft);
  }

  const sliderMax = max && max > 0 ? max : 100;
  const sliderMin = min;
  const commitSlider = () => requestSave(Number(draft === "" ? 0 : draft));

  const gateOpen = pending?.raw === true;
  // **والمفتاحُ والخيارُ لا يحتاجان نموذجاً** — وطريقةُ الحصّة تُعرض داخل نموذج رقمها، ونموذجٌ داخل نموذج خطأ.
  const Wrap = s.kind === "geo" || s.kind === "bool" || s.kind === "choice" ? "div" : "form";
  const boolOn = s.value === true;

  const confirmTitle = gate
    ? (gateOpen ? U.gate.openTitle : U.gate.closeTitle).replace("{name}", label(s.key))
    : dayStart
      ? U.dayStart.confirmTitle
      : s.risk === "money"
        ? U.confirmTitle
        : U.confirmTitleGeneric;
  const confirmNote = dayStart ? U.dayStart.confirmNote : s.risk === "money" ? U.confirmNote : U.securityHint;

  const body = (
    <>
      {/* **والموقعُ ليس نموذجاً** (فحصُ المتصفّح ٢٠٢٦-١٠-٠٥): لاقطُه فيه نموذجُ بحثه، **ونموذجٌ
          داخلَ نموذجٍ خطأُ ترطيبٍ في الطرفيّة** — وهو يُحفظ بالضغط على الخريطة لا بزرّ. */}
      <Wrap onSubmit={Wrap === "form" ? submit : undefined}>
        <div className="mb-1 flex flex-wrap items-center gap-2">
          <span className="font-bold">{label(s.key)}</span>
          {s.risk && (
            <Badge variant={s.risk === "money" ? "warning" : "info"}>
              <span className="flex items-center gap-1">
                <IconWarning size={12} />
                {s.risk === "money" ? U.riskMoney : U.riskSecurity}
              </span>
            </Badge>
          )}
          {zeroNote(s.key) && Number(s.value) === 0 && (
            <Badge variant="warning">{zeroNote(s.key)}</Badge>
          )}
          {saved && (
            <Badge variant="success">
              <span className="flex items-center gap-1">
                <IconCheck size={12} strokeWidth={3} />
                {S.saved}
              </span>
            </Badge>
          )}
          {/* **رابطُ «السجل»** (البند ٩) — تاريخُ هذا المفتاح وحدَه. */}
          <a
            href={`/dashboard/audit?entity=setting&entity_id=${encodeURIComponent(s.key)}`}
            className="ms-auto text-xs text-ink-muted underline hover:text-ink"
          >
            {U.history}
          </a>
        </div>
        {hint(s.key) && s.kind !== "geo" && (
          <p className="mb-3 text-xs leading-relaxed text-ink-muted">{hint(s.key)}</p>
        )}
        {dayStart && <p className="mb-3 text-xs font-medium text-warning">{U.dayStart.hint}</p>}

        {/* **طريقةُ الحصّة فوق رقمها** — نسبةٌ أو مبلغٌ ثابت، وتأكيدُها من بطاقتها. */}
        {companion && (
          <div className="mb-3 rounded-control bg-field p-3">
            <SettingRow
              s={companion}
              all={all}
              editable={companion.editable ?? false}
              onSaved={onSaved}
              bare
            />
          </div>
        )}

        <div className="flex flex-wrap items-end gap-2">
          {s.kind === "bool" ? (
            <div className="flex items-center gap-3">
              <Switch
                id={s.key}
                label=""
                checked={boolOn}
                disabled={!editable || busy}
                onChange={(next) => requestSave(next)}
              />
              <span className={`text-sm font-bold ${boolOn ? "text-success" : "text-ink-muted"}`}>
                {boolText(s.key, boolOn ? "on" : "off")}
              </span>
            </div>
          ) : s.kind === "choice" ? (
            <div className="flex flex-wrap gap-2">
              {(s.options ?? []).map((o) => {
                const on = String(s.value ?? "") === o;
                return (
                  <button
                    key={o}
                    type="button"
                    disabled={!editable || busy}
                    aria-pressed={on}
                    onClick={() => {
                      if (on) return;
                      requestSave(o);
                    }}
                    className={`rounded-full border px-3 py-1.5 text-sm transition-colors disabled:opacity-50 ${
                      on
                        ? "border-accent bg-accent font-bold text-on-bright"
                        : "border-line-soft text-ink-muted hover:bg-row-hover hover:text-ink"
                    }`}
                  >
                    {choiceText(o)}
                  </button>
                );
              })}
            </div>
          ) : s.kind === "percent" ? (
            /* **والشريطُ يحترم حدَّ الفهرس** (البند ١٣) — حصّةُ المنصّة تسعون لا
               مئة. **ويُحفظ عند الإفلات وحدَه، وبلا تغييرٍ لا يُحفظ** (البند ١٤). */
            <div className="flex flex-1 items-center gap-3">
              <input
                id={s.key}
                type="range"
                min={sliderMin}
                max={sliderMax}
                value={draft === "" ? 0 : Number(draft)}
                disabled={!editable || busy}
                onChange={(e) => setDraft(e.target.value)}
                onMouseUp={commitSlider}
                onTouchEnd={commitSlider}
                onKeyUp={(e) => {
                  if (e.key.startsWith("Arrow") || e.key === "Home" || e.key === "End") commitSlider();
                }}
                className="h-1.5 flex-1 cursor-pointer appearance-none rounded-badge bg-field accent-accent"
              />
              <span className="figure w-20 shrink-0 text-end text-primary-dark">
                {fmtNum(draft === "" ? 0 : Number(draft))}
                {unitText("percent")}
              </span>
            </div>
          ) : s.kind === "file" ? (
            /* **والرفعُ يذهب إلى مفتاحه** (البند ٥) — كان يذهب إلى المفتاح القديم. */
            <div className="w-full">
              <FileUpload
                label=""
                hint=""
                accept=".apk"
                present={typeof s.value === "string" && s.value !== ""}
                path={`/api/v1/admin/app-file?key=${encodeURIComponent(s.key)}`}
                api={api}
                errorText={(e) => (e instanceof ApiError ? errorText(e) : m.errors.internal)}
                onChange={onSaved}
              />
            </div>
          ) : s.kind === "media" ? (
            <ImageUpload
              kind={MEDIA_KIND[s.key] ?? "platform_logo"}
              label=""
              initialUrl={typeof s.value === "string" && s.value ? s.media_url : null}
              onChange={(id) => requestSave(id)}
            />
          ) : s.kind === "longtext" ? (
            <div className="w-full space-y-2">
              <Textarea
                id={s.key}
                rows={12}
                value={draft}
                disabled={!editable || busy}
                onChange={(e) => {
                  setDraft(e.target.value);
                  setError("");
                }}
                placeholder={S.longTextHint}
              />
              <p className="text-xs leading-relaxed text-ink-muted">{S.longTextHint}</p>
            </div>
          ) : s.kind === "geo" ? (
            <div className="w-full space-y-2">
              <div className="overflow-hidden rounded-control border border-line">
                <PickMap
                  api={api}
                  lat={geoOf(draft)?.[0] ?? null}
                  lng={geoOf(draft)?.[1] ?? null}
                  height="h-56"
                  onPick={(la, ln) => {
                    if (!editable || busy) return;
                    requestSave(`${la.toFixed(6)},${ln.toFixed(6)}`);
                  }}
                />
              </div>
              {draft ? (
                <div className="flex items-center gap-2">
                  <p className="text-xs text-ink-muted" dir="ltr">
                    {draft}
                  </p>
                  {editable && (
                    <Button variant="ghost" disabled={busy} onClick={() => requestSave("")}>
                      {U.clear}
                    </Button>
                  )}
                </div>
              ) : null}
            </div>
          ) : (
            <>
              <Input
                id={s.key}
                type={numeric && !decimal ? "number" : "text"}
                inputMode={decimal ? "decimal" : numeric ? "numeric" : undefined}
                dir={decimal ? "ltr" : undefined}
                min={numeric ? min : undefined}
                max={max}
                value={draft}
                disabled={!editable || busy}
                onChange={(e) => {
                  setDraft(e.target.value);
                  setError("");
                }}
                className={numeric ? "w-40" : "w-full"}
                wrapperClassName={numeric ? "" : "min-w-0 flex-1"}
              />
              {unit && <span className="pb-2 text-sm font-bold text-ink-muted">{unitText(unit)}</span>}
            </>
          )}
          {editable && dirty && s.kind !== "percent" && s.kind !== "geo" && (
            <Button type="submit" disabled={busy} className="w-full sm:w-auto">
              {m.common.save}
            </Button>
          )}
        </div>

        {/* **المدى والافتراضيّ لكلّ رقم** — كان يختفي حين لا يُرسَل الحدُّ الأدنى (صفر). */}
        {(numeric || s.kind === "percent") && (
          <p className="mt-1.5 text-xs text-ink-muted">
            {max !== undefined && (
              <>
                {S.range.replace("{min}", fmtNum(min)).replace("{max}", fmtNum(max))}
                {" · "}
              </>
            )}
            {S.defaultIs.replace("{v}", valueText(s, s.default, unit))}
          </p>
        )}

        {error && <p className="mt-1.5 text-xs text-danger">{error}</p>}
        {note && <p className="mt-1.5 text-xs text-ink-muted">{note}</p>}
        {/* **ورقمُ النسخة واسمُها يُقرآن من الملفّ المرفوع** (٢٠٢٦-١٠-٠٦) — لا صلاحيّةَ تفتحهما. */}
        {!editable && (
          <p className="mt-1.5 text-xs text-ink-muted">
            {s.read_only ? U.fromFile : U.readOnly.replace("{cap}", capName(capOf(s)))}
          </p>
        )}
        {s.risk && editable && (dirty || s.kind === "bool" || s.kind === "choice" || s.kind === "percent") && !bare && (
          <Alert tone="warning" className="mt-2">
            {s.risk === "money" ? U.moneyHint : U.securityHint}
          </Alert>
        )}
      </Wrap>

      {/* **«كان ← يصير» قبل الحفظ** (البند ٨)، **وتأكيدٌ لكلّ باب إطلاق** (البند ١٠). */}
      <Confirm
        open={pending !== null}
        title={confirmTitle}
        body={
          gate ? (
            gateOpen ? U.gate.openBody : U.gate.closeBody
          ) : (
            <>
              <p className="font-medium">
                {label(s.key)}:{" "}
                {U.confirmBody
                  .replace("{from}", valueText(s, s.value, unit))
                  .replace("{to}", valueText(s, pending?.raw, unit))}
              </p>
              <p className="mt-1 text-sm text-ink-muted">{confirmNote}</p>
            </>
          )
        }
        confirmLabel={gate ? (gateOpen ? U.gate.open : U.gate.close) : U.confirmAction}
        tone={gate && !gateOpen ? "danger" : "primary"}
        busy={busy}
        onConfirm={() => void doSave(pending?.raw)}
        onCancel={() => {
          setPending(null);
          setDraft(String(s.value ?? ""));
        }}
      />
    </>
  );

  if (bare) return <div>{body}</div>;
  return (
    <Card tone={s.risk ? "accent" : "default"} className={WIDE.has(s.kind) ? "col-span-full" : ""}>
      {body}
    </Card>
  );
}
