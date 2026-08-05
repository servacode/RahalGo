"use client";

/**
 * الرئيسية — **أقسامٌ وأصناف، لا متاجر.**
 *
 * # لماذا انقلبت
 *
 * كان التصفّح «اختر متجراً ثمّ صنفاً»، **والزبونُ لا يفكّر هكذا**: يشتهي
 * شاورما ولا يعرف من يصنع أفضلَها.
 *
 * **وأخطرُ منه أنه يكشف المصدر.** في مدينةٍ يعرف أهلُها بعضهم، **زبونٌ رأى اسمَ
 * المطعم يتّصل به مباشرةً في المرّة القادمة** — يوفّر رسمَ التوصيل والمطعمُ
 * يوفّر عمولتنا. **وكلُّ منصةِ توصيلٍ تموت من هذا الباب لا من غيره.**
 *
 * # والبحثُ صار بالأصناف
 *
 * كان يعيد متاجر ويقول «طابق في: شاورما، فروج» — **فيُقرأ اسمُ المتجر أوّلاً
 * وهو ما نخفيه**، ويُطلب من الزبون خطوةٌ زائدة: يفتح المتجرَ ثمّ يبحث فيه.
 */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  CategoryIcon,
  EmptyState,
  useFavorites,
  Input,
  BannerSlider,
  IconSearch,
  IconClose,
} from "@rahalgo/ui";
import ItemCard, { type BrowseItem } from "@/components/ItemCard";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);

interface Banner {
  id: string;
  title: string;
  image_url: string | null;
  /** وجهةُ الضغط — **ولافتةٌ بلا وجهةٍ تُقرأ ولا تُفتح**، وهي حالٌ مشروعة. */
  target: string | null;
}
interface Category {
  id: string;
  name: string;
  icon: string;
}
export interface HomeData {
  banners: Banner[];
  categories: Category[];
  sections: Section[];
}
interface Section {
  id: string;
  name: string;
  icon: string;
  /**
   * **صورةُ القسم — هويّتُه.** والسوقُ يُتصفَّح بالصور لا بالرموز.
   *
   * **والأصلُ لا المصغَّرة**: المصغَّرةُ حدُّها ٤٠٠ بكسل، **وبطاقةٌ تمطّها
   * تبهت.** والمصغَّرةُ تبقى بديلاً لما رُفع قبل هذا التغيير.
   */
  image_url: string | null;
  image_thumb_url: string | null;
  /** **عددُ المتاح الآن لا كلُّ ما سُجّل** — قسمٌ يقول ١٢ ثمّ يُفتح على ثلاثة
   *  يجعل الزبونَ يشكّ في كلّ رقمٍ بعده. */
  count: number;
}

