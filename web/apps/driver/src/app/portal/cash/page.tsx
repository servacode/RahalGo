"use client";

/**
 * **كشفُ صندوقي — مالٌ في ذمّتي أقرؤه.**
 *
 * كان السائقُ يرى المجموعَ وحدَه في رأس لوحته: `123,000 / 500,000`. **ولا يرى
 * تفصيلَه**: من أيّ طلبٍ قبض، ومتى سلّم، وكم بقي.
 *
 * **ومالٌ في ذمّة إنسانٍ بلا كشفٍ يقرؤه خلافٌ ينتظر**: يقول «سلّمتُ» وتقول
 * المنصةُ «لم يصل»، **ولا ورقةَ بينهما.**
 *
 * # وإعادةُ الهيكلة
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «أعد هيكلة الصفحة بطريقةٍ واضحةٍ مفهومة».)
 *
 * **الصفُّ كان يقول ثلاثة أشياءٍ بلا ترتيب**: نوعاً إنكليزيّاً خامّاً، ثمّ
 * ملاحظةً تقول ما يقوله النوعُ بالعربيّة، ثمّ تاريخاً — **والمبلغُ يطفو
 * وحدَه في الطرف بلا ما يربطه بها.**
 *
 * **وصار الصفُّ جملةً**: سهمٌ يقول الاتّجاه · ما وقع · على أيّ طلب · بكم ·
 * ومتى. **والاتّجاهُ يُقرأ قبل الرقم** — قبضٌ يزيد ذمّتَك، وتسليمٌ يُنقصها.
 *
 * **والحركاتُ مجموعةٌ بالأيّام**: من سلّم صندوقَه مساءً يريد أن يرى «اليوم»
 * وحدَه، **وكشفٌ متّصلٌ من مئة سطرٍ لا يُراجَع.**
 *
 * # والسقفُ يُقال بما يبقى لا بما مضى
 *
 * **كان `204,000 / 5,000,000`** — كسرٌ يُحسب في الرأس. **والسائقُ يسأل سؤالاً
 * واحداً: كم أقدر أن أقبض بعد؟** فيُقال له بالرقم.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtTime, fmtLongDate } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  PageContainer,
  PageHeader,
  LoadingState,
  EmptyState,
  useLiveRefresh,
  IconBalance,
  IconOrder,
  IconArrowIn,
  IconArrowOut,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.driver.cashbox;

interface Entry {
  kind: string;
  amount: number;
  note: string;
  order_number: number | null;
  created_at: string;
}

interface Me {
  cash_held: number;
  cash_limit: number;
}

/** يومُ الحركة — **و«اليوم» أوضحُ من تاريخٍ كامل** لمن يراجع وردِيّتَه. */
function dayLabel(iso: string): string {
  const d = new Date(iso);
  const today = new Date();
  const diff = Math.floor(
    (new Date(today.toDateString()).getTime() - new Date(d.toDateString()).getTime()) / 86_400_000,
  );
  if (diff <= 0) return m.shared.notifications.today;
  if (diff === 1) return m.shared.notifications.yesterday;
  return fmtLongDate(d);
}

