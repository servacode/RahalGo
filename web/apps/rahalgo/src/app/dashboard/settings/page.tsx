"use client";

/**
 * الإعدادات — أخطر شاشة في المنصة.
 *
 * **والتعريفُ كلُّه من الخادم**: نوعُ الحقل ومداه وخياراته، **وخطورتُه**
 * (`risk` من قائمة الخطورة الواحدة) **وبيتُه في العمود الجانبيّ** (`topic`).
 * ولو كُتب شيءٌ منها هنا لانحرف عن المحرّك يوماً.
 *
 * ══════════════════════════════════════════════════════════════════════
 * **قراراتُ المالك ٢٠٢٦-١٠-٠٤ (قسم الإعدادات)**
 * ══════════════════════════════════════════════════════════════════════
 *
 *  - **عمودٌ جانبيٌّ بالموضوع** بدل ستّة عشر تبويباً (البند ١٧)، **وكلُّ إعدادٍ
 *    في مكانٍ واحد** — المكرّرُ يُضبط في لوحه المخصَّص (البند ١٢).
 *  - **مثالٌ ماليٌّ حيٌّ يحسبه المحرّك** فوق مفاتيح المال.
 *  - **«كان ← يصير» قبل حفظ أيّ مفتاحٍ ماليّ** ثمّ كلمةُ المرور (البند ٨).
 *  - **رابطُ «السجل» على كلّ بطاقة** (البند ٩).
 *  - **كلُّ بابِ إطلاقٍ يُبدَّل بتأكيد** (البند ١٠).
 *  - **رسالةٌ لكلّ حالة فشل، ورجوعُ الحقل إلى المحفوظ، ولا حفظَ بلا تغيير.**
 *  - **بحثٌ لا يفهم التشكيل يدوّر في الألواح أيضاً.**
 *  - **ومن لا يملك التعديل يُقال له أيُّ صلاحيّةٍ تلزم.**
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Tabs,
  Alert,
  Confirm,
  PageHeader, Button, Input, Textarea, Switch, Badge, Card, EmptyState,
  IconSettings, IconWarning, IconCheck,
  LoadingState,
} from "@rahalgo/ui";
import ImageUpload from "@/components/admin/ImageUpload";
import { FileUpload } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import dynamic from "next/dynamic";
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

/** **«عرض,طول» ← رقمان** — وفارغٌ أو مشوَّهٌ يعني «لا موقع». */
function geoOf(v: string): [number, number] | null {
  const p = v.split(",");
  if (p.length !== 2) return null;
  const la = Number(p[0]);
  const ln = Number(p[1]);
  return Number.isFinite(la) && Number.isFinite(ln) ? [la, ln] : null;
}

const m = getMessages(defaultLocale);

/* **والخريطةُ لا تُصيَّر في الخادم** — `leaflet` يقرأ `window` عند التحميل. */
const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), {
  ssr: false,
});
const S = m.admin.settings;
const U = S.ui;

/** **يطابق أنواعَ الكتالوج في المحرّك** — ونوعٌ يُضاف هناك ولا يُضاف هنا يسقط إلى الحقل النصّيّ. */
type Kind =
  | "int"
  | "money"
  | "bool"
  | "choice"
  | "text"
  | "media"
  | "percent"
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
  /** **بيتُه في العمود الجانبيّ** — بقواعد البادئة في المحرّك. */
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
const boolText = (k: string, side: "on" | "off") =>
  (S.boolStates as Record<string, { on: string; off: string }>)[k]?.[side] ??
  (side === "on" ? S.boolOn : S.boolOff);
const topicLabel = (t: string) => (S.topics as Record<string, string>)[t] ?? t;
const topicSectionLabel = (t: string) => (S.topicSections as Record<string, string>)[t] ?? "";
const sectionLabel = (name: string) => (S.sections as Record<string, string>)?.[name] ?? name;

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

