"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store — «متجري»: لوحةُ قيادة المتجر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StoreHome.kt` — ملاحظةُ المالك ٦، ٢٠٢٦-١٠-٢٨؛ وإعادةُ التصميم ٢٠٢٦-١١-٠٧:
 *  «لسّا التصميم ضعيف جداً — لازم يكون أقوى واحترافي أكثر».)
 *
 * **من فوق إلى تحت (جوّالاً: عمودٌ واحد · حاسوباً: ثلاثةُ أعمدة):**
 *
 *   ١ · هويّةُ المتجر (شعارٌ واسمٌ) ‖ **حالُه** — مفتوحٌ · مغلقٌ بالساعات · إغلاقٌ طارئ · موقوف،
 *       وزرُّ الإغلاق الطارئ/إعادة الفتح كبيرٌ بحجم لمسةٍ كاملة.
 *   ٢ · **طلباتُ الآن** (جديدة · قيد التحضير · بانتظار السائق) من سياق الطلبات الحيّ نفسِه
 *       — بلا نداءٍ جديد، وتتحدّث مع البثّ.
 *   ٣ · **أرقامُ اليوم**: الطلبات · المبيعات · المستحق · الملغاة (من تقرير اليوم).
 *   ٤ · **يحتاج انتباهك**: أصنافٌ بلا صورة · غير متوفرة · بلا قسم (الأمين) · وما نقص من الإعداد.
 *   ٥ · **الإجراءات السريعة** · ٦ · **الإعدادات** في مجموعتين.
 *
 * **ولا مفتاحَ «مفتوح» يدويّ**: الحالُ تتبع ساعاتِ الدوام، **والإغلاقُ الطارئُ وحدَه بيده.**
 */

import { AmeenStatusTile } from "@/components/store/AmeenStatus";
import Link from "next/link";
import { useCallback, useEffect, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale, damascusDay, errorText, fmtMoney, fmtNum, fmtSince } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Confirm,
  FormActions,
  IconAdd,
  IconBanknote,
  IconBlock,
  IconCash,
  IconCheck,
  IconEdit,
  IconHourglass,
  IconCamera,
  IconDate,
  IconLocation,
  IconLock,
  IconMoto,
  IconNext,
  IconOrder,
  IconPhone,
  IconPromos,
  IconReports,
  IconReturn,
  IconRotate,
  IconStore,
  IconTrendUp,
  IconUsers,
  IconViewOff,
  IconWallet,
  IconWarning,
  IconCatOther,
  Input,
  Kpi,
  KpiGrid,
  LoadingState,
  Modal,
  Select,
  useLiveData,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { useStoreOrders } from "@/lib/store/orders";
