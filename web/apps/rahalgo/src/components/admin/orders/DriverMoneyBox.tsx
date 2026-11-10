"use client";

import { getMessages, defaultLocale, fmtNum, fmtUsd } from "@rahalgo/i18n";
import { Money } from "@rahalgo/ui";

const m = getMessages(defaultLocale);

/** **ما يحتاجه صندوقُ حساب السائق من الطلب** — أرقامُ الخادم كما هي. */
export interface DriverMoneyInput {
  payment_method: string;
  /** **«التوصيل من المحفظة والباقي كاش»** (٢٠٢٦-١١-٠٩) — طريقتُه `wallet` ونقدُه ثمنُ البضاعة. */
  wallet_scope?: string;
  cash_due?: number;
  platform_commission?: number;
  commission_percent_exact?: number;
  pay_merchant?: number;
  /** **ما يدفعه السائقُ لكلّ متجر** — والدولارُ (`currency: "USD"`) بالسنتات ولا يدخل `pay_merchant` بالليرة. */
  pay_merchants?: { merchant_name: string; amount: number; currency?: string }[];
  driver_fee?: number;
  free_waived?: number;
}

/**
 * **حسابُ السائق والمتجر في الطلب** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «لازم يكون واضح شقد العمولة مشان السائق
 * يعرف شقد يعطي للمتجر وشقد يسلّم للمكتب»). من أرقام الخادم نفسِها: عمولةُ المنصّة من المتجر، وما
 * يدفعه السائقُ للمتجر عند الاستلام (`pay_merchant` — للمتاجر النقديّة)، وما يقبضه من الزبون، والباقي
 * للمكتب — **وأجرتُه كاملةً ولو كان التوصيلُ مجّانيّاً على المنصّة.**
 */
export function DriverMoneyBox({ o, hideCommission = false }: {
  o: DriverMoneyInput;
  /** **بطاقةُ اللوحة تقول العمولةَ في سطرها فوقه** (٢٠٢٦-١١-٠٣) — فلا تُكرَّر هنا. */
  hideCommission?: boolean;
}) {
  const OP = m.admin.ordersPage;
  const com = hideCommission ? 0 : o.platform_commission ?? 0;
  const pay = o.pay_merchant ?? 0;
  // **النقدُ من `cash_due` لا من الطريقة** (قرارُ المالك ٢٠٢٦-١١-٠٩): «التوصيل من المحفظة» طريقتُه
  // `wallet` والكابتن يقبض ثمنَ البضاعة؛ و«الكل من المحفظة» نقدُه صفر أصلاً.
  const cash = o.cash_due ?? 0;
  if (com <= 0 && pay <= 0 && cash <= 0 && (o.pay_merchants ?? []).every((p) => p.currency !== "USD")) return null;
  const office = Math.max(0, cash - pay);
  const row = "flex items-center justify-between gap-2";
  /* **ما يدفعه السائقُ بالدولار يُكتب بالدولار** — منفصلاً عن الليرة، لا يُجمع معها ولا يُحوَّل. */
  const usdPays = (o.pay_merchants ?? []).filter((p) => p.currency === "USD" && p.amount > 0);
  return (
    <li className="mt-2 list-none rounded-control border border-line-soft bg-field p-2.5 text-sm">
      <p className="mb-1.5 text-xs font-bold">{OP.moneyBox}</p>
      <ul className="space-y-1">
        {com > 0 && (
          <li className={row}>
            <span className="text-ink-muted">
              {OP.storeCommission.replace("{p}", fmtNum(o.commission_percent_exact ?? 0))}
            </span>
            <Money value={com} small />
          </li>
        )}
        {cash > 0 && (
          <li className={row}>
            <span className="text-ink-muted">{OP.driverCollects}</span>
            <Money value={cash} small />
          </li>
        )}
        {pay > 0 && (
          <li className={row}>
            <span className="text-ink-muted">{OP.driverPaysStore}</span>
            <span className="flex items-center gap-1">
              −<Money value={pay} small />
            </span>
          </li>
        )}
        {usdPays.map((p, i) => (
          <li key={`usd-${i}`} className={row}>
            <span className="text-ink-muted">{p.merchant_name ? `${OP.driverPaysStore} — ${p.merchant_name}` : OP.driverPaysStore}</span>
            <span dir="ltr" className="tabular-nums">
              −{fmtUsd(p.amount)}
            </span>
          </li>
        ))}
        {cash > 0 && (
          <li className={`${row} border-t border-line-soft pt-1 font-bold`}>
            <span>{OP.driverToOffice}</span>
            <Money value={office} small />
          </li>
        )}
        {(o.driver_fee ?? 0) > 0 && (
          <li className={`${row} text-success`}>
            <span>{(o.free_waived ?? 0) > 0 ? OP.driverEarnsFree : OP.driverEarns}</span>
            <Money value={o.driver_fee ?? 0} small />
          </li>
        )}
      </ul>
    </li>
  );
}