export default function CashPage() {
  const [rows, setRows] = useState<Entry[] | null>(null);
  const [me, setMe] = useState<Me | null>(null);
  /**
   * **وفشلُ القراءة ليس «لا نقدَ بذمّتك».**
   *
   * **وهي عائلةُ «لم أصل غير لا شيء» نفسُها** — **وهذه أخطرُ مواضعها لأنّها
   * مال.**
   */
  const [failed, setFailed] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    Promise.all([
      api<Me>("/api/v1/driver/me"),
      api<Entry[] | { entries: Entry[] }>("/api/v1/driver/cash"),
    ])
      .then(([m2, r]) => {
        setMe(m2);
        setRows(Array.isArray(r) ? r : (r?.entries ?? []));
      })
      .catch(() => setFailed(true));
  }, []);

  useEffect(load, [load]);
  // **وتسويةُ المالية تصل بلا تحديثِ صفحة** — من سلّم صندوقَه يريد أن يراه صفراً.
  useLiveRefresh(["wallet", "order"], load);

  const groups = useMemo(() => {
    const out: { day: string; rows: Entry[] }[] = [];
    /* **ولا يُفترض ترتيبٌ لم يُطلب** — التجميعُ بالجوار يفترض أنّ الأحدثَ
       أوّلاً. **وردٌّ غيرُ مرتّبٍ يُنتج يوماً يتكرّر مرّتين في الكشف**،
       فيُقرأ عطباً في المال. فيُرتَّب هنا ولا يُنتظر. */
    const sorted = [...(rows ?? [])].sort((a, b) => b.created_at.localeCompare(a.created_at));
    for (const e of sorted) {
      const day = dayLabel(e.created_at);
      const last = out[out.length - 1];
      if (last && last.day === day) last.rows.push(e);
      else out.push({ day, rows: [e] });
    }
    return out;
  }, [rows]);

  // **والخطأُ يُقال ويُعاد المحاولة** — لا يُترك على صمت.
  if (failed) {
    return (
      <PageContainer>
        <PageHeader icon={IconBalance} title={C.title} />
        <Alert tone="warning" title={m.errors.offline}>
          {m.errors.offlineHint}
        </Alert>
        <Button variant="secondary" onClick={load}>
          {m.common.retry}
        </Button>
      </PageContainer>
    );
  }
  if (!rows || !me) return <LoadingState />;

  const ratio = me.cash_limit > 0 ? me.cash_held / me.cash_limit : 0;
  const left = me.cash_limit - me.cash_held;

  return (
    <PageContainer>
      <PageHeader icon={IconBalance} title={C.title} subtitle={C.hint} />

      {/* ══════════════════════════════════════════════════════════════
          **ما في ذمّتي — وكم أقدر أن أقبض بعد**
          ══════════════════════════════════════════════════════════════

          **والسقفُ ليس زينة**: من بلغه لا يُعرض عليه طلبٌ نقديٌّ جديد، **فيقف
          عملُه ولا يعرف لماذا.** */}
      <section className="surface p-5">
        <p className="text-sm text-ink-muted">{C.held}</p>
        <p dir="ltr" className="figure mt-1 text-warning">
          {fmtNum(me.cash_held)}{" "}
          <span className="text-sm font-normal text-ink-muted">{m.common.currency}</span>
        </p>

        <div className="mt-4 h-2 overflow-hidden rounded-badge bg-field">
          <div
            className={`h-full rounded-badge transition-[width] ${
              ratio > 0.8 ? "bg-danger" : ratio > 0.5 ? "bg-warning" : "bg-success"
            }`}
            style={{ width: `${Math.min(100, ratio * 100)}%` }}
          />
        </div>

        {/* **ويُقال ما يبقى لا ما مضى** — «يبقى لك ٤٫٧٩٦٫٠٠٠» جوابُ سؤاله،
            **و«٢٠٤٬٠٠٠ / ٥٬٠٠٠٬٠٠٠» كسرٌ يُحسب في الرأس.** */}
        <div className="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs">
          <span className="text-ink-muted">
            {C.limit}{" "}
            <span dir="ltr" className="tabular-nums">
              {fmtNum(me.cash_limit)}
            </span>
          </span>
          <span className={left > 0 ? "font-medium text-success" : "font-bold text-danger"}>
            {left > 0 ? C.remaining : C.over}{" "}
            <span dir="ltr" className="tabular-nums">
              {fmtNum(Math.abs(left))}
            </span>
          </span>
        </div>

        {/* **والتحذيرُ عند الحدّ لا بعده** — من بلغ السقفَ وقف طابورُه. */}
        {ratio >= 1 ? (
          <Alert tone="error" className="mt-4">
            {C.full}
          </Alert>
        ) : ratio > 0.8 ? (
          <Alert tone="warning" className="mt-4">
            {C.near}
          </Alert>
        ) : null}
      </section>

      {rows.length === 0 ? (
        <EmptyState icon={IconBalance} title={C.empty} />
      ) : (
        groups.map((g) => (
          <section key={g.day}>
            <h2 className="mb-2 text-xs font-bold text-ink-muted">{g.day}</h2>
            <ul className="surface divide-y divide-line-soft">
              {g.rows.map((e, i) => {
                // **الموجبُ قبضٌ يزيد الذمّة، والسالبُ تسليمٌ يُنقصها.**
                const inbound = e.amount > 0;
                const label = (C.kinds as Record<string, string>)[e.kind];
                return (
                  <li key={i} className="flex items-center gap-3 px-3 py-3">
                    {/* **والاتّجاهُ يُقرأ قبل الرقم** — سهمٌ يقول أزاد أم نقص. */}
                    <span
                      className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-control ${
                        inbound ? "bg-warning-tint text-warning" : "bg-success-tint text-success"
                      }`}
                    >
                      {inbound ? <IconArrowIn size={17} /> : <IconArrowOut size={17} />}
                    </span>

                    <span className="min-w-0 flex-1">
                      {/* **والاسمُ عربيٌّ دائماً** — ورمزٌ لا اسمَ له تقوله
                          ملاحظتُه، **ولا يُعرض رمزٌ إنكليزيٌّ على سائق.** */}
                      <span className="block truncate text-sm font-medium">
                        {label ?? e.note ?? ""}
                      </span>
                      <span className="mt-0.5 flex flex-wrap items-center gap-x-2 text-xs text-ink-muted">
                        {e.order_number != null && (
                          <span dir="ltr" className="inline-flex items-center gap-1 tabular-nums">
                            <IconOrder size={12} />#{fmtRef(e.order_number)}
                          </span>
                        )}
                        {/* **والملاحظةُ لا تُكرّر الاسم** — تظهر إن قالت زيادة. */}
                        {e.note && e.note !== label && <span className="truncate">{e.note}</span>}
                        <span dir="ltr" className="tabular-nums">
                          {fmtTime(e.created_at)}
                        </span>
                      </span>
                    </span>

                    <span
                      dir="ltr"
                      className={`shrink-0 text-base font-bold tabular-nums ${
                        inbound ? "text-warning" : "text-success"
                      }`}
                    >
                      {inbound ? "+" : ""}
                      {fmtNum(e.amount)}
                    </span>
                  </li>
                );
              })}
            </ul>
          </section>
        ))
      )}
    </PageContainer>
  );
}
