"use client";

/**
 * كشف حساب المحفظة — ورقة قابلة للطباعة، مركزية لكل من له محفظة.
 *
 * ليست جدولاً مزيّناً: هي **مستند مالي**. ولذلك ثلاثة قرارات تحكمها:
 *
 *  1. **يتوازن حسابياً**: الافتتاحي + مجموع الحركات = الختامي. من يراجع كشفاً
 *     يجمع أسطره؛ فإن لم تُغلق المعادلة فقد الثقة بالمنصة كلها.
 *  2. **يقول ما لا يعرضه**: إن قُصّت النتيجة عند السقف أُعلن ذلك في الورقة
 *     نفسها. كشفٌ ناقص يصمت عن نقصه أسوأ من لا كشف.
 *  3. **يُقرأ بلا شاشة**: عليه اسم صاحبه ورقمه والمدى وتاريخ الإصدار — الورقة
 *     تُطبع وتُسلَّم وتُراجَع بعد شهر، ولا أحد يتذكّر أي فلتر كان مفتوحاً.
 */

import { useMemo } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtDateTime } from "@rahalgo/i18n";
import { Button, Input } from "./components";
import { IconPrint } from "./icons";

const m = getMessages(defaultLocale);
const S = m.shared.statement;
const KIND_LABELS: Record<string, string> = m.shared.txKinds;

export interface StatementTx {
  id: number | string;
  kind: string;
  amount: number;
  note: string;
  order_number?: number | null;
  created_at: string;
}

export interface StatementData {
  opening: number;
  closing: number;
  truncated: boolean;
  transactions: StatementTx[];
}

/** يوم بصيغة YYYY-MM-DD بتوقيت المستخدم — لا ISO فيزحزحه UTC ليوم كامل. */
function ymd(d: Date): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

/** مدى الشهر الجاري — الافتراض الأكثر طلباً، فلا يُجبَر أحد على ملء حقلين. */
export function currentMonthRange(): { from: string; to: string } {
  const now = new Date();
  return { from: ymd(new Date(now.getFullYear(), now.getMonth(), 1)), to: ymd(now) };
}