/** **اسمُ الصلاحيّة اللازمة لتحرير المفتاح** — بالمنطق نفسِه في المحرّك. */
function capOf(s: Setting): string {
  if (s.key.startsWith("security.")) return "settings.security.manage";
  if (s.risk) return "settings.financial.manage";
  return "settings.general.manage";
}
const capName = (c: string) => (U.capNames as Record<string, string>)[c] ?? c;

const GRID = "grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(20rem,1fr))]";
const WIDE: ReadonlySet<Kind> = new Set<Kind>(["geo", "media", "longtext", "file"]);

const MEDIA_KIND: Record<string, "platform_logo" | "auth_background" | "site_background"> = {
  "platform.logo": "platform_logo",
  "auth.background": "auth_background",
  "auth.background_mobile": "auth_background",
  "platform.background": "site_background",
  "platform.background_mobile": "site_background",
};

/** **الألواحُ المخصَّصةُ في العمود** — والبحثُ يجدها بأسمائها. */
const PANELS: { key: string; topic: string; label: string; adminOnly?: boolean }[] = [
  { key: "appStatus", topic: "launch", label: m.admin.appStatus.title },
  { key: "hours", topic: "launch", label: m.admin.platformHours.title, adminOnly: true },
  { key: "release", topic: "apps", label: m.admin.release.title },
  { key: "slider", topic: "site", label: m.admin.sliderPanel.title },
  { key: "divisions", topic: "coverage", label: m.admin.divisions.title, adminOnly: true },
  { key: "cities", topic: "coverage", label: m.admin.cities.title, adminOnly: true },
  { key: "zones", topic: "coverage", label: m.terms.zones, adminOnly: true },
  { key: "whatsapp", topic: "whatsapp", label: m.admin.nav.whatsapp, adminOnly: true },
  { key: "broadcast", topic: "whatsapp", label: m.admin.broadcast.title, adminOnly: true },
];

/** **الألواحُ تُضاف بعد مواضيع المفاتيح.** */
const PANEL_TOPICS = ["coverage", "whatsapp"];

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

