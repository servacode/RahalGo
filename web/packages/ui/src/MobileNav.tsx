"use client";

/**
 * **التنقّلُ السفليُّ على الجوّال** — حيث يصل الإبهام.
 *
 * # المسألة
 *
 * أقسامُ الزبون والسائق كلُّها في **الشريط العلويّ**، وهو ينزلق أفقيّاً على
 * الضيّق (أصلحتُ فيضَه قبل موجتين). **والإبهامُ يصل الثلثَ السفليَّ من الشاشة
 * وحدَه** — وهي قاعدةٌ فيزيائيّةٌ لا ذوق: هاتفٌ بستّ بوصاتٍ يُمسك بيدٍ واحدة،
 * **وأعلى الشاشة يحتاج أن يُزحلَق الجهازُ في الكفّ.**
 *
 * **وسائقُنا يمسك هاتفَه بيدٍ وهو واقفٌ في الشارع** — واليدُ الأخرى على
 * الدرّاجة.
 *
 * # ولماذا أربعةٌ لا سبعة
 *
 * **العرضُ يقسَّم على العدد**: أربعةٌ تُعطي كلَّ بندٍ نحوَ تسعين بكسلاً — تكفي
 * لأيقونةٍ واسمٍ يُقرأ. **وسبعةٌ تُعطيه خمسين**، فيُقصّ الاسمُ أو تُحذف
 * التسمية، **وأيقونةٌ بلا اسمٍ تُفهم بالتجربة لا بالنظر.**
 *
 * **والباقي يبقى في الشريط العلويّ**: ما يُفتح مرّةً في الأسبوع لا يستحقّ
 * ثُمنَ الشاشة الدائم.
 *
 * # ولا يظهر على الواسع
 *
 * **شريطٌ سفليٌّ على لابتوب يأكل مساحةً بلا سبب** — الفأرةُ تصل كلَّ نقطةٍ
 * بالكلفة نفسِها، **والتنقّلُ السفليُّ حلٌّ لمشكلةٍ لا توجد هناك.**
 */

import type { ComponentType, ReactNode } from "react";

export interface NavItem {
  href: string;
  label: string;
  icon: ComponentType<{ size?: number; className?: string }>;
  /** عدّادٌ فوق الأيقونة — **والصفرُ لا يُعرض** فبقعةٌ فارغةٌ تُقلق بلا سبب. */
  count?: number;
}

/**
 * MobileNav شريطُ الأقسام السفليّ.
 *
 * @param active المسارُ الحاليّ — **يُطابَق ببدايته** لتبقى الأقسامُ الفرعيّة
 *   مُبرِزةً أصلَها: من فتح `‎/orders/5` ما يزال في «طلباتي».
 */
export function MobileNav({
  items,
  active,
  Link,
}: {
  items: readonly NavItem[];
  active: string;
  Link: ComponentType<{
    href: string;
    className?: string;
    "aria-current"?: "page";
    children: ReactNode;
  }>;
}) {
  return (
    <nav
      className="elev-4 fixed inset-x-0 bottom-0 z-40 flex border-t border-line bg-raised md:hidden"
      /* **وحافّةُ الأمان تحت الشريط** — هواتفُ آيفون تضع خطَّ الإيماءة أسفلَ
         الشاشة، **وزرٌّ تحته يُضغط فيُغلق التطبيق** بدل أن يُفتح القسم. */
      style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      {items.map((it) => {
        // **والجذرُ يُطابَق تماماً**: `‎/` بدايةٌ لكلّ مسارٍ في الدنيا،
        // **فلو طُوبق ببدايته لَبقيت «الرئيسية» مُبرَزةً في كلّ صفحة.**
        const on = it.href === "/" ? active === "/" : active.startsWith(it.href);
        const Icon = it.icon;
        return (
          <Link
            key={it.href}
            href={it.href}
            aria-current={on ? "page" : undefined}
            /* **وأربعةٌ وأربعون بكسلاً حدُّ اللمس** — والارتفاعُ هنا ستّون:
               الإصبعُ لا يصيب أقلَّ منها من أوّل مرّة، **ومن أخطأ مرّتين ترك
               الشريطَ واستعمل رجوعَ المتصفّح.** */
            className={`relative flex min-h-[3.75rem] flex-1 flex-col items-center justify-center gap-1 py-2 text-2xs transition-colors ${
              on ? "font-bold text-accent-text" : "text-ink-muted"
            }`}
          >
            {/* **والنشِطُ له خيطٌ علويّ** — لونٌ وحدَه لا يكفي لمن لا يميّز
                الألوان، **وموضعُ الخيط يقول «هذا القسم» بلا قراءة.** */}
            <span
              aria-hidden
              className={`absolute inset-x-4 top-0 h-0.5 rounded-badge transition-colors ${
                on ? "bg-accent" : "bg-transparent"
              }`}
            />
            <span className="relative">
              <Icon size={21} />
              {it.count != null && it.count > 0 && (
                <span
                  dir="ltr"
                  className="absolute -top-1.5 -end-2 flex h-4 min-w-4 items-center justify-center rounded-badge bg-danger-solid px-1 text-2xs font-bold text-on-solid"
                >
                  {it.count > 9 ? "9+" : it.count}
                </span>
              )}
            </span>
            {it.label}
          </Link>
        );
      })}
    </nav>
  );
}

/**
 * MobileNavSpacer فراغٌ بارتفاع الشريط — **يُوضع آخرَ الصفحة.**
 *
 * **والشريطُ ثابتٌ فوق المحتوى** (`fixed`)، فبلا هذا الفراغ **يختفي آخرُ سطرٍ
 * تحته** — وهو غالباً زرُّ الحسم: «أكّد الطلب» أو «سلّمت».
 *
 * **ومن مرّر إلى الآخر فلم يجد الزرَّ ظنّه غيرَ موجود.**
 */
export function MobileNavSpacer() {
  return (
    <div
      aria-hidden
      className="h-[3.75rem] md:hidden"
      style={{ marginBottom: "env(safe-area-inset-bottom)" }}
    />
  );
}
