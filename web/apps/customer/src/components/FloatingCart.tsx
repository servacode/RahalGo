"use client";

/**
 * السلّة العائمة — تُرافق التصفّح ولا تختفي به.
 *
 * كانت رمزاً في الشريط العلوي، والشريط يمضي مع التمرير. **فتغيب السلّة في
 * اللحظة التي تُستعمل فيها**: حين يكون الزبون غارقاً في قائمةٍ طويلة يُضيف
 * منها صنفاً بعد صنف، فيرفع رأسه ليرى ما جمع فلا يجد شيئاً — ويصعد إلى الأعلى
 * ليطمئنّ ثم ينزل ليُكمل.
 *
 * وتحمل ما لا يحمله رمز: **العدد والمبلغ معاً**. ورقمٌ عارٍ بجانب رمز يقول
 * «ثلاثة» ولا يقول «ثلاثةَ ماذا» ولا «بكم» — وقد قُرئ «ثلاثة طلبات» فعلاً
 * (R-88).
 *
 * وثلاثةُ شروطٍ لظهورها:
 *
 *  1. **فيها شيء.** سلّةٌ فارغة لا تحتاج باباً، وزرٌّ يقول «صفر» يشغل مكاناً
 *     بلا عمل.
 *  2. **لسنا في السلّة.** زرٌّ يقودك إلى حيث أنت ضجيجٌ لا اختصار.
 *  3. **الزبون داخل.** الضيف يجمع سلّته ويُساق إلى الدخول عند التأكيد — وهذا
 *     سلوكٌ قائم لا نغيّره هنا.
 */

import Link from "next/link";
import { usePathname } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconCart, IconNext } from "@rahalgo/ui";
import { useCart } from "@/lib/cart";

const m = getMessages(defaultLocale);

export function FloatingCart() {
  const { cart, count } = useCart();
  const pathname = usePathname();

  if (!cart || count === 0) return null;
  if (pathname.startsWith("/cart")) return null;

  const subtotal = cart.lines.reduce((s, l) => s + l.price * l.qty, 0);

  return (
    // `end-4` لا `left-4`: في العربية النهايةُ يسارٌ وفي الإنكليزية يمين —
    // والخاصيّةُ المنطقية تتبع اتجاه الصفحة بلا شرطٍ في الشيفرة.
    <Link
      href="/cart"
      aria-label={m.terms.cart}
      className="fixed bottom-4 end-4 z-50 flex items-center gap-3 rounded-card bg-primary px-5 py-3.5 text-white shadow-lg transition-transform hover:scale-[1.02] active:scale-95"
    >
      <span className="relative flex items-center">
        <IconCart size={24} />
        {/* العدّاد على الرمز: يُقرأ قبل النصّ ويُفهم بلا قراءة */}
        <span className="absolute -top-2 -end-2 flex h-5 min-w-5 items-center justify-center rounded-badge bg-white px-1 text-xs font-bold text-primary-dark">
          {fmtNum(count)}
        </span>
      </span>

      <span className="flex flex-col leading-tight">
        <span className="text-xs opacity-90">
          {m.site.cart.itemsCount.replace("{n}", fmtNum(cart.lines.length))}
        </span>
        <span className="font-bold">
          {fmtNum(subtotal)} {m.common.currency}
        </span>
      </span>

      <IconNext size={18} className="opacity-80" />
    </Link>
  );
}