export default function SettingsPage() {
  const { can } = useAuth();
  const isAdmin = can("settings.general.manage");
  const [list, setList] = useState<Setting[] | null>(null);
  const [topics, setTopics] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [topic, setTopic] = useState("");
  const [sub, setSub] = useState("");
  const [q, setQ] = useState("");

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

  useEffect(() => {
    void load();
    // **ورابطٌ من الرئيسيّة يفتح موضوعَه** (`?topic=site`).
    if (typeof window !== "undefined") {
      const t = new URLSearchParams(window.location.search).get("topic");
      if (t) setTopic(t);
    }
  }, [load]);

  // **وموضوعا اللوحات لمن يملك الإعدادَ العامّ** (فحصُ المتصفّح ٢٠٢٦-١٠-٠٥): الماليّةُ
  // ترى صفحةَ الإعدادات بمفاتيحها الماليّة، **وكانت التغطيةُ وواتساب وساعاتُ المنصّة
  // تُفتح لها فتُردّ ٤٠٣** — لوحاتُها تنادي أبواباً بـ`settings.general.manage`.
  const allTopics = useMemo(
    () => [...topics, ...(isAdmin ? PANEL_TOPICS : [])],
    [topics, isAdmin],
  );

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
    const panels = PANELS.filter(
      (p) => (!p.adminOnly || isAdmin) && norm(p.label).includes(needle),
    );
    return { keys, panels };
  }, [q, list, isAdmin]);

  if (!list && error) return <Alert>{error}</Alert>;
  if (!list) return <LoadingState variant="text" />;

  const active = topic && allTopics.includes(topic) ? topic : (allTopics[0] ?? "");
  const readOnlyAll = list.length > 0 && list.every((s) => !(s.editable ?? false));
  const goPanel = (t: string, panel: string) => {
    setTopic(t);
    setSub(panel);
    setQ("");
  };

  const row = (s: Setting) => (
    <SettingRow key={s.key} s={s} all={list} editable={s.editable ?? false} onSaved={load} />
  );

  return (
    <div>
      <PageHeader icon={IconSettings} title={m.admin.settingsPage.title} />
      <p className="mb-4 text-sm text-ink-muted">{m.admin.settingsPage.hint}</p>

      {error && <Alert className="mb-4">{error}</Alert>}
      {readOnlyAll && (
        <Alert tone="info" className="mb-4">
          {U.readOnly.replace("{cap}", capName("settings.general.manage"))}
        </Alert>
      )}

      <div className="mb-4">
        <Input
          id="settings-search"
          label=""
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={S.searchPlaceholder}
          wrapperClassName="max-w-md"
        />
      </div>

      {found !== null ? (
        found.keys.length === 0 && found.panels.length === 0 ? (
          <EmptyState icon={IconSettings} title={S.searchNoResults} />
        ) : (
          <div className="space-y-3">
            <p className="text-xs text-ink-muted">
              {S.searchCount.replace("{n}", fmtNum(found.keys.length + found.panels.length))}
            </p>
            {found.panels.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {found.panels.map((p) => (
                  <Button key={p.key} variant="secondary" onClick={() => goPanel(p.topic, p.key)}>
                    {p.label} · {U.searchIn.replace("{topic}", topicLabel(p.topic))}
                  </Button>
                ))}
              </div>
            )}
            <div className={GRID}>
              {found.keys.map((s) =>
                s.panel ? (
                  <Card key={s.key}>
                    <p className="font-medium">{label(s.key)}</p>
                    <p className="mt-1 text-xs text-ink-muted">
                      {U.searchIn.replace("{topic}", topicLabel(s.topic))}
                    </p>
                    <Button
                      variant="secondary"
                      className="mt-2"
                      onClick={() => goPanel(s.topic, s.panel!)}
                    >
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
        <div className="flex flex-col gap-4 md:flex-row">
          {/* **العمودُ الجانبيّ بالموضوع** — وفي الشاشة الضيّقة شريطٌ يُمرَّر أفقيّاً. */}
          <nav className="flex shrink-0 gap-1 overflow-x-auto md:w-56 md:flex-col md:overflow-visible">
            {allTopics.map((t) => (
              <button
                key={t}
                type="button"
                aria-current={active === t}
                onClick={() => {
                  setTopic(t);
                  setSub("");
                }}
                className={`whitespace-nowrap rounded-control px-3 py-2 text-start text-sm transition-colors ${
                  active === t
                    ? "bg-primary font-bold text-on-bright elev-1"
                    : "text-ink-muted hover:bg-row-hover hover:text-ink"
                }`}
              >
                {topicLabel(t)}
              </button>
            ))}
          </nav>

          <div className="min-w-0 flex-1 space-y-4">
            <TopicBody
              topic={active}
              list={list}
              sub={sub}
              setSub={setSub}
              isAdmin={isAdmin}
              row={row}
            />
          </div>
        </div>
      )}
    </div>
  );
}

function TopicBody({
  topic,
  list,
  sub,
  setSub,
  isAdmin,
  row,
}: {
  topic: string;
  list: Setting[];
  sub: string;
  setSub: (v: string) => void;
  isAdmin: boolean;
  row: (s: Setting) => React.ReactNode;
}) {
  const items = list.filter(
    (s) => s.topic === topic && !s.hidden && !s.panel && visibleIn(s, list),
  );
  // **والسلايدرُ بابُه `content.manage`** — لا مفتاحُ الإعدادات.
  const canContent = useAuth().can("content.manage");

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
        <WhatsAppPanel />
        {isAdmin && (
          <div className="space-y-3">
            <h2 className="heading-card">{m.admin.broadcast.title}</h2>
            <p className="text-sm text-ink-muted">{m.admin.broadcast.hint}</p>
            <BroadcastPanel />
            <CampaignsPanel />
          </div>
        )}
      </>
    );
  }

  /* **صفحاتُ الموقع تبويباتٌ بأسمائها** — ومنها «من نحن» وتعليماتُ السائق
     والمندوب والمتجر (البند ٢٧): كانت لا تُبلَغ إلّا بالبحث. */
  const sections = groupBy(items, (s) => s.topic_section ?? "");
  return (
    <>
      {topic === "money" && <MoneyExample version={list} />}
      {topic === "launch" && <AppStatusPanel />}
      {topic === "apps" && <ReleasePanel />}
      {/* **وسلايدرُ التطبيق رجع إلى بيته** (قرارُ المالك ٢٠٢٦-١٠-٠٥): كان في قسم
          الموقع، **فلمّا خُبّئت إعداداتُ الموقع ذهب معها** — والتطبيقُ ما زال
          يقرأ لافتاتِه، فبقيت سبعُ صورٍ تُعرض ولا بابَ يُبدّلها. */}
      {topic === "site" && canContent && (
        <section className="space-y-2">
          <h2 className="heading-card">{m.admin.sliderPanel.title}</h2>
          <p className="text-sm text-ink-muted">{m.admin.sliderPanel.hint}</p>
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
            <div className={GRID}>{sec.items.map(row)}</div>
          </section>
        );
      })}

      {topic === "launch" && isAdmin && <HoursPanel />}
      {sections.length === 0 && topic !== "launch" && topic !== "apps" && (
        <EmptyState icon={IconSettings} title={S.emptyGroup} />
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
 * (`/admin/settings/money-example`)، ويُعاد بعد كلّ حفظ.
 */
function MoneyExample({ version }: { version: unknown }) {
  const [ex, setEx] = useState<Example | null>(null);
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    api<Example>("/api/v1/admin/settings/money-example?amount=50000")
      .then((r) => {
        setEx(r);
        setFailed(false);
      })
      .catch(() => setFailed(true));
  }, [version]);
  const money = (n: number) => fmtMoney(n);
  if (failed) return <Alert tone="warning">{U.example.failed}</Alert>;
  if (!ex) return <LoadingState variant="text" />;
  const lines: [string, number][] = [
    [U.example.customer, ex.customer_pays],
    [U.example.merchant, ex.merchant_gets],
    [U.example.driver, ex.driver_gets],
    [U.example.rep, ex.rep_gets],
    [U.example.platform, ex.platform_gets],
  ];
  return (
    <Card tone="accent">
      <p className="mb-1 font-medium">{U.example.title}</p>
      <p className="mb-2 text-sm text-ink-muted">
        {U.example.intro.replace("{amount}", money(ex.amount)).replace("{fee}", money(ex.delivery_fee))}
      </p>
      <ul className="grid grid-cols-1 gap-1 text-sm sm:grid-cols-2">
        {lines.map(([k, v]) => (
          <li key={k} className="flex justify-between gap-3">
            <span className="text-ink-muted">{k}</span>
            <span className="font-medium tabular-nums">{money(v)}</span>
          </li>
        ))}
      </ul>
    </Card>
  );
}

