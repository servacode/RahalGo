"use client";

/**
 * السلّة العائمة — عربةٌ تُلاحق الزبون.
 *
 * **بلا كرتٍ ولا حدٍّ ولا صندوق**: عربةٌ كبيرة وحدها. والصندوقُ حولها يجعلها
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
 * **تتقدّم ثم تلتفّ ثلاث دوراتٍ ثم تعود** — حركةٌ لها معنى لا زخرفةٌ عامّة:
 * عربةُ تسوّقٍ تُدفع خطوةً، تدور حول نفسها، ثم ترجع إلى مكانها.
 *
 * **والدورة في أوّل ثلث المدّة والباقي سكون**: حركةٌ متّصلة في زاوية الشاشة
 * تُتعب العين وتُفقد أثرها بالتكرار — **والسكونُ بين اللفّات هو ما يجعل اللفّة
 * تُلحَظ**.
 *
 * **ونبضةٌ عند كل إضافة** مع حلقةٍ تتمدّد خلفها: تقول «وصلَك شيء» في اللحظة
 * التي يُضاف فيها، فلا يحتاج الزبون أن يتحقّق. **ولا تهتزّ** — الاهتزازُ يُقرأ
 * خطأً لا ترحيباً.
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

  // ══════════════════════════════════════════════════════════════════
  // **ولا تظهر إلّا في صفحة التسوّق**
  // ══════════════════════════════════════════════════════════════════
  //
  // (قرارُ المالك ٢٠٢٦-٠٨-١٥ — وهو قرارُه في التطبيق نفسُه: «لازم
  //  السلّة تظهر فقط بصفحة التسوّق».)
  //
  // **وكانت تُرافقه في كلّ صفحة** — في «طلباتي» و«حسابي» والصفحات
  // القانونيّة. **وعربةُ تسوّقٍ في صفحة شروطِ استخدامٍ زينةٌ لا دعوة**،
  // **وما يُرى في كلّ مكانٍ لا يُرى في أيّ مكان.**
  //
  // **وحيث تُملأ تُعرض** — في الممرّ بين الأصناف.
  if (!pathname.startsWith("/shop")) return null;

  return (
    // `end-8` بالخاصيّة المنطقية لا باليسار الصريح: تتبع اتجاه الصفحة بلا شرطٍ
    // في الشيفرة. والمسافةُ من الأسفل ثلاثةُ أضعافها (٩٦ بكسلاً) كي تبتعد
    // العربة عن حافّة المحتوى، **ولئلّا يبتلعها شريطُ المتصفّح في الجوال**.
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
      // التكبير عند المرور على الرابط لا على العربة: الحركةُ تكتب `transform`
      // باستمرار، ولو وُضعا على عنصرٍ واحد **لابتلعت الحركةُ التكبير**.
      className="fixed bottom-24 end-8 z-50 block transition-transform hover:scale-110 active:scale-95"
    >
      <span
        /* **السلّةُ بالنبرة — وهي أظهرُ ما تتحرّك في الموقع.**

             قاعدةُ العلامة: الهادئُ ما ثبت والنبرةُ لِما يتحرّك. وهي تدور
             وتتقدّم وتعود — **فنبرتُها ليست زينةً، هي تطبيقُ القاعدة على
             أوضح مثالٍ لها.** وتُميَّز بها عن هدوء الشاشة كلِّه. */
        className={`relative block text-accent-text drop-shadow-float ${
          pop ? "cart-pop" : "cart-roll"
        }`}
      >
        {/* الحلقة خلف العربة — أثرُ لمسةٍ يختفي، لا زخرفة دائمة */}
        {pop && (
          <span className="cart-ring pointer-events-none absolute inset-0 -z-10 rounded-badge bg-accent-fill" />
        )}

        <IconCart size={56} strokeWidth={1.7} />

        {count > 0 && (
          // العدّاد على العربة: يُقرأ قبل النصّ ويُفهم بلا قراءة
          <span className="absolute -top-1 -end-1 flex h-7 min-w-7 items-center justify-center rounded-badge bg-danger px-1.5 text-sm font-bold text-on-bright elev-1">
            {fmtNum(count)}
          </span>
        )}
      </span>
    </Link>
  );
}
