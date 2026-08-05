"use client";

/**
 * بطاقةُ صنفٍ في التصفّح — **بلا مصدرها.**
 *
 * الزبونُ يشتري «من رحّال» لا «من مطعم فلان»: **الاسمُ والصورةُ والسعرُ وحدَها،
 * ولا شيءَ يدلّ على المطبخ.**
 *
 * **وسببُ الغياب يُقال بموعده**: «متاح من ١٠ صباحاً» موعدٌ يُعاد إليه، و«غير
 * متاح» طريقٌ مسدود.
 *
 * # وشكلُها شكلُ القسم — والصنفِ في اللوحة
 *
 * كانت **سطراً أفقيّاً بمصغَّرةٍ ٦٤ بكسل** بينما القسمُ فوقها بطاقةٌ بصورةٍ
 * تملأ عرضَها. **فيُقرأ القسمُ سلعةً والصنفُ سطرَ جدول** — وهو مقلوب: الصنفُ
 * هو ما يُشترى.
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «شكلُ العرض بالموقع للأصناف يجب أن يكون موحّداً».)
 *
 * **والشكلُ الواحدُ ليس ذوقاً**: من يمسح السوقَ بالعين ثمّ يفتح قسماً **لا
 * يُعيد تعلُّمَ أين يقع الاسمُ وأين السعرُ وأين الحال**. وشاشةُ اللوحة تعرض
 * البطاقةَ نفسَها، **فمراجعةُ ما يراه الزبونُ لا تصير تخميناً.**
 */

import { useCallback, useState } from "react";
import { Modal, LoadingState, FavoriteButton } from "@rahalgo/ui";
import ItemClient, { type Group } from "@/app/i/[id]/ItemClient";
import { api } from "@/lib/api";
import { getMessages, defaultLocale, fmtNum, fmtTime } from "@rahalgo/i18n";
import { Badge } from "@rahalgo/ui";
import { mediaUrl } from "@/lib/api";

const m = getMessages(defaultLocale);

export interface BrowseItem {
  id: string;
  name: string;
  description: string;
  price: number;
  /**
   * **الأصلُ للبطاقة، والمصغَّرةُ بديلُها.**
   *
   * المصغَّرةُ حدُّها ٤٠٠ بكسل، **وبطاقةٌ تمطّها تبهت.** والمصغَّرةُ تبقى لما
   * رُفع قبل هذا التغيير ولنافذةِ الصنف.
   */
  image_url?: string | null;
  image_thumb_url: string | null;
  available: boolean;
  source_closed: boolean;
  source_opens_at: string | null;
  section_id: string;
  section_name: string;
  /**
   * **سعرُ ما قبل الخصم — وفارغٌ حين لا خصم.**
   *
   * **والمشطوبُ هو ما يجعل الخصمَ خصماً**: «٧٧٬٢٥٠» وحدَه رقمٌ، **و«١٠٣٬٠٠٠»
   * مشطوبةً فوقه توفيرٌ يُرى.**
   */
  price_before?: number | null;
  /** نسبةُ الحسم — **تُقرأ بلمحةٍ قبل أن يُقارَن الرقمان.** */
  discount_percent?: number | null;
}

