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

import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import { Button } from "./components";
import { SheetHeader } from "./layout";
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
  merchant_name?: string;
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
  /**
   * إظهارُ مصدر البضاعة — **للمتجر والعمليات لا للزبون.**
   *
   * **وافتراضُه الإخفاء لا الإظهار.** من نسي تمريرَه في شاشةٍ جديدة يُخفي —
   * **وخطأُ الإخفاء يُكتشف بسؤالٍ من موظّف، وخطأُ الإظهار لا يُكتشف أبداً**:
   * يمضي في آلاف الفواتير قبل أن ينتبه أحد.
   */
  showSource = false,
}: {
  order: InvoiceOrder;
  showMerchantSettlement?: boolean;
  showSource?: boolean;
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

      <div data-print="sheet" className="surface p-6 text-sm">
        {/* العلامةُ والاسمُ ووقتُ الطباعة — ترويسةٌ واحدة للفاتورة والكشف */}
        <SheetHeader />

        {/* **سطرُ التعريف**: رقمُ الطلب ومن هو صاحبُه وأين — ما يُبحث به.
            وتاريخُ الطلب هنا لا في الترويسة: تلك تحمل وقتَ الطباعة، **وخلطُهما
            يجعل ورقةً تُطبع بعد شهرٍ تبدو طلباً وقع اليوم.** */}
        <div className="mb-4 grid grid-cols-1 gap-x-6 gap-y-1 border-b border-line-soft pb-3 text-xs sm:grid-cols-2">
          <p className="text-base font-bold">
            {V.title} <span dir="ltr">#{fmtRef(order.number)}</span>
          </p>
          <p className="sm:text-end">
            <span className="text-ink-muted">{V.issuedAt} </span>
            <span dir="ltr">{fmtDateTime(order.created_at)}</span>
          </p>
          {order.customer_name && (
            <p>
              <span className="text-ink-muted">{V.customer} </span>
              <span className="font-medium">{order.customer_name}</span>
              {order.customer_phone && (
                <span dir="ltr" className="ms-2 text-ink-muted">
                  {order.customer_phone}
                </span>
              )}
            </p>
          )}
          {order.delivered_at && (
            <p className="sm:text-end">
              <span className="text-ink-muted">{V.deliveredAt} </span>
              <span dir="ltr">{fmtDateTime(order.delivered_at)}</span>
            </p>
          )}
          <p className="sm:col-span-2">
            <span className="text-ink-muted">{V.address} </span>
            {order.address_text}
          </p>
          {/* **الفاتورةُ صادرةٌ من «رحّال غو» لا من المتجر.**

              الزبونُ اشترى منّا: نحن من عرض السعرَ وقبض الثمنَ وأوصل. **واسمُ
              المتجر في الفاتورة يقول له من أين نشتري** — فيتّصل به في المرّة
              القادمة **ويوفّر رسمَ التوصيل والمتجرُ يوفّر عمولتنا.** وكلُّ
              منصةِ توصيلٍ تموت من هذا الباب لا من غيره.

              **والورقةُ أبقى من الشاشة**: صفحةٌ تُغلق، **وفاتورةٌ تُطبع تبقى
              في البيت شهراً وتُقرأ مرّةً بعد مرّة.**

              وتُعرض للمتجر والعمليات كما هي: `showSource` تُمرَّر من شاشتهما. */}
          {showSource && order.merchant_name && (
            <p className="sm:col-span-2 text-ink-muted">{order.merchant_name}</p>
          )}
        </div>

        {items.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead>
                {/* **رأسٌ يُميَّز بحدٍّ لا بلونٍ**: الألوانُ لا تُطبع، ورأسُ
                    جدولٍ يذوب في صفوفه يجعل العمودَ الأوّل يُقرأ مبلغاً. */}
                <tr className="border-b-2 border-line-soft text-2xs uppercase tracking-wide text-ink-muted">
                  <th className="py-2 text-start font-bold">{V.colItem}</th>
                  <th className="w-16 py-2 text-end font-bold">{V.colQty}</th>
                  <th className="w-24 py-2 text-end font-bold">{V.colUnit}</th>
                  <th className="w-28 py-2 text-end font-bold">{V.colLine}</th>
                </tr>
              </thead>
              <tbody>
                {items.map((it) => (
                  <tr key={it.id} className="border-b border-line-soft">
                    <td className="py-2 align-top">
                      {it.name}
                      {!!it.options?.length && (
                        <span className="block text-xs text-ink-muted">
                          {it.options.map((o) => o.name).join(m.common.listSeparator)}
                        </span>
                      )}
                      {it.note && <span className="block text-xs text-ink-muted">{it.note}</span>}
                    </td>
                    <td className="py-2 text-end align-top tabular-nums" dir="ltr">
                      {fmtNum(it.qty)}
                    </td>
                    <td className="py-2 text-end align-top tabular-nums text-ink-muted" dir="ltr">
                      {fmtNum(it.unit_price)}
                    </td>
                    <td className="py-2 text-end align-top font-bold tabular-nums" dir="ltr">
                      {fmtNum(it.unit_price * it.qty)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* الحساب: كل سطر يُجمع مع ما قبله فيبلغ الإجمالي — يُراجَع لا يُصدَّق */}
        {/* **المجاميعُ كتلةٌ إلى المنتهى لا شريطٌ بعرض الورقة.**

            سطرٌ ممتدٌّ من الحافة إلى الحافة يُبعد اللفظَ عن رقمه شبراً، **فتُقرأ
            الأرقامُ في عمودٍ واحدٍ ويُبحث عن أسمائها**. وجمعُهما في كتلةٍ ضيّقة
            يجعل كلَّ لفظٍ ملاصقاً لمبلغه. */}
        <div className="mt-5 flex justify-end" data-print-keep>
          <dl className="w-full max-w-xs space-y-1.5 text-sm">
            <Row label={V.subtotal} value={order.subtotal} />
            <Row label={V.deliveryFee} value={order.delivery_fee} />
            {order.discount > 0 && (
              <Row label={V.discount} value={-order.discount} tone="success" />
            )}
            <div className="figure flex items-center justify-between border-t-2 border-line-soft pt-2">
              <dt>{V.total}</dt>
              <dd dir="ltr" className="tabular-nums">
                {fmtNum(order.total)}{" "}
                <span className="text-sm font-normal">{m.common.currency}</span>
              </dd>
            </div>
          </dl>
        </div>

        <p className="mt-3 rounded-control bg-field px-3 py-2 text-xs text-ink-muted">
          {order.wallet_paid > 0 && order.cash_due === 0
            ? V.paidWallet
            : V.paidCash.replace("{n}", fmtNum(order.cash_due))}
        </p>

        {/* **آخرُ ما تقع عليه العين.**

            ورقةٌ تنتهي برقمٍ تنتهي جافّة، **والفاتورةُ آخرُ ما يبقى من الطلب
            في يد الزبون** — فتقول كلمةً قبل أن تُطوى. وهي في المعجم لا في
            الشيفرة: يبدّلها المالكُ متى شاء بلا نشر. */}
        <p className="mt-5 border-t border-line-soft pt-4 text-center text-sm font-medium text-primary">
          {V.thanks}
        </p>

        {showMerchantSettlement && (
          <dl className="mt-4 space-y-1.5 border-t border-line-soft pt-3 text-sm">
            <p className="mb-1 text-xs font-medium text-ink-muted">{V.settlementTitle}</p>
            <Row label={V.commission} value={-commission} tone="danger" />
            <div className="flex items-center justify-between border-t border-line-soft pt-2 font-bold">
              <dt>{V.netDue}</dt>
              <dd className="text-success" dir="ltr">
                {fmtNum(net)} {m.common.currency}
              </dd>
            </div>
            <p className="pt-1 text-2xs leading-relaxed text-ink-muted">{V.settlementHint}</p>
          </dl>
        )}

        <p className="mt-4 border-t border-line-soft pt-3 text-xs leading-relaxed text-ink-muted">
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
        className={`tabular-nums ${
          tone === "success" ? "text-success" : tone === "danger" ? "text-danger" : ""
        }`}
        dir="ltr"
      >
        {fmtNum(value)}
      </dd>
    </div>
  );
}
