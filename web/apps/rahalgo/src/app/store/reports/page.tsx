"use client";

/** تقارير المتجر: ملخص بمدى زمني + أعمدة يومية (منهجية dataviz الموحدة). */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Alert, IconStatus, IconOrder, IconSuccess, IconError, IconWallet, Input,
  Money,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useStore } from "@/lib/store";

const m = getMessages(defaultLocale);

interface Report {
  summary: {
    orders: number;
    delivered: number;
    cancelled: number;
    /** ما خرج من يده — **وهو ما قبض عليه**. */
    sold: number;
    /** ما عاد إليه فرُدّ ثمنُه. */
    returned: number;
    sales: number;
    platform_commission: number;
  };
  days: { date: string; orders: number; delivered: number; sales: number }[];
  /** تفصيلُ الأصناف — **«ماذا بعتُ؟» لا «كم بعتُ؟»**. */
  /** **ثلاثةُ أرقامٍ لكلّ صنف** — الأساسيُّ والعمولةُ وما بعدها. */
  items: { name: string; qty: number; revenue: number; commission: number; net: number }[];
}

function isoDaysAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d.toISOString().slice(0, 10);
}

export default function MerchantReportsPage() {
  const { store } = useStore();
  const [from, setFrom] = useState(isoDaysAgo(6));
  const [to, setTo] = useState(isoDaysAgo(0));
  const [report, setReport] = useState<Report | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    if (!store) return;
    try {
      setReport(
        await api<Report>(`/api/v1/merchant/stores/${store.id}/reports?from=${from}&to=${to}`)
      );
      setError("");
    } catch (err) {
      setError(err instanceof ApiError ? m.errors.internal : m.errors.internal);
    }
  }, [store, from, to]);

  useEffect(() => {
    void load();
  }, [load]);

  const s = report?.summary;
  const net = s ? s.sales - s.platform_commission : 0;
  const maxSales = Math.max(1, ...(report?.days.map((d) => d.sales) ?? [1]));

  const cards = s
    ? [
        { label: m.merchant.reports.orders, value: fmtNum(s.orders), icon: <IconOrder /> },
        {
          label: m.merchant.reports.delivered,
          value: fmtNum(s.delivered),
          icon: <IconSuccess className="text-success" />,
        },
        {
          label: m.merchant.reports.cancelled,
          value: fmtNum(s.cancelled),
          icon: <IconError className="text-danger" />,
        },
        // **ما خرج من يده هو ما قبض عليه** — لا ما وصل الزبون.
        //
        // كان الرقمُ يُحسب على `delivered`، **فطلبٌ استُلم منه ثمّ تعذّر
        // تسليمُه مالُه في محفظته وتقريرُه يقول لم يبع شيئاً.** ورقمان
        // يختلفان لمعنًى واحد أسوأُ من رقمٍ ناقص: **يرى رصيدَه أكبرَ من
        // مبيعاته فلا يعرف أيَّهما يصدّق.**
        {
          label: m.merchant.reports.sold,
          value: fmtNum(s.sold),
          icon: <IconSuccess className="text-success" />,
        },
        {
          label: m.merchant.reports.returned,
          value: fmtNum(s.returned),
          icon: <IconError className="text-warning" />,
        },
        {
          label: `${m.merchant.reports.sales} (${m.common.currency})`,
          value: fmtNum(s.sales),
          icon: <IconWallet />,
        },
        {
          label: `${m.merchant.reports.net} (${m.common.currency})`,
          value: fmtNum(net),
          icon: <IconWallet className="text-success" />,
        },
      ]
    : [];

  return (
    <div>
      <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
        <h1 className="heading-section flex items-center gap-2">
          <IconStatus className="text-primary" />
          {m.merchant.reports.title}
        </h1>
        <div className="flex gap-2">
          <Input
            id="from"
            type="date"
            dir="ltr"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          />
          <Input id="to" type="date" dir="ltr" value={to} onChange={(e) => setTo(e.target.value)} />
        </div>
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}

      <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-5">
        {cards.map((c) => (
          <div key={c.label} className="surface p-4">
            <div className="mb-1 flex items-center gap-2 text-ink-muted">{c.icon}</div>
            <p className="figure">{c.value}</p>
            <p className="text-xs text-ink-muted">{c.label}</p>
          </div>
        ))}
      </div>

      {report && (
        <section className="surface p-4">
          <h2 className="mb-4 font-bold">{m.merchant.reports.daily}</h2>
          <div className="flex items-end gap-2" style={{ height: 160 }}>
            {report.days.map((d) => (
              <div key={d.date} className="group relative flex-1">
                <div
                  className="mx-auto w-3/5 rounded-t-[4px] bg-primary transition-colors group-hover:bg-primary-dark"
                  style={{ height: Math.max(3, (d.sales / maxSales) * 140) }}
                />
                <div className="pointer-events-none absolute bottom-full start-1/2 z-10 mb-1 hidden -translate-x-1/2 whitespace-nowrap surface-raised rounded-control px-2 py-1 text-xs text-ink group-hover:block rtl:translate-x-1/2">
                  <Money value={d.sales} /> — {fmtNum(d.delivered)}/
                  {fmtNum(d.orders)}
                </div>
              </div>
            ))}
          </div>
          <div className="mt-1 flex gap-2 border-t border-line-soft pt-1">
            {report.days.map((d) => (
              <span key={d.date} className="flex-1 text-center text-2xs text-ink-muted" dir="ltr">
                {d.date.slice(8)}
              </span>
            ))}
          </div>
        </section>
      )}

      {/* **تفصيلُ الأصناف.**

          المجاميعُ تقول إنّ الأسبوع كان جيّداً **ولا تقول لماذا.** وصاحبُ
          المتجر لا يُدير مطبخَه برقمٍ واحد: يسأل أيُّ صنفٍ يمشي وأيُّه راكد،
          فيزيد من هذا ويوقف ذاك. (قرارُ المالك ٢٠٢٦-٠٨-٠٣) */}
      {report && report.items.length > 0 && (
        <section className="mt-6 surface p-4">
          <h2 className="mb-3 font-bold">{m.merchant.reports.itemsTitle}</h2>
          {/* **وعلى الجوّال بطاقاتٌ لا أعمدة** — انظر `.table-stack`.
              (شهده المالك ٢٠٢٦-٠٨-١١: «فايت ببعضه جدولُ الطلبات بالمتجر».)
              **خمسةُ أعمدةٍ عناوينُها جملٌ** — «مبيعاتك – قبل العمولة
              (ل.س)» — **في ٣٦٠ بكسلاً تتراكب حروفُها.** */}
          <div className="sm:overflow-x-auto">
            <table className="table-stack w-full text-sm">
              <thead>
                <tr className="border-b border-line-soft text-xs text-ink-muted">
                  <th className="py-2 text-start font-medium">{m.merchant.reports.itemName}</th>
                  <th className="py-2 text-center font-medium">{m.merchant.reports.itemQty}</th>
                  {/* ══════════════════════════════════════════════════
                      **وثلاثةُ أعمدةٍ لا واحد**
                      ══════════════════════════════════════════════════

                      (قرارُ المالك ٢٠٢٦-٠٨-١٠: «يجب أن يكون الجدولُ السعرَ
                       الأساسيَّ والعمولةَ والسعرَ بعد العمولة، ليكون كلُّ
                       شيءٍ واضحاً».)

                      **ورقمٌ واحدٌ يترك الحسابَ لصاحبه**: يرى ١٨٬٠٠٠ ويقرأ
                      «عمولة» في مكانٍ آخر، **فيطرح بيده ويُخطئ** — أو لا
                      يطرح فيظنّ أنّه يقبضها كلَّها.

                      **ولا هامشَ المنصّة في شيءٍ منها** — لا علاقةَ له به. */}
                  <th className="py-2 text-end font-medium">
                    {m.merchant.reports.itemRevenue} ({m.common.currency})
                  </th>
                  <th className="py-2 text-end font-medium">
                    {m.merchant.reports.itemCommission}
                  </th>
                  <th className="py-2 text-end font-medium">
                    {m.merchant.reports.itemNet}
                  </th>
                </tr>
              </thead>
              <tbody>
                {report.items.map((it) => (
                  <tr key={it.name} className="border-b border-line-soft last:border-0">
                    <td className="py-2" data-label={m.merchant.reports.itemName}>{it.name}</td>
                    <td className="py-2 text-center tabular-nums" dir="ltr" data-label={m.merchant.reports.itemQty}>
                      {fmtNum(it.qty)}
                    </td>
                    {/* **السعرُ الأساسيُّ** — سعرُه هو، دون هامش المنصّة. */}
                    <td className="py-2 text-end tabular-nums text-ink-muted" dir="ltr" data-label={`${m.merchant.reports.itemRevenue} (${m.common.currency})`}>
                      {fmtNum(it.revenue)}
                    </td>
                    {/* **والعمولةُ تُقرأ خصماً** — بإشارتها ونبرتها. */}
                    <td className="py-2 text-end tabular-nums text-danger" dir="ltr" data-label={m.merchant.reports.itemCommission}>
                      −{fmtNum(it.commission)}
                    </td>
                    {/* **وما يقبضه فعلاً** — وهو الرقمُ الذي يهمّه. */}
                    <td className="py-2 text-end font-bold tabular-nums" dir="ltr" data-label={m.merchant.reports.itemNet}>
                      {fmtNum(it.net)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  );
}
