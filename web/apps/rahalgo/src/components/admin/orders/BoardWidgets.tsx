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
  IconWhatsApp,
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
  alerts,
  alertsFailed,
  onRetryAlerts,
  now,
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
  // **ولا قسمَ «جديدة»** (قرارُ المالك ٢٠٢٦-١١-٠٣: «هي مالا لازم اصلا») — الجديدُ ظاهرٌ بكرته وعدّاده،
  // **والرنينُ يسكت أوّلَ ما ينقبل أو ينرفض.**
  if (alerts.length === 0 && !alertsFailed) return null;

  return (
    <section className="surface mb-4 overflow-hidden" role="alert">
      <h2 className="heading-card flex items-center gap-2 border-b border-line-soft px-4 py-3">
        <span className="h-2.5 w-2.5 animate-pulse rounded-badge bg-danger" />
        {B.attentionTitle}
      </h2>

      {alerts.length > 0 && (
        <div className="bg-danger-tint px-4 py-3">
          <p className="mb-2 flex flex-wrap items-center gap-2 text-sm font-bold text-danger">
            {B.attentionStuck}
            <Badge variant="danger">{fmtNum(alerts.length)}</Badge>
          </p>
          <ul className="space-y-2">
            {alerts.map((a) => (
              <li key={a.order_id + a.reason} className="flex flex-wrap items-center gap-2 text-sm">
                <span className="font-bold text-danger">#{a.number}</span>
                <Badge variant="danger">{alertReasons[a.reason] ?? a.reason}</Badge>
                <span className="min-w-0 truncate">{a.merchant_name}</span>
                {(a.customer_phone || a.customer_phone_masked) && (
                  <span dir="ltr" className="text-xs text-ink-muted">
                    {a.customer_phone || a.customer_phone_masked}
                  </span>
                )}
                {/* **والدقائقُ تُعدّ في الشاشة من لحظة السبب** (المشكلة ١٩). */}
                <span className="text-xs text-ink-muted">
                  {B.since.replace("{n}", fmtNum(minutesSince(a.since, now) ?? a.minutes))}
                </span>
                {/* **بلفظ المكتب لا الزبون** (المشكلة ٢٣). */}
                <Badge variant="warning">{officeStatus[a.status] ?? a.status}</Badge>
                <Button variant="secondary" className="ms-auto" onClick={() => onOpen(a.order_id, a.number)}>
                  {B.open}
                </Button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* **وفشلُ جلب التنبيهات يُقال** (المشكلة ٣١) — لا يختفي القسمُ كأنّ شيئاً لم يعلق. */}
      {alertsFailed && (
        <div className="flex flex-wrap items-center gap-2 px-4 py-3 text-sm text-warning">
          {B.alertsFailed}
          <Button variant="secondary" onClick={onRetryAlerts}>
            {B.retry}
          </Button>
        </div>
      )}
    </section>
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
        <span dir="ltr" className="text-xs text-ink-muted">
          {phone}
        </span>
      )}
      <ContactLinks phone={phone} callLabel={m.admin.ordersPage.callDriver} />
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

/** **رقمٌ للواتساب** — أرقامُه وحدها بلا «+» (`wa.me`). */
function waDigits(phone: string): string {
  return phone.replace(/[^0-9]/g, "");
}

/**
 * **واتساب ويب مباشرةً** (قرارُ المالك ٢٠٢٦-١٠-٠٦) — `wa.me` يفتح صفحةَ واتساب الرسميّة
 * «التطبيق أم الويب» أوّلاً، **و`web.whatsapp.com/send` يفتح المحادثةَ والرسالةُ جاهزة.**
 * (نُقلت من `OrdersScreen` ليستعملها زرُّ إرسال كلّ متجرٍ في الطلب المشترك أيضاً.)
 */
export function toWhatsAppWeb(link: string): string {
  try {
    const u = new URL(link);
    if (u.hostname !== "wa.me" && u.hostname !== "api.whatsapp.com") return link;
    const phone = u.hostname === "wa.me" ? u.pathname.replace(/\//g, "") : (u.searchParams.get("phone") ?? "");
    const out = new URL("https://web.whatsapp.com/send");
    out.searchParams.set("phone", phone);
    const text = u.searchParams.get("text");
    if (text) out.searchParams.set("text", text);
    return out.toString();
  } catch {
    return link;
  }
}

/**
 * **زرّا اتّصالٍ وواتساب جنب كلّ طرف** (قرارُ المالك ٢٠٢٦-١١-٠٣) — المتجرُ والزبونُ والسائق
 * على البطاقة نفسِها، بلا فتحِ لوحةٍ أخرى.
 */
export function ContactLinks({ phone, callLabel }: { phone: string | null | undefined; callLabel: string }) {
  if (!phone) return null;
  const P = m.admin.ordersPage;
  const cls =
    "inline-flex h-8 w-8 items-center justify-center rounded-control border border-line-soft bg-surface text-accent-text hover:bg-row-hover";
  return (
    <span className="flex shrink-0 items-center gap-1" onClick={(e) => e.stopPropagation()}>
      <a href={`tel:${phone}`} aria-label={callLabel} title={callLabel} className={cls}>
        <IconPhone size={15} />
      </a>
      <a
        href={`https://wa.me/${waDigits(phone)}`}
        target="_blank"
        rel="noreferrer"
        aria-label={P.waChat}
        title={P.waChat}
        className={cls}
      >
        <IconWhatsApp size={15} />
      </a>
    </span>
  );
}

/**
 * **عمرُ الطلب بالرأس** (قرارُ المالك ٢٠٢٦-١١-٠٣) — «من ١٢ د»، **وأحمرُ إن تأخّر** (حكمُ المحرّك على
 * المرحلة الحاليّة) وأصفرُ إن فات نصفَ مهلة التوصيل.
 */
export function AgeBadge({
  createdAt,
  now,
  late,
  warnAfterMin,
}: {
  createdAt: string;
  now: number;
  late: boolean;
  warnAfterMin: number;
}) {
  const mins = minutesSince(createdAt, now) ?? 0;
  const P = m.admin.ordersPage;
  const text =
    mins >= 60
      ? P.ageHour.replace("{h}", fmtNum(Math.floor(mins / 60))).replace("{n}", fmtNum(mins % 60))
      : P.ageMin.replace("{n}", fmtNum(mins));
  const tone = late ? "danger" : warnAfterMin > 0 && mins >= warnAfterMin ? "warning" : "neutral";
  return <Badge variant={tone}>{text}</Badge>;
}
