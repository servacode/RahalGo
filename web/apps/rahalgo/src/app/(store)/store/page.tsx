"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store — «متجري»: لوحةُ قيادة المتجر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StoreHome.kt` — ملاحظةُ المالك ٦، ٢٠٢٦-١٠-٢٨؛ وإعادةُ التصميم ٢٠٢٦-١١-٠٨ بالتصميم المعتمد:
 *  لابتوب بمقام Stripe/Shopify وآيفون بمقام Apple HIG.)
 *
 * **حاسوباً**: رأسٌ بهويّة المتجر ومفتاحِ «مفتوح/مغلق» وزرّ «أضف صنفاً» · أربعةُ مؤشّراتٍ برسمٍ مصغّر ·
 * رسمُ المبيعات وطلباتُ الآن · «يحتاج انتباهك» والإجراءاتُ السريعة · الإعداداتُ في مجموعتين.
 * **جوّالاً**: عنوانٌ كبيرٌ «متجري» · بطاقةُ المتجر ومفتاحُ الفتح · «اليوم» (مبيعاتٌ وأعمدةُ أسبوع) ·
 * «الطلبات الحيّة» · «يحتاج انتباهك» · أربعةُ إجراءاتٍ · الإعدادات — مجموعاتٌ مُدرجةٌ كإعدادات آيفون.
 *
 * **ومصدرُ كلّ رقمٍ نقطةٌ يعرفها المحرّك أصلاً**: تقريرُ اليوم، وتقريرُ كلِّ يومٍ للرسم والنسبة (`reports?from&to`)،
 * وملخّصُ الأصناف، وسياقُ الطلبات الحيّ. **وما لا مصدرَ له لا يُرسم.**
 *
 * **ولا مفتاحَ «مفتوح» يدويّ**: الحالُ تتبع ساعاتِ الدوام، **والإغلاقُ الطارئُ وحدَه بيده.**
 */