export default function ItemCard({
  item,
  favorite = false,
  onFavorite,
  onRequireLogin,
}: {
  item: BrowseItem;
  /**
   * **حالُ القلب تأتي من فوق لا تُجلب هنا.**
   *
   * **وبطاقةٌ تجلب مفضّلتَها بنفسها تعني عشرين نداءً في شبكةٍ من عشرين
   * بطاقة** — والقائمةُ واحدةٌ لكلّ الصفحة. (انظر `useFavorites`.)
   */
  favorite?: boolean;
  onFavorite?: (itemID: string) => void;
  /** **ومن لم يدخل يُساق إلى الدخول لا يُمنع صامتاً.** */
  onRequireLogin?: () => void;
}) {
  /**
   * **الصنفُ يُفتح في نافذةٍ لا في صفحة.**
   *
   * الزبونُ في قسمٍ يتصفّح عشرةَ أصناف: يفتح واحداً، يقرأ خياراتِه، **يرجع
   * ليفتح غيرَه** — وكلُّ رجعةٍ تعيده إلى أعلى القائمة فيبحث عن موضعه.
   * **والنافذةُ تُبقيه حيث هو.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «بعد أن يدخل إلى قسم معيّن ويضغط على نوع،
   * هنا تكون نافذة منبثقة».)
   */
  const [openItem, setOpenItem] = useState<{ item: BrowseItem; modifiers: Group[] } | null>(null);
  const [loading, setLoading] = useState(false);

  const open = useCallback(async () => {
    // **الخياراتُ تُجلب عند الفتح** — بطاقةُ التصفّح لا تحملها، وجلبُها لكلّ
    // بطاقةٍ في القائمة عشرةُ نداءاتٍ لينظر في واحد.
    setLoading(true);
    setOpenItem(null);
    try {
      const d = await api<{ item?: BrowseItem; modifiers?: Group[] }>(
        `/api/v1/public/items/${item.id}`,
      );
      if (d.item) setOpenItem({ item: d.item, modifiers: d.modifiers ?? [] });
    } catch {
      /* تعذّر الجلب — تُغلق النافذة ولا تُفتح فارغة */
    } finally {
      setLoading(false);
    }
  }, [item.id]);

  const img = mediaUrl(item.image_url ?? item.image_thumb_url);
  const off = !item.available || item.source_closed;
  // **وخصمٌ بلا سعرٍ سابقٍ لا يُعرض** — الرقمُ وحدَه لا يقول إنّه أرخص.
  const discounted = !!item.price_before && item.price_before > item.price;

  return (
    <>
      {/* **البطاقةُ غلافٌ والزرُّ داخلَه.**

          كانت البطاقةُ نفسُها `<button>`، **وقلبُ المفضّلة زرٌّ** — وزرٌّ
          داخل زرٍّ لا يجوز: المتصفّحُ يفكّه كما يشاء **فتضيع إحدى
          الضغطتين.** */}
      <div
        /* **والبطاقةُ ترتفع لا تُضاء وحدَها.**

           كان `hover:elev-2` — ظلٌّ يزيد ولا شيءَ يتحرّك، **وهو أثرٌ لا
           يُلحَظ في شبكةٍ من عشرين.** والرفعُ بكسلين مع الظلّ **يقول
           «هذه تحت مؤشّرك»** قبل أن تُقرأ.

           **والحدُّ يتلوّن تركوازاً** — لا يسمك ولا يبيضّ: حدٌّ يغلظ عند
           التحويم يُزحزح ما حوله بكسلاً، **وشبكةٌ تهتزّ عند مرور الفأرة.**

           `group` كي تعرف الصورةُ في جوفها متى يُحوَّم على البطاقة. */
        className={`group relative flex flex-col overflow-hidden rounded-card border border-line bg-surface transition-[transform,box-shadow,border-color] duration-[--duration-base] ease-[--ease-out] hover:-translate-y-0.5 hover:border-primary/30 hover:elev-3 ${
          off ? "opacity-60" : ""
        }`}
      >
      <button
        type="button"
        onClick={open}
        className="flex flex-1 flex-col text-start"
      >
        {/* **الصورةُ أوّلاً وتملأ العرض** — كبطاقة القسم فوقها تماماً. */}
        <span className="relative flex aspect-[4/3] items-center justify-center overflow-hidden bg-page">
          {img ? (
            // eslint-disable-next-line @next/next/no-img-element
            /* **والصورةُ تتقدّم قليلاً عند التحويم.**

               **الصورةُ هي البضاعة** — وحركتُها هي ما يقول «انظر إليّ»،
               **وثلاثةٌ بالمئةِ تكبيراً تُحسّ ولا تُلحَظ حيلةً.** وهي داخل
               `overflow-hidden` فلا تتجاوز زواياها. */
            <img
              src={img}
              alt=""
              loading="lazy"
              className="h-full w-full object-cover transition-transform duration-[--duration-slow] ease-[--ease-out] group-hover:scale-[1.03]"
            />
          ) : (
            /* **وحرفُ الاسم لا رمزٌ رماديّ** — الرمزُ الواحدُ لعشرة أصنافٍ
               يجعلها شيئاً واحداً، **والحرفُ يفرّق بينها ويبقى لها.** */
            <span className="text-3xl font-bold text-primary-dark">{item.name.charAt(0)}</span>
          )}
          {/* **وحجابٌ متدرّجٌ من الأعلى تحت الشارات.**

              **الشارةُ على صورةٍ لا يُعرف لونُها**: طبقٌ فاتحٌ يبتلع «−٢٥٪»
              وطبقٌ داكنٌ يبتلع «نفد». **وشارةٌ تُقرأ في صورةٍ وتختفي في
              أخرى ليست شارة.**

              **والحجابُ من الأعلى وحدَه** حيث تقف الشارات — ولو عمّ الصورةَ
              **لَأبهتها كلَّها**، وهي البضاعة. */}
          <span
            aria-hidden
            /* **والحجابُ صار من الحبر لا من الأرض.**

               كان `from-shell/55` يُعتم أعلى الصورة. **ولمّا صارت الأرضُ
               فاتحةً صار يُبيّضه** — وبياضٌ على صورة طعامٍ يغسلها. **والحبرُ
               داكنٌ في اللوحتين**، فيبقى الحجابُ حجاباً. */
            className="pointer-events-none absolute inset-x-0 top-0 h-14 bg-gradient-to-b from-scrim/45 to-transparent"
          />

          {/* **«نفد» و«نائم» خبران مختلفان** — الأوّلُ لا موعدَ له والثاني له
              موعد. **وموضعُهما فوق الصورة** كشارة القسم: تُقرأ قبل الاسم. */}
          {discounted && item.discount_percent ? (
            <span className="absolute end-1.5 top-1.5">
              <Badge variant="danger">−{item.discount_percent}%</Badge>
            </span>
          ) : !item.available ? (
            <span className="absolute end-1.5 top-1.5">
              <Badge variant="warning">{m.site.menu.unavailable}</Badge>
            </span>
          ) : item.source_closed ? (
            <span className="absolute end-1.5 top-1.5">
              <Badge variant="neutral">
                {item.source_opens_at
                  ? m.site.menu.availableFrom.replace("{t}", fmtTime(item.source_opens_at))
                  : m.site.menu.unavailable}
              </Badge>
            </span>
          ) : null}
        </span>

        {/* **الاسمُ يميناً والسعرُ يساراً** — كالقسم: اسمُه يميناً وعددُه يساراً. */}
        <span className="flex min-w-0 flex-1 flex-col gap-1 p-3">
          <span className="truncate font-bold">{item.name}</span>
          <span className="flex items-baseline justify-between gap-2">
            <span className="truncate text-xs text-ink-muted">{item.description}</span>
            {/* **والخصمُ يُرى في البطاقة نفسِها** — لا في شاشةٍ ثانيةٍ بشكلٍ
                ثانٍ. **والمشطوبُ هو ما يجعل الخصمَ خصماً.** */}
            <span className="flex shrink-0 items-baseline gap-1.5" dir="ltr">
              {discounted && (
                <span className="text-2xs text-ink-muted line-through">
                  {fmtNum(item.price_before!)}
                </span>
              )}
              {/* **والسعرُ المخصومُ جمرةٌ لا خُضرة.**

                  كان أخضرَ — **ولونُ النجاح في هذه المنصة «تمّ» و«وصل»**،
                  فسعرٌ أخضرُ يُقرأ حالةَ طلبٍ لا توفيراً. **والجمرُ لونُ
                  الخصم وشارتِه والزرّ** — وثلاثتُها شيءٌ واحد: اضغط. */}
              <span className={`font-bold ${discounted ? "text-accent-text" : "text-primary-strong"}`}>
                {fmtNum(item.price)} {m.common.currency}
              </span>
            </span>
          </span>
        </span>
      </button>

      {/* **والقلبُ فوق الصورة في زاويتها.**

          **وكان غائباً عن هذه البطاقة كلَّها** — وهي شكلُ الأصناف في السوق
          والأقسام والبحث، **فالمفضّلةُ لا تُملأ إلّا من شاشة المتجر وحدَها**
          وهي آخرُ ما يفتحه المتصفّح.

          **وموضعُه بدايةُ السطر لا نهايتُه**: نهايتُه للشارة («نفد» أو
          «متاح من ١٠»)، **وشيئان في زاويةٍ واحدةٍ يتزاحمان.** */}
      {/* **ولا قلبَ حيث لا يُحفظ** — البطاقةُ تُستعمل في مواضعَ لا مفضّلةَ
          فيها (معاينةُ اللوحة)، **وزرٌّ لا يفعل شيئاً يُقرأ عطباً.**

          **والزائرُ يراه ويُساق إلى الدخول**: أوّلُ صياغةٍ أخفته عنه لأنّ
          `onFavorite` تغيب، **فيفوته أنّ الميزةَ له إن دخل.** */}
      {(onFavorite || onRequireLogin) && (
      <span className="absolute start-1.5 top-1.5">
        <FavoriteButton
          size="sm"
          itemID={item.id}
          on={favorite}
          onToggle={onFavorite ?? (() => undefined)}
          onRequireLogin={onRequireLogin}
          className="border-transparent bg-scrim/45 text-on-solid backdrop-blur-md"
        />
      </span>
      )}
      </div>

      <Modal
        open={loading || !!openItem}
        onClose={() => setOpenItem(null)}
        title={openItem?.item.name ?? item.name}
        size="lg"
      >
        {loading || !openItem ? (
          <LoadingState />
        ) : (
          <ItemClient item={openItem.item} modifiers={openItem.modifiers} />
        )}
      </Modal>
    </>
  );
}
