"use client";

/**
 * فاتورة الطلب — ورقة واحدة تخدم الزبون والمتجر.
 *
 * **لماذا واحدة لا اثنتان**: الفاتورة سجلٌّ لواقعةٍ واحدة — بيعٌ جرى بين طرفين
 * بوساطة المنصة. فلو صُنعت نسختان لاختلفتا يوماً وصار لكل طرف حقيقته، وهذا أول
 * ما ينهار عند الخلاف. النسخة واحدة والقارئ يختلف.
 *
 * وما يُعرض للمتجر ولا يُعرض للزبون **سطرٌ واحد**: عمولة المنصة وصافي مستحقّه.
 * عقدٌ بين المتجر والمنصة لا شأن للزبون به — ولا يُخفى عن المتجر لأنه طرفه.
 */

import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import { Button } from "./components";
import { IconPrint } from "./icons";

const m = getMessages(defaultLocale);
const V = m.shared.invoice;

export interface InvoiceItem {
  id: string;
  name: string;
  unit_price: number;
  qty: number;
  note?: string;
  options?: { group: string; name: string; price_delta: number }[];
}

export interface InvoiceOrder {
  number: number;
  status: string;
  merchant_name: string;
  customer_name?: string;
  customer_phone?: string;
  address_text: string;
  payment_method: string;
  subtotal: number;
  delivery_fee: number;
  discount: number;
  total: number;
  wallet_paid: number;
  cash_due: number;
  platform_commission?: number;
  items?: InvoiceItem[];
  created_at: string;
  delivered_at?: string | null;
}

export function Invoice({
  order,
  /** عرضُ سطر العمولة وصافي المستحقّ — للمتجر وحده */
  showMerchantSettlement = false,
}: {
  order: InvoiceOrder;
  showMerchantSettlement?: boolean;
}) {
  const items = order.items ?? [];
  const commission = order.platform_commission ?? 0;
  const net = order.subtotal - commission;

  return (
    <div className="space-y-4">
      <div className="no-print flex justify-end">
        <Button onClick={() => window.print()} className="flex items-center gap-2">
          <IconPrint size={16} />
          {V.print}
        </Button>
      </div>

      <div data-print="sheet" className="rounded-card border border-line bg-surface p-6 text-sm">
        {/* الترويسة: من، ولمن، ومتى — ورقةٌ تُقرأ بعد شهر بلا شاشة */}
        <div className="mb-5 flex flex-wrap items-start justify-between gap-4 border-b border-line pb-4">
          <div>
            <p className="text-lg font-bold">{V.title}</p>
            <p className="mt-1 text-ink-muted">
              {V.number} <span dir="ltr">#{fmtNum(order.number)}</span>
            </p>
            <p className="text-xs text-ink-muted">
              {V.issuedAt} <span dir="ltr">{fmtDateTime(order.created_at)}</span>
            </p>
            {order.delivered_at && (
              <p className="text-xs text-ink-muted">
                {V.deliveredAt} <span dir="ltr">{fmtDateTime(order.delivered_at)}</span>
              </p>
            )}
          </div>
          <div className="text-end">
            <p className="font-bold">{m.common.appName}</p>
            <p className="text-ink-muted">{order.merchant_name}</p>
            {order.customer_name && <p className="text-xs text-ink-muted">{order.customer_name}</p>}
            {order.customer_phone && (
              <p className="text-xs text-ink-muted" dir="ltr">
                {order.customer_phone}
              </p>
            )}
          </div>
        </div>

        <p className="mb-4 text-xs text-ink-muted">
          {V.address} {order.address_text}
        </p>

        {items.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead>
                <tr className="border-b border-line text-xs text-ink-muted">
                  <th className="py-2 text-start font-medium">{V.colItem}</th>
                  <th className="py-2 text-end font-medium">{V.colQty}</th>
                  <th className="py-2 text-end font-medium">{V.colUnit}</th>
                  <th className="py-2 text-end font-medium">{V.colLine}</th>
                </tr>
              </thead>
              <tbody>
                {items.map((it) => (
                  <tr key={it.id} className="border-b border-line/60">
                    <td className="py-2 align-top">
                      {it.name}
                      {!!it.options?.length && (
                        <span className="block text-xs text-ink-muted">
                          {it.options.map((o) => o.name).join(m.common.listSeparator)}
                        </span>
                      )}
                      {it.note && <span className="block text-xs text-ink-muted">{it.note}</span>}
                    </td>
                    <td className="py-2 text-end align-top" dir="ltr">
                      {fmtNum(it.qty)}
                    </td>
                    <td className="py-2 text-end align-top" dir="ltr">
                      {fmtNum(it.unit_price)}
                    </td>
                    <td className="py-2 text-end align-top font-medium" dir="ltr">
                      {fmtNum(it.unit_price * it.qty)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* الحساب: كل سطر يُجمع مع ما قبله فيبلغ الإجمالي — يُراجَع لا يُصدَّق */}
        <dl className="mt-4 space-y-1.5 border-t border-line pt-3 text-sm">
          <Row label={V.subtotal} value={order.subtotal} />
          <Row label={V.deliveryFee} value={order.delivery_fee} />
          {order.discount > 0 && <Row label={V.discount} value={-order.discount} tone="success" />}
          <div className="flex items-center justify-between border-t border-line pt-2 text-base font-bold">
            <dt>{V.total}</dt>
            <dd dir="ltr">
              {fmtNum(order.total)} {m.common.currency}
            </dd>
          </div>
        </dl>

        <p className="mt-3 rounded-control bg-page px-3 py-2 text-xs text-ink-muted">
          {order.wallet_paid > 0 && order.cash_due === 0
            ? V.paidWallet
            : V.paidCash.replace("{n}", fmtNum(order.cash_due))}
        </p>

        {showMerchantSettlement && (
          <dl className="mt-4 space-y-1.5 border-t border-line pt-3 text-sm">
            <p className="mb-1 text-xs font-medium text-ink-muted">{V.settlementTitle}</p>
            <Row label={V.commission} value={-commission} tone="danger" />
            <div className="flex items-center justify-between border-t border-line pt-2 font-bold">
              <dt>{V.netDue}</dt>
              <dd className="text-success" dir="ltr">
                {fmtNum(net)} {m.common.currency}
              </dd>
            </div>
            <p className="pt-1 text-[11px] leading-relaxed text-ink-muted">{V.settlementHint}</p>
          </dl>
        )}

        <p className="mt-4 border-t border-line pt-3 text-xs leading-relaxed text-ink-muted">
          {V.footer}
        </p>
      </div>
    </div>
  );
}

function Row({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone?: "success" | "danger";
}) {
  return (
    <div className="flex items-center justify-between">
      <dt className="text-ink-muted">{label}</dt>
      <dd
        className={tone === "success" ? "text-success" : tone === "danger" ? "text-danger" : ""}
        dir="ltr"
      >
        {fmtNum(value)}
      </dd>
    </div>
  );
}
