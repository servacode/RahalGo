"use client";

/**
 * **أقسامُ الزبون أسفلَ الشاشة** — على الجوّال وحدَه.
 *
 * # ولماذا هذه الأربعة
 *
 * رحلةُ الزبون أربعُ محطّاتٍ لا أكثر:
 *
 *	تصفّحٌ  ←  طلبٌ  ←  تتبّعٌ  ←  عودةٌ إلى ما أحبّ
 *
 * **والرئيسيّةُ والطلباتُ يوميّتان**، والمفضّلةُ هي ما يختصر التصفّحَ في
 * المرّة القادمة، والحسابُ بابُ المحفظة والعناوين.
 *
 * **وما بقي في الشريط العلويّ**: الإشعاراتُ والرصيدُ والعروضُ والدعوةُ
 * والشكاوى — **تُفتح مرّةً في الأسبوع، ولا تستحقّ ثُمنَ الشاشة الدائم.**
 *
 * # والسلّةُ ليست بنداً
 *
 * **هي عربةٌ عائمةٌ بقرار المالك** (`FloatingCart`) — «العربةُ تُلتقط عند
 * الباب لا بعد اختيار أوّل صنف». **فتُرفع فوق الشريط ولا تُستبدل به.**
 *
 * # ولا يظهر لزائرٍ لم يدخل
 *
 * **ثلاثةٌ من أربعةٍ تسوقه إلى الدخول** — وشريطٌ كلُّه أبوابٌ مقفلةٌ يُقرأ
 * حاجزاً لا تنقّلاً. **والزائرُ يتصفّح أوّلاً**، فإن أراد شيئاً ساقه الفعلُ
 * نفسُه إلى الدخول.
 */

import Link from "next/link";
import { usePathname } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  MobileNav,
  MobileNavSpacer,
  IconGrid,
  IconStore,
  IconOrder,
  IconHeart,
  IconUser,
  type NavItem,
} from "@rahalgo/ui";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);

const ITEMS: readonly NavItem[] = [
  { href: "/", label: m.site.nav.home, icon: IconGrid },
  /* **والتسوّقُ ثانياً — بعد الرئيسيّة مباشرةً.**
     **وهو سببُ وجود الموقع**: الرئيسيّةُ تُعرّف بالمنصة، **والتسوّقُ ما يأتي
     الزبونُ من أجله.** فموضعُه حيث يصل الإبهامُ أوّلاً بعد البيت.
     (شهده المالك ٢٠٢٦-٠٨-٠٦: «لم تظهر أيُّ أيقونةٍ تدلّ على صفحة التسوّق؟») */
  { href: "/shop", label: m.site.nav.shop, icon: IconStore },
  { href: "/orders", label: m.terms.orders, icon: IconOrder },
  { href: "/favorites", label: m.customer.favorites.title, icon: IconHeart },
  { href: "/account", label: m.terms.account, icon: IconUser },
];

export function BottomNav() {
  const pathname = usePathname();
  const { user } = useAuth();
  if (!isLoggedIn(user)) return null;
  return <MobileNav items={ITEMS} active={pathname} Link={Link} />;
}

/** الفراغُ تحت المحتوى — **بلاه يختفي آخرُ سطرٍ خلف الشريط.** */
export function BottomNavSpacer() {
  const { user } = useAuth();
  if (!isLoggedIn(user)) return null;
  return <MobileNavSpacer />;
}
