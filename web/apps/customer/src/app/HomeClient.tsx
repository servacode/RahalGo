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
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { CategoryIcon, Input, IconSearch, IconClose } from "@rahalgo/ui";
import ItemCard, { type BrowseItem } from "@/components/ItemCard";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);

interface Banner {
  id: string;
  title: string;
  image_url: string | null;
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
        .catch(() => setHits([]));
    }, 300);
    return () => clearTimeout(t);
  }, [q]);

  return (
    <div>
      <h1 className="mb-4 text-2xl font-bold">{m.site.hero}</h1>

      {banners.length > 0 && (
        <div className="mb-6 flex gap-3 overflow-x-auto pb-1">
          {banners.map((b) => (
            <div key={b.id} className="relative h-36 w-80 shrink-0 overflow-hidden rounded-card">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={mediaUrl(b.image_url) ?? ""}
                alt={b.title}
                className="h-full w-full object-cover"
              />
              <span className="absolute bottom-0 start-0 end-0 bg-gradient-to-t from-ink/70 to-transparent p-2 text-sm font-bold text-white">
                {b.title}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* البحث فوق كل شيء: من يعرف ما يريد لا يتصفّح */}
      <div className="relative mb-5">
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

      {hits !== null ? (
        hits.length === 0 ? (
          <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-ink-muted">
            {m.site.search.empty}
          </p>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {hits.map((it) => (
              <ItemCard key={it.id} item={it} />
            ))}
          </div>
        )
      ) : (
        <>
          <h2 className="mb-3 text-lg font-bold">{m.site.sections.title}</h2>
          <p className="mb-4 text-sm text-ink-muted">{m.site.sections.hint}</p>
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
                className={`flex flex-col overflow-hidden rounded-card border border-line bg-surface transition-shadow hover:shadow-md ${
                  sec.count === 0 ? "opacity-60" : ""
                }`}
              >
                <span className="flex aspect-[4/3] items-center justify-center bg-page">
                  {sec.image_url || sec.image_thumb_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={mediaUrl(sec.image_url ?? sec.image_thumb_url) ?? ""}
                      alt={sec.name}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    /* **والأيقونةُ تبقى للزبون وحدَه** — لا لنا.

                       شاشتُنا تقول «أضف صورة» لأنّنا من يضيف، **وشاشةُ الزبون
                       لا تعرض له نقصَنا**: بطاقةٌ فارغةٌ تُقرأ عطباً. */
                    <CategoryIcon name={sec.icon} size={26} />
                  )}
                </span>
                <div className="min-w-0 p-3">
                  <p className="truncate font-bold">{sec.name}</p>
                  <p className="text-xs text-ink-muted">
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
