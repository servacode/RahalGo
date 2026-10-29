"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قِطَعُ الإعداد المشتركة — للمعالج ولصفحات «متجري» معاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StoreParts.kt` في تطبيق المتجر، ملاحظاتُ المالك ٢٠٢٦-١٠-٢٨.)
 * **والقطعةُ الواحدةُ في الموضعين** — فما يراه في المعالج يراه حين يعدّل لاحقاً.
 *
 * - **شبكةُ الأقسام**: ثلاثةُ مربّعاتٍ بصورها وبحثٌ فوقها، والمختارُ بحدٍّ وعلامةِ صحّ.
 * - **الخريطةُ الصغيرة**: لاقطُ الموقع نفسُه (`PickMap`) ظاهراً في الصفحة، لا نافذةً فوقها.
 * - **بطاقتا اختيارٍ بأيقونة**: الاستلامُ والترجيعُ والجمعة.
 * - **مخطّطُ الدوام**: «٢٤ ساعة» أو «ساعات محددة»، سطرٌ لكلّ يوم، وسؤالُ الجمعة.
 */

import Link from "next/link";
import { lazy, Suspense, useCallback, useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, IconCheck, IconClose, IconPrev, IconSearch, Modal, FormActions, Input, Switch } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import type { DayHours, SectionRef } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const H = m.store.hours;
const W = m.store.wizard;
const HM = m.store.home;

/** **الجمعة** — كما يرقّمها المحرّك (٠ الأحد … ٦ السبت). */
export const FRIDAY = 5;

// ══════════════════════════════════════════════════════════════════════
// رأسُ الصفحة الفرعيّة
// ══════════════════════════════════════════════════════════════════════

/** **سهمُ الرجوع وحدَه** — لرأسٍ مشتركٍ يأخذه قبل عنوانه (الإشعارات). */
export function BackArrow({ href = "/store" }: { href?: string }) {
  return (
    <Link
      href={href}
      aria-label={HM.back}
      title={HM.back}
      className="flex h-11 w-11 shrink-0 items-center justify-center rounded-control text-ink transition-colors hover:bg-row-hover"
    >
      <IconPrev size={22} />
    </Link>
  );
}

/**
 * **سهمُ رجوعٍ وعنوان** — مرآةُ `PageHeader`، **ورأسٌ واحدٌ لكلّ صفحةٍ فرعيّة**
 * (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩: صفحاتٌ كانت بسهمٍ وأخرى بلا طريقِ رجوع).
 * و`actions` في آخر السطر إن وُجدت.
 */
export function PageHeader({ title, href = "/store", actions }: { title: string; href?: string; actions?: ReactNode }) {
  return (
    <header className="flex items-center gap-2">
      <BackArrow href={href} />
      <h1 className="heading-page min-w-0 flex-1 truncate">{title}</h1>
      {actions}
    </header>
  );
}

