"use client";

/**
 * قائمة مهامّ السائق — لا لوحة تحكّم.
 *
 * اللوحات لمن يجلس إلى مكتب فيمسح بعينه شبكة أرقام. والسائق واقفٌ عند درّاجته
 * يمسك هاتفه بيدٍ واحدة، وربّما تحت الشمس: فما يحتاجه سطرٌ واحد يقول **ماذا
 * الآن**، وزرٌّ واحد كبير يُنهيه. ولذلك:
 *
 * - لا صفحة تفاصيل منفصلة: البطاقة **هي** التفصيل، وكلُّ ضغطةٍ زائدة على درّاجة
 *   ضغطةٌ لن تُضغط.
 * - الزرّ الكبير واحدٌ في كل بطاقة، ونصُّه **الفعل التالي** لا اسم الحالة:
 *   «استلمتُ الطلب» لا «picked_up».
 * - «تعذّر التسليم» زرٌّ ثانويّ صغير تحته: هو مخرجٌ حقيقي لكنه ليس الطريق.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtTime } from "@rahalgo/i18n";
import {
  Button,
  Badge,
  Modal,
  Input,
  Card,
  EmptyState,
  StatGrid,
  StatCard,
  useLiveRefresh,
  IconOrder,
  IconStore,
  IconLocation,
  IconPhone,
  IconCheck,
  IconBalance,
  IconWallet,
  IconDriver,
  IconWarning,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const D = m.driver;

/** الفعل التالي لكل حالة — مصدرٌ واحد يقابل خارطة الحالات في الخادم. */
const NEXT: Record<string, string> = {
  assigned: "at_pickup",
  at_pickup: "picked_up",
  picked_up: "on_the_way",
  on_the_way: "at_dropoff",
  at_dropoff: "delivered",
};

interface DriverOrder {
  id: string;
  number: number;
  status: string;
  merchant_name: string;
  merchant_phone: string | null;
  address_text: string;
  lat: number;
  lng: number;
  customer_name: string;
  customer_phone: string;
  total: number;
  cash_due: number;
  items_count: number;
  ready_at: string | null;
  prep_minutes: number | null;
  accepted_at: string | null;
  created_at: string;
}

interface Me {
  full_name: string;
  on_shift: boolean;
  shift_started_at: string | null;
  cash_held: number;
  cash_limit: number;
  balance: number;
  today_delivered: number;
  today_earned: number;
  active_orders: number;
}

