"use client";

/**
 * **المفضّلة** — أصنافٌ يعود إليها صاحبُها.
 *
 * # وشبكتُها شبكةُ السوق نفسُها
 *
 * **كانت قائمةً بمصغَّرةٍ ٥٦ بكسل ورابطٍ ينقل إلى صفحةِ الصنف** — فيُقرأ
 * الصنفُ في المفضّلة سطرَ جدولٍ وفي القسم سلعة، **ويخرج صاحبُها من شاشتها
 * ليعود إليها.** (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «عند النقر على الصنف تفتح صفحة
 * جديدة وهذا غلط — اتّفقنا نافذةٌ منبثقةٌ نفسَ نظام الصفحة الرئيسيّة».)
 *
 * **و`ItemGrid` هي عينُها في القسم والعروض**: البطاقةُ نفسُها، والنافذةُ
 * نفسُها، والقلبُ نفسُه — **فما يُتعلَّم في شاشةٍ يُعرف في الباقي.**
 *
 * # وخطّافٌ واحدٌ يُمرَّر
 *
 * **`ItemGrid` تحمل `useFavorites` لقلوبها** — وهي هنا القائمةُ المعروضةُ
 * نفسُها. **ولو جلبت الشبكةُ نسختَها لَافترقتا**: يُنزع القلبُ عن صنفٍ
 * فتُحدَّث نسخةُ الشبكة **وتبقى البطاقةُ معروضةً في الصفحة.**
 */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  EmptyState,
  LoadingState,
  PageContainer,
  PageHeader,
  IconHeart,
  useFavorites,
} from "@rahalgo/ui";
import ItemGrid from "@/components/ItemGrid";
import type { BrowseItem } from "@/components/ItemCard";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);
const F = m.customer.favorites;

export default function Page() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const signedIn = isLoggedIn(user);
  const fav = useFavorites<BrowseItem>(api, signedIn);
  const { rows } = fav;

  useEffect(() => {
    if (!loading && !signedIn) router.replace("/login?next=/favorites");
  }, [signedIn, loading, router]);

  if (loading || !signedIn || rows === null) return <LoadingState />;

  return (
    <PageContainer>
      <PageHeader icon={IconHeart} title={F.title} subtitle={F.subtitle} />
      {rows.length === 0 ? (
        /* **وزرُّ تصفّحٍ خيرٌ من جملةٍ يائسة**: من لم يحفظ شيئاً بعد لا يحتاج
           خبراً بل طريقاً. */
        <EmptyState
          icon={IconHeart}
          title={F.empty}
          action={<Button onClick={() => router.push("/")}>{F.browse}</Button>}
        />
      ) : (
        <ItemGrid items={rows} next="/favorites" favorites={fav} />
      )}
    </PageContainer>
  );
}
