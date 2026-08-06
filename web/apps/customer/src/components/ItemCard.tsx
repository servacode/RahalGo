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
        /* **قوسٌ أعلى وحافّةٌ مستقيمةٌ أسفل.**

           (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «أريد شكلاً فريداً يستحقّ التميّز».)

           **والقوسُ على الغلاف لا على الصورة**: لو رُوّست الصورةُ وحدَها داخل
           غلافٍ بزوايا `card` **لَقُصّ القوسُ عند حدّ الغلاف** فيظهر خطّان
           متداخلان. **فالغلافُ يحمل الشكلَ والصورةُ تُقصّ فيه.**

           **والأسفلُ يبقى مستقيماً** — عليه يقف الاسمُ والوصفُ وثلاثةُ أرقامٍ
           في سطر، **ولا سطرَ يستقيم على منحنى.** */
        className={`relative flex flex-col overflow-hidden rounded-b-card rounded-t-arch border border-line bg-surface transition-shadow hover:elev-2 ${
          off ? "opacity-60" : ""
        }`}
      >
      <button
        type="button"
        onClick={open}
        className="flex flex-1 flex-col text-start"
      >
        {/* **والصورةُ مربّعةٌ لا عريضة.**

           كانت ٤:٣ — **والصحنُ مستديرٌ في الغالب**، فالإطارُ العريضُ يقصّ
           جانبيه ويترك فوقه وتحته فراغاً. **والمربّعُ يحيط بالصحن.**

           **ويخدم القوسَ**: انحناءةٌ فوق إطارٍ عريضٍ تأكل ثُلثَ ارتفاعه،
           **وفوق مربّعٍ تُقرأ تتويجاً.** */}
        <span className="relative flex aspect-square items-center justify-center bg-page">
          {img ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={img} alt="" loading="lazy" className="h-full w-full object-cover" />
          ) : (
            /* **وحرفُ الاسم لا رمزٌ رماديّ** — الرمزُ الواحدُ لعشرة أصنافٍ
               يجعلها شيئاً واحداً، **والحرفُ يفرّق بينها ويبقى لها.** */
            <span className="text-3xl font-bold text-primary-dark">{item.name.charAt(0)}</span>
          )}
          {/* **«نفد» و«نائم» خبران مختلفان** — الأوّلُ لا موعدَ له والثاني له
              موعد. **وموضعُهما فوق الصورة** كشارة القسم: تُقرأ قبل الاسم. */}
          {discounted && item.discount_percent ? (
            <span className="absolute end-3 top-3">
              <Badge variant="danger">−{item.discount_percent}%</Badge>
            </span>
          ) : !item.available ? (
            <span className="absolute end-3 top-3">
              <Badge variant="warning">{m.site.menu.unavailable}</Badge>
            </span>
          ) : item.source_closed ? (
            <span className="absolute end-3 top-3">
              <Badge variant="neutral">
                {item.source_opens_at
                  ? m.site.menu.availableFrom.replace("{t}", fmtTime(item.source_opens_at))
                  : m.site.menu.unavailable}
              </Badge>
            </span>
          ) : null}
        </span>

        {/* ══════════════════════════════════════════════════════════════
            **والوصفُ خرج من البطاقة**
            ══════════════════════════════════════════════════════════════

            كان سطراً ثالثاً تحت الاسم. **وفي بطاقةٍ بمئةٍ وستّةٍ وسبعين بكسلاً
            لا يظهر منه إلّا ثلاثُ كلماتٍ مقطوعة** — «بخبز الصاج مع…» — **وهي
            لا تُفيد ولا تُقرأ، إنّما تزاحم السعر.**

            **والوصفُ كاملٌ في النافذة** التي تُفتح بضغطةٍ واحدة، **فما ضاع
            شيءٌ وربحت البطاقةُ سطراً.**

            **وبطاقةٌ صغيرةٌ تحتمل شيئين: صورةً واسماً وسعراً.** والثالثُ
            يُنقصها ولا يزيدها. */}
        <span className="flex min-w-0 flex-1 flex-col gap-1.5 p-2.5 sm:p-3">
          {/* **والاسمُ سطرٌ واحدٌ لا سطران** — سطران يجعلان بطاقاتِ الصفّ
              مختلفةَ الارتفاع، **فتُقرأ الشبكةُ مهتزّة.** */}
          <span className="truncate text-sm font-bold">{item.name}</span>
          <span className="flex items-baseline justify-between gap-2">
            {/* **والخصمُ يُرى في البطاقة نفسِها** — لا في شاشةٍ ثانيةٍ بشكلٍ
                ثانٍ. **والمشطوبُ هو ما يجعل الخصمَ خصماً.** */}
            <span className="flex shrink-0 items-baseline gap-1.5" dir="ltr">
              {discounted && (
                <span className="text-2xs text-ink-muted line-through">
                  {fmtNum(item.price_before!)}
                </span>
              )}
              <span className={`font-bold ${discounted ? "text-success" : "text-primary-strong"}`}>
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
          className="bg-surface/90 backdrop-blur-sm"
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
