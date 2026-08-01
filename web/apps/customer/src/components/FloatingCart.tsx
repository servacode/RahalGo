"use client";

/**
 * السلّة العائمة — عربةٌ تُلاحق الزبون.
 *
 * **بلا كرتٍ ولا حدٍّ ولا صندوق**: عربةٌ كبيرة وحدها. والصندوقُ حولها يجعلها
 * زرّاً من أزرار الواجهة، والمقصودُ أن تكون **شيئاً في المشهد** — عربةَ تسوّقٍ
 * تمشي مع صاحبها في الممرّ.
 *
 * **وتظهر دائماً ولو كانت فارغة.** وكان إخفاؤها عند الفراغ منطقاً سليماً على
 * الورق — «بابٌ بلا شيء خلفه» — وهو خطأ في محلٍّ يُتجوَّل فيه: **العربة تُلتقط
 * عند الباب لا بعد اختيار أوّل صنف**. ووجودُها دعوةٌ إلى الشراء، وغيابُها حتى
 * يُشترى يجعلها أثراً لا سبباً.
 *
 * وتُخفى في صفحة السلّة وحدها: زرٌّ يقودك إلى حيث أنت ضجيجٌ لا اختصار، وقد
 * يحجب زرّ التأكيد.
 *
 * **والظلّ على الرمز لا خلفه** (`drop-shadow` لا `shadow`): بلا صندوقٍ يحمل
 * الظلّ، يتبع الظلُّ حدودَ العربة نفسها — فتُقرأ فوق البطاقات البيضاء وفوق
 * صور اللافتات معاً.
 */

import Link from "next/link";
import { usePathname } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconCart } from "@rahalgo/ui";
import { useCart } from "@/lib/cart";

const m = getMessages(defaultLocale);

export function FloatingCart() {
  const { count } = useCart();
  const pathname = usePathname();

  if (pathname.startsWith("/cart")) return null;

  return (
    // `end-5` لا `left-5`: الخاصيّةُ المنطقية تتبع اتجاه الصفحة بلا شرطٍ في
    // الشيفرة — في العربية النهايةُ يسارٌ وفي الإنكليزية يمين.
    <Link
      href="/cart"
      aria-label={
        count > 0
          ? m.site.cart.badgeTitle.replace(
              "{n}",
              m.site.cart.itemsCount.replace("{n}", fmtNum(count)),
            )
          : m.terms.cart
      }
      title={m.terms.cart}
      className="fixed bottom-5 end-5 z-50 block text-primary transition-transform hover:scale-110 active:scale-95 [filter:drop-shadow(0_4px_10px_rgb(0_0_0/0.28))]"
    >
      <span className="relative block">
        <IconCart size={56} strokeWidth={1.7} />
        {count > 0 && (
          // العدّاد على قبضة العربة: يُقرأ قبل النصّ ويُفهم بلا قراءة
          <span className="absolute -top-1 -end-1 flex h-7 min-w-7 items-center justify-center rounded-badge bg-danger px-1.5 text-sm font-bold text-white shadow-sm">
            {fmtNum(count)}
          </span>
        )}
      </span>
    </Link>
  );
}
