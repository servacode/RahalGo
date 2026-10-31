"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أجزاءُ لوحة العمل — قراراتُ المالك ٢٠٢٦-١٠-٠٤ (قسمُ «الطلبات»)**
 * ══════════════════════════════════════════════════════════════════════
 *
 *	عدّاداتُ المراحل        البند ٣ — كلُّ عدّادٍ فلترٌ بشرطه نفسِه في الخادم
 *	الصوتُ و«استلمتها»     البند ٥ — رنينٌ متكرّرٌ للطلب الجديد حتّى يُستلم
 *	سطرُ السائق            البند ٤ — اسمٌ · اتّصال · آخرُ ظهورٍ بلون، تحت الترويسة
 *	العرضُ الحيّ            «معروضٌ على فلان — باقي كذا ث — مرّ على كذا سائق»
 *	الدقائقُ الحيّة          تُعدّ في الشاشة من لحظة السبب، بالمنسِّق المركزيّ
 *
 * **والشروطُ كلُّها في الخادم** (`orders/board.go`) — **وهنا عرضُها وحده.**
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  AdminCard,
  Badge,
  Button,
  IconBell,
  IconBellOff,
  IconPhone,
  IconWarning,
  Row,
  RowGroupTitle,
  RowIcon,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const B = m.admin.ordersPage.board;

/**
 * **ساعةٌ تدقّ في الشاشة** — والأوقاتُ تُعاد حسبتُها بلا جلب (المشكلة ١٨).
 *
 * كان «آخرُ ظهورٍ قبل ١ د» يبقى كما هو عشرين دقيقة **لأنّه يُحسب لحظةَ الرسم
 * وحدَها**، وزرُّ الإسناد لا يظهر بعد مهلته ما لم يصل حدث.
 */
export function useNow(everyMs = 30_000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), everyMs);
    return () => clearInterval(id);
  }, [everyMs]);
  return now;
}

/** **دقائقُ منذ لحظةٍ** — وصفرٌ لما لم يمضِ. */
export function minutesSince(iso: string | null | undefined, now: number): number | null {
  if (!iso) return null;
  const at = Date.parse(iso);
  if (!Number.isFinite(at)) return null;
  return Math.max(0, Math.floor((now - at) / 60_000));
}

/** فلاترُ العدّادات الظاهرة — **بترتيب القرار** (البند ٣) ثمّ «عالقة». */
export const COUNTER_FILTERS = [
  "awaiting_accept",
  "awaiting_driver",
  "on_the_way",
  "at_door",
  "reports",
  "stuck",
] as const;

/**
 * **زرُّ الصوت** — المتصفّحُ لا يُسمع شيئاً قبل ضغطة، **فيُقال ذلك بزرٍّ لا بصمت.**
 */
export function SoundButton({
  enabled,
  unlocked,
  onEnable,
  onToggle,
}: {
  enabled: boolean;
  unlocked: boolean;
  onEnable: () => void;
  onToggle: (on: boolean) => void;
}) {
  if (enabled && !unlocked) {
    return (
      <Button variant="primary" size="sm" onClick={onEnable} title={B.soundHint}>
        <span className="flex items-center gap-1.5">
          <IconBell size={15} />
          {B.soundEnable}
        </span>
      </Button>
    );
  }
  return (
    <Button
      variant="secondary"
      size="sm"
      aria-pressed={enabled}
      onClick={() => onToggle(!enabled)}
    >
      <span className="flex items-center gap-1.5">
        {enabled ? <IconBell size={15} /> : <IconBellOff size={15} />}
        {enabled ? B.soundOn : B.soundOff}
      </span>
    </Button>
  );
}

/** طلبٌ جديدٌ يرنّ — من قائمة الخادم بفلتر `new`. */
export interface NewOrder {
  id: string;
  number: number;
  merchant_name: string;
  created_at: string;
}