/** **علامةُ «مختار»** — دائرةٌ بلون العلامة فيها صحّ. */
export function CheckBadge({ className = "" }: { className?: string }) {
  return (
    <span
      className={`flex h-6 w-6 items-center justify-center rounded-badge bg-primary-solid text-on-solid ${className}`}
      aria-hidden
    >
      <IconCheck size={14} />
    </span>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ١ · شبكةُ أقسام السوق
// ══════════════════════════════════════════════════════════════════════

export function SectionGrid({
  all,
  picked,
  disabled,
  onToggle,
}: {
  all: SectionRef[];
  picked: Set<string>;
  disabled?: boolean;
  onToggle: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const q = query.trim().toLowerCase();
  const shown = q ? all.filter((s) => s.name.toLowerCase().includes(q)) : all;
  return (
    <div className="flex flex-col gap-3" data-testid="section-grid">
      <Input
        id="section-search"
        aria-label={W.sectionsSearch}
        placeholder={W.sectionsSearch}
        value={query}
        maxLength={40}
        onChange={(e) => setQuery(e.target.value)}
        icon={<IconSearch size={18} />}
      />
      <p className={`text-sm font-bold ${picked.size > 0 ? "text-primary" : "text-ink-muted"}`}>
        {fill(W.sectionsPicked, { n: picked.size })}
      </p>
      {shown.length === 0 ? (
        <p className="py-3 text-sm text-ink-muted">{W.sectionsNoMatch}</p>
      ) : (
        <ul className="grid grid-cols-3 gap-3 sm:grid-cols-4 lg:grid-cols-6">
          {shown.map((s) => {
            const on = picked.has(s.id);
            const img = mediaUrl(s.image_thumb_url || s.image_url || null);
            return (
              <li key={s.id}>
                <button
                  type="button"
                  role="checkbox"
                  aria-checked={on}
                  disabled={disabled}
                  onClick={() => onToggle(s.id)}
                  className="flex w-full flex-col items-center gap-1 rounded-control p-1 text-center disabled:opacity-60"
                >
                  <span
                    className={`relative block aspect-square w-full overflow-hidden rounded-control bg-field ${
                      on ? "border-2 border-primary-solid" : "border border-line"
                    }`}
                  >
                    {img ? (
                      <img src={img} alt="" className="h-full w-full object-cover" loading="lazy" />
                    ) : (
                      <span className="figure flex h-full w-full items-center justify-center text-ink-muted">
                        {s.name.slice(0, 1)}
                      </span>
                    )}
                    {on && <CheckBadge className="absolute end-1 top-1" />}
                  </span>
                  <span className={`line-clamp-2 text-sm ${on ? "font-bold text-primary" : ""}`}>{s.name}</span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ٢ · الخريطةُ الصغيرة
// ══════════════════════════════════════════════════════════════════════

const PickMap = lazy(() => import("@rahalgo/ui/map").then((mod) => ({ default: mod.PickMap })));

/**
 * **موقعُ المتجر على خريطةٍ ظاهرة** — لاقطُ المنصّة الواحد (`PickMap`) بزرّ
 * «تحديد موقعي» فوقه، **لا نافذةَ ملءَ الشاشة.** وتحتها حالُ النقطة.
 */
export function StorePointMap({
  lat,
  lng,
  disabled,
  onPick,
}: {
  lat: number | null;
  lng: number | null;
  disabled?: boolean;
  onPick: (lat: number, lng: number) => void;
}) {
  const set = lat != null && lng != null;
  return (
    <div className={`flex flex-col gap-2 ${disabled ? "pointer-events-none opacity-60" : ""}`} data-testid="store-point-map">
      <Suspense fallback={<div className="h-64 w-full animate-pulse rounded-card bg-field" />}>
        <PickMap lat={lat} lng={lng} height="h-64" api={api} onPick={onPick} onLocated={onPick} />
      </Suspense>
      <p className={`text-sm font-bold ${set ? "text-success" : "text-danger"}`}>{set ? W.pointSet : W.pointNone}</p>
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ٣ · بطاقاتُ الاختيار
// ══════════════════════════════════════════════════════════════════════

export interface IconChoice {
  icon: ComponentType<{ size?: number; className?: string }>;
  title: string;
  selected: boolean;
  onClick: () => void;
}

/** **بطاقتان كبيرتان متجاورتان** — أيقونةٌ في دائرةٍ وعنوان، والمختارةُ بحدّ العلامة وعلامةِ صحّ. */
export function IconChoicePair({ items, disabled }: { items: readonly [IconChoice, IconChoice]; disabled?: boolean }) {
  return (
    <div className="grid grid-cols-2 gap-3" role="radiogroup">
      {items.map((c) => {
        const Icon = c.icon;
        return (
          <button
            key={c.title}
            type="button"
            role="radio"
            aria-checked={c.selected}
            disabled={disabled}
            onClick={c.onClick}
            className={`relative flex min-h-36 flex-col items-center justify-center gap-3 rounded-card px-3 py-4 text-center transition-colors disabled:opacity-60 ${
              c.selected ? "border-2 border-primary-solid bg-primary-tint" : "border border-line bg-surface hover:bg-row-hover"
            }`}
          >
            <span
              className={`flex h-14 w-14 items-center justify-center rounded-badge ${
                c.selected ? "bg-primary-solid text-on-solid" : "bg-primary-tint text-primary"
              }`}
            >
              <Icon size={28} />
            </span>
            <span className="text-base font-bold">{c.title}</span>
            {c.selected && <CheckBadge className="absolute end-2 top-2" />}
          </button>
        );
      })}
    </div>
  );
}

/** **بطاقةُ اختيارٍ عريضة** — دائرةٌ وعنوان، والمختارةُ بحدّ العلامة. */
export function ChoiceCard({
  title,
  selected,
  disabled,
  onClick,
}: {
  title: string;
  selected: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      disabled={disabled}
      onClick={onClick}
      className={`flex min-h-14 w-full items-center gap-3 rounded-control px-4 py-3 text-start transition-colors disabled:opacity-60 ${
        selected ? "border-2 border-primary-solid bg-primary-tint" : "border border-line bg-surface hover:bg-row-hover"
      }`}
    >
      <span
        className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-badge border-2 ${
          selected ? "border-primary-solid" : "border-line"
        }`}
        aria-hidden
      >
        {selected && <span className="h-2.5 w-2.5 rounded-badge bg-primary-solid" />}
      </span>
      <span className="min-w-0 flex-1 text-base font-bold">{title}</span>
    </button>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ٤ · مخطّطُ الدوام
// ══════════════════════════════════════════════════════════════════════

const hhmm = (t: string) => (t ?? "").slice(0, 5);
const DEFAULT_OPEN = "09:00";
const DEFAULT_CLOSE = "23:00";

/** **مرآةُ `clock12`** — «٩:٠٠ م» لا «21:00». */
export function clock12(t: string): string {
  const h = Number(t.slice(0, 2));
  if (!Number.isFinite(h)) return t;
  const mm = t.slice(3, 5) || "00";
  const h12 = h === 0 ? 12 : h > 12 ? h - 12 : h;
  return `${h12}:${mm} ${h < 12 ? H.am : H.pm}`;
}

/** **«مفتوح ٢٤ ساعة»** — الرايةُ من الخادم، أو ٠٠:٠٠ ← ٠٠:٠٠ من خادمٍ أقدم. */
export function is24h(d: DayHours): boolean {
  return d.open_24h ?? (hhmm(d.open_time) === "00:00" && hhmm(d.close_time) === "00:00");
}

function with24h(d: DayHours, on: boolean): DayHours {
  if (on) return { ...d, closed: false, open_24h: true, open_time: "00:00", close_time: "00:00" };
  return { ...d, open_24h: false, open_time: DEFAULT_OPEN, close_time: DEFAULT_CLOSE };
}

/** **مرآةُ `opensAtNight`** — يفتح بين ٠ و٥ صباحاً، إلّا الدوامَ الكاملَ و«٢٤ ساعة». */
export function nightOpen(d: DayHours): boolean {
  if (d.closed || is24h(d)) return false;
  const o = Number(d.open_time.slice(0, 2));
  const c = Number(d.close_time.slice(0, 2));
  const oo = Number.isFinite(o) ? o : 9;
  const cc = Number.isFinite(c) ? c : 23;
  return oo >= 0 && oo <= 5 && !(oo === 0 && cc >= 23);
}

/** **أيتساوى الدوامان معنًى؟** — فلا يظهر «حفظ» على ما لم يتغيّر. */
export function sameHours(a: DayHours[], b: DayHours[]): boolean {
  if (a.length !== b.length) return false;
  const bb = new Map(b.map((d) => [d.day_of_week, d]));
  return a.every((x) => {
    const y = bb.get(x.day_of_week);
    if (!y || x.closed !== y.closed) return false;
    if (x.closed) return true;
    if (is24h(x) || is24h(y)) return is24h(x) && is24h(y);
    return hhmm(x.open_time) === hhmm(y.open_time) && hhmm(x.close_time) === hhmm(y.close_time);
  });
}

/** **وقتُ اليوم في سطر** — «٢٤ ساعة» أو «9:00 ص – 11:00 م». */
export function rangeText(d: DayHours): string {
  return is24h(d) ? H.day24 : fill(HM.hoursRange, { from: clock12(hhmm(d.open_time)), to: clock12(hhmm(d.close_time)) });
}

/** **أيّامُ الأسبوع السبعةُ من المحرّك** — مرتّبةً وبوقتٍ «HH:MM». */
export function useStoreHours(storeId: string) {
  const [saved, setSaved] = useState<DayHours[] | null>(null);
  const [error, setError] = useState<unknown>(null);
  const load = useCallback(async () => {
    if (!storeId) return;
    try {
      const rows = await api<DayHours[]>(`/api/v1/merchant/stores/${storeId}/hours`);
      setSaved(
        [...(rows ?? [])]
          .map((d) => ({ ...d, open_time: hhmm(d.open_time), close_time: hhmm(d.close_time) }))
          .map((d) => ({ ...d, open_24h: is24h(d) }))
          .sort((a, b) => a.day_of_week - b.day_of_week),
      );
      setError(null);
    } catch (e) {
      setError(e);
    }
  }, [storeId]);
  useEffect(() => {
    void load();
  }, [load]);
  return { saved, setSaved, error, reload: load };
}

/** **حفظُ الأيّام السبعة** — `PUT /hours`؛ والمحرّكُ يريدها معاً. */
export async function putHours(storeId: string, days: DayHours[]): Promise<void> {
  await api(`/api/v1/merchant/stores/${storeId}/hours`, { method: "PUT", body: JSON.stringify({ days }) });
}

export interface HoursPlan {
  mode: "" | "24" | "custom";
  friday: "" | "on" | "off";
  draft: DayHours[];
  chosen: boolean;
  chooseMode: (m: "24" | "custom") => void;
  chooseFriday: (on: boolean) => void;
  update: (d: DayHours) => void;
  applyAll: (sample: DayHours) => void;
  days: () => DayHours[];
}

/**
 * **حالُ مخطّط الدوام** — مرآةُ `HoursPlan`.
 *
 * - `mode`: «» لم يختر · «24» مفتوحٌ ٢٤ ساعة · «custom» ساعاتٌ محدّدة.
 * - `friday`: «» لم يُجب · «on» دوام · «off» عطلة — **يُسأل صراحةً في الحالين.**
 *
 * **ويُبنى مرّةً من المحفوظ** — ومن أراد إعادته يغيّر `key` المكوّن.
 */
export function useHoursPlan(saved: DayHours[], hasHours: boolean): HoursPlan {
  const [mode, setMode] = useState<HoursPlan["mode"]>(() => {
    if (!hasHours) return "";
    const o = saved.filter((d) => d.day_of_week !== FRIDAY && !d.closed);
    return o.length > 0 && o.every(is24h) ? "24" : "custom";
  });
  const [friday, setFriday] = useState<HoursPlan["friday"]>(() => {
    if (!hasHours) return "";
    return saved.find((d) => d.day_of_week === FRIDAY)?.closed ? "off" : "on";
  });
  // **ولمن لم يضبط دوامَه بعد: ٩ص–١١م لكلّ يوم** — والجمعةُ حتّى يجيب.
  const [draft, setDraft] = useState<DayHours[]>(() =>
    hasHours
      ? saved
      : saved.map((d) => ({ ...d, closed: false, open_24h: false, open_time: DEFAULT_OPEN, close_time: DEFAULT_CLOSE })),
  );
  const fridayEdited = useRef(hasHours);

  function chooseMode(next: "24" | "custom") {
    let d = draft;
    if (next === "custom" && mode !== "custom") {
      const others = d.filter((x) => x.day_of_week !== FRIDAY && !x.closed);
      if (others.length > 0 && others.every(is24h)) {
        d = d.map((x) => (is24h(x) && (x.day_of_week !== FRIDAY || !fridayEdited.current) ? with24h(x, false) : x));
      }
    }
    if (next === "24" && !fridayEdited.current) {
      d = d.map((x) => (x.day_of_week === FRIDAY ? with24h(x, true) : x));
    }
    setDraft(d);
    setMode(next);
  }

  function chooseFriday(on: boolean) {
    setFriday(on ? "on" : "off");
    if (!on) return;
    const fri = draft.find((x) => x.day_of_week === FRIDAY);
    if (fri && (fri.closed || !fri.open_time || (!fridayEdited.current && mode === "24" && !is24h(fri)))) {
      // **والجمعةُ تبدأ بما في بقيّة الأسبوع** — ٢٤ ساعة أو ساعاتُ أوّل يوم دوام.
      const like = draft.find((x) => x.day_of_week !== FRIDAY && !x.closed);
      const auto =
        mode === "24" || !like
          ? with24h(fri, true)
          : { ...fri, closed: false, open_24h: like.open_24h, open_time: like.open_time, close_time: like.close_time };
      setDraft(draft.map((x) => (x.day_of_week === FRIDAY ? auto : x)));
    }
  }

  function update(d: DayHours) {
    if (d.day_of_week === FRIDAY) fridayEdited.current = true;
    setDraft((all) => all.map((x) => (x.day_of_week === d.day_of_week ? d : x)));
  }

  /** **«طبّق نفس الساعات على كل الأيام»** — على أيّام الدوام وحدَها، والعطلةُ تبقى عطلة. */
  function applyAll(sample: DayHours) {
    setDraft((all) =>
      all.map((x) =>
        x.closed ? x : { ...x, open_24h: sample.open_24h, open_time: sample.open_time, close_time: sample.close_time },
      ),
    );
  }

  function days(): DayHours[] {
    return draft.map((d) => {
      if (d.day_of_week === FRIDAY) return { ...d, closed: friday === "off" };
      if (mode === "24") return with24h(d, true);
      return d;
    });
  }

  return {
    mode,
    friday,
    draft,
    chosen: mode !== "" && friday !== "",
    chooseMode,
    chooseFriday,
    update,
    applyAll,
    days,
  };
}

/** **سطرُ يوم** — اسمُه · مفتاحُ دوام/عطلة · رقاقةُ وقته. ويومُ العطلة رماديّ. */
function DayRow({
  d,
  disabled,
  onToggle,
  onTime,
}: {
  d: DayHours;
  disabled?: boolean;
  onToggle?: (open: boolean) => void;
  onTime: () => void;
}) {
  const open = !d.closed;
  const name = H.days[d.day_of_week] ?? "";
  return (
    <div
      className={`flex min-h-14 items-center gap-3 px-3 py-2 ${open ? "" : "bg-field"}`}
      data-testid="hours-day"
    >
      <span className={`min-w-0 flex-1 text-sm font-bold ${open ? "" : "text-ink-muted"}`}>{name}</span>
      {onToggle && (
        <div className="shrink-0" aria-label={name}>
          <Switch checked={open} disabled={disabled} onChange={onToggle} label="" />
        </div>
      )}
      {open ? (
        <button
          type="button"
          disabled={disabled}
          onClick={onTime}
          className="min-w-36 shrink-0 rounded-control border border-primary-edge bg-primary-tint px-3 py-2 text-center text-sm font-bold text-primary disabled:opacity-60"
        >
          {rangeText(d)}
        </button>
      ) : (
        <span className="min-w-36 shrink-0 px-3 py-2 text-center text-sm text-ink-muted">{H.off}</span>
      )}
    </div>
  );
}

/** **نافذةُ وقت اليوم** — «٢٤ ساعة» أو «ساعات محددة» من/إلى. لا تحفظ؛ تعيد اليومَ إلى المخطّط. */
function DayTimeDialog({
  title,
  start,
  onClose,
  onDone,
}: {
  title: string;
  start: DayHours;
  onClose: () => void;
  onDone: (d: DayHours) => void;
}) {
  const [all24, setAll24] = useState(is24h(start));
  const [from, setFrom] = useState(is24h(start) ? DEFAULT_OPEN : hhmm(start.open_time) || DEFAULT_OPEN);
  const [to, setTo] = useState(is24h(start) ? DEFAULT_CLOSE : hhmm(start.close_time) || DEFAULT_CLOSE);
  return (
    <Modal open onClose={onClose} title={title}>
      <div className="flex flex-col gap-3" role="radiogroup">
        <ChoiceCard title={H.day24} selected={all24} onClick={() => setAll24(true)} />
        <ChoiceCard title={H.daySpecific} selected={!all24} onClick={() => setAll24(false)} />
        {!all24 && (
          <div className="flex flex-wrap items-center gap-2 pt-1">
            <span className="text-sm text-ink-muted">{H.from}</span>
            <Input
              id="day-from"
              type="time"
              dir="ltr"
              step={60}
              aria-label={H.from}
              value={from}
              onChange={(e) => e.target.value && setFrom(e.target.value)}
              wrapperClassName="w-32"
            />
            <span className="text-sm text-ink-muted">{H.to}</span>
            <Input
              id="day-to"
              type="time"
              dir="ltr"
              step={60}
              aria-label={H.to}
              value={to}
              onChange={(e) => e.target.value && setTo(e.target.value)}
              wrapperClassName="w-32"
            />
          </div>
        )}
        <FormActions
          onSave={() =>
            onDone(
              all24
                ? with24h(start, true)
                : { ...start, closed: false, open_24h: false, open_time: from, close_time: to },
            )
          }
          onCancel={onClose}
          saveLabel={H.done}
        />
      </div>
    </Modal>
  );
}

/**
 * **مخطّطُ الدوام** — خياران فوق، وفي «ساعات محددة» سطرٌ لكلّ يوم، **ثمّ سؤالُ
 * الجمعة صراحةً في الحالين** ودوامُها في سطرها.
 */
export function HoursPlanner({ plan, disabled }: { plan: HoursPlan; disabled?: boolean }) {
  const [editing, setEditing] = useState<DayHours | null>(null);
  const [applying, setApplying] = useState(false);
  const week = plan.draft
    .filter((d) => d.day_of_week !== FRIDAY)
    .sort((a, b) => ((a.day_of_week + 1) % 7) - ((b.day_of_week + 1) % 7));
  const fri = plan.draft.find((d) => d.day_of_week === FRIDAY);
  const sample = plan.draft.find((d) => d.day_of_week !== FRIDAY && !d.closed) ?? plan.draft[0];

  return (
    <div className="flex flex-col gap-3" data-testid="hours-planner">
      <div className="flex flex-col gap-2" role="radiogroup">
        <ChoiceCard title={H.mode24} selected={plan.mode === "24"} disabled={disabled} onClick={() => plan.chooseMode("24")} />
        <ChoiceCard
          title={H.modeCustom}
          selected={plan.mode === "custom"}
          disabled={disabled}
          onClick={() => plan.chooseMode("custom")}
        />
      </div>

      {plan.mode === "custom" && (
        <>
          <h3 className="heading-card pt-2">{H.daysTitle}</h3>
          <div className="surface flex flex-col divide-y divide-line overflow-hidden rounded-card p-0">
            {week.map((d) => (
              <DayRow
                key={d.day_of_week}
                d={d}
                disabled={disabled}
                onToggle={(open) => plan.update({ ...d, closed: !open })}
                onTime={() => setEditing(d)}
              />
            ))}
          </div>
          <Button
            variant="secondary"
            className="w-full font-bold"
            disabled={disabled || !plan.draft.some((d) => !d.closed)}
            onClick={() => setApplying(true)}
          >
            {H.applySame}
          </Button>
        </>
      )}

      {plan.mode !== "" && (
        <>
          <h3 className="heading-card pt-2">{H.fridayQ}</h3>
          <IconChoicePair
            disabled={disabled}
            items={[
              { icon: IconCheck, title: H.on, selected: plan.friday === "on", onClick: () => plan.chooseFriday(true) },
              { icon: IconClose, title: H.off, selected: plan.friday === "off", onClick: () => plan.chooseFriday(false) },
            ]}
          />
          {plan.friday === "on" && fri && (
            <div className="surface overflow-hidden rounded-card p-0">
              <DayRow d={{ ...fri, closed: false }} disabled={disabled} onTime={() => setEditing({ ...fri, closed: false })} />
            </div>
          )}
        </>
      )}

      {editing && (
        <DayTimeDialog
          title={H.days[editing.day_of_week] ?? ""}
          start={editing}
          onClose={() => setEditing(null)}
          onDone={(d) => {
            plan.update(d);
            setEditing(null);
          }}
        />
      )}
      {applying && sample && (
        <DayTimeDialog
          title={H.applySameTitle}
          start={{ ...sample, closed: false }}
          onClose={() => setApplying(false)}
          onDone={(d) => {
            plan.applyAll(d);
            setApplying(false);
          }}
        />
      )}
    </div>
  );
}
