"use client";

/**
 * السلّة العائمة — حقيبةٌ تُلاحق الزبون.
 *
 * **بلا كرتٍ ولا حدٍّ ولا صندوق**: حقيبةٌ كبيرة وحدها. والصندوقُ حولها يجعلها
 * زرّاً من أزرار الواجهة، والمقصودُ أن تكون **شيئاً في المشهد** — سلّةَ تسوّقٍ
 * تمشي مع صاحبها في الممرّ.
 *
 * **وتظهر دائماً ولو كانت فارغة.** وكان إخفاؤها عند الفراغ منطقاً سليماً على
 * الورق — «بابٌ بلا شيء خلفه» — وهو خطأ في محلٍّ يُتجوَّل فيه: **العربة تُلتقط
 * عند الباب لا بعد اختيار أوّل صنف**. ووجودُها دعوةٌ إلى الشراء، وغيابُها حتى
 * يُشترى يجعلها أثراً لا سبباً.
 *
 * ## الحركة
 *
 * **طفوٌ بطيءٌ دائم** يلفت العين بلا أن يشغلها — سبعةُ بكسلات في ثلاث ثوانٍ:
 * أقلُّ ما يُلحَظ وأكثرُ ما لا يُزعج.
 *
 * **ونبضةٌ عند كل إضافة** مع حلقةٍ تتمدّد خلفها: تقول «وصلَك شيء» في اللحظة
 * التي يُضاف فيها، فلا يحتاج الزبون أن يتحقّق. **ولا تدور ولا تهتزّ** —
 * الاهتزازُ يُقرأ خطأً لا ترحيباً.
 *
 * **ولا نبضةَ عند أوّل رسم**: سلّةٌ محفوظةٌ من جلسةٍ سابقة تُحمَّل مع الصفحة،
 * ونبضُها يقول «أُضيف الآن» وهو كذب.
 */

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconCart } from "@rahalgo/ui";
import { useCart } from "@/lib/cart";

const m = getMessages(defaultLocale);

export function FloatingCart() {
  const { count } = useCart();
  const pathname = usePathname();

  const [pop, setPop] = useState(false);
  const prev = useRef<number | null>(null);

  useEffect(() => {
    // أوّل رسمٍ يُسجَّل ولا يُنبض — والنقصان لا يُحتفى به
    if (prev.current === null) {
      prev.current = count;
      return;
    }
    if (count > prev.current) {
      setPop(true);
      const t = setTimeout(() => setPop(false), 700);
      prev.current = count;
      return () => clearTimeout(t);
    }
    prev.current = count;
  }, [count]);

  if (pathname.startsWith("/cart")) return null;

  return (
    // `end-8 bottom-8` بالخاصيّة المنطقية لا باليسار الصريح: تتبع اتجاه الصفحة
    // بلا شرطٍ في الشيفرة. والمسافةُ أوسع كي لا تلتصق الحقيبة بحافّة المحتوى.
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
      className="fixed bottom-8 end-8 z-50 block"
    >
      <span
        className={`relative block text-primary transition-transform hover:scale-110 active:scale-95 [filter:drop-shadow(0_5px_12px_rgb(0_0_0/0.3))] ${
          pop ? "cart-pop" : "cart-float"
        }`}
      >
        {/* الحلقة خلف الحقيبة — أثرُ لمسةٍ يختفي، لا زخرفة دائمة */}
        {pop && (
          <span className="cart-ring pointer-events-none absolute inset-0 -z-10 rounded-badge bg-primary/35" />
        )}

        <IconCart size={56} strokeWidth={1.7} />

        {count > 0 && (
          // العدّاد على الحقيبة: يُقرأ قبل النصّ ويُفهم بلا قراءة
          <span className="absolute -top-1 -end-1 flex h-7 min-w-7 items-center justify-center rounded-badge bg-danger px-1.5 text-sm font-bold text-white shadow-sm">
            {fmtNum(count)}
          </span>
        )}
      </span>
    </Link>
  );
}