/** **طلبٌ عالق** — من `GET /orders/alerts`، والهاتفُ يحجبه الخادمُ لمن لا يملك تفاصيلَ الزبون. */
export interface StuckAlert {
  order_id: string;
  number: number;
  status: string;
  merchant_name: string;
  customer_phone: string;
  customer_phone_masked?: string;
  reason: string;
  minutes: number;
  since?: string;
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«بدها انتباهك» — بطاقةٌ واحدةٌ بقسمين** (تدقيقُ الطلبات ٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * كانت الجديدةُ شريطاً والعالقةُ شريطاً وفشلُ التنبيهات شريطاً — **فوق بعضها قبل أن تصل
 * الطلبات.** وصارت: «جديدة» بالأزرق و«عالقة» بالأحمر، وكلُّ سطرٍ رقمُه وسببُه ومدّتُه وزرُّ «افتح».
 *
 * **وفتحُ الجديد يُسكت رنينَه** (`POST /orders/{id}/seen`) — بلا زرِّ «استلمتها» (المالك ٢٠٢٦-١٠-٢٢:
 * «مابدي هالزر اصلا») — فيسكت عند المكتب كلِّه لا عند من ضغط وحدَه.
 */
export function AttentionCard({
  newOrders,
  newTotal,
  alerts,
  alertsFailed,
  onRetryAlerts,
  now,
  canAck,
  onAcked,
  onOpen,
  officeStatus,
  alertReasons,
}: {
  newOrders: NewOrder[];
  newTotal: number;
  alerts: StuckAlert[];
  alertsFailed: boolean;
  onRetryAlerts: () => void;
  now: number;
  canAck: boolean;
  onAcked: () => void;
  onOpen: (id: string, number: number) => void;
  officeStatus: Record<string, string>;
  alertReasons: Record<string, string>;
}) {
  const [err, setErr] = useState("");
  if (newOrders.length === 0 && alerts.length === 0 && !alertsFailed) return null;

  async function openNew(o: NewOrder) {
    setErr("");
    if (canAck) {
      try {
        await api(`/api/v1/admin/orders/${o.id}/seen`, { method: "POST", body: "{}" });
        onAcked();
      } catch (e) {
        setErr(errorText(e) || B.ackFailed);
      }
    }
    onOpen(o.id, o.number);
  }

  return (
    <div role="alert">
      <AdminCard title={B.attentionTitle} padded={false}>
        <ul className="admin-rows border-t border-line-soft">
          {newOrders.length > 0 && (
            <RowGroupTitle icon={IconBell}>
              {B.attentionNew} · {fmtNum(Math.max(newTotal, newOrders.length))}
            </RowGroupTitle>
          )}
          {newOrders.map((o) => (
            <Row
              key={o.id}
              leading={<RowIcon icon={IconBell} tone="attention" />}
              title={
                <span className="flex flex-wrap items-center gap-x-2">
                  <span dir="ltr" className="tabular-nums">#{o.number}</span>
                  <span className="min-w-0 truncate font-medium">{o.merchant_name}</span>
                </span>
              }
              meta={B.since.replace("{n}", fmtNum(minutesSince(o.created_at, now) ?? 0))}
              trailing={
                <Button size="sm" onClick={() => void openNew(o)}>
                  {B.open}
                </Button>
              }
            />
          ))}

          {alerts.length > 0 && (
            <RowGroupTitle icon={IconWarning}>
              {B.attentionStuck} · {fmtNum(alerts.length)}
            </RowGroupTitle>
          )}
          {alerts.map((a) => (
            <Row
              key={a.order_id + a.reason}
              leading={<RowIcon icon={IconWarning} tone="danger" />}
              title={
                <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
                  <span dir="ltr" className="tabular-nums">#{a.number}</span>
                  <span className="min-w-0 truncate font-medium">{a.merchant_name}</span>
                  <Badge variant="danger">{alertReasons[a.reason] ?? a.reason}</Badge>
                </span>
              }
              meta={
                <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
                  {/* **بلفظ المكتب لا الزبون** (المشكلة ٢٣). */}
                  <span>{officeStatus[a.status] ?? a.status}</span>
                  {(a.customer_phone || a.customer_phone_masked) && (
                    <span dir="ltr">{a.customer_phone || a.customer_phone_masked}</span>
                  )}
                  {/* **والدقائقُ تُعدّ في الشاشة من لحظة السبب** (المشكلة ١٩). */}
                  <span>{B.since.replace("{n}", fmtNum(minutesSince(a.since, now) ?? a.minutes))}</span>
                </span>
              }
              trailing={
                <Button size="sm" variant="secondary" onClick={() => onOpen(a.order_id, a.number)}>
                  {B.open}
                </Button>
              }
            />
          ))}

          {/* **وفشلُ جلب التنبيهات يُقال** (المشكلة ٣١) — لا يختفي القسمُ كأنّ شيئاً لم يعلق. */}
          {alertsFailed && (
            <Row
              leading={<RowIcon icon={IconWarning} tone="attention" />}
              title={<span className="font-medium">{B.alertsFailed}</span>}
              trailing={
                <Button size="sm" variant="secondary" onClick={onRetryAlerts}>
                  {B.retry}
                </Button>
              }
            />
          )}
        </ul>
        {err && <p className="border-t border-line-soft px-4 py-3 text-sm text-danger lg:px-5">{err}</p>}
      </AdminCard>
    </div>
  );
}

/** **لونُ آخر ظهور** — أخضرُ قريب، أصفرُ قديم، أحمرُ لم يظهر أو انقطع. */
function seenTone(mins: number | null, staleMin: number): "success" | "warning" | "danger" {
  if (mins === null) return "danger";
  if (mins <= 2) return "success";
  if (mins <= staleMin) return "warning";
  return "danger";
}

/**
 * **سطرُ السائق تحت ترويسة البطاقة** (البند ٤ — بدل الشارة في العمود الضيّق).
 *
 * اسمُه · زرُّ اتّصال · آخرُ ظهورٍ بلونه — **ويُحسب مع ساعة الشاشة لا لحظةَ
 * الرسم وحدَها**، فسائقٌ اختفى عشرين دقيقة يُرى أحمرَ ولو لم يقع حدث.
 */
export function DriverLine({
  name,
  phone,
  seenAt,
  now,
  staleMin,
  hideSeen = false,
}: {
  name: string;
  phone: string | null;
  seenAt: string | null | undefined;
  now: number;
  staleMin: number;
  /** **لا شارةَ ظهورٍ على طلبٍ انتهى** — «لم يظهر موقعه» عن طلبٍ سُلّم أمس يُقرأ عطلاً. */
  hideSeen?: boolean;
}) {
  const mins = minutesSince(seenAt, now);
  const tone = seenTone(mins, staleMin);
  return (
    <span
      className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm"
      onClick={(e) => e.stopPropagation()}
    >
      <span className="font-medium text-ink">{name}</span>
      {phone && (
        <a
          href={`tel:${phone}`}
          aria-label={m.admin.ordersPage.callDriver}
          title={m.admin.ordersPage.callDriver}
          className="inline-flex items-center gap-1 text-accent-text"
        >
          <IconPhone size={14} />
          <span dir="ltr" className="text-xs">
            {phone}
          </span>
        </a>
      )}
      {!hideSeen && (
      <Badge variant={tone}>
        {mins === null
          ? B.driverNever
          : mins === 0
            ? B.driverSeenNow
            : B.driverSeen.replace("{n}", fmtNum(mins))}
      </Badge>
      )}
    </span>
  );
}

/**
 * **لمن معروضٌ الطلبُ الآن** (المشكلة ٢٤) — والثواني تُعدّ في الشاشة.
 *
 * **وساعتُها ثانيةٌ لا نصفُ دقيقة** — عرضٌ مهلتُه ستّون ثانية.
 */
export function OfferLine({
  name,
  expiresAt,
  passed,
}: {
  name: string | null;
  expiresAt: string | null | undefined;
  passed: number;
}) {
  const now = useNow(1_000);
  const left = expiresAt ? Math.max(0, Math.ceil((Date.parse(expiresAt) - now) / 1000)) : 0;
  const parts: string[] = [];
  if (name && left > 0) {
    parts.push(B.offeredTo.replace("{name}", name));
    parts.push(B.offerLeft.replace("{n}", fmtNum(left)));
  }
  if (passed > 0) parts.push(B.offerPassed.replace("{n}", fmtNum(passed)));
  if (parts.length === 0) return null;
  return (
    <span className="text-xs text-ink-muted">{parts.join(m.common.listSeparator)}</span>
  );
}
