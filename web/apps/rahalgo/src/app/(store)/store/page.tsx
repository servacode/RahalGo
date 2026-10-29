"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store — «متجري»: الشاشةُ نفسُها التي في التطبيق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StoreHome.kt` — ملاحظةُ المالك ٦، ٢٠٢٦-١٠-٢٨.)
 *
 * **من فوق إلى تحت**: رأسٌ بالشعار والاسم · **شريطُ الحال ملوّناً** (أخضر مفتوح ·
 * رماديّ مغلق بالساعات · برتقاليّ إغلاقٌ طارئ · أحمر موقوف) وزرُّه تحته ·
 * **أربعةُ أزرارٍ سريعة** · **أرقامُ اليوم** · **ثمّ الإعداداتُ في مجموعتين**:
 * «المتجر» و«الدوام والمال»، كلُّ سطرٍ بأيقونته وقيمته ونقطةٍ حمراءَ إن نقص.
 *
 * **ولا مفتاحَ «مفتوح» يدويّ**: الحالُ تتبع ساعاتِ الدوام، **والإغلاقُ الطارئُ
 * وحدَه بيده.**
 */

import Link from "next/link";
import { useCallback, useEffect, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale, damascusDay, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Confirm,
  FormActions,
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
  IconStore,
  IconTrendUp,
  IconUsers,
  IconViewOff,
  IconWallet,
  IconCatOther,
  Input,
  LoadingState,
  Modal,
  Select,
  useLiveData,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { pointSet, useStoreSections } from "@/lib/store/setup";
import type { DayHours, MenuSection, MerchantStore, StoreReport } from "@/lib/store/types";
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

/** **لونُ الأيقونة** — توكنز المنصّة لا ألوانٌ مكتوبة. */
const TINT = {
  brand: "bg-primary-tint text-primary",
  accent: "bg-accent-tint text-accent-text",
  success: "bg-success-tint text-success",
  danger: "bg-danger-tint text-danger",
} as const;
type Tint = keyof typeof TINT;

// ══════════════════════════════════════════════════════════════════════
// الرأس
// ══════════════════════════════════════════════════════════════════════

function HomeHeader({ store, suspended, onLogo }: { store: MerchantStore; suspended: boolean; onLogo: () => void }) {
  const { stores, select } = useStore();
  const { busy, save } = useSaveSettings();
  const [draft, setDraft] = useState<string | null>(null);
  const logo = mediaUrl(store.logo_thumb_url);
  return (
    <section className="flex flex-col gap-3 rounded-card bg-primary-solid p-4 text-on-solid">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={onLogo}
          disabled={suspended}
          aria-label={HM.changeLogo}
          className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden figure rounded-card border-2 border-on-solid bg-paper text-primary-solid"
        >
          {logo ? (
            <img src={logo} alt="" className="h-full w-full object-cover" />
          ) : (
            store.name.slice(0, 1) || "?"
          )}
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
            <h1 className="heading-page line-clamp-2">{store.name}</h1>
          )}
          {store.address_text.trim() && <p className="truncate text-sm opacity-85">{store.address_text}</p>}
          {!suspended && (
            <button type="button" onClick={onLogo} className="mt-1 text-sm font-bold underline-offset-2 hover:underline">
              {logo ? HM.changeLogo : HM.addLogo}
            </button>
          )}
        </div>
        {!suspended && draft == null && (
          <button
            type="button"
            onClick={() => setDraft(store.name)}
            aria-label={T.rename}
            title={T.rename}
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-control hover:bg-scrim"
          >
            <IconEdit size={20} />
          </button>
        )}
      </div>
      {draft != null && (
        <form
          className="flex flex-col gap-2 rounded-control bg-surface p-3 text-ink"
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
// شريطُ الحال
// ══════════════════════════════════════════════════════════════════════

function StatusBand({ store, suspended }: { store: MerchantStore; suspended: boolean }) {
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
  const tone = suspended
    ? "bg-danger-solid"
    : store.emergency_closed
      ? "bg-warning-solid"
      : !store.open_now
        ? "bg-disabled"
        : "bg-success-solid";
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
    <section className={`flex flex-col gap-3 rounded-card p-4 text-on-solid ${tone}`} data-testid="store-status">
      <div className="flex items-center gap-3">
        <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-badge bg-on-solid-tint">
          <Icon size={24} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="heading-section">{label}</p>
          <p className="text-sm opacity-90">{sub}</p>
        </div>
      </div>
      {!suspended && (
        <button
          type="button"
          disabled={busy}
          onClick={() => (reopen ? void setClosed(false) : setAsking(true))}
          className={`flex h-12 w-full items-center justify-center gap-2 rounded-control bg-paper text-base font-bold disabled:opacity-60 ${
            reopen ? "text-success-solid" : "text-danger-solid"
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
// الأزرارُ السريعة · أرقامُ اليوم · سطورُ الإعداد
// ══════════════════════════════════════════════════════════════════════

function QuickAction({ href, icon: Icon, label, tint, badge = 0 }: { href: string; icon: IconT; label: string; tint: Tint; badge?: number }) {
  return (
    <Link
      href={href}
      className="surface relative flex min-w-0 flex-1 flex-col items-center gap-2 rounded-card px-1 py-3 text-center transition-colors hover:bg-row-hover"
    >
      <span className={`flex h-11 w-11 items-center justify-center rounded-badge ${TINT[tint]}`}>
        <Icon size={24} />
      </span>
      <span className="text-xs font-bold sm:text-sm">{label}</span>
      {badge > 0 && (
        <span className="absolute end-1 top-1 rounded-badge bg-danger-solid px-2 py-0.5 text-xs font-bold text-on-solid">
          {fmtNum(badge)}
        </span>
      )}
    </Link>
  );
}

function Tile({ icon: Icon, tint, value, label, sub, href }: { icon: IconT; tint: Tint; value: string; label: string; sub?: string; href?: string }) {
  const body = (
    <>
      <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-control ${TINT[tint]}`}>
        <Icon size={20} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="figure truncate tabular-nums">{value}</p>
        <p className="truncate text-xs text-ink-muted">{label}</p>
        {sub && <p className="truncate text-xs text-ink-muted">{sub}</p>}
      </div>
    </>
  );
  // **وطلباتُ اليوم تُضغط فتُري الطلبات** (قرارُ المالك ٢٠٢٦-١٠-٢٩).
  return href ? (
    <Link href={href} className="surface flex items-center gap-3 rounded-card p-3 transition-colors hover:bg-row-hover">
      {body}
    </Link>
  ) : (
    <div className="surface flex items-center gap-3 rounded-card p-3">{body}</div>
  );
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2" aria-label={title}>
      <h2 className="px-1 text-sm font-bold text-ink-muted">{title}</h2>
      <div className="surface flex flex-col divide-y divide-line overflow-hidden rounded-card p-0">{children}</div>
    </section>
  );
}

function Row({
  icon: Icon,
  tint,
  label,
  value,
  warn,
  href,
  onClick,
  disabled,
}: {
  icon: IconT;
  tint: Tint;
  label: string;
  value?: string;
  warn?: boolean;
  href?: string;
  onClick?: () => void;
  disabled?: boolean;
}) {
  const body = (
    <>
      <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-control ${TINT[tint]}`}>
        <Icon size={20} />
      </span>
      <span className="shrink-0 text-base">{label}</span>
      <span className={`min-w-0 flex-1 truncate text-end text-sm ${warn ? "text-danger" : "text-ink-muted"}`}>{value}</span>
      {warn && <span className="h-2.5 w-2.5 shrink-0 rounded-badge bg-danger-solid" aria-label={HM.missing} />}
      <IconNext size={18} className="shrink-0 text-ink-muted" />
    </>
  );
  const cls = "flex min-h-15 w-full items-center gap-3 px-3 py-2 text-start transition-colors hover:bg-row-hover disabled:opacity-60";
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
  const { store } = useStore();
  const id = store?.id ?? "";
  const loadReport = useCallback(() => {
    const today = damascusDay();
    return api<StoreReport>(`/api/v1/merchant/stores/${id}/reports?from=${today}&to=${today}`);
  }, [id]);
  const { data: report, loading } = useLiveData(loadReport, ["order"], [id]);
  const sec = useStoreSections(id);
  const { saved: hours } = useStoreHours(id);
  const [unavailable, setUnavailable] = useState(0);
  const [logoOpen, setLogoOpen] = useState(false);

  useEffect(() => {
    if (!id) return;
    api<MenuSection[]>(`/api/v1/merchant/stores/${id}/menu`)
      .then((secs) => setUnavailable((secs ?? []).reduce((n, s) => n + s.items.filter((i) => !i.available).length, 0)))
      .catch(() => undefined);
  }, [id]);

  if (!store) return null;
  const suspended = store.status === "suspended";
  const point = pointSet(store);
  const notSet = HM.notSet;

  return (
    <div className="flex flex-col gap-4">
      <HomeHeader store={store} suspended={suspended} onLogo={() => setLogoOpen(true)} />
      <StatusBand store={store} suspended={suspended} />
      {suspended && <Alert tone="error">{S.suspendedBanner}</Alert>}

      {!suspended && (
        <div className="flex gap-2">
          <QuickAction href="/store/offers" icon={IconPromos} label={HM.quickOffers} tint="brand" />
          <QuickAction
            href="/store/menu?filter=unavailable"
            icon={IconViewOff}
            label={HM.quickUnavailable}
            tint="accent"
            badge={unavailable}
          />
          <QuickAction href="/store/menu/prices" icon={IconPromos} label={HM.quickPrices} tint="success" />
          <QuickAction href="/store/delivery" icon={IconMoto} label={HM.quickDelivery} tint="brand" />
        </div>
      )}

      <section className="flex flex-col gap-2" aria-label={HM.today}>
        <h2 className="px-1 text-sm font-bold text-ink-muted">{HM.today}</h2>
        {loading && !report ? (
          <LoadingState variant="stats" rows={4} />
        ) : (
          report && (
            <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
              <Tile
                icon={IconOrder}
                tint="brand"
                value={fmtNum(report.summary.orders)}
                label={HM.todayOrders}
                sub={fill(HM.cancelledSub, { n: fmtNum(report.summary.cancelled) })}
                href="/store/history"
              />
              <Tile icon={IconCheck} tint="success" value={fmtNum(report.summary.delivered)} label={HM.deliveredToday} />
              <Tile icon={IconReports} tint="accent" value={fmtMoney(report.summary.sales)} label={HM.salesToday} />
              <Tile
                icon={IconWallet}
                tint="brand"
                value={fmtMoney(report.summary.sales - report.summary.platform_commission)}
                label={HM.dueToday}
              />
            </div>
          )
        )}
      </section>

      <Group title={HM.groupStore}>
        <Row
          icon={IconCatOther}
          tint="accent"
          label={HM.rowSections}
          value={sec.mine.map((s) => s.name).join(" · ") || notSet}
          warn={sec.known && sec.mine.length === 0}
          href="/store/settings/sections"
        />
        <Row
          icon={IconLocation}
          tint="danger"
          label={HM.rowAddress}
          value={point ? store.address_text : notSet}
          warn={!point}
          href="/store/settings/address"
        />
        <Row
          icon={IconCamera}
          tint="success"
          label={HM.rowLogo}
          value={store.logo_thumb_url ? HM.logoSet : HM.logoNone}
          disabled={suspended}
          onClick={() => setLogoOpen(true)}
        />
        <Row icon={IconUsers} tint="brand" label={HM.rowStaff} href="/store/team" />
        <Row icon={IconPhone} tint="accent" label={HM.rowOrderPhones} href="/store/order-phones" />
      </Group>

      <Group title={HM.groupHoursMoney}>
        <Row
          icon={IconDate}
          tint="brand"
          label={HM.rowHours}
          value={hoursValue(store, hours) ?? notSet}
          warn={!store.has_hours}
          href="/store/settings/hours"
        />
        <Row
          icon={store.settlement_method === "wallet" ? IconWallet : IconCash}
          tint="success"
          label={HM.rowSettlement}
          value={!store.settlement_confirmed ? notSet : store.settlement_method === "wallet" ? HM.wallet : HM.cash}
          warn={!store.settlement_confirmed}
          href="/store/settings/settlement"
        />
        <Row
          icon={IconReturn}
          tint="accent"
          label={HM.rowReturns}
          value={!store.returns_confirmed ? notSet : store.accepts_returns ? HM.returnsYes : HM.returnsNo}
          warn={!store.returns_confirmed}
          href="/store/settings/returns"
        />
        <Row
          icon={IconTrendUp}
          tint="brand"
          label={HM.rowRate}
          value={store.exchange_rate ? usdText(store.exchange_rate) : notSet}
          href="/store/settings/rate"
        />
        <Row
          icon={IconHourglass}
          tint="accent"
          label={HM.rowPrep}
          value={`${fmtNum(store.default_prep_minutes)} ${T.prepUnit}`}
          href="/store/settings/prep"
        />
      </Group>

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