export function StatementSheet({
  data,
  from,
  to,
  onFrom,
  onTo,
  onQuick,
  holderName,
  holderPhone,
  loading,
}: {
  data: StatementData | null;
  from: string;
  to: string;
  onFrom: (v: string) => void;
  onTo: (v: string) => void;
  /** اختصارات المدى — الشهر الجاري/الماضي */
  onQuick: (from: string, to: string) => void;
  holderName?: string;
  holderPhone?: string;
  loading?: boolean;
}) {
  const rows = data?.transactions ?? [];

  // من الأقدم إلى الأحدث: الكشف يُقرأ تصاعدياً كي يتراكم الرصيد الجاري أمام
  // القارئ سطراً بعد سطر — عكسُه يجعل عمود الرصيد بلا معنى.
  const ordered = useMemo(() => [...rows].reverse(), [rows]);

  const running: number[] = [];
  let acc = data?.opening ?? 0;
  for (const t of ordered) {
    acc += t.amount;
    running.push(acc);
  }

  const credits = ordered.reduce((s, t) => (t.amount > 0 ? s + t.amount : s), 0);
  const debits = ordered.reduce((s, t) => (t.amount < 0 ? s + t.amount : s), 0);

  const thisMonth = currentMonthRange();
  const lastMonthStart = new Date();
  lastMonthStart.setDate(1);
  lastMonthStart.setMonth(lastMonthStart.getMonth() - 1);
  const lastMonthEnd = new Date(lastMonthStart.getFullYear(), lastMonthStart.getMonth() + 1, 0);

  return (
    <div className="space-y-4">
      {/* أدوات المدى — لا تُطبع: الورقة تحمل المدى نصّاً لا حقولاً */}
      <div className="no-print flex flex-wrap items-end gap-3 rounded-card border border-line bg-surface p-4">
        <Input
          id="st-from"
          label={S.from}
          type="date"
          dir="ltr"
          value={from}
          onChange={(e) => onFrom(e.target.value)}
          className="w-44"
        />
        <Input
          id="st-to"
          label={S.to}
          type="date"
          dir="ltr"
          value={to}
          onChange={(e) => onTo(e.target.value)}
          className="w-44"
        />
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" onClick={() => onQuick(thisMonth.from, thisMonth.to)}>
            {S.thisMonth}
          </Button>
          <Button
            variant="secondary"
            onClick={() => onQuick(ymd(lastMonthStart), ymd(lastMonthEnd))}
          >
            {S.lastMonth}
          </Button>
        </div>
        <Button
          onClick={() => window.print()}
          disabled={loading || !rows.length}
          className="ms-auto flex items-center gap-2"
        >
          <IconPrint size={16} />
          {S.print}
        </Button>
      </div>

      {/* الورقة */}
      <div
        data-print="sheet"
        className="rounded-card border border-line bg-surface p-6 text-sm"
      >
        <div className="mb-5 flex flex-wrap items-start justify-between gap-4 border-b border-line pb-4">
          <div>
            <p className="text-lg font-bold">{S.title}</p>
            <p className="mt-1 text-ink-muted">
              {S.period} <span dir="ltr">{fmtDate(from)}</span> — <span dir="ltr">{fmtDate(to)}</span>
            </p>
          </div>
          <div className="text-end">
            <p className="font-bold">{m.common.appName}</p>
            {holderName && <p className="text-ink-muted">{holderName}</p>}
            {holderPhone && (
              <p className="text-ink-muted" dir="ltr">
                {holderPhone}
              </p>
            )}
            <p className="mt-1 text-xs text-ink-muted">
              {S.issuedAt} <span dir="ltr">{fmtDateTime(new Date())}</span>
            </p>
          </div>
        </div>

        {loading ? (
          <p className="py-8 text-center text-ink-muted">{m.common.loading}</p>
        ) : !rows.length ? (
          <p className="py-8 text-center text-ink-muted">{S.empty}</p>
        ) : (
          <>
            {data?.truncated && (
              <p className="mb-3 rounded-control bg-warning/10 px-3 py-2 text-xs text-warning">
                {S.truncated}
              </p>
            )}

            <div className="overflow-x-auto">
              <table className="w-full border-collapse text-start">
                <thead>
                  <tr className="border-b border-line text-xs text-ink-muted">
                    <th className="py-2 text-start font-medium">{S.colDate}</th>
                    <th className="py-2 text-start font-medium">{S.colKind}</th>
                    <th className="py-2 text-start font-medium">{S.colNote}</th>
                    <th className="py-2 text-end font-medium">{S.colAmount}</th>
                    <th className="py-2 text-end font-medium">{S.colRunning}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-b border-line font-medium">
                    <td className="py-2" dir="ltr">
                      {fmtDate(from)}
                    </td>
                    <td className="py-2" colSpan={2}>
                      {S.opening}
                    </td>
                    <td className="py-2" />
                    <td className="py-2 text-end" dir="ltr">
                      {fmtNum(data?.opening ?? 0)}
                    </td>
                  </tr>

                  {ordered.map((t, i) => (
                    <tr key={t.id} className="border-b border-line/60">
                      <td className="py-2 align-top" dir="ltr">
                        {fmtDate(t.created_at)}
                      </td>
                      <td className="py-2 align-top">{KIND_LABELS[t.kind] ?? t.kind}</td>
                      <td className="py-2 align-top text-ink-muted">
                        {t.note}
                        {!!t.order_number && (
                          <span className="ms-1" dir="ltr">
                            #{fmtNum(t.order_number)}
                          </span>
                        )}
                      </td>
                      <td
                        className={`py-2 text-end align-top font-medium ${
                          t.amount >= 0 ? "text-success" : "text-danger"
                        }`}
                        dir="ltr"
                      >
                        {t.amount >= 0 ? "+" : ""}
                        {fmtNum(t.amount)}
                      </td>
                      <td className="py-2 text-end align-top" dir="ltr">
                        {fmtNum(running[i] ?? 0)}
                      </td>
                    </tr>
                  ))}

                  <tr className="border-t-2 border-line font-bold">
                    <td className="py-2" dir="ltr">
                      {fmtDate(to)}
                    </td>
                    <td className="py-2" colSpan={2}>
                      {S.closing}
                    </td>
                    <td className="py-2" />
                    <td className="py-2 text-end" dir="ltr">
                      {fmtNum(data?.closing ?? 0)}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div className="mt-4 flex flex-wrap justify-end gap-x-8 gap-y-1 border-t border-line pt-3 text-sm">
              <span>
                {S.totalIn}{" "}
                <span className="font-bold text-success" dir="ltr">
                  +{fmtNum(credits)}
                </span>
              </span>
              <span>
                {S.totalOut}{" "}
                <span className="font-bold text-danger" dir="ltr">
                  {fmtNum(debits)}
                </span>
              </span>
              <span>
                {S.net}{" "}
                <span className="font-bold" dir="ltr">
                  {fmtNum(credits + debits)} {m.common.currency}
                </span>
              </span>
            </div>

            <p className="mt-4 border-t border-line pt-3 text-xs leading-relaxed text-ink-muted">
              {S.footer}
            </p>
          </>
        )}
      </div>
    </div>
  );
}
