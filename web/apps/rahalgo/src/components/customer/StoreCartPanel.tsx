"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحةُ السلّة اللاصقة في صفحة المتجر — على الحاسوب وحدَه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ترى السلّةَ وأنت تتصفّح** بلا أن تغادر المتجر (مخطّطُ ٢٠٢٦-١١-٠٨). **عرضٌ فوق السلّة الموجودة**:
 * `useCart` نفسُها، وعدّادُ كلِّ سطرٍ يستعمل `setQty` كما تفعل صفحةُ السلّة، **والحسمُ يبقى في `/cart`**
 * (العنوانُ والدفعُ والتأكيد) — فلا سعرَ يُحسب هنا ولا طلبَ يُرسل.
 *
 * **ولا تُرسم إلّا لسلّةٍ من هذا المتجر**: سلّةُ متجرٍ آخر تُقال («سلتك من متجر آخر») في نافذة الصنف.
 */

import Link from "next/link";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconCart, IconDelete, IconMinus, IconAdd, Money } from "@rahalgo/ui";
import { lineKey, lineTotal, useCart } from "@/lib/customer/cart";
import { fill } from "@/lib/customer/texts";
import { ShopPic } from "./ShopPic";

const m = getMessages(defaultLocale);
const SK = m.shop.skin;

export function StoreCartPanel({ storeId, storeName }: { storeId: string; storeName: string }) {
  const cart = useCart();
  const mine = cart.merchantId === storeId ? cart.lines : [];
  return (
    <aside className="shop-cartpanel" aria-label={SK.yourCart}>
      <div className="shop-cartpanel-head">
        <div className="min-w-0">
          <h2 className="heading-card">{SK.yourCart}</h2>
          <p className="truncate text-xs text-ink-muted">{fill(SK.fromStore, { name: storeName })}</p>
        </div>
        {mine.length > 0 && <span className="shop-cap" data-tone="brand">{fill(SK.itemsN, { n: fmtNum(cart.count) })}</span>}
      </div>
      {mine.length === 0 ? (
        <div className="flex flex-col items-center gap-2 px-6 py-10 text-center">
          <IconCart size={28} className="text-ink-muted" />
          <p className="font-bold">{SK.cartEmpty}</p>
          <p className="text-sm text-ink-muted">{SK.cartEmptyHint}</p>
        </div>
      ) : (
        <>
          <ul className="shop-cartpanel-lines">
            {mine.map((l) => {
              const key = lineKey(l);
              const note = l.options.map((o) => o.name).join(" · ");
              return (
                <li key={key} className="shop-cartpanel-line">
                  <ShopPic
                    src={l.item.image_thumb_url ?? l.item.image_url}
                    name={l.item.name}
                    icon="food"
                    iconSize={18}
                    className="h-14 w-14 shrink-0 rounded-control"
                  />
                  <div className="min-w-0 flex-1">
                    <p className="line-clamp-2 text-sm font-bold">{l.item.name}</p>
                    {note && <p className="truncate text-xs text-ink-muted">{note}</p>}
                    <Money value={lineTotal(l)} className="text-sm" />
                  </div>
                  <span className="shop-stepper" dir="ltr">
                    <button type="button" aria-label={SK.plus} onClick={() => cart.setQty(key, l.qty + 1)}>
                      <IconAdd size={16} />
                    </button>
                    <span>{fmtNum(l.qty)}</span>
                    <button
                      type="button"
                      aria-label={l.qty <= 1 ? SK.remove : SK.minus}
                      onClick={() => cart.setQty(key, l.qty - 1)}
                    >
                      {l.qty <= 1 ? <IconDelete size={16} /> : <IconMinus size={16} />}
                    </button>
                  </span>
                </li>
              );
            })}
          </ul>
          <div className="shop-cartpanel-foot">
            <div className="flex items-center justify-between">
              <span className="text-ink-muted">{SK.subtotal}</span>
              <Money value={cart.subtotal} className="figure" />
            </div>
            <Link href="/cart" className="shop-cta">
              {SK.continue}
            </Link>
          </div>
        </>
      )}
    </aside>
  );
}