function errText(e: unknown): string {
  const key =
    e instanceof ApiError ? (e.body.message_key ?? "").split(".").pop() ?? "" : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

/** رابط الملاحة — يفتحه تطبيق الخرائط في الهاتف مباشرةً. */
function mapsHref(lat: number, lng: number): string {
  return `https://www.google.com/maps/dir/?api=1&destination=${lat},${lng}`;
}

export default function TasksPage() {
  const [me, setMe] = useState<Me | null>(null);
  const [mine, setMine] = useState<DriverOrder[]>([]);
  const [queue, setQueue] = useState<DriverOrder[]>([]);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [failing, setFailing] = useState<DriverOrder | null>(null);
  const [reason, setReason] = useState("");

  const load = useCallback(() => {
    api<Me>("/api/v1/driver/me").then(setMe).catch(() => undefined);
    api<DriverOrder[]>("/api/v1/driver/orders").then(setMine).catch(() => undefined);
    api<DriverOrder[]>("/api/v1/driver/queue").then(setQueue).catch(() => undefined);
  }, []);

  useEffect(load, [load]);

  // الطابور حيٌّ: طلبٌ يظهر لسائقين في اللحظة نفسها، ومن أخذه سبق. فبلا بثٍّ
  // يرى السائق طلباً أُخذ قبل دقيقتين فيضغط ثم يُردّ عليه — وهذا يُتعِب لا يُفيد.
  useLiveRefresh(["order", "wallet"], load);

  async function act(o: DriverOrder, to: string, note = "") {
    setBusy(o.id);
    setError("");
    try {
      await api(`/api/v1/driver/orders/${o.id}/transition`, {
        method: "POST",
        body: JSON.stringify({ to, note }),
      });
      load();
    } catch (e) {
      setError(errText(e));
      load();
    } finally {
      setBusy("");
    }
  }

  async function accept(o: DriverOrder) {
    setBusy(o.id);
    setError("");
    try {
      await api(`/api/v1/driver/orders/${o.id}/accept`, { method: "POST" });
      load();
    } catch (e) {
      setError(errText(e));
      load();
    } finally {
      setBusy("");
    }
  }

  async function toggleShift() {
    if (!me) return;
    setBusy("shift");
    setError("");
    try {
      await api("/api/v1/driver/shift", {
        method: "POST",
        body: JSON.stringify({ on: !me.on_shift }),
      });
      load();
    } catch (e) {
      setError(errText(e));
    } finally {
      setBusy("");
    }
  }

  if (!me) return <p className="p-6 text-center text-ink-muted">{m.common.loading}</p>;

  const cashRatio = me.cash_limit > 0 ? me.cash_held / me.cash_limit : 0;

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      {error && (
        <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      {/* الدوام: العلَم الذي يرفعه هو — لا يُستنتج عنه من آخر ظهور */}
      <Card>
        <div className="flex items-center gap-3">
          <span
            className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-badge ${
              me.on_shift ? "bg-success/10 text-success" : "bg-page text-ink-muted"
            }`}
          >
            <IconDriver size={20} />
          </span>
          <div className="min-w-0 flex-1">
            <p className="font-bold">{me.on_shift ? D.shift.on : D.shift.off}</p>
            <p className="text-xs text-ink-muted">
              {me.on_shift && me.shift_started_at
                ? D.shift.since.replace("{t}", fmtTime(me.shift_started_at))
                : D.shift.offHint}
            </p>
          </div>
          <Button
            variant={me.on_shift ? "secondary" : "primary"}
            disabled={busy === "shift"}
            onClick={toggleShift}
          >
            {me.on_shift ? D.shift.end : D.shift.start}
          </Button>
        </div>
      </Card>

      {/* النقد بذمّته — يُعرض دائماً لا عند الامتلاء: الرقم الذي يُرى يُسلَّم */}
      <Card>
        <div className="mb-2 flex items-center justify-between text-sm">
          <span className="flex items-center gap-2 font-medium">
            <IconBalance size={17} className="text-ink-muted" />
            {D.cash.title}
          </span>
          <span dir="ltr" className="font-bold">
            {fmtNum(me.cash_held)} / {fmtNum(me.cash_limit)}
          </span>
        </div>
        <div className="h-2 overflow-hidden rounded-badge bg-page">
          <div
            className={`h-full rounded-badge ${
              cashRatio >= 1 ? "bg-danger" : cashRatio >= 0.8 ? "bg-warning" : "bg-success"
            }`}
            style={{ width: `${Math.min(100, cashRatio * 100)}%` }}
          />
        </div>
        <p
          className={`mt-1.5 text-xs ${
            cashRatio >= 1 ? "text-danger" : cashRatio >= 0.8 ? "text-warning" : "text-ink-muted"
          }`}
        >
          {cashRatio >= 1 ? D.cash.full : cashRatio >= 0.8 ? D.cash.near : D.cash.hint}
        </p>
      </Card>

      <StatGrid>
        <StatCard
          icon={IconCheck}
          label={D.today.delivered}
          value={fmtNum(me.today_delivered)}
        />
        <StatCard
          icon={IconWallet}
          label={D.today.earned}
          value={`${fmtNum(me.today_earned)} ${m.common.currency}`}
        />
      </StatGrid>

      {/* مهامّي */}
      <section>
        <h2 className="mb-2 flex items-center gap-2 font-bold">
          <IconOrder size={18} className="text-ink-muted" />
          {D.tasks.title}
        </h2>
        {mine.length === 0 ? (
          <EmptyState icon={IconOrder} title={D.tasks.empty} action={<span className="text-sm text-ink-muted">{D.tasks.emptyHint}</span>} />
        ) : (
          <div className="space-y-3">
            {mine.map((o) => (
              <TaskCard
                key={o.id}
                o={o}
                busy={busy === o.id}
                onAct={(to) => act(o, to)}
                onFail={() => {
                  setReason("");
                  setFailing(o);
                }}
                onRelease={() => act(o, "dispatching")}
              />
            ))}
          </div>
        )}
      </section>

      {/* الطابور */}
      <section>
        <h2 className="mb-2 flex items-center gap-2 font-bold">
          <IconLocation size={18} className="text-ink-muted" />
          {D.queue.title}
        </h2>
        {!me.on_shift ? (
          <EmptyState icon={IconDriver} title={D.queue.offEmpty} />
        ) : queue.length === 0 ? (
          <EmptyState icon={IconOrder} title={D.queue.empty} />
        ) : (
          <div className="space-y-3">
            {queue.map((o) => (
              <QueueCard key={o.id} o={o} busy={busy === o.id} onAccept={() => accept(o)} />
            ))}
          </div>
        )}
      </section>

      {/* تعذّر التسليم — السبب إلزامي: طلبٌ يسقط بلا سبب خلافٌ مؤجَّل */}
      <Modal
        open={failing !== null}
        onClose={() => setFailing(null)}
        title={D.act.failedTitle}
      >
        <div className="space-y-3">
          <Input
            id="fail-reason"
            label={D.act.failedReason}
            required
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
          <p className="text-xs text-ink-muted">{D.act.failedHint}</p>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setFailing(null)}>
              {m.common.cancel}
            </Button>
            <Button
              variant="danger"
              disabled={!reason.trim() || busy !== ""}
              onClick={() => {
                const o = failing;
                setFailing(null);
                if (o) act(o, "failed", reason.trim());
              }}
            >
              {D.act.failed}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

/** بطاقة المهمّة — كلُّ ما يحتاجه في مكانٍ واحد، والزرّ الكبير آخرها. */
function TaskCard({
  o,
  busy,
  onAct,
  onFail,
  onRelease,
}: {
  o: DriverOrder;
  busy: boolean;
  onAct: (to: string) => void;
  onFail: () => void;
  onRelease: () => void;
}) {
  const next = NEXT[o.status];
  // قبل الاستلام وجهتُه المتجر، وبعده وجهتُه الزبون — الملاحة تتبع الرحلة
  const heading = o.status === "assigned" || o.status === "at_pickup";
  const cash = o.cash_due > 0;

  return (
    <Card>
      <div className="mb-3 flex items-center gap-2">
        <Badge variant="primary">#{fmtNum(o.number)}</Badge>
        <Badge variant="neutral">{m.orders.status[o.status as keyof typeof m.orders.status]}</Badge>
        <span className="ms-auto text-sm font-bold" dir="ltr">
          {fmtNum(o.total)} {m.common.currency}
        </span>
      </div>

      <Leg
        icon={IconStore}
        label={D.order.pickup}
        name={o.merchant_name}
        detail={D.order.items.replace("{n}", fmtNum(o.items_count))}
        phone={o.merchant_phone}
        callLabel={D.order.callMerchant}
        dim={!heading}
      />
      <Leg
        icon={IconLocation}
        label={D.order.dropoff}
        name={o.customer_name}
        detail={o.address_text}
        phone={o.customer_phone}
        callLabel={D.order.callCustomer}
        href={mapsHref(o.lat, o.lng)}
        hrefLabel={D.order.navigate}
        dim={heading}
      />

      {/* ما يقبضه: يُقال قبل التسليم لا بعده */}
      <p
        className={`mt-3 flex items-center gap-2 rounded-control px-3 py-2 text-sm ${
          cash ? "bg-warning/10 font-bold text-warning" : "bg-page text-ink-muted"
        }`}
      >
        <IconBalance size={16} />
        {cash ? (
          <>
            {D.order.cashCollect}:{" "}
            <span dir="ltr">
              {fmtNum(o.cash_due)} {m.common.currency}
            </span>
          </>
        ) : (
          D.order.walletPaid
        )}
      </p>

      {next && (
        <Button
          size="lg"
          className="mt-3 w-full"
          disabled={busy}
          onClick={() => onAct(next)}
        >
          {D.act[next as keyof typeof D.act]}
        </Button>
      )}

      <div className="mt-2 flex items-center justify-between">
        {/* الإعادة للطابور متاحة قبل بلوغ المتجر فقط: بعد الاستلام صار الطلب بيده */}
        {o.status === "assigned" ? (
          <Button variant="ghost" disabled={busy} onClick={onRelease} title={D.order.releaseHint}>
            {D.order.release}
          </Button>
        ) : (
          <span />
        )}
        {/* **المخرجُ عند الطرفين — لا عند الزبون وحده.**

            كان الإفشالُ متاحاً عند باب الزبون فقط. فلو وصل السائقُ إلى المطعم
            ووجده **مغلقاً**، أو لم تصله الرسالةُ أصلاً، أو رفض التحضير —
            **بقي الطلبُ معلّقاً بلا نهاية ممكنة**: مالُ الزبون محجوز، والسائقُ
            مربوطٌ بطلبٍ لا يُقفل وسقفُه النقديّ مشغولٌ به.

            **وهو من هناك، فهو من يقول.** واللفظُ يختلف بالموضع: عند المطعم
            «المطعم لم يسلّمني»، وعند الزبون «الزبون لم يستلم». و«فشل» وحدها
            تُخفي ثلاثة أخطاءٍ في ثلاث جهات. */}
        {(o.status === "at_pickup" || o.status === "at_dropoff") && (
          <Button variant="ghost" disabled={busy} onClick={onFail} className="text-danger">
            <span className="flex items-center gap-1.5">
              <IconWarning size={15} />
              {o.status === "at_pickup" ? D.act.failedAtPickup : D.act.failed}
            </span>
          </Button>
        )}
      </div>
    </Card>
  );
}

/** طرفُ الرحلة — المتجر أو الزبون. الطرف الذي ليس وجهتَه الآن يبهت. */
function Leg({
  icon: Icon,
  label,
  name,
  detail,
  phone,
  callLabel,
  href,
  hrefLabel,
  dim,
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  name: string;
  detail: string;
  phone: string | null;
  callLabel: string;
  href?: string;
  hrefLabel?: string;
  dim: boolean;
}) {
  return (
    <div className={`flex items-start gap-3 py-1.5 ${dim ? "opacity-50" : ""}`}>
      <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-control bg-page">
        <Icon size={16} className="text-ink-muted" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-xs text-ink-muted">{label}</p>
        <p className="truncate font-medium">{name}</p>
        <p className="truncate text-xs text-ink-muted">{detail}</p>
      </div>
      <div className="flex shrink-0 gap-1">
        {href && (
          <a
            href={href}
            target="_blank"
            rel="noreferrer"
            title={hrefLabel}
            aria-label={hrefLabel}
            className="flex h-9 w-9 items-center justify-center rounded-control bg-primary-light text-primary-dark"
          >
            <IconLocation size={17} />
          </a>
        )}
        {phone && (
          <a
            href={`tel:${phone}`}
            title={callLabel}
            aria-label={callLabel}
            className="flex h-9 w-9 items-center justify-center rounded-control bg-page text-ink-muted"
          >
            <IconPhone size={17} />
          </a>
        )}
      </div>
    </div>
  );
}

/** بطاقة الطابور — أقلّ ممّا في المهمّة: قرار الأخذ لا يحتاج رقم هاتف. */
function QueueCard({ o, busy, onAccept }: { o: DriverOrder; busy: boolean; onAccept: () => void }) {
  const readyLeft = o.ready_at
    ? 0
    : o.accepted_at && o.prep_minutes
      ? Math.max(
          0,
          Math.round(
            (new Date(o.accepted_at).getTime() + o.prep_minutes * 60_000 - Date.now()) / 60_000,
          ),
        )
      : null;

  return (
    <Card>
      <div className="mb-2 flex items-center gap-2">
        <Badge variant="primary">#{fmtNum(o.number)}</Badge>
        {readyLeft !== null && (
          <Badge variant={readyLeft === 0 ? "success" : "warning"}>
            {readyLeft === 0 ? D.queue.readyNow : D.queue.readyIn.replace("{n}", fmtNum(readyLeft))}
          </Badge>
        )}
        <span className="ms-auto text-sm font-bold" dir="ltr">
          {fmtNum(o.total)} {m.common.currency}
        </span>
      </div>
      <p className="flex items-center gap-2 text-sm font-medium">
        <IconStore size={16} className="text-ink-muted" />
        {o.merchant_name}
      </p>
      <p className="mt-1 flex items-start gap-2 text-sm text-ink-muted">
        <IconLocation size={16} className="mt-0.5 shrink-0" />
        <span className="min-w-0 flex-1">{o.address_text}</span>
      </p>
      {o.cash_due > 0 && (
        <p className="mt-2 flex items-center gap-1.5 text-xs font-medium text-warning">
          <IconBalance size={14} />
          {D.order.cashCollect}: <span dir="ltr">{fmtNum(o.cash_due)}</span>
        </p>
      )}
      <Button size="lg" className="mt-3 w-full" disabled={busy} onClick={onAccept}>
        {D.queue.accept}
      </Button>
    </Card>
  );
}
