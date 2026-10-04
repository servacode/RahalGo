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
  Alert,
  Badge,
  Button,
  IconBell,
  IconBellOff,
  IconPhone,
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
 * **عدّاداتُ المراحل** — كلٌّ يُضغط فيفتح القائمةَ بالشرط الذي عدّه بعينه.
 *
 * **والضغطةُ الثانيةُ لا تُطفئ** — زرُّ «عرض الكلّ» بجانب البحث يقولها صراحةً.
 */
export function BoardCounters({
  counts,
  active,
  onPick,
  failed,
  onRetry,
}: {
  counts: Record<string, number> | null;
  active: string;
  onPick: (filter: string) => void;
  failed: boolean;
  onRetry: () => void;
}) {
  if (failed && !counts) {
    return (
      <Alert className="mb-4" tone="warning">
        <span className="flex flex-wrap items-center gap-2">
          {B.countersFailed}
          <Button variant="secondary" onClick={onRetry}>
            {B.retry}
          </Button>
        </span>
      </Alert>
    );
  }
  return (
    <div
      role="group"
      aria-label={B.countersLabel}
      className="mb-4 grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6"
    >
      {COUNTER_FILTERS.map((k) => {
        const on = active === k;
        const n = counts?.[k];
        const hot = (k === "stuck" || k === "reports") && (n ?? 0) > 0;
        return (
          <button
            key={k}
            type="button"
            aria-pressed={on}
            onClick={() => onPick(k)}
            className={`rounded-card border p-3 text-start transition-colors ${
              on
                ? "border-accent bg-accent-tint"
                : hot
                  ? "border-danger-edge bg-danger-tint"
                  : "border-line-soft bg-surface hover:bg-field"
            }`}
          >
            <span className="block text-xs text-ink-muted">
              {(B.counters as Record<string, string>)[k]}
            </span>
            <span
              className={`figure block ${
                on ? "text-accent-text" : hot ? "text-danger" : "text-ink"
              }`}
            >
              {n === undefined ? "—" : fmtNum(n)}
            </span>
          </button>
        );
      })}
    </div>
  );
}

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
      <Button variant="primary" onClick={onEnable} title={B.soundHint}>
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

/**
 * **شريطُ الطلبات الجديدة — يرنّ حتّى «استلمتها»** (البند ٥).
 *
 * **والضغطةُ تُكتب في الطلب** (`POST /orders/{id}/seen`) — فيسكت الرنينُ عند
 * المكتب كلِّه لا عند من ضغط وحدَه.
 */
export function NewOrdersBar({
  orders,
  now,
  canAck,
  onAcked,
  onOpen,
}: {
  orders: NewOrder[];
  now: number;
  canAck: boolean;
  onAcked: () => void;
  onOpen: (o: NewOrder) => void;
}) {
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  if (orders.length === 0) return null;

  async function ack(ids: string[], key: string) {
    setBusy(key);
    setErr("");
    try {
      for (const id of ids) {
        await api(`/api/v1/admin/orders/${id}/seen`, { method: "POST", body: "{}" });
      }
      onAcked();
    } catch (e) {
      setErr(errorText(e) || B.ackFailed);
    } finally {
      setBusy("");
    }
  }

  return (
    <div
      role="alert"
      className="mb-4 rounded-card border-2 border-accent bg-accent-tint p-3 text-sm"
    >
      <p className="mb-2 flex items-center gap-2 font-bold text-accent-text">
        <span className="h-2.5 w-2.5 animate-pulse rounded-badge bg-accent" />
        {B.newTitle} ({fmtNum(orders.length)})
      </p>
      <ul className="space-y-1.5">
        {orders.map((o) => (
          <li key={o.id} className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => onOpen(o)}
              className="font-bold text-accent-text underline-offset-2 hover:underline"
            >
              #{o.number}
            </button>
            <span className="min-w-0 flex-1 truncate text-ink">{o.merchant_name}</span>
            <span className="text-xs text-ink-muted">
              {B.since.replace("{n}", fmtNum(minutesSince(o.created_at, now) ?? 0))}
            </span>
            {canAck && (
              <Button
                variant="primary"
                disabled={busy !== ""}
                onClick={() => void ack([o.id], o.id)}
              >
                {B.newAck}
              </Button>
            )}
          </li>
        ))}
      </ul>
      {err && <p className="mt-2 text-xs text-danger">{err}</p>}
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