/** **قيمةٌ بكلماتها** — للتأكيد «كان ← يصير». */
function valueText(s: Setting, v: unknown): string {
  if (s.kind === "bool") return boolText(s.key, v === true ? "on" : "off");
  if (s.kind === "choice") return choiceText(String(v ?? ""));
  if (s.kind === "percent") return `${fmtNum(Number(v ?? 0))}${unitText("percent")}`;
  if (s.kind === "int" || s.kind === "money") {
    const u = unitText(s.unit);
    return u ? `${fmtNum(Number(v ?? 0))} ${u}` : fmtNum(Number(v ?? 0));
  }
  return String(v ?? "");
}

/** **أهي القيمةُ المحفوظةُ نفسُها؟** — فلا يُحفظ ما لم يتغيّر. */
function same(s: Setting, raw: unknown): boolean {
  if (s.kind === "int" || s.kind === "money" || s.kind === "percent") {
    return Number(raw) === Number(s.value);
  }
  return raw === s.value || String(raw) === String(s.value ?? "");
}

/** **رسالةٌ لكلّ حالة فشل** (البند ١١) — لا «المدى المسموح» لكلّ رفض. */
function saveError(s: Setting, err: unknown): string {
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
    if ((s.kind === "int" || s.kind === "money" || s.kind === "percent") && s.max !== undefined) {
      return U.errors.range
        .replace("{min}", fmtNum(s.min ?? 0))
        .replace("{max}", fmtNum(s.max));
    }
    return U.errors.invalid;
  }
  return errorText(err);
}

