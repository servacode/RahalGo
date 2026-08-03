"use client";

/**
 * **«طلبات قادمة» — ما لم يُسنَد إلى سائقٍ بعد.**
 *
 * # ولماذا قسمٌ قائمٌ بذاته
 *
 * كان تحت «مهامّي» في الصفحة نفسها، **فيقرؤه السائقُ بعد أن يمرّ على مهامّه**
 * — وهو أوّلُ ما يحتاجه لا آخرُه. **وقرارُ الأخذ يُتّخذ في ثوانٍ**، ومن نزل
 * بإصبعه ليراه فاته.
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «اجعله قسماً منفصلاً لوحده فقط مختصّاً بالطلبات
 * القادمة · **مجرّد البدء يتحوّل إلى مهامّي** · قسم الطلبات يعرض كلّ الطلبات
 * التي لم تُسنَد إلى سائقٍ بعد».
 *
 * # والحدُّ بين القسمين حدُّ ملكيةٍ لا حدُّ عرض
 *
 *	طلبات قادمة  ←  بلا سائق — **قرارٌ يُتّخذ**
 *	مهامّي        ←  بيدي — **عملٌ يُنفَّذ**
 *
 * فلا يظهر طلبٌ في القسمين معاً، **ولا يختفي من أحدهما بلا أن يظهر في الآخر.**
 */

import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Card,
  Badge,
  Button,
  IconStore,
  IconLocation,
  IconBalance,
} from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const D = m.driver;

/** طلبٌ كما يراه السائق — الحقولُ نفسُها في القسمين. */
export interface DriverOrder {
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
  /** نقطةُ استلامٍ بديلة — البضاعةُ مع سائقٍ سابقٍ وقع له طارئ، لا في المتجر. */
  pickup_lat?: number | null;
  pickup_lng?: number | null;
  pickup_note?: string;
  items_count: number;
  ready_at: string | null;
  prep_minutes: number | null;
  accepted_at: string | null;
  created_at: string;
}

/** بطاقةُ الطلب القادم — أقلّ ممّا في المهمّة: قرارُ الأخذ لا يحتاج رقمَ هاتف. */
export function IncomingCard({
  o,
  busy,
  onAccept,
}: {
  o: DriverOrder;
  busy: boolean;
  onAccept: () => void;
}) {
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
      {/* **والبضاعةُ قد لا تكون في المتجر** — طارئٌ وقع لسائقٍ قبله وهي في يده.
          فيُقال له قبل أن ينطلق، **لا بعد أن يقف أمام مطبخٍ سلّم.** */}
      {o.pickup_note ? (
        <p className="mt-1 rounded-control bg-warning/10 px-2 py-1 text-xs text-warning">
          {o.pickup_note}
        </p>
      ) : null}
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
