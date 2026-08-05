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
    <div className="space-y-8 sm:space-y-10">
      {/* ══════════════════════════════════════════════════════════════════
          **صدرُ الصفحة** — لوحٌ واحدٌ يحمل الجملةَ والبحثَ معاً
          ══════════════════════════════════════════════════════════════════

          # ما كان

          ثلاثةُ عناصرَ متجاورةٍ بلا رابط: **عنوانٌ عائم**، ثمّ سلايدرُ لافتات،
          ثمّ حقلُ بحثٍ ملتصقٌ في منتصف الصفحة. **ولا واحدٌ منها يقول للزائر
          الجديد أين وقع** — والحقلُ الملتصقُ خصوصاً كان يبدو **دخيلاً معلَّقاً**
          لا جزءاً من شيء.

          # ولماذا البحثُ في الصدر

          **من يعرف ما يريد لا يتصفّح.** وحقلُ البحث هو الفعلُ الأوّلُ في سوقٍ
          فيه ألفُ صنف — **فموضعُه أوّلُ الشاشة لا منتصفُها.** وكان يُبلَغ بعد
          السلايدر (١٧٥ بكسلاً) والعنوان (٦٠) — **أي بعد ثلثِ شاشةِ الجوّال.**

          # والهالةُ من الأسفل

          توهّجٌ جمريٌّ خافتٌ في قاع اللوح — **نارُ الرحّال في آخر الطريق.**
          وهو التوقيعُ الوحيدُ المسموحُ هنا: **ما يُحسّ ولا يُلحَظ حيلةً.**
          ══════════════════════════════════════════════════════════════════ */}
      <section className="surface-lit relative overflow-hidden rounded-card border border-line bg-surface px-5 py-8 sm:px-8 sm:py-12">
        <span
          aria-hidden
          className="pointer-events-none absolute inset-x-0 -bottom-24 h-48 bg-[radial-gradient(60%_100%_at_50%_100%,var(--color-accent)_0%,transparent_70%)] opacity-[0.09]"
        />
        <div className="relative mx-auto max-w-2xl text-center">
          <h1 className="text-2xl font-bold sm:text-4xl">{m.site.hero}</h1>
          <span
            aria-hidden
            className="brand-rule mx-auto mt-3 mb-6 block h-0.5 w-20 rounded-badge sm:mt-4 sm:mb-8"
          />

          {/* **والحقلُ كبيرٌ هنا لا كحقلٍ في نموذج** — هو الفعلُ الأوّل،
              **وحجمُ العنصر يقول رتبتَه** قبل أن يُقرأ ما فيه. */}
          <div className="relative">
            <Input
              id="site-search"
              icon={<IconSearch size={18} />}
              placeholder={m.site.search.itemsPlaceholder}
              value={q}
              onChange={(e) => setQ(e.target.value)}
              className="!py-3.5 !text-base"
            />
            {q && (
              <button
                type="button"
                onClick={() => setQ("")}
                aria-label={m.common.cancel}
                className="taparea absolute inset-block-0 end-2.5 my-auto flex h-8 w-8 items-center justify-center rounded-badge text-ink-muted transition-colors hover:bg-raised hover:text-ink"
              >
                <IconClose size={16} />
              </button>
            )}
          </div>
        </div>
      </section>

      {/* **واللافتاتُ بعد الصدر** — خبرٌ يُقرأ بنظرة، **ولا تسبق الفعلَ.** */}
      {banners.length > 0 && (
        <BannerSlider
          Link={Link}
          items={banners.map((b) => ({
            id: b.id,
            title: b.title,
            imageUrl: mediaUrl(b.image_url) ?? null,
            href: b.target || undefined,
          }))}
        />
      )}

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
          {/* **وعنوانُ القسم له عينٌ صغيرةٌ فوقه.**

              **سطرٌ جمريٌّ قصيرٌ ثمّ العنوان** — وهو ما يفصل قسماً عن قسمٍ في
              صفحةٍ طويلةٍ **بلا خطٍّ يقطعها عرضاً.** والخطُّ العارضُ يقول
              «انتهى»، **والعينُ تقول «بدأ شيءٌ جديد».** */}
          <div className="mb-5 flex items-end justify-between gap-4">
            <div>
              <span aria-hidden className="mb-2 block h-0.5 w-8 rounded-badge bg-accent" />
              <h2 className="text-xl font-bold sm:text-2xl">{m.site.sections.title}</h2>
              <p className="mt-1 text-sm text-ink-muted">{m.site.sections.hint}</p>
            </div>
          </div>
          {/* **صورةُ القسم هويّتُه — لا رمزٌ رماديّ.**

              **والسوقُ يُتصفَّح بالصور**: الزبونُ يعرف الشاورما من صورتها قبل
              أن يقرأ اسمَها، **ورمزٌ واحدٌ لعشرة أقسامٍ يجعلها كلَّها شيئاً
              واحداً** فتُمسح العينُ فوقها بلا أن تقف.

              (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «رح نرفع صورةً معبّرةً عن القسم، ما
              بدّي أيقوناتٍ عادية».) */}
          {/* ══════════════════════════════════════════════════════════
              **وإيقاعُ الشبكة ليس واحداً**
              ══════════════════════════════════════════════════════════

              كانت خمسةَ أعمدةٍ من متساوياتٍ — **صفٌّ رتيبٌ تمسحه العينُ ولا
              تقف عند شيء.** ولا شيءَ في الصفحة يقول «ابدأ من هنا».

              **والأوّلُ يأخذ عمودين وصفّين**: قسمٌ واحدٌ يتصدّر فتقع عليه
              العينُ أوّلاً، **ثمّ تنزل إلى البقيّة.** وهو الفرقُ بين رفٍّ
              مرتَّبٍ وجدولِ بيانات.

              **ولا يقع هذا على الجوّال**: عمودان لا يحتملان تصديراً،
              **وبطاقةٌ بضعفِ الحجم في شاشةٍ ضيّقةٍ تدفع البقيّةَ تحت الطيّة.**
              ══════════════════════════════════════════════════════════ */}
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4">
            {sections.map((sec, i) => (
              <Link
                key={sec.id}
                href={`/s/${sec.id}`}
                /* **وبطاقةُ القسم كبطاقة الصنف حرفاً بحرف** — الرفعُ نفسُه
                   والحدُّ نفسُه وتكبيرُ الصورة نفسُه. **ومن مسح السوقَ بالعين
                   ثمّ فتح قسماً لا يجد الشبكةَ تغيّرت تحته.** */
                className={`group flex flex-col overflow-hidden rounded-card border border-line bg-surface transition-[transform,box-shadow,border-color] duration-[--duration-base] ease-[--ease-out] hover:-translate-y-0.5 hover:border-primary/30 hover:elev-3 ${
                  i === 0 ? "sm:col-span-2 sm:row-span-2" : ""
                } ${sec.count === 0 ? "opacity-60" : ""}`}
              >
                <span
                  className={`flex items-center justify-center overflow-hidden bg-page ${
                    i === 0 ? "aspect-[4/3] sm:aspect-square" : "aspect-[4/3]"
                  }`}
                >
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
                {/* **والمتصدِّرُ يكبر اسمُه** — الحجمُ يقول الرتبةَ، **وبطاقةٌ
                    بضعفِ المساحة واسمٍ بحجم أخواتها تُقرأ خطأً في التنضيد** لا
                    تصديراً مقصوداً. */}
                <div className="flex min-w-0 items-baseline justify-between gap-2 p-3 sm:p-4">
                  <p className={`truncate font-bold ${i === 0 ? "sm:text-lg" : ""}`}>{sec.name}</p>
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
