"use client";

/**
 * **شبكةُ الأصناف** — شكلٌ واحدٌ لكلّ مكانٍ يُعرض فيه صنف.
 *
 * # لماذا وُجدت
 *
 * `ItemCard` بطاقةٌ واحدةٌ موحَّدة، **لكنّ قلبَ المفضّلة يحتاج قائمةً**:
 * `useFavorites` نداءٌ واحدٌ للصفحة، **وبطاقةٌ تجلب مفضّلتَها بنفسها تعني
 * عشرين نداءً في شبكةٍ من عشرين بطاقة.**
 *
 * **فالشبكةُ تحمل القائمةَ والبطاقاتُ تقرأ منها.**
 *
 * # وصفحةُ القسم خادميّة
 *
 * **و`useFavorites` خطّافٌ لا يعمل في الخادم** — فهذه غلافُها العميل.
 *
 * # وأعمدتُها أعمدةُ الأقسام
 *
 * **من فتح قسماً لا يجد الشبكةَ تغيّرت تحته.** (قرارُ المالك ٢٠٢٦-٠٨-٠٤:
 * «شكلُ العرض للأصناف يجب أن يكون موحّداً».)
 */

import { useRouter } from "next/navigation";
import { useFavorites } from "@rahalgo/ui";
import ItemCard, { type BrowseItem } from "@/components/ItemCard";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

export default function ItemGrid({
  items,
  /** إلى أين يعود بعد الدخول — **ومن ساقه زرٌّ يعود إلى حيث كان.** */
  next = "/",
  favorites,
}: {
  items: BrowseItem[];
  next?: string;
  /**
   * **قائمةٌ جاهزةٌ حين تكون هي المعروضة** — صفحةُ المفضّلة تعرض ما حُفظ،
   * **فلو جلبته الشبكةُ ثانيةً لَصارتا نسختين**: يُنزع القلبُ عن صنفٍ
   * **فتُحدَّث نسخةُ الشبكة وتبقى البطاقةُ معروضة.**
   */
  favorites?: ReturnType<typeof useFavorites<BrowseItem>>;
}) {
  const router = useRouter();
  const { user } = useAuth();
  const signedIn = isLoggedIn(user);
  // **ولا تُجلب مرّتين**: حين تُمرَّر القائمةُ يُعطَّل الجلبُ هنا — **والخطّافُ
  // يُنادى دائماً** لأنّ ترتيبَ الخطّافات لا يحتمل شرطاً.
  const own = useFavorites<BrowseItem>(api, signedIn && !favorites);
  const { has, toggle } = favorites ?? own;

  return (
    /* **وأربعةُ أعمدةٍ لا خمسة.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦.)

       خمسةٌ على شاشةٍ بألفٍ وأربعمئة تجعل البطاقةَ **مئتين وخمسين بكسلاً**،
       **فيصير الصحنُ نقطةً** ويُقرأ الاسمُ بمشقّة. وأربعةٌ تعطيها **ثلاثمئةً
       وعشرين** — والفرقُ سبعون بكسلاً في الصورة وحدَها.

       **والفجوةُ تتّسع مع الشاشة**: بطاقاتٌ ملتصقةٌ تُقرأ شريطاً واحداً. */
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4">
      {items.map((it) => (
        <ItemCard
          key={it.id}
          item={it}
          favorite={has(it.id)}
          onFavorite={signedIn ? toggle : undefined}
          onRequireLogin={
            signedIn ? undefined : () => router.push(`/login?next=${encodeURIComponent(next)}`)
          }
        />
      ))}
    </div>
  );
}