import { AmeenStatusTile } from "@/components/store/AmeenStatus";
import Link from "next/link";
import { useEffect, useState, type ComponentType } from "react";
import {
  getMessages,
  defaultLocale,
  damascusDay,
  errorText,
  fmtCalendarDay,
  fmtMoney,
  fmtNum,
  fmtSince,
} from "@rahalgo/i18n";
import {
  Alert,
  Button,
  ButtonLink,
  Confirm,
  Delta,
  FormActions,
  IGroup,
  IRow,
  IconAdd,
  IconBanknote,
  IconCamera,
  IconCash,
  IconCatOther,
  IconCheck,
  IconClose,
  IconDate,
  IconEdit,
  IconHourglass,
  IconImageOff,
  IconLocation,
  IconMoto,
  IconNext,
  IconOrder,
  IconPhone,
  IconPromos,
  IconReturn,
  IconRotate,
  IconTrendUp,
  IconUser,
  IconUsers,
  IconViewOff,
  IconWallet,
  IconButton,
  Input,
  Kpi,
  KpiGrid,
  LoadingState,
  MiniBars,
  Modal,
  PageHead,
  Pill,
  SalesChart,
  Segmented,
  SwitchTrack,
  useIsDesktop,
  useLiveData,
  type SalesPoint,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { useStoreOrders } from "@/lib/store/orders";
import { pointSet, useStoreSections } from "@/lib/store/setup";
import { pctChange, useSalesSeries, type SalesRange, type SalesRow } from "@/lib/store/sales";
import { groupOf } from "@/lib/store/types";
import type { DayHours, MenuSummary, MerchantOrder, MerchantStore, StoreReport } from "@/lib/store/types";
import { backAt, fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";
import { LogoCard, useSaveSettings } from "@/components/store/StoreSettings";
import { FRIDAY, is24h, rangeText, useStoreHours } from "@/components/store/SetupParts";
import { usdText } from "@/lib/store/usd";

const m = getMessages(defaultLocale);
const S = m.store;
const T = S.settings;
const HM = S.home;
const SH = S.shell;

type IconT = ComponentType<{ size?: number; className?: string }>;

/** دقائقُ الانتظار التي يصير بعدها الطلبُ الجديدُ «متأخّراً» (كصفحة الطلبات). */
const LATE_MIN = 5;

// ══════════════════════════════════════════════════════════════════════
// الهويّةُ والحال
// ══════════════════════════════════════════════════════════════════════

/** **حالُ المتجر نصّاً** — للمفتاح والسطر الوصفيّ (دالةٌ لا خُطّاف). */
function storeState(store: MerchantStore, suspended: boolean) {
  const next = backAt(store.next_open);
  const label = suspended
    ? HM.suspended
    : store.emergency_closed
      ? HM.tempClosed
      : !store.open_now
        ? next
          ? fill(HM.closedUntil, { when: next })
          : HM.closed
        : HM.openNow;
  const sub = suspended
    ? HM.suspendedSub
    : store.emergency_closed
      ? HM.tempClosedSub
      : !store.open_now
        ? HM.closedHours
        : HM.byHours;
  const isOpen = !suspended && !store.emergency_closed && store.open_now;
  return { label, sub, isOpen };
}

/** **الإغلاقُ الطارئ وإعادةُ الفتح** — النداءُ نفسُه الذي كان، وبنافذة تأكيدٍ للإغلاق. */
function useEmergency(store: MerchantStore) {
  const { patch, reload } = useStore();
  const toast = useShopToast();
  const [asking, setAsking] = useState(false);
  const [busy, setBusy] = useState(false);
  async function setClosed(closed: boolean) {
    setBusy(true);
    try {
      const r = await api<{ emergency_closed: boolean }>(`/api/v1/merchant/stores/${store.id}/emergency`, {
        method: "POST",
        body: JSON.stringify({ closed }),
      });
      patch({ emergency_closed: r.emergency_closed });
      toast(closed ? T.closedOk : HM.reopened, "success");
      void reload();
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
      setAsking(false);
    }
  }
  const dialog = (
    <Confirm
      open={asking}
      title={T.emergencyQ}
      body={T.emergencyHint}
      confirmLabel={T.emergencyDo}
      busy={busy}
      onConfirm={() => void setClosed(true)}
      onCancel={() => setAsking(false)}
    />
  );
  return { asking, busy, ask: () => setAsking(true), setClosed, dialog };
}

/** **اسمُ المتجر وتعديلُه** — نموذجٌ سطريٌّ كما كان (الإغلاقُ بلا حفظ). */
function RenameForm({ store, onDone }: { store: MerchantStore; onDone: () => void }) {
  const { busy, save } = useSaveSettings();
  const [draft, setDraft] = useState(store.name);
  return (
    <form
      className="flex w-full max-w-md flex-col gap-2 rounded-control bg-field p-3"
      onSubmit={(e) => {
        e.preventDefault();
        const name = draft.trim();
        if (!name) return;
        void save({ name }, T.nameSaved, { name }).then((ok) => ok && onDone());
      }}
    >
      <Input
        id="store-name"
        label={T.storeName}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        error={draft.trim() ? undefined : T.vStoreName}
        autoFocus
      />
      <FormActions submit onCancel={onDone} busy={busy} />
    </form>
  );
}

/** **شعارُ المتجر** — زرٌّ يفتح نافذةَ تغيير الشعار؛ والحرفُ الأوّلُ بتدرّج العلامة إن لم يوجد. */
function StoreLogo({ store, suspended, onLogo, size }: { store: MerchantStore; suspended: boolean; onLogo: () => void; size?: "lg" }) {
  const logo = mediaUrl(store.logo_thumb_url);
  return (
    <button type="button" onClick={onLogo} disabled={suspended} aria-label={HM.changeLogo} className="hero-logo shrink-0" data-size={size}>
      {logo ? <img src={logo} alt="" className="h-full w-full object-cover" /> : store.name.slice(0, 1) || SH.noName}
    </button>
  );
}

// ══════════════════════════════════════════════════════════════════════
// طلباتُ الآن
// ══════════════════════════════════════════════════════════════════════

function useLive() {
  const { orders } = useStoreOrders();
  const now = Date.now();
  const list = orders ?? [];
  const by = (g: string) => list.filter((o) => groupOf(o) === g);
  const fresh = by("new");
  const prep = by("preparing");
  const wait = by("waiting");
  const sentAt = (o: MerchantOrder) => new Date(o.sent_to_merchant_at ?? o.created_at).getTime();
  const lateNew = fresh.filter((o) => now - sentAt(o) > LATE_MIN * 60_000);
  return { fresh, prep, wait, lateNew, now, sentAt, all: [...fresh, ...prep, ...wait] };
}

function LiveOrders() {
  const { fresh, prep, wait, now, sentAt, all } = useLive();
  const cells = [
    { key: "new", label: SH.newOrders, n: fresh.length, tone: "info" as const },
    { key: "prep", label: SH.preparing, n: prep.length, tone: "warn" as const },
    { key: "wait", label: SH.ready, n: wait.length, tone: undefined },
  ];
  return (
    <section className="panel flex flex-col" aria-label={HM.liveTitle} data-testid="store-live">
      <div className="panel-head">
        <h2 className="console-h2">{HM.liveTitle}</h2>
        <Pill tone="good">{SH.live}</Pill>
        <Link href="/store/orders" className="ms-auto flex items-center gap-1 text-sm font-bold text-primary hover:underline">
          {SH.viewAll}
          <IconNext size={14} />
        </Link>
      </div>
      <div className="stat-row mx-4 mt-2 rounded-control border border-line-soft">
        {cells.map((c, i) => (
          <Link
            key={c.key}
            href="/store/orders"
            className={`px-3 py-2 transition-colors hover:bg-row-hover ${i > 0 ? "border-s border-line-soft" : ""}`}
          >
            <span className="flex items-center gap-1.5 text-sm text-ink-muted">
              <span className="dot" data-tone={c.tone} aria-hidden />
              {c.label}
            </span>
            <span className="stat-num block">{fmtNum(c.n)}</span>
          </Link>
        ))}
      </div>
      {all.length > 0 ? (
        <ul className="my-2 flex flex-col">
          {all.slice(0, 4).map((o) => {
            const late = groupOf(o) === "new" && now - sentAt(o) > LATE_MIN * 60_000;
            return (
              <li key={o.id} className="border-t border-line-soft first:border-0">
                <Link href="/store/orders" className="flex items-center gap-2.5 px-4 py-2.5 text-sm transition-colors hover:bg-row-hover">
                  <b className="w-12 shrink-0" dir="ltr">
                    #{o.number}
                  </b>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-bold">{o.items.map((l) => l.name).join(m.common.listSep)}</span>
                    <span className="console-meta">
                      {fill(SH.itemsCount, { n: fmtNum(o.items.length) })} · {fmtMoney(o.subtotal)}
                    </span>
                  </span>
                  <span className={`shrink-0 text-xs ${late ? "font-bold text-danger" : "text-ink-dim"}`}>
                    {late ? `${SH.late} · ` : ""}
                    {fmtSince(o.sent_to_merchant_at ?? o.created_at, now)}
                  </span>
                  <Pill tone={groupOf(o) === "new" ? "info" : groupOf(o) === "preparing" ? "attention" : "good"} dot={false}>
                    {S.orders.status[o.status as keyof typeof S.orders.status] ?? S.orders.readyDone}
                  </Pill>
                </Link>
              </li>
            );
          })}
        </ul>
      ) : (
        <p className="px-4 py-6 text-center text-sm text-ink-dim">{HM.liveNone}</p>
      )}
    </section>
  );
}

// ══════════════════════════════════════════════════════════════════════
// يحتاج انتباهك
// ══════════════════════════════════════════════════════════════════════

interface Need {
  key: string;
  text: string;
  cta?: string;
  href: string;
  /** `danger`: نقصٌ في الإعداد يمنع عملاً · `warning`: أصنافٌ تحتاج مراجعة. */
  tone: "danger" | "warning";
  icon: IconT;
  /** رقمٌ بارزٌ في الصفّ (عددُ الأصناف). */
  num?: number;
}

function NeedsPanel({ needs }: { needs: Need[] }) {
  return (
    <section className="panel flex min-w-0 flex-col" aria-label={HM.alertsTitle} data-testid="store-needs">
      <div className="panel-head">
        <h2 className="console-h2">{HM.alertsTitle}</h2>
        {needs.length > 0 && <Pill tone="danger" dot={false}>{fmtNum(needs.length)}</Pill>}
      </div>
      {needs.length === 0 ? (
        <p className="flex items-center justify-center gap-2 px-4 py-7 text-center text-sm text-ink-dim">
          <IconCheck size={18} className="text-success" />
          {HM.alertsClear}
        </p>
      ) : (
        <ul className="my-1.5 flex flex-col">
          {needs.map((n) => {
            const Ico = n.icon;
            return (
              <li key={n.key} className="border-t border-line-soft first:border-0">
                <Link href={n.href} className="flex items-center gap-3 px-4 py-2.5 transition-colors hover:bg-row-hover">
                  <span className="icon-chip" data-tone={n.tone === "danger" ? "danger" : "attention"} aria-hidden>
                    <Ico size={17} />
                  </span>
                  <span className="min-w-0 flex-1 text-sm font-bold">{n.text}</span>
                  {n.num != null && <span className="text-base font-bold tabular-nums">{fmtNum(n.num)}</span>}
                  {n.cta ? (
                    <span className="surface shrink-0 px-2.5 py-1 text-xs font-bold">{n.cta}</span>
                  ) : (
                    <IconNext size={15} className="shrink-0 text-ink-dim" />
                  )}
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

function QuickActions({ unavailable, noPhoto }: { unavailable: number; noPhoto: number }) {
  const items: { href: string; icon: IconT; label: string; badge?: number }[] = [
    { href: "/store/menu/new", icon: IconAdd, label: HM.quickAdd },
    { href: "/store/menu/photos", icon: IconCamera, label: SH.quickPhotos, badge: noPhoto },
    { href: "/store/offers?new=1", icon: IconPromos, label: HM.quickOffers },
    { href: "/store/delivery", icon: IconMoto, label: HM.quickDelivery },
    { href: "/store/menu?filter=unavailable", icon: IconViewOff, label: HM.quickUnavailable, badge: unavailable },
    { href: "/store/menu/prices", icon: IconBanknote, label: HM.quickPrices },
  ];
  return (
    <section className="panel flex flex-col" aria-label={HM.quickTitle}>
      <div className="panel-head">
        <h2 className="console-h2">{HM.quickTitle}</h2>
      </div>
      <div className="grid grid-cols-2 gap-2 px-4 pb-4 pt-2">
        {items.map((q) => {
          const Ico = q.icon;
          return (
            <Link
              key={q.href}
              href={q.href}
              className="group flex h-11 items-center gap-2.5 rounded-control border border-line-soft bg-field px-3 text-sm font-bold transition-colors hover:border-primary-edge hover:bg-primary-soft hover:text-primary"
            >
              <Ico size={17} className="text-ink-dim group-hover:text-primary" />
              <span className="min-w-0 flex-1 truncate">{q.label}</span>
              {!!q.badge && <span className="text-xs font-bold tabular-nums text-ink-dim">{fmtNum(q.badge)}</span>}
            </Link>
          );
        })}
      </div>
    </section>
  );
}

/** **قيمةُ الدوام في سطرٍ واحد** — أو فراغٌ إن لم يُضبط بعد. */
function hoursValue(store: MerchantStore, hours: DayHours[] | null): string | null {
  if (!store.has_hours || !hours) return null;
  const open = hours.filter((d) => !d.closed);
  if (open.length === 0) return null;
  if (open.every(is24h)) return S.hours.open24;
  const first = open.find((d) => d.day_of_week !== FRIDAY) ?? open[0];
  if (!first) return null;
  return open.every((d) => rangeText(d) === rangeText(first)) ? rangeText(first) : HM.hoursMixed;
}

// ══════════════════════════════════════════════════════════════════════
// رسمُ المبيعات (حاسوباً)
// ══════════════════════════════════════════════════════════════════════

/** فهرسُ يومِ الأسبوع (٠ الأحد) من تاريخ `YYYY-MM-DD`. */
function dowOf(day: string): number {
  const [y = 0, mo = 1, d = 1] = day.split("-").map(Number);
  return new Date(Date.UTC(y, mo - 1, d)).getUTCDay();
}
/** اسمُ اليوم كاملاً (الاثنين…) — لمحور الرسم والتلميح. */
function weekdayOf(day: string): string {
  return m.common.dates.days[dowOf(day)] ?? "";
}
/** اختصارُ اليوم (اث · ث · أر …) — لأعمدة الهاتف. */
function weekdayShort(day: string): string {
  return SH.daysShort[dowOf(day)] ?? "";
}

function toPoints(rows: SalesRow[], range: SalesRange): SalesPoint[] {
  return rows.map((r, i) => ({
    label: range === 7 ? (i === rows.length - 1 ? HM.today : weekdayOf(r.from)) : fmtNum(Number(r.to.slice(8, 10))),
    value: r.sales,
    title: range === 7 ? fmtCalendarDay(r.from) : `${fmtCalendarDay(r.from)} – ${fmtCalendarDay(r.to)}`,
    detail: `${fmtMoney(r.sales)} · ${fill(SH.ordersCount, { n: fmtNum(r.orders) })}`,
  }));
}

function SalesCard({ storeId, wk }: { storeId: string; wk: { cur: SalesRow[] | null; prev: SalesRow[] | null } }) {
  const [range, setRange] = useState<SalesRange>(7);
  const mo = useSalesSeries(storeId, 30, range === 30);
  const s = range === 7 ? wk : mo;
  const total = (s.cur ?? []).reduce((a, r) => a + r.sales, 0);
  const days = range;
  return (
    <section className="panel flex min-w-0 flex-col" aria-label={SH.chartTitle} data-testid="store-week">
      <div className="panel-head">
        <h2 className="console-h2">{SH.chartTitle}</h2>
        <span className="console-meta">{fill(SH.lastDays, { n: fmtNum(days) })}</span>
        <span className="ms-auto">
          <Segmented
            ariaLabel={SH.chartTitle}
            value={String(range) as "7" | "30"}
            onChange={(k) => setRange(k === "7" ? 7 : 30)}
            items={[
              { key: "7", label: SH.range7 },
              { key: "30", label: SH.range30 },
            ]}
          />
        </span>
      </div>
      <div className="flex flex-wrap items-end gap-7 px-4 pt-2">
        <div>
          <p className="console-meta">{range === 7 ? SH.weekTotal : SH.monthTotal}</p>
          <p className="figure">
            {fmtNum(total)}
            <small className="ms-1 text-sm font-medium text-ink-dim">{m.common.currency}</small>
          </p>
        </div>
        <div>
          <p className="console-meta">{SH.dayAvg}</p>
          <p className="stat-num">
            {fmtNum(Math.round(total / days))}
            <small className="ms-1 text-sm font-medium text-ink-dim">{m.common.currency}</small>
          </p>
        </div>
        <div className="ms-auto flex gap-3.5 pb-1 text-xs text-ink-dim">
          <span className="flex items-center gap-1.5">
            <i className="inline-block h-0.5 w-2.5 rounded bg-primary" />
            {SH.thisPeriod}
          </span>
          {s.prev && (
            <span className="flex items-center gap-1.5">
              <i className="inline-block h-0.5 w-2.5 rounded bg-line" />
              {SH.prevPeriod}
            </span>
          )}
        </div>
      </div>
      <div className="px-4 pb-4 pt-1.5">
        {s.cur ? (
          <>
            <SalesChart
              ariaLabel={HM.weekChart}
              points={toPoints(s.cur, range)}
              prev={s.prev?.map((r) => r.sales)}
              tickLabel={(v) => (v >= 1000 ? `${fmtNum(v / 1000)}${SH.kUnit}` : fmtNum(v))}
            />
            {total === 0 && <p className="console-meta mt-2 text-center">{HM.weekNone}</p>}
          </>
        ) : (
          <LoadingState variant="inline" />
        )}
      </div>
    </section>
  );
}

// ══════════════════════════════════════════════════════════════════════
// الصفحة
// ══════════════════════════════════════════════════════════════════════

export default function StoreHome() {
  const { store } = useStore();
  return store ? <HomeBody store={store} /> : null;
}

function HomeBody({ store }: { store: MerchantStore }) {
  const { selfManage } = useStore();
  const id = store.id;
  const desk = useIsDesktop();
  const loadReport = () => {
    const today = damascusDay();
    return api<StoreReport>(`/api/v1/merchant/stores/${id}/reports?from=${today}&to=${today}`);
  };
  const { data: report, loading } = useLiveData(loadReport, ["order"], [id]);
  const sec = useStoreSections(id);
  const { saved: hours } = useStoreHours(id);
  const [menu, setMenu] = useState<MenuSummary | null>(null);
  /** **أصنافُ الأمين بلا قسم** — من `price-sync/groups`، وللمتجر الذي يستعمل الأمين وحدَه. */
  const [unsorted, setUnsorted] = useState(0);
  const [logoOpen, setLogoOpen] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const usesAmeen = store.uses_ameen === true;
  const wk = useSalesSeries(id, 7);
  const live = useLive();

  useEffect(() => {
    if (!id) return;
    // **The counts from the light summary** (2026-11-07) — not the whole menu of a big mall.
    api<MenuSummary>(`/api/v1/merchant/stores/${id}/menu/summary`)
      .then(setMenu)
      .catch(() => undefined);
  }, [id]);

  useEffect(() => {
    if (!id || !usesAmeen) {
      setUnsorted(0);
      return;
    }
    api<{ pending: number }>(`/api/v1/merchant/stores/${id}/price-sync/groups`)
      .then((r) => setUnsorted(r.pending ?? 0))
      .catch(() => undefined);
  }, [id, usesAmeen]);

  const emergency = useEmergency(store);

  const suspended = store.status === "suspended";
  const point = pointSet(store);
  const notSet = HM.notSet;
  const unavailable = menu?.unavailable ?? 0;
  const noPhoto = menu?.no_photo ?? 0;
  const state = storeState(store, suspended);

  /* **ما يحتاج انتباهاً** — النقصُ في الإعداد أوّلاً (يمنع عملاً)، ثمّ الأصناف. */
  const needs: Need[] = [];
  if (sec.known && sec.mine.length === 0)
    needs.push({ key: "sections", text: `${HM.rowSections}: ${notSet}`, href: "/store/settings/sections", tone: "danger", icon: IconCatOther });
  if (!point) needs.push({ key: "address", text: `${HM.rowAddress}: ${notSet}`, href: "/store/settings/address", tone: "danger", icon: IconLocation });
  if (!store.has_hours) needs.push({ key: "hours", text: `${HM.rowHours}: ${notSet}`, href: "/store/settings/hours", tone: "danger", icon: IconDate });
  if (!store.settlement_confirmed)
    needs.push({ key: "settle", text: `${HM.rowSettlement}: ${notSet}`, href: "/store/settings/settlement", tone: "danger", icon: IconWallet });
  if (!store.returns_confirmed)
    needs.push({ key: "returns", text: `${HM.rowReturns}: ${notSet}`, href: "/store/settings/returns", tone: "danger", icon: IconReturn });
  if (!suspended && noPhoto > 0)
    needs.push({ key: "photo", text: fill(HM.alertNoPhoto, { n: fmtNum(noPhoto) }), cta: HM.alertNoPhotoCta, href: "/store/menu/photos", tone: "warning", icon: IconImageOff });
  if (!suspended && unsorted > 0)
    needs.push({ key: "unsorted", text: fill(HM.alertUnsorted, { n: fmtNum(unsorted) }), cta: HM.alertUnsortedCta, href: "/store/price-sync", tone: "warning", icon: IconRotate });
  if (unavailable > 0)
    needs.push({ key: "off", text: fill(HM.alertOff, { n: fmtNum(unavailable) }), cta: HM.alertOffCta, href: "/store/menu?filter=unavailable", tone: "warning", icon: IconViewOff });

  // ── أرقامُ اليوم ونسبُها ──
  const rows = wk.cur;
  const prev = wk.prev;
  const yesterday = rows && rows.length >= 2 ? rows[rows.length - 2] : null;
  const lastWeekSameDay = prev ? prev[prev.length - 1] : null;
  const S0 = report?.summary;
  const ordersPct = S0 && yesterday ? pctChange(S0.orders, yesterday.orders) : null;
  const salesPct = S0 && lastWeekSameDay ? pctChange(S0.sales, lastWeekSameDay.sales) : null;
  const sparkOf = (f: (r: SalesRow) => number) => (rows ? rows.map(f) : undefined);
  const due = S0 ? S0.sales - S0.platform_commission : 0;
  const homeSub = fill(SH.homeSub, { n: fmtNum(menu?.total ?? 0), date: fmtCalendarDay(damascusDay()) });

  const needsNode = needs.length > 0 || desk ? <NeedsPanel needs={needs} /> : null;

  // ── مجموعتا الإعدادات (كما كانتا) ──
  const settings = (
    <>
      <IGroup title={HM.groupStore} testId="store-group-store">
        <IRow
          icon={IconCatOther}
          tone="info"
          title={HM.rowSections}
          value={sec.mine.map((s) => s.name).join(" · ") || notSet}
          valueTone={sec.known && sec.mine.length === 0 ? "danger" : undefined}
          href="/store/settings/sections"
          Link={Link}
        />
        <IRow icon={IconLocation} tone="danger" title={HM.rowAddress} value={point ? store.address_text : notSet} valueTone={!point ? "danger" : undefined} href="/store/settings/address" Link={Link} />
        <IRow
          icon={IconCamera}
          tone="violet"
          title={HM.rowLogo}
          value={store.logo_thumb_url ? HM.logoSet : HM.logoNone}
          disabled={suspended}
          onClick={() => setLogoOpen(true)}
        />
        <IRow icon={IconUsers} tone="good" title={HM.rowStaff} href="/store/team" Link={Link} />
        <IRow icon={IconPhone} tone="attention" title={HM.rowOrderPhones} href="/store/order-phones" Link={Link} />
      </IGroup>
      <IGroup title={HM.groupHoursMoney} testId="store-group-money">
        <IRow
          icon={IconDate}
          tone="attention"
          title={HM.rowHours}
          value={hoursValue(store, hours) ?? notSet}
          valueTone={!store.has_hours ? "danger" : undefined}
          href="/store/settings/hours"
          Link={Link}
        />
        <IRow
          icon={store.settlement_method === "wallet" ? IconWallet : IconCash}
          tone="good"
          title={HM.rowSettlement}
          value={!store.settlement_confirmed ? notSet : store.settlement_method === "wallet" ? HM.wallet : HM.cash}
          valueTone={!store.settlement_confirmed ? "danger" : undefined}
          href="/store/settings/settlement"
          Link={Link}
        />
        <IRow
          icon={IconReturn}
          tone="info"
          title={HM.rowReturns}
          value={!store.returns_confirmed ? notSet : store.accepts_returns ? HM.returnsYes : HM.returnsNo}
          valueTone={!store.returns_confirmed ? "danger" : undefined}
          href="/store/settings/returns"
          Link={Link}
        />
        <IRow icon={IconCheck} tone="violet" title={HM.rowAmeen} value={store.uses_ameen ? HM.ameenYesShort : HM.ameenNoShort} href="/store/settings/ameen" Link={Link} />
        <IRow icon={IconTrendUp} tone="gray" title={HM.rowRate} value={store.exchange_rate ? usdText(store.exchange_rate) : notSet} href="/store/settings/rate" Link={Link} />
        <IRow icon={IconHourglass} tone="attention" title={HM.rowPrep} value={`${fmtNum(store.default_prep_minutes)} ${T.prepUnit}`} href="/store/settings/prep" Link={Link} />
      </IGroup>
    </>
  );

  const logoModal = (
    <Modal open={logoOpen} onClose={() => setLogoOpen(false)} title={T.logo}>
      <div className="flex flex-col gap-3">
        <LogoCard />
        <Button variant="secondary" onClick={() => setLogoOpen(false)}>
          {m.common.close}
        </Button>
      </div>
    </Modal>
  );

  // ══════════════════════════ الحاسوب ══════════════════════════
  if (desk) {
    return (
      <div className="console-page" data-testid="store-home">
        <header className="flex flex-wrap items-center gap-3.5">
          <StoreLogo store={store} suspended={suspended} onLogo={() => setLogoOpen(true)} />
          <div className="min-w-0">
            {renaming ? (
              <RenameForm store={store} onDone={() => setRenaming(false)} />
            ) : (
              <>
                <h1 className="console-title flex items-center gap-1.5">
                  <span className="truncate">{store.name}</span>
                  {!suspended && (
                    <IconButton icon={IconEdit} label={SH.editName} onClick={() => setRenaming(true)} size={15} />
                  )}
                </h1>
                <p className="console-meta line-clamp-1">{[point ? store.address_text : "", homeSub].filter(Boolean).join(" · ")}</p>
              </>
            )}
          </div>
          <div className="ms-auto flex flex-wrap items-center gap-2.5">
            {suspended ? (
              <Pill tone="danger">{HM.suspended}</Pill>
            ) : (
              <>
                <Segmented
                  ariaLabel={HM.statusLabel}
                  value={state.isOpen ? "open" : "closed"}
                  onChange={(k) => {
                    if (k === "closed" && state.isOpen) emergency.ask();
                    else if (k === "open" && store.emergency_closed) void emergency.setClosed(false);
                  }}
                  items={[
                    { key: "open", label: <><span className="dot" aria-hidden />{SH.segOpen}</> },
                    { key: "closed", label: SH.segClosed },
                  ]}
                  testId="store-status"
                />
                <span className="console-meta">{state.sub}</span>
              </>
            )}
            {!suspended && (
              <ButtonLink Link={Link} href="/store/menu/new" data-testid="store-home-add">
                <IconAdd size={15} />
                {HM.quickAdd}
              </ButtonLink>
            )}
          </div>
        </header>
        {suspended && <Alert tone="error">{S.suspendedBanner}</Alert>}

        {/* **حالةُ برنامج الأمين قرب الأعلى** (ملاحظةُ المالك ٢٠٢٦-١١-٠٧) — لمتاجر الأمين وحدَها. */}
        <AmeenStatusTile />

        {loading && !report ? (
          <LoadingState variant="stats" rows={4} />
        ) : (
          report && (
            <KpiGrid>
              <Kpi
                icon={IconOrder}
                label={HM.todayOrders}
                value={report.summary.orders}
                spark={sparkOf((r) => r.orders)}
                sub={
                  ordersPct != null && yesterday ? (
                    <Delta pct={ordersPct} label={fill(SH.vsYesterday, { n: fmtNum(yesterday.orders) })} />
                  ) : (
                    `${HM.deliveredToday}: ${fmtNum(report.summary.delivered)}`
                  )
                }
                href="/store/history"
                Link={Link}
                testId="kpi-orders"
              />
              <Kpi
                icon={IconTrendUp}
                label={HM.salesToday}
                value={fmtNum(report.summary.sales)}
                unit={m.common.currency}
                spark={sparkOf((r) => r.sales)}
                sub={salesPct != null ? <Delta pct={salesPct} label={SH.vsLastWeek} /> : undefined}
                testId="kpi-sales"
              />
              <Kpi
                icon={IconWallet}
                label={HM.dueToday}
                value={fmtNum(due)}
                unit={m.common.currency}
                tone="accent"
                spark={sparkOf((r) => r.sales - r.commission)}
                testId="kpi-due"
              />
              <Kpi
                icon={IconClose}
                label={HM.cancelledToday}
                value={report.summary.cancelled}
                tone={report.summary.cancelled > 0 ? "danger" : undefined}
                spark={sparkOf((r) => r.cancelled)}
                sub={
                  report.summary.orders > 0
                    ? fill(SH.ofOrders, { p: fmtNum(Math.round((report.summary.cancelled / report.summary.orders) * 100)) })
                    : undefined
                }
                testId="kpi-cancelled"
              />
            </KpiGrid>
          )
        )}

        {selfManage ? (
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1.85fr)_minmax(0,1fr)]">
            <SalesCard storeId={id} wk={wk} />
            <LiveOrders />
          </div>
        ) : (
          <SalesCard storeId={id} wk={wk} />
        )}

        <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
          {needsNode}
          {!suspended && <QuickActions unavailable={unavailable} noPhoto={noPhoto} />}
        </div>

        <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-2">{settings}</div>
        {emergency.dialog}
        {logoModal}
      </div>
    );
  }

  // ══════════════════════════ الهاتف ══════════════════════════
  const bars = (rows ?? []).map((r, i, a) => ({
    label: i === a.length - 1 ? HM.today : weekdayShort(r.from),
    value: r.sales,
    title: `${fmtCalendarDay(r.from)} — ${fmtMoney(r.sales)}`,
  }));
  return (
    <div className="console-page tab-grid" data-testid="store-home">
      <PageHead
        title={S.nav.store}
        navEnd={<IconButton icon={IconUser} label={S.nav.account} href="/store/account" Link={Link} size={24} testId="phone-account" />}
      />
      {suspended && (
        <Alert tone="error">{S.suspendedBanner}</Alert>
      )}
      <div className="igroup tab-span">
        <div className="flex items-center gap-3.5 px-4 py-3.5">
          <StoreLogo store={store} suspended={suspended} onLogo={() => setLogoOpen(true)} size="lg" />
          <div className="min-w-0 flex-1">
            {renaming ? (
              <RenameForm store={store} onDone={() => setRenaming(false)} />
            ) : (
              <>
                <h2 className="hero-name flex items-center gap-1">
                  <span className="line-clamp-2">{store.name}</span>
                  {!suspended && <IconButton icon={IconEdit} label={SH.editName} onClick={() => setRenaming(true)} size={16} />}
                </h2>
                <p className="console-meta line-clamp-1">{point ? store.address_text : S.nav.panel}</p>
              </>
            )}
          </div>
        </div>
        {!suspended && (
          <button
            type="button"
            role="switch"
            aria-checked={state.isOpen}
            disabled={emergency.busy || (!state.isOpen && !store.emergency_closed)}
            onClick={() => (state.isOpen ? emergency.ask() : void emergency.setClosed(false))}
            data-testid="store-status"
            className="irow disabled:opacity-60"
          >
            <span className="min-w-0 flex-1 py-2">
              <span className="block text-base font-medium">{state.label}</span>
              <span className="console-meta block">{state.sub}</span>
            </span>
            <SwitchTrack on={state.isOpen} />
          </button>
        )}
      </div>

      <div className="tab-span">
        <AmeenStatusTile />
      </div>

      <div className="tab-span">
        <div className="sec-head">{SH.todayHead}</div>
        <div className="igroup">
          <div className="px-4 pb-1 pt-3.5">
            <p className="flex items-center gap-1.5 text-sm text-ink-dim">
              <IconTrendUp size={16} />
              {HM.salesToday}
            </p>
            <p className="hero-figure">
              {fmtNum(S0?.sales ?? 0)}
              <small className="ms-1.5 text-base font-medium text-ink-dim">{m.common.currency}</small>
            </p>
            {salesPct != null && (
              <p className="flex items-center text-sm">
                <Delta pct={salesPct} label={SH.vsLastWeek} />
              </p>
            )}
            {bars.length > 0 && (
              <div className="mt-2 pb-2.5">
                <MiniBars ariaLabel={HM.weekChart} data={bars} height={(S0?.sales ?? 0) > 0 || (rows ?? []).some((r) => r.sales > 0) ? 54 : 28} />
              </div>
            )}
          </div>
          <div className="stat-row relative before:absolute before:inset-x-4 before:top-0 before:h-px before:bg-line-soft">
            <div className="px-4 py-3">
              <p className="console-meta">{HM.todayOrders}</p>
              <p className="stat-num">{fmtNum(S0?.orders ?? 0)}</p>
            </div>
            <div className="border-s border-line-soft px-4 py-3">
              <p className="console-meta">{HM.dueToday}</p>
              <p className="stat-num">{fmtNum(due)}</p>
            </div>
            <div className="border-s border-line-soft px-4 py-3">
              <p className="console-meta">{HM.cancelledToday}</p>
              <p className="stat-num">{fmtNum(S0?.cancelled ?? 0)}</p>
            </div>
          </div>
        </div>
      </div>

      {selfManage && (
        <IGroup title={HM.liveTitle} testId="store-live">
          <IRow
            icon={IconOrder}
            tone="info"
            title={SH.newOrders}
            value={<b className="rounded-full bg-info-solid px-2 text-sm text-on-solid">{fmtNum(live.fresh.length)}</b>}
            href="/store/orders"
            Link={Link}
          />
          <IRow
            icon={IconHourglass}
            tone="attention"
            title={SH.preparing}
            sub={live.lateNew.length > 0 ? `${SH.late} · ${fmtNum(live.lateNew.length)}` : undefined}
            value={fmtNum(live.prep.length)}
            href="/store/orders"
            Link={Link}
          />
          <IRow icon={IconCheck} tone="good" title={SH.ready} value={fmtNum(live.wait.length)} href="/store/orders" Link={Link} />
        </IGroup>
      )}

      {needs.length > 0 && (
        <IGroup title={HM.alertsTitle} testId="store-needs">
          {needs.map((n) => (
            <IRow key={n.key} icon={n.icon} tone={n.tone === "danger" ? "danger" : "attention"} title={n.text} value={n.num != null ? fmtNum(n.num) : undefined} href={n.href} Link={Link} />
          ))}
        </IGroup>
      )}

      {!suspended && (
        <div className="tab-span">
        <IGroup title={HM.quickTitle}>
          <div className="qa-row px-1.5 py-3.5">
            {[
              { href: "/store/menu/new", icon: IconAdd, label: HM.quickAdd, tone: undefined },
              { href: "/store/menu/photos", icon: IconCamera, label: SH.quickPhotos, tone: "violet" as const },
              { href: "/store/offers?new=1", icon: IconPromos, label: HM.quickOffers, tone: "attention" as const },
              { href: "/store/delivery", icon: IconMoto, label: HM.quickDelivery, tone: "info" as const },
            ].map((q) => {
              const Ico = q.icon;
              return (
                <Link key={q.href} href={q.href} className="flex flex-col items-center gap-1.5 text-center text-xs">
                  <span className="tile" data-size="lg" data-tone={q.tone}>
                    <Ico size={22} />
                  </span>
                  {q.label}
                </Link>
              );
            })}
          </div>
        </IGroup>
        </div>
      )}

      {settings}
      {emergency.dialog}
      {logoModal}
    </div>
  );
}
