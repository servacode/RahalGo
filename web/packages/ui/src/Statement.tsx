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
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDate, withPlatform } from "@rahalgo/i18n";
import { Button, Input } from "./components";
import { Alert } from "./feedback";
import { SheetHeader, LoadingState } from "./layout";
import { usePlatform } from "./platform";
import { IconPrint, IconPrev } from "./icons";

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
  onBack,
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
  /**
   * **بابُ الرجوع** — وفارغٌ يعني لا زرّ.
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «عند الدخول على كشف حساب لا يوجد زرُّ رجوعٍ
   *  للمحفظة».)
   *
   * **وكان الرجوعُ بالزرّ الذي جاء منه** — يُضغط ثانيةً فيُغلق. **ومن دخل
   * باباً يبحث عن بابٍ يخرج منه، لا عن الذي دخل منه.**
   */
  onBack?: () => void;
}) {
  /* **واسمُ المنصة من الإعدادات** — كالفاتورة، والكشفُ يُسلَّم كذلك. */
  const { name: platformName, supportPhone: platformSupport } = usePlatform();
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
      <div className="no-print flex flex-wrap items-end gap-3 surface p-4">
        {onBack && (
          <Button variant="ghost" onClick={onBack} className="flex items-center gap-1.5">
            <IconPrev size={16} />
            {m.common.back}
          </Button>
        )}
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
        className="surface p-6 text-sm"
      >
        {/* **الترويسةُ نفسها** — الفاتورةُ والكشفُ ورقتان من دارٍ واحدة. */}
        <SheetHeader />

        {/* وسطرُ التعريف: ما هذه الورقة، ولمن، وعن أيّ مدى. */}
        <div className="mb-4 grid grid-cols-1 gap-x-6 gap-y-1 border-b border-line-soft pb-3 text-xs sm:grid-cols-2">
          <p className="text-base font-bold">{S.title}</p>
          <p className="sm:text-end">
            <span className="text-ink-muted">{S.period} </span>
            <span dir="ltr">{fmtDate(from)}</span> — <span dir="ltr">{fmtDate(to)}</span>
          </p>
          {holderName && (
            <p>
              <span className="font-medium">{holderName}</span>
              {holderPhone && (
                <span dir="ltr" className="ms-2 text-ink-muted">
                  {holderPhone}
                </span>
              )}
            </p>
          )}
        </div>

        {loading ? (
          <LoadingState variant="text" />
        ) : !rows.length ? (
          <p className="py-8 text-center text-ink-muted">{S.empty}</p>
        ) : (
          <>
            {data?.truncated && (
              <Alert tone="warning" className="mb-3">
                {S.truncated}
              </Alert>
            )}

            <div className="overflow-x-auto">
              <table className="w-full border-collapse text-start">
                <thead>
                  <tr className="border-b-2 border-line-soft text-2xs uppercase tracking-wide text-ink-muted">
                    <th className="py-2 text-start font-medium">{S.colDate}</th>
                    <th className="py-2 text-start font-bold">{S.colKind}</th>
                    <th className="py-2 text-start font-bold">{S.colNote}</th>
                    <th className="w-28 py-2 text-end font-bold">{S.colAmount}</th>
                    <th className="w-28 py-2 text-end font-bold">{S.colRunning}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-b border-line-soft font-medium">
                    <td className="py-2" dir="ltr">
                      {fmtDate(from)}
                    </td>
                    <td className="py-2" colSpan={2}>
                      {S.opening}
                    </td>
                    <td className="py-2" />
                    {/* **و`dir` على الرقم لا على الخانة** — «النهاية» تتبع
                        الاتّجاه، فخانةٌ `ltr` تحاذي يميناً ورأسُها يحاذي
                        يساراً. (شكوى المالك ٢٠٢٦-٠٨-٠٩ على الفاتورة، وهذا
                        الجدولُ أخوها.) */}
                    <td className="py-2 text-end tabular-nums">
                      <span dir="ltr">{fmtNum(data?.opening ?? 0)}</span>
                    </td>
                  </tr>

                  {ordered.map((t, i) => (
                    <tr key={t.id} className="border-b border-line-soft">
                      <td className="py-2 align-top" dir="ltr">
                        {fmtDate(t.created_at)}
                      </td>
                      <td className="py-2 align-top">{KIND_LABELS[t.kind] ?? t.kind}</td>
                      <td className="py-2 align-top text-ink-muted">
                        {t.note}
                        {!!t.order_number && (
                          <span className="ms-1" dir="ltr">
                            #{fmtRef(t.order_number)}
                          </span>
                        )}
                      </td>
                      <td
                        className={`py-2 text-end align-top font-bold tabular-nums ${
                          t.amount >= 0 ? "text-success" : "text-danger"
                        }`}
                      >
                        <span dir="ltr">
                          {t.amount >= 0 ? "+" : ""}
                          {fmtNum(t.amount)}
                        </span>
                      </td>
                      <td className="py-2 text-end align-top tabular-nums text-ink-muted">
                        <span dir="ltr">{fmtNum(running[i] ?? 0)}</span>
                      </td>
                    </tr>
                  ))}

                  <tr className="border-t-2 border-line-soft font-bold">
                    <td className="py-2" dir="ltr">
                      {fmtDate(to)}
                    </td>
                    <td className="py-2" colSpan={2}>
                      {S.closing}
                    </td>
                    <td className="py-2" />
                    <td className="py-2 text-end tabular-nums">
                      <span dir="ltr">{fmtNum(data?.closing ?? 0)}</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div className="mt-4 flex flex-wrap justify-end gap-x-8 gap-y-1 border-t border-line-soft pt-3 text-sm">
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

            {/* **وبابُ الاستفسار على الورق** — كالفاتورة، **والكشفُ يُطوى
                ويُراجَع بعد أيّام.** وفارغٌ يُخفي السطر. */}
            {platformSupport && (
              <p className="mt-3 text-center text-xs font-medium">
                {S.support}{" "}
                <span dir="ltr" className="tabular-nums">
                  {platformSupport}
                </span>
              </p>
            )}

            <p className="mt-4 border-t border-line-soft pt-3 text-xs leading-relaxed text-ink-muted">
              {withPlatform(S.footer, platformName)}
            </p>
          </>
        )}
      </div>
    </div>
  );
}