import { pointSet, useStoreSections } from "@/lib/store/setup";
import { groupOf } from "@/lib/store/types";
import type { DayHours, MenuSummary, MerchantStore, StoreReport } from "@/lib/store/types";
import { backAt, fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";
import { LogoCard, useSaveSettings } from "@/components/store/StoreSettings";
import { FRIDAY, is24h, rangeText, useStoreHours } from "@/components/store/SetupParts";
import { usdText } from "@/lib/store/usd";

const m = getMessages(defaultLocale);
const S = m.store;
const T = S.settings;
const HM = S.home;

type IconT = ComponentType<{ size?: number; className?: string }>;

/**
 * **لونُ الأيقونة واحدٌ هادئ** — لونُ المنصّة لكلّ سطرٍ وزرّ (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩).
 * **والأحمرُ والبرتقاليُّ والأخضرُ للحال وحدَها**: نقطةُ «ناقص» وشريطُ الحال.
 */
const ICON_TINT = "bg-primary-tint text-primary";

// ══════════════════════════════════════════════════════════════════════
// الهويّة
// ══════════════════════════════════════════════════════════════════════

function Identity({ store, suspended, onLogo }: { store: MerchantStore; suspended: boolean; onLogo: () => void }) {
  const { stores, select } = useStore();
  const { busy, save } = useSaveSettings();
  const [draft, setDraft] = useState<string | null>(null);
  const logo = mediaUrl(store.logo_thumb_url);
  return (
    <section className="panel flex flex-col gap-3 p-4 lg:col-span-2 lg:justify-center" aria-label={HM.groupStore}>
      <div className="flex items-center gap-4">
        <button
          type="button"
          onClick={onLogo}
          disabled={suspended}
          aria-label={HM.changeLogo}
          className="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-card border border-line bg-field heading-page text-primary lg:size-20"
        >
          {logo ? <img src={logo} alt="" className="h-full w-full object-cover" /> : store.name.slice(0, 1) || "?"}
        </button>
        <div className="min-w-0 flex-1">
          {/* **والاسمُ مرّةً واحدة** (تجربةُ المالك ٢٠٢٦-١٠-٢٩) — ومن له فروعٌ يبدّل منه. */}
          {stores.length > 1 ? (
            <Select aria-label={S.pickStore} value={store.id} onChange={(e) => select(e.target.value)} className="font-bold">
              {stores.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </Select>
          ) : (
            <h1 className="console-title line-clamp-2">{store.name}</h1>
          )}
          <p className="console-meta mt-1">{S.nav.panel}</p>
        </div>
        {!suspended && draft == null && (
          <button
            type="button"
            onClick={() => setDraft(store.name)}
            aria-label={T.rename}
            title={T.rename}
            className="flex size-11 shrink-0 items-center justify-center rounded-control text-ink-muted hover:bg-row-hover hover:text-ink"
          >
            <IconEdit size={20} />
          </button>
        )}
      </div>
      {draft != null && (
        <form
          className="flex flex-col gap-2 rounded-control bg-field p-3"
          onSubmit={(e) => {
            e.preventDefault();
            const name = draft.trim();
            if (!name) return;
            void save({ name }, T.nameSaved, { name }).then((ok) => ok && setDraft(null));
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
          <FormActions submit onCancel={() => setDraft(null)} busy={busy} />
        </form>
      )}
    </section>
  );
}

// ══════════════════════════════════════════════════════════════════════
// الحال — والإغلاقُ الطارئ
// ══════════════════════════════════════════════════════════════════════

function StatusPanel({ store, suspended }: { store: MerchantStore; suspended: boolean }) {
  const { patch, reload } = useStore();
  const toast = useShopToast();
  const [asking, setAsking] = useState(false);
  const [busy, setBusy] = useState(false);
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
  /* **أرضٌ فاتحةٌ من لون الحال وحرفٌ بلونه** — و«إغلاق طارئ» زرٌّ بإطار لا تعبئة (لا يصرخ). */
  const tone = suspended
    ? "border-danger-edge bg-danger-tint text-danger"
    : store.emergency_closed
      ? "border-warning-edge bg-warning-tint text-warning"
      : !store.open_now
        ? "border-line bg-field text-ink-muted"
        : "border-success-edge bg-success-tint text-success";
  const Icon: IconT = suspended ? IconLock : store.emergency_closed || !store.open_now ? IconBlock : IconStore;

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

  const reopen = store.emergency_closed;
  return (
    <section className={`flex flex-col justify-center gap-3 rounded-card border p-4 ${tone}`} data-testid="store-status" aria-label={HM.statusLabel}>
      <div className="flex items-center gap-3">
        <span className="flex size-11 shrink-0 items-center justify-center rounded-badge bg-surface">
          <Icon size={24} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="heading-card">{label}</p>
          <p className="console-meta">{sub}</p>
        </div>
      </div>
      {!suspended && (
        <button
          type="button"
          disabled={busy}
          onClick={() => (reopen ? void setClosed(false) : setAsking(true))}
          className={`flex min-h-11 w-full items-center justify-center gap-2 rounded-control border bg-surface px-4 text-sm font-bold transition-colors hover:bg-row-hover disabled:opacity-60 ${
            reopen ? "border-success-edge text-success" : "border-danger-edge text-danger"
          }`}
        >
          <IconBlock size={18} />
          {reopen ? HM.reopen : T.emergency}
        </button>
      )}
      <Confirm
        open={asking}
        title={T.emergencyQ}
        body={T.emergencyHint}
        confirmLabel={T.emergencyDo}
        busy={busy}
        onConfirm={() => void setClosed(true)}
        onCancel={() => setAsking(false)}
      />
    </section>
  );
}

// ══════════════════════════════════════════════════════════════════════
// طلباتُ الآن
// ══════════════════════════════════════════════════════════════════════

function LiveOrders() {
  const { orders } = useStoreOrders();
  const now = Date.now();
  const list = orders ?? [];
  const fresh = list.filter((o) => groupOf(o) === "new");
  const cells: { key: string; label: string; n: number; urgent?: boolean }[] = [
    { key: "new", label: HM.liveNew, n: fresh.length, urgent: true },
    { key: "preparing", label: HM.livePrep, n: list.filter((o) => groupOf(o) === "preparing").length },
    { key: "waiting", label: HM.liveWait, n: list.filter((o) => groupOf(o) === "waiting").length },
  ];
  const total = cells.reduce((a, c) => a + c.n, 0);
  return (
    <section className="panel flex flex-col lg:col-span-2" aria-label={HM.liveTitle} data-testid="store-live">
      <div className="panel-head">
        <h2 className="console-h2 flex items-center gap-2">
          <IconOrder size={18} className="text-primary" />
          {HM.liveTitle}
        </h2>
        <Link href="/store/orders" className="flex min-h-11 items-center gap-1 text-sm font-bold text-primary hover:underline">
          {HM.liveAll}
          <IconNext size={16} />
        </Link>
      </div>
      <div className="flex gap-2 p-3 lg:gap-3 lg:p-4">
        {cells.map((c) => (
          <Link
            key={c.key}
            href="/store/orders"
            className={`flex min-w-0 flex-1 flex-col gap-1 rounded-control border p-3 text-start transition-colors hover:bg-row-hover ${
              c.urgent && c.n > 0 ? "border-danger-edge bg-danger-tint" : "border-line-soft bg-field"
            }`}
          >
            <span className="kpi-figure">{fmtNum(c.n)}</span>
            <span className="console-meta line-clamp-2">{c.label}</span>
          </Link>
        ))}
      </div>
      {fresh.length > 0 ? (
        <ul className="flex flex-col border-t border-line-soft">
          {fresh.slice(0, 3).map((o) => (
            <li key={o.id} className="border-b border-line-soft last:border-0">
              <Link href="/store/orders" className="flex min-h-14 items-center gap-3 px-4 py-2 transition-colors hover:bg-row-hover">
                <span className="heading-card" dir="ltr">
                  #{o.number}
                </span>
                <span className="console-meta">{fmtSince(o.sent_to_merchant_at ?? o.created_at, now)}</span>
                <span className="ms-auto text-sm font-bold tabular-nums">{fmtMoney(o.subtotal)}</span>
                <IconNext size={16} className="shrink-0 text-ink-muted" />
              </Link>
            </li>
          ))}
        </ul>
      ) : (
        total === 0 && <p className="border-t border-line-soft px-4 py-4 text-center text-sm text-ink-muted">{HM.liveNone}</p>
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
}

function NeedsPanel({ needs, wide }: { needs: Need[]; wide: boolean }) {
  return (
    <section className={`panel flex flex-col ${wide ? "lg:col-span-3" : ""}`} aria-label={HM.alertsTitle} data-testid="store-needs">
      <div className="panel-head">
        <h2 className="console-h2 flex items-center gap-2">
          <IconWarning size={18} className="text-primary" />
          {HM.alertsTitle}
        </h2>
        {needs.length > 0 && <span className="chip-count">{fmtNum(needs.length)}</span>}
      </div>
      {needs.length === 0 ? (
        <p className="flex items-center justify-center gap-2 px-4 py-6 text-center text-sm text-ink-muted">
          <IconCheck size={18} className="text-success" />
          {HM.alertsClear}
        </p>
      ) : (
        <ul className="flex flex-col">
          {needs.map((n) => {
            const Ico = n.icon;
            return (
              <li key={n.key} className="border-b border-line-soft last:border-0">
                <Link href={n.href} className="flex min-h-14 items-center gap-3 px-4 py-2 transition-colors hover:bg-row-hover">
                  <span
                    className={`flex size-9 shrink-0 items-center justify-center rounded-control ${
                      n.tone === "danger" ? "bg-danger-tint text-danger" : "bg-warning-tint text-warning"
                    }`}
                  >
                    <Ico size={18} />
                  </span>
                  <span className="min-w-0 flex-1 text-sm font-medium">{n.text}</span>
                  {n.cta && <span className="shrink-0 text-sm font-bold text-primary">{n.cta}</span>}
                  <IconNext size={16} className="shrink-0 text-ink-muted" />
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

// ══════════════════════════════════════════════════════════════════════
// الإجراءاتُ السريعة · سطورُ الإعداد
// ══════════════════════════════════════════════════════════════════════

function QuickAction({
  href,
  icon: Icon,
  label,
  badge = 0,
  primary = false,
  className = "",
}: {
  href: string;
  icon: IconT;
  label: string;
  badge?: number;
  primary?: boolean;
  className?: string;
}) {
  return (
    <Link
      href={href}
      className={`relative flex min-h-16 min-w-0 items-center gap-3 rounded-card border px-4 py-3 text-start transition-colors ${
        primary
          ? "border-accent bg-accent text-on-bright hover:opacity-90"
          : "border-line-soft bg-surface shadow-[var(--shadow-e1)] hover:bg-row-hover"
      } ${className}`}
    >
      <span className={`flex size-10 shrink-0 items-center justify-center rounded-control ${primary ? "bg-on-bright text-accent-text" : ICON_TINT}`}>
        <Icon size={22} />
      </span>
      <span className="min-w-0 flex-1 text-sm font-bold">{label}</span>
      {badge > 0 && (
        <span className="rounded-badge bg-danger-solid px-2 py-0.5 text-xs font-bold tabular-nums text-on-solid">{fmtNum(badge)}</span>
      )}
    </Link>
  );
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="panel flex flex-col overflow-hidden" aria-label={title}>
      <div className="panel-head">
        <h2 className="console-h2">{title}</h2>
      </div>
      <div className="flex flex-col divide-y divide-line-soft">{children}</div>
    </section>
  );
}

function Row({
  icon: Icon,
  label,
  value,
  warn,
  href,
  onClick,
  disabled,
}: {
  icon: IconT;
  label: string;
  value?: string;
  warn?: boolean;
  href?: string;
  onClick?: () => void;
  disabled?: boolean;
}) {
  const body = (
    <>
      <span className={`flex size-9 shrink-0 items-center justify-center rounded-control ${ICON_TINT}`}>
        <Icon size={20} />
      </span>
      <span className="shrink-0 text-sm font-medium">{label}</span>
      <span className={`min-w-0 flex-1 truncate text-end text-sm ${warn ? "text-danger" : "text-ink-muted"}`}>{value}</span>
      {warn && <span className="size-2.5 shrink-0 rounded-badge bg-danger-solid" aria-label={HM.missing} />}
      <IconNext size={18} className="shrink-0 text-ink-muted" />
    </>
  );
  const cls = "flex min-h-14 w-full items-center gap-3 px-4 py-2 text-start transition-colors hover:bg-row-hover disabled:opacity-60";
  return href ? (
    <Link href={href} className={cls}>
      {body}
    </Link>
  ) : (
    <button type="button" onClick={onClick} disabled={disabled} className={cls}>
      {body}
    </button>
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
// الصفحة
// ══════════════════════════════════════════════════════════════════════

export default function StoreHome() {
  const { store, selfManage } = useStore();
  const id = store?.id ?? "";
  const loadReport = useCallback(() => {
    const today = damascusDay();
    return api<StoreReport>(`/api/v1/merchant/stores/${id}/reports?from=${today}&to=${today}`);
  }, [id]);
  const { data: report, loading } = useLiveData(loadReport, ["order"], [id]);
  const sec = useStoreSections(id);
  const { saved: hours } = useStoreHours(id);
  const [menu, setMenu] = useState<MenuSummary | null>(null);
  /** **أصنافُ الأمين بلا قسم** — من `price-sync/groups`، وللمتجر الذي يستعمل الأمين وحدَه. */
  const [unsorted, setUnsorted] = useState(0);
  const [logoOpen, setLogoOpen] = useState(false);
  const usesAmeen = store?.uses_ameen === true;

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

  if (!store) return null;
  const suspended = store.status === "suspended";
  const point = pointSet(store);
  const notSet = HM.notSet;
  const unavailable = menu?.unavailable ?? 0;
  const noPhoto = menu?.no_photo ?? 0;

  /* **ما يحتاج انتباهاً** — النقصُ في الإعداد أوّلاً (يمنع عملاً)، ثمّ الأصناف. */
  const needs: Need[] = [];
  if (sec.known && sec.mine.length === 0)
    needs.push({ key: "sections", text: `${HM.rowSections}: ${notSet}`, href: "/store/settings/sections", tone: "danger", icon: IconCatOther });
  if (!point)
    needs.push({ key: "address", text: `${HM.rowAddress}: ${notSet}`, href: "/store/settings/address", tone: "danger", icon: IconLocation });
  if (!store.has_hours)
    needs.push({ key: "hours", text: `${HM.rowHours}: ${notSet}`, href: "/store/settings/hours", tone: "danger", icon: IconDate });
  if (!store.settlement_confirmed)
    needs.push({ key: "settle", text: `${HM.rowSettlement}: ${notSet}`, href: "/store/settings/settlement", tone: "danger", icon: IconWallet });
  if (!store.returns_confirmed)
    needs.push({ key: "returns", text: `${HM.rowReturns}: ${notSet}`, href: "/store/settings/returns", tone: "danger", icon: IconReturn });
  if (!suspended && noPhoto > 0)
    needs.push({ key: "photo", text: fill(HM.alertNoPhoto, { n: fmtNum(noPhoto) }), cta: HM.alertNoPhotoCta, href: "/store/menu/photos", tone: "warning", icon: IconCamera });
  if (!suspended && unsorted > 0)
    needs.push({ key: "unsorted", text: fill(HM.alertUnsorted, { n: fmtNum(unsorted) }), cta: HM.alertUnsortedCta, href: "/store/price-sync", tone: "warning", icon: IconRotate });
  if (unavailable > 0)
    needs.push({ key: "off", text: fill(HM.alertOff, { n: fmtNum(unavailable) }), cta: HM.alertOffCta, href: "/store/menu?filter=unavailable", tone: "warning", icon: IconViewOff });

  return (
    <div className="console-page lg:grid lg:grid-cols-3" data-testid="store-home">
      <Identity store={store} suspended={suspended} onLogo={() => setLogoOpen(true)} />
      <StatusPanel store={store} suspended={suspended} />
      {suspended && (
        <div className="lg:col-span-3">
          <Alert tone="error">{S.suspendedBanner}</Alert>
        </div>
      )}

      {/* **حالةُ برنامج الأمين قرب الأعلى** (ملاحظةُ المالك ٢٠٢٦-١١-٠٧) — لمتاجر الأمين وحدَها. */}
      <AmeenStatusTile className="lg:col-span-3" />

      <section className="flex flex-col gap-2 lg:col-span-3" aria-label={HM.today}>
        <h2 className="console-h2 px-1">{HM.today}</h2>
        {loading && !report ? (
          <LoadingState variant="stats" rows={4} />
        ) : (
          report && (
            <KpiGrid>
              <Kpi
                icon={IconOrder}
                label={HM.todayOrders}
                value={report.summary.orders}
                sub={`${HM.deliveredToday}: ${fmtNum(report.summary.delivered)}`}
                href="/store/history"
                Link={Link}
                testId="kpi-orders"
              />
              <Kpi
                icon={IconReports}
                label={HM.salesToday}
                value={fmtNum(report.summary.sales)}
                unit={m.common.currency}
                tone="accent"
                testId="kpi-sales"
              />
              <Kpi
                icon={IconWallet}
                label={HM.dueToday}
                value={fmtNum(report.summary.sales - report.summary.platform_commission)}
                unit={m.common.currency}
                tone="good"
                testId="kpi-due"
              />
              <Kpi
                icon={IconBlock}
                label={HM.cancelledToday}
                value={report.summary.cancelled}
                tone={report.summary.cancelled > 0 ? "attention" : undefined}
                testId="kpi-cancelled"
              />
            </KpiGrid>
          )
        )}
      </section>

      {selfManage && <LiveOrders />}
      {selfManage && <NeedsPanel needs={needs} wide={false} />}

      {/* الوضع «المنصّة تدير» لا تعرض الطلبات الحيّة، فيأخذ «يحتاج انتباهك» العرضَ كلَّه. */}
      {!selfManage && <NeedsPanel needs={needs} wide />}

      {!suspended && (
        <section className="flex flex-col gap-2 lg:col-span-3" aria-label={HM.quickTitle}>
          <h2 className="console-h2 px-1">{HM.quickTitle}</h2>
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
            <QuickAction href="/store/menu/new" icon={IconAdd} label={HM.quickAdd} primary className="col-span-2 lg:col-span-1" />
            <QuickAction href="/store/offers" icon={IconPromos} label={HM.quickOffers} />
            <QuickAction href="/store/menu?filter=unavailable" icon={IconViewOff} label={HM.quickUnavailable} badge={unavailable} />
            <QuickAction href="/store/menu/prices" icon={IconBanknote} label={HM.quickPrices} />
            <QuickAction href="/store/delivery" icon={IconMoto} label={HM.quickDelivery} />
          </div>
        </section>
      )}

      <div className="flex flex-col gap-4 lg:col-span-3 lg:grid lg:grid-cols-2 lg:items-start lg:gap-5">
        <Group title={HM.groupStore}>
          <Row
            icon={IconCatOther}
            label={HM.rowSections}
            value={sec.mine.map((s) => s.name).join(" · ") || notSet}
            warn={sec.known && sec.mine.length === 0}
            href="/store/settings/sections"
          />
          <Row
            icon={IconLocation}
            label={HM.rowAddress}
            value={point ? store.address_text : notSet}
            warn={!point}
            href="/store/settings/address"
          />
          <Row
            icon={IconCamera}
            label={HM.rowLogo}
            value={store.logo_thumb_url ? HM.logoSet : HM.logoNone}
            disabled={suspended}
            onClick={() => setLogoOpen(true)}
          />
          <Row icon={IconUsers} label={HM.rowStaff} href="/store/team" />
          <Row icon={IconPhone} label={HM.rowOrderPhones} href="/store/order-phones" />
        </Group>

        <Group title={HM.groupHoursMoney}>
          <Row
            icon={IconDate}
            label={HM.rowHours}
            value={hoursValue(store, hours) ?? notSet}
            warn={!store.has_hours}
            href="/store/settings/hours"
          />
          <Row
            icon={store.settlement_method === "wallet" ? IconWallet : IconCash}
            label={HM.rowSettlement}
            value={!store.settlement_confirmed ? notSet : store.settlement_method === "wallet" ? HM.wallet : HM.cash}
            warn={!store.settlement_confirmed}
            href="/store/settings/settlement"
          />
          <Row
            icon={IconReturn}
            label={HM.rowReturns}
            value={!store.returns_confirmed ? notSet : store.accepts_returns ? HM.returnsYes : HM.returnsNo}
            warn={!store.returns_confirmed}
            href="/store/settings/returns"
          />
          <Row
            icon={IconCheck}
            label={HM.rowAmeen}
            value={store.uses_ameen ? HM.ameenYesShort : HM.ameenNoShort}
            href="/store/settings/ameen"
          />
          <Row
            icon={IconTrendUp}
            label={HM.rowRate}
            value={store.exchange_rate ? usdText(store.exchange_rate) : notSet}
            href="/store/settings/rate"
          />
          <Row
            icon={IconHourglass}
            label={HM.rowPrep}
            value={`${fmtNum(store.default_prep_minutes)} ${T.prepUnit}`}
            href="/store/settings/prep"
          />
        </Group>
      </div>

      <Modal open={logoOpen} onClose={() => setLogoOpen(false)} title={T.logo}>
        <div className="flex flex-col gap-3">
          <LogoCard />
          <Button variant="secondary" onClick={() => setLogoOpen(false)}>
            {m.common.close}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