/**
 * صفٌّ واحد — يحرّر نفسه في مكانه.
 */
function SettingRow({
  s,
  all,
  editable,
  onSaved,
}: {
  s: Setting;
  all: Setting[];
  editable: boolean;
  onSaved: () => void;
}) {
  const [draft, setDraft] = useState<string>(() => String(s.value ?? ""));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [note, setNote] = useState("");
  const [saved, setSaved] = useState(false);
  /** **ما ينتظر تأكيداً** — قيمةٌ ماليّةٌ أو بابُ إطلاق. */
  const [pending, setPending] = useState<{ raw: unknown } | null>(null);
  void all;

  useEffect(() => {
    setDraft(String(s.value ?? ""));
  }, [s.value]);

  const numeric = s.kind === "int" || s.kind === "money";
  const gate = s.topic_section === "launch.gates" && s.kind === "bool";
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
      setError(saveError(s, err));
      // **والحقلُ يرجع إلى المحفوظ** (البند ١٥) — لا يبقى عارضاً ما لم يُحفظ.
      setDraft(String(s.value ?? ""));
    } finally {
      setBusy(false);
      setPending(null);
    }
  }

  /** **الحفظُ يمرّ من هنا** — بلا تغييرٍ لا نداء، والماليُّ والبابُ بتأكيد. */
  function requestSave(raw: unknown) {
    if (same(s, raw)) {
      setDraft(String(s.value ?? ""));
      setNote(U.unchanged);
      setTimeout(() => setNote(""), 1500);
      return;
    }
    if (s.risk === "money" || gate) {
      setPending({ raw });
      return;
    }
    void doSave(raw);
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (numeric) {
      const n = Number(draft);
      if (!Number.isInteger(n)) return setError(U.errors.invalid);
      if ((s.min !== undefined && n < s.min) || (s.max !== undefined && n > s.max)) {
        return setError(
          U.errors.range.replace("{min}", fmtNum(s.min ?? 0)).replace("{max}", fmtNum(s.max ?? 0)),
        );
      }
      return requestSave(n);
    }
    requestSave(draft);
  }

  const sliderMax = s.max && s.max > 0 ? s.max : 100;
  const sliderMin = s.min ?? 0;
  const commitSlider = () => requestSave(Number(draft === "" ? 0 : draft));

  const gateOpen = pending?.raw === true;
  const Wrap = s.kind === "geo" ? "div" : "form";

  return (
    <Card
      tone={s.risk ? "accent" : "default"}
      className={WIDE.has(s.kind) ? "col-span-full" : ""}
    >
      {/* **والموقعُ ليس نموذجاً** (فحصُ المتصفّح ٢٠٢٦-١٠-٠٥): لاقطُه فيه نموذجُ بحثه، **ونموذجٌ
          داخلَ نموذجٍ خطأُ ترطيبٍ في الطرفيّة** — وهو يُحفظ بالضغط على الخريطة لا بزرّ. */}
      <Wrap onSubmit={s.kind === "geo" ? undefined : submit}>
        <div className="mb-1 flex flex-wrap items-center gap-2">
          <span className="font-medium">{label(s.key)}</span>
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
          {/* **رابطُ «السجل»** (البند ٩) — تاريخُ هذا المفتاح وحدَه، بلا سطر «آخر تغيير». */}
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

        <div className="flex flex-wrap items-end gap-2">
          {s.kind === "bool" ? (
            <Switch
              id={s.key}
              label=""
              checked={s.value === true}
              disabled={!editable || busy}
              onChange={(next) => requestSave(next)}
            />
          ) : s.kind === "choice" ? (
            <div className="flex flex-wrap gap-1 rounded-control border border-line bg-field p-1">
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
                    className={`rounded-control px-3 py-1.5 text-sm transition-colors disabled:opacity-50 ${
                      on
                        ? "bg-primary font-bold text-on-bright elev-1"
                        : "text-ink-muted hover:bg-surface hover:text-ink"
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
              <span className="w-14 shrink-0 text-end text-sm font-medium tabular-nums text-primary-dark">
                {fmtNum(draft === "" ? 0 : Number(draft))}
                {unitText("percent")}
              </span>
            </div>
          ) : s.kind === "file" ? (
            /* **والرفعُ يذهب إلى مفتاحه** (البند ٥) — كان يذهب إلى المفتاح القديم. */
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
                type={numeric ? "number" : "text"}
                inputMode={numeric ? "numeric" : undefined}
                min={s.min}
                max={s.max}
                value={draft}
                disabled={!editable || busy}
                onChange={(e) => {
                  setDraft(e.target.value);
                  setError("");
                }}
                className={numeric ? "w-40" : "w-full"}
                wrapperClassName={numeric ? "" : "min-w-0 flex-1"}
              />
              {s.unit && <span className="pb-2 text-sm text-ink-muted">{unitText(s.unit)}</span>}
            </>
          )}
          {editable && dirty && s.kind !== "percent" && s.kind !== "geo" && (
            <Button type="submit" disabled={busy}>
              {m.common.save}
            </Button>
          )}
        </div>

        {(numeric || s.kind === "percent") && s.min !== undefined && s.max !== undefined && (
          <p className="mt-1.5 text-xs text-ink-muted">
            {S.range.replace("{min}", fmtNum(s.min)).replace("{max}", fmtNum(s.max))}
            {" · "}
            {S.defaultIs.replace("{v}", valueText(s, s.default))}
          </p>
        )}

        {error && <p className="mt-1.5 text-xs text-danger">{error}</p>}
        {note && <p className="mt-1.5 text-xs text-ink-muted">{note}</p>}
        {!editable && (
          <p className="mt-1.5 text-xs text-ink-muted">
            {U.readOnly.replace("{cap}", capName(capOf(s)))}
          </p>
        )}
        {s.risk && editable && (dirty || s.kind === "bool" || s.kind === "choice" || s.kind === "percent") && (
          <Alert tone="warning" className="mt-2">
            {s.risk === "money" ? U.moneyHint : U.securityHint}
          </Alert>
        )}
      </Wrap>

      {/* **«كان ← يصير» قبل حفظ المال** (البند ٨)، **وتأكيدٌ لكلّ باب إطلاق** (البند ١٠).
          ثمّ يطلب المحرّكُ كلمةَ المرور لمفاتيح المال. */}
      <Confirm
        open={pending !== null}
        title={
          gate
            ? (gateOpen ? U.gate.openTitle : U.gate.closeTitle).replace("{name}", label(s.key))
            : U.confirmTitle
        }
        body={
          gate ? (
            gateOpen ? U.gate.openBody : U.gate.closeBody
          ) : (
            <>
              <p className="font-medium">
                {label(s.key)}:{" "}
                {U.confirmBody
                  .replace("{from}", valueText(s, s.value))
                  .replace("{to}", valueText(s, pending?.raw))}
              </p>
              <p className="mt-1 text-sm text-ink-muted">{U.confirmNote}</p>
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
    </Card>
  );
}