export default function HomeClient({ initial }: { initial: HomeData }) {
  const { banners, sections } = initial;
  const router = useRouter();
  const { user } = useAuth();
  const signedIn = isLoggedIn(user);
  // **وقائمةُ المفضّلة واحدةٌ للصفحة كلِّها** — لا لكلّ بطاقة.
  const { has, toggle } = useFavorites(api, signedIn);
  const [q, setQ] = useState("");
  const [hits, setHits] = useState<BrowseItem[] | null>(null);

  // البحث بعد سكون الكتابة لا مع كل حرف: كلُّ حرفٍ طلبٌ للخادم، والكاتب لم
  // ينتهِ من كلمته بعد. ٣٠٠ مللي ثانية هي حدُّ ما يُحسّ به المستخدم تأخيراً.
  useEffect(() => {
    const term = q.trim();
    if (term.length < 2) {
      setHits(null);
      return;
    }
    const t = setTimeout(() => {
      fetch(
        `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/v1/public/search/items?q=${encodeURIComponent(term)}`,
      )
        .then((r) => r.json())
        .then((j) => setHits(j.data?.items ?? []))
        // @empty-ok — **البحثُ يُعاد بحرفٍ واحد**: من كتب فلم يجد يُضيف حرفاً
        // فيُعاد النداء، **والأقسامُ تحته لم تُمسّ.**
        .catch(() => setHits([]));
    }, 300);
    return () => clearTimeout(t);
  }, [q]);

  return (
    <div>
      {/* **والعنوانُ يصغر على الجوّال.**

          كان `text-2xl` ثابتاً بهامشٍ أربعةٍ — **نحو ستّين بكسلاً من أوّل
          شاشةٍ يراها الزبون** لجملةٍ تسويقيّةٍ لا يُضغط عليها. **والطيّةُ
          الأولى على هاتفٍ ستُّمئة بكسلٍ لا أكثر**، والسلايدرُ يأخذ مئةً
          وخمسةً وسبعين، والبحثُ خمسين — **فلا تبدأ الأقسامُ إلّا تحت
          الطيّة.**

          **ولا يُحذف**: هو ما يقول للزائر الجديد أين وقع. **إنّما يُقاس
          بالشاشة لا بالذوق.** */}
      {/* **والعنوانُ الأوّلُ صار له صوت.**

          كان `text-xl font-bold` — **نصَّ متنٍ مكبَّراً مغمَّقاً**، وهو ما
          يفعله كلُّ من لا يملك خطَّ عناوين. **وقد صار كوفيّاً بالثيم** (قاعدةُ
          `:where(h1,h2,h3)`) فبقي أن يأخذ الحجمَ الذي يستحقّه.

          **وخيطُ العلامة تحته** — الشريطُ الذي يمضي من التركواز إلى الجمر،
          **وهو الطريقُ في اللوغو.** ويُستعمل في ثلاثة مواضعَ لا أكثر:
          **والتدرّجُ إن تكرّر صار زخرفةً، وإن قلّ صار توقيعاً.** */}
      <h1 className="text-2xl font-bold sm:text-3xl">{m.site.hero}</h1>
      <span aria-hidden className="brand-rule mt-2.5 mb-5 block h-0.5 w-16 rounded-badge sm:mb-6" />

      {/* **سلايدرٌ لا شريطٌ يُسحب.**

          كان شريطاً أفقياً، **ومن لا يسحب لا يرى إلّا الأولى** — فالثانيةُ
          والثالثةُ تُنشَران ولا يراهما أحد. (قرارُ المالك ٢٠٢٦-٠٨-٠٥.) */}
      <BannerSlider
        className="mb-6"
        Link={Link}
        items={banners.map((b) => ({
          id: b.id,
          title: b.title,
          imageUrl: mediaUrl(b.image_url) ?? null,
          href: b.target || undefined,
        }))}
      />

      {/* **البحثُ فوق كلّ شيءٍ ويلتصق.**

          من يعرف ما يريد لا يتصفّح. **والأقسامُ تطول** — عشرون قسماً في
          شبكةٍ من عمودين على الجوّال عشرةُ صفوف، **ومن نزل فيها ثمّ قرّر أن
          يبحث يصعد إلى الأعلى من جديد.**

          **والالتصاقُ تحت الشريط لا فوقه** (`top-[4.5rem]`): الشريطُ لاصقٌ
          أصلاً بارتفاعه، **ولو التصق البحثُ عند الصفر لَاختفى تحته.** */}
      {/* **واللاصقُ زجاجٌ كالشريط فوقه** — كان `bg-surface` مصمتاً على أرضٍ
          هي `shell`، **فيُقرأ لوحاً دخيلاً معلَّقاً في منتصف الصفحة.** */}
      <div className="sticky top-[4.5rem] z-30 -mx-1 mb-5 rounded-card bg-shell/85 px-1 py-2 backdrop-blur-xl">
      <div className="relative">
        <Input
          id="site-search"
          icon={<IconSearch size={16} />}
          placeholder={m.site.search.itemsPlaceholder}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        {q && (
          <button
            type="button"
            onClick={() => setQ("")}
            aria-label={m.common.cancel}
            className="absolute inset-block-0 end-2 my-auto flex h-7 w-7 items-center justify-center rounded-badge text-ink-muted hover:bg-page"
          >
            <IconClose size={16} />
          </button>
        )}
      </div>
      </div>

      {hits !== null ? (
        hits.length === 0 ? (
          /* **وفراغُ البحث حالُ فراغٍ لا بطاقةٌ مرتجَلة** — بالشكل نفسِه
             الذي يراه في كلّ شاشةٍ فارغة. */
          <EmptyState icon={IconSearch} title={m.site.search.empty} />
        ) : (
          /* **ونتائجُ البحث بشبكة الأقسام نفسِها** — من كتب في الحقل لا يجد
             الصفحةَ صارت شيئاً آخر تحته. */
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
            {hits.map((it) => (
              <ItemCard
                key={it.id}
                item={it}
                favorite={has(it.id)}
                onFavorite={signedIn ? toggle : undefined}
                onRequireLogin={signedIn ? undefined : () => router.push("/login?next=/")}
              />
            ))}
          </div>
        )
      ) : (
        <>
          {/* **والعنوانُ ووصفُه كتلةٌ واحدةٌ لا كتلتان متباعدتان.**

              كانا سطرين بينهما فراغان مختلفان (`mb-3` ثمّ `mb-4`) — **فيُقرأ
              الوصفُ نصّاً مستقلّاً لا شرحاً للعنوان.** والقربُ هو ما يقول
              «هذان واحد». */}
          <div className="mb-4">
            <h2 className="text-xl font-bold">{m.site.sections.title}</h2>
            <p className="mt-1 text-sm text-ink-muted">{m.site.sections.hint}</p>
          </div>
          {/* **صورةُ القسم هويّتُه — لا رمزٌ رماديّ.**

              **والسوقُ يُتصفَّح بالصور**: الزبونُ يعرف الشاورما من صورتها قبل
              أن يقرأ اسمَها، **ورمزٌ واحدٌ لعشرة أقسامٍ يجعلها كلَّها شيئاً
              واحداً** فتُمسح العينُ فوقها بلا أن تقف.

              (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «رح نرفع صورةً معبّرةً عن القسم، ما
              بدّي أيقوناتٍ عادية».) */}
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
            {sections.map((sec) => (
              <Link
                key={sec.id}
                href={`/s/${sec.id}`}
                /* **وبطاقةُ القسم كبطاقة الصنف حرفاً بحرف** — الرفعُ نفسُه
                   والحدُّ نفسُه وتكبيرُ الصورة نفسُه. **ومن مسح السوقَ بالعين
                   ثمّ فتح قسماً لا يجد الشبكةَ تغيّرت تحته.** */
                className={`group flex flex-col overflow-hidden rounded-card border border-line bg-surface transition-[transform,box-shadow,border-color] duration-[--duration-base] ease-[--ease-out] hover:-translate-y-0.5 hover:border-primary/30 hover:elev-3 ${
                  sec.count === 0 ? "opacity-60" : ""
                }`}
              >
                <span className="flex aspect-[4/3] items-center justify-center overflow-hidden bg-page">
                  {sec.image_url || sec.image_thumb_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={mediaUrl(sec.image_url ?? sec.image_thumb_url) ?? ""}
                      alt={sec.name}
                      className="h-full w-full object-cover transition-transform duration-[--duration-slow] ease-[--ease-out] group-hover:scale-[1.03]"
                    />
                  ) : (
                    /* **والأيقونةُ تبقى للزبون وحدَه** — لا لنا.

                       شاشتُنا تقول «أضف صورة» لأنّنا من يضيف، **وشاشةُ الزبون
                       لا تعرض له نقصَنا**: بطاقةٌ فارغةٌ تُقرأ عطباً. */
                    <CategoryIcon name={sec.icon} size={26} />
                  )}
                </span>
                {/* **الاسمُ يميناً والعددُ يساراً** — كما في لوحة الإدارة.
                    **وشكلٌ يختلف بين الشاشتين يجعل المراجعةَ تخميناً.** */}
                <div className="flex min-w-0 items-baseline justify-between gap-2 p-3">
                  <p className="truncate font-bold">{sec.name}</p>
                  <p className="shrink-0 text-xs text-ink-muted">
                    {m.site.sections.count.replace("{n}", fmtNum(sec.count))}
                  </p>
                </div>
              </Link>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
