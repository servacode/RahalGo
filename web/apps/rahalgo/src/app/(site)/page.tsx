/**
 * **الصفحةُ الرئيسيّة — واجهةُ الموقع، لا صورةٌ في خلفيّة.**
 *
 * # ما كانت
 *
 * **كانت `/` صفحةَ التسوّق نفسَها**، ثمّ أُفرغت بقرار المالك (٢٠٢٦-٠٨-٠٦)،
 * **ثمّ صارت صورةً واحدةً تُرفع من الإعدادات** (٢٠٢٦-٠٨-٠٩).
 *
 * **والصورةُ لم تُرفع قطّ**: ناداها القياسُ فردَّت `{"image":null}` —
 * **فكانت الصفحةُ بيضاءَ عند كلّ زائرٍ منذ كُتبت.** ولم تكن في اللوحة شاشةٌ
 * ترفعها أصلاً.
 *
 * # وما صارت
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦: «الموقع لازم يكون الرئيسية · تسوّق · من نحن ·
 *  تواصل معنا · ميزاتنا · لماذا اسم منصّتنا… رح نلغي موضوع الصورة ونخلّيه
 *  موقعاً احترافيّاً».)
 *
 * **صفحةٌ واحدةٌ طويلةٌ فيها كلُّ ما يعرّف بالمنصّة** — واختارها المالك على
 * صفحاتٍ متفرّقة: **الزائرُ الذي لا يعرفنا لا يضغط ليعرف، يمرّر.**
 *
 * # ولماذا تُقدَّم من الخادم
 *
 * **لا نداءَ فيها ولا حالة** — نصٌّ وأيقونات. **وصفحةٌ تُبنى في المتصفّح
 * يقرؤها غوغل فارغةً**، وهي الصفحةُ الوحيدةُ التي يهمّنا أن تُفهرَس.
 * **واسمُ المنصّة وحدَه يأتي من الخادم** (`fetchPlatform`) كما في الغلاف.
 *
 * # والحشوةُ الجانبيّة هنا لا في السوق
 *
 * **قاعدةُ المالك «لا حشوةَ يميناً ويساراً» للسوق** — وأكّدها للموقع
 * التعريفيّ بتخصيص (٢٠٢٦-٠٨-١٦): **الأشرطةُ تبلغ الحافّة، والنصُّ داخلها
 * يُحدَّد.** **وسطرٌ يمتدّ عبر شاشةٍ عريضةٍ لا يُقرأ** — تضيع العينُ في
 * رجوعها إلى أوّل السطر التالي.
 */

import Link from "next/link";
import { getMessages, defaultLocale, withPlatform } from "@rahalgo/i18n";
import {
  fetchPlatform,
  BannerSlider,
  ButtonLink,
  IconMoto,
  IconLocation,
  IconWallet,
  IconChat,
  IconPromos,
  IconSupport,
  IconSearch,
  IconCart,
  IconDriver,
  IconStore,
  IconUsers,
} from "@rahalgo/ui";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const m = getMessages(defaultLocale);
const H = m.site.homePage;

/** **والأيقوناتُ من لوسيد بأسماءٍ موحّدة** — ولا نوعَ مصدَّراً لها، فيُوصف بما يُستعمل. */
type Icon = React.ComponentType<{ size?: number; className?: string }>;

/** **بطاقةٌ واحدةٌ تخدم الميزات والانضمام** — أيقونةٌ وعنوانٌ وسطر. */
function Tile({ Icon, title, body }: { Icon: Icon; title: string; body: string }) {
  return (
    <div className="surface flex flex-col gap-2 p-5">
      <Icon size={28} className="text-accent-text" />
      <h3 className="heading-card">{title}</h3>
      <p className="text-sm text-ink-muted">{body}</p>
    </div>
  );
}

/**
 * **شريطٌ يبلغ الحافّة ونصُّه محدَّد.**
 *
 * **و`-mx-3` تُلغي حشوةَ الغلاف** — الغلافُ يحشو كلَّ صفحاته `px-3`،
 * **وشريطٌ ملوّنٌ يقف قبل الحافّة بثلاثة أرباعِ سنتيمترٍ يُقرأ عطباً**، لا
 * تصميماً.
 */
function Band({
  children,
  tinted,
  flush,
}: {
  children: React.ReactNode;
  tinted?: boolean;
  /**
   * **بلا حشوةٍ ولا حدٍّ للعرض** — (قاعدةُ المالك، أعادها ٢٠٢٦-٠٨-١٧:
   * «ما زال هناك بادينغ على اليمين»).
   *
   * **والافتتاحيّةُ وحدَها**: الأقسامُ تحتها بطاقاتٌ في شبكة، **وشبكةٌ
   * تلامس الحافّةَ تُقرأ مقصوصة.**
   */
  flush?: boolean;
}) {
  return (
    <section
      className={`-mx-3 py-14 sm:-mx-4 sm:py-20 ${flush ? "" : "px-3 sm:px-4"} ${tinted ? "bg-raised" : ""}`}
    >
      <div className={flush ? "w-full" : "mx-auto w-full max-w-6xl"}>{children}</div>
    </section>
  );
}

/** **عنوانُ قسمٍ وسطرُه** — واحدٌ لكلّ شريطٍ فلا تفترق أشكالُها. */
function BandHead({ title, lead }: { title: string; lead?: string }) {
  return (
    <div className="mb-8 text-center">
      <h2 className="heading-page">{title}</h2>
      {lead && <p className="mt-2 text-sm text-ink-muted">{lead}</p>}
    </div>
  );
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لافتاتُ العرض الافتتاحيّ — من مكانها لا من مكانٍ ثانٍ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٧: «أوّلُ قسمٍ خلّيه سلايدر بالرئيسيّة، نعرض عليه
 *  صوراً يطلع احترافيّ أكثر».)
 *
 * **ولافتاتُها غيرُ لافتات التسوّق** — (تصحيحُ المالك ٢٠٢٦-٠٨-١٧: «بانرات
 * صفحة التسوّق مختلفة برأيي عن الرئيسيّة»): **صفحةٌ تعرّف بالمنصّة وصفحةٌ
 * تبيع، وصورةٌ تصلح لإحداهما لا تصلح للأخرى.**
 *
 * **والجدولُ والشاشةُ والسلايدرُ واحدةٌ ويفرّقها عمودُ موضع** — **وجدولان
 * بالحقول نفسِها يعنيان شاشتين ومسارين** لأجل كلمة.
 *
 * **وفشلُ الجلب يُرجع فراغاً**: الرئيسيّةُ تبقى بعنوانها ودعوتها،
 * **وشريطُ خطأٍ في أوّل ما يراه زائرٌ أسوأُ من سلايدرٍ غائب.**
 */
async function fetchBanners(): Promise<Slide[]> {
  try {
    const res = await fetch(`${API}/api/v1/public/banners?at=home`, {
      next: { revalidate: 300 },
    });
    if (!res.ok) return [];
    const j = (await res.json()) as {
      data?: { banners?: { id: string; title: string; image_url: string | null; target: string | null }[] };
    };
    return (j.data?.banners ?? [])
      .filter((b) => b.image_url)
      .map((b) => ({
        id: b.id,
        title: b.title,
        imageUrl: API + b.image_url,
        href: b.target || undefined,
      }));
  } catch {
    // @empty-ok — انظر أعلاه: الفراغُ قرارٌ لا صمت.
    return [];
  }
}

interface Slide {
  id: string;
  title: string;
  imageUrl: string;
  href?: string;
}

export default async function HomePage() {
  const [brand, slides] = await Promise.all([fetchPlatform(API), fetchBanners()]);
  const name = brand.name;

  const features: [Icon, string, string][] = [
    [IconMoto, H.f1t, H.f1d],
    [IconLocation, H.f2t, H.f2d],
    [IconWallet, H.f3t, H.f3d],
    [IconChat, H.f4t, H.f4d],
    [IconPromos, H.f5t, H.f5d],
    [IconSupport, H.f6t, H.f6d],
  ];
  const steps: [Icon, string, string][] = [
    [IconSearch, H.h1t, H.h1d],
    [IconCart, H.h2t, H.h2d],
    [IconMoto, H.h3t, H.h3d],
  ];
  const joins: [Icon, string, string][] = [
    [IconDriver, H.j1t, H.j1d],
    [IconStore, H.j2t, H.j2d],
    [IconUsers, H.j3t, H.j3d],
  ];

  return (
    <>
      {/* ══════════════════════════════════════════════════════════════
          **العرضُ الافتتاحيّ — جملةٌ تقول ما نفعل وزرٌّ يبدأ**
          ══════════════════════════════════════════════════════════════

          **ولا زرَّ «حمّل التطبيق» بعد**: التطبيقاتُ لم تُنشر على غوغل بلاي
          **وزرٌّ يعد بما لا يوجد يُفقد الثقةَ في أوّل شاشة.** يُضاف يومَ
          النشر. */}
      <Band flush>
        {/* **والصورةُ فوق الكلام** — **وعنوانٌ يُكتب فوق صورةٍ يرفعها
            صاحبُها لا يُضمَن أن يُقرأ**: صورةٌ فاتحةٌ تبتلع الحرفَ الأبيضَ
            وداكنةٌ تبتلع الأسود. **فالسلايدرُ يعلو والكلامُ تحته على أرضِ
            الصفحة** — كلٌّ منهما يُقرأ على حدة.

            **ولا يُرسم إن لم تُرفع لافتة** — إطارٌ فارغٌ يُقرأ عطباً. */}
        {slides.length > 0 && (
          <BannerSlider className="mb-10" items={slides} Link={Link} />
        )}
        {/* **والنصُّ إلى اليمين لا في الوسط** — (طلبُ المالك ٢٠٢٦-٠٨-١٧:
            «النصّ كامل يصبح محاذاة إلى اليمين»).

            **و`items-start` لا `items-end`**: البدايةُ منطقيّةٌ تنقلب مع
            اللغة — **ومن كتب «يمين» بالحرف كسر صفحتَه يومَ تُقرأ
            بالإنجليزيّة.** */}
        <div className="flex flex-col items-start text-start">
          {/* **ولا شعارَ في العرض الافتتاحيّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٧:
              «اللوغو شيلو من هون»).

              **وهو في الشريط فوقَه مباشرةً** — وشعارٌ مرّتين في شاشةٍ
              واحدةٍ يزاحم العنوانَ الذي جاء الزائرُ ليقرأه. */}
          {/* **واسمُ المنصّة يُحقن ولا يُكتب** — قاعدةُ المالك: من بدّله من
              اللوحة بدّله في كلّ موضع. */}
          {/* ══════════════════════════════════════════════════════════
              **وكلمتان في العنوان لهما لونُهما**
              ══════════════════════════════════════════════════════════

              (اختيارُ المالك ٢٠٢٦-٠٨-١٧: أخضرُ مزرقٌّ للمدينة وبرتقاليٌّ
               للاسم.)

              **والعنوانُ يبقى جملةً واحدةً في المعجم** بموضعَين يُملآن —
              **وتقطيعُه ثلاثةَ مفاتيحَ يجعل من يبدّله يبدّل ثلاثةً
              ويخطئ في الفراغات بينها.**

              **والاسمُ يُحقن هنا في عنصرٍ ملوّنٍ مستقلّ**، فلا تصلح
              `withPlatform` التي تردّ نصّاً واحداً. */}
          <h1 className="heading-hero">
            {(() => {
              /* @platform-ok — يُحقن أدناه في عنصره الملوّن. */
              const [head, rest = ""] = H.heroTitle.split("{city}");
              const [mid, tail = ""] = rest.split("{platform}");
              return (
                <>
                  {head}
                  <span className="text-gradient-city">{H.cityName}</span>
                  {mid}
                  <span className="text-gradient-platform">{name}</span>
                  {tail}
                </>
              );
            })()}
          </h1>
          {/* **وثلاثةُ أسطرٍ لا فقرةٌ واحدة** — (نصُّ المالك ٢٠٢٦-٠٨-١٧):
              كلُّ سطرٍ وعدٌ قائمٌ بنفسه، **وجمعُها في فقرةٍ يجعلها تُقرأ
              كلاماً متّصلاً فيضيع الثالث.** */}
          <div className="hero-lead mt-5 flex max-w-prose flex-col gap-1 text-ink-muted">
            {/* **وثلاثُ كلماتٍ في مربّعاتها** — (طلبُ المالك ٢٠٢٦-٠٨-١٧).
                **والأيقونةُ تسبق الكلمةَ في القراءة**: من رأى متجراً
                وعدسةً وسلّةً عرف الخطواتِ الثلاثَ قبل أن يقرأها. */}
            <div className="flex flex-wrap items-center gap-2">
              {[
                [IconStore, H.heroChip1],
                [IconSearch, H.heroChip2],
                [IconCart, H.heroChip3],
              ].map(([Ico, label]) => {
                const I = Ico as Icon;
                return (
                  <span
                    key={label as string}
                    className="surface flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-ink"
                  >
                    <I size={18} className="text-accent-text" />
                    {label as string}
                  </span>
                );
              })}
            </div>
            <p>{H.heroLine1}</p>
            <p>{H.heroLine2}</p>
            <p>{H.heroLine3}</p>
          </div>
          {/* **والدعوةُ إلى السوق تُخفى مع بابَيه الآخرَين** — (طلبُ
              المالك ٢٠٢٦-٠٨-١٧). **ودعوةٌ باقيةٌ بعد إخفاء الزرَّين
              تنقض الإخفاءَ كلَّه**، وهي أظهرُ الثلاثة.

              **و«من نحن» يصير الزرَّ الرئيسيَّ حينَها** — فلا تبقى
              الواجهةُ بلا وجهةٍ تُضغط. */}
          <div className="mt-8 flex flex-wrap items-center justify-start gap-3">
            {/* **ولا زرَّ «من نحن» هنا** — (قرارُ المالك ٢٠٢٦-٠٨-١٧:
                «احذفها من الصفحة، لا أريدها»).

                **وهو في الشريط العلويّ** — وبابان لصفحةٍ واحدةٍ في شاشةٍ
                واحدة، **والافتتاحيّةُ موضعُ دعوةٍ واحدةٍ لا قائمةِ
                أبواب.** */}
            {brand.showShop && (
              <ButtonLink href="/shop" size="lg">
                {H.ctaShop}
              </ButtonLink>
            )}
          </div>
        </div>
      </Band>

      {/* **ميزاتنا** — (طلبُ المالك ٢٠٢٦-٠٨-١٦). */}
      <Band tinted>
        <div id="features" className="scroll-mt-20">
          <BandHead title={H.featuresTitle} lead={withPlatform(H.featuresLead, name)} />
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {features.map(([Icon, title, body]) => (
              <Tile key={title} Icon={Icon} title={title} body={body} />
            ))}
          </div>
        </div>
      </Band>

      {/* **كيف تطلب** — **والرقمُ هنا يعني ترتيباً حقيقيّاً**: لا تُتابع قبل
          أن تطلب. */}
      <Band>
        <BandHead title={H.howTitle} lead={H.howLead} />
        <ol className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {steps.map(([Icon, title, body], i) => (
            <li key={title} className="surface flex flex-col gap-2 p-5">
              <span className="flex items-center gap-2 text-accent-text">
                <Icon size={28} />
                <span className="figure">{i + 1}</span>
              </span>
              <h3 className="heading-card">{title}</h3>
              <p className="text-sm text-ink-muted">{body}</p>
            </li>
          ))}
        </ol>
      </Band>

      {/* ══════════════════════════════════════════════════════════════
          **لماذا اسمُنا**
          ══════════════════════════════════════════════════════════════

          **وهذا النصُّ مسوّدةٌ لا قصّة**: سألتُ المالكَ عن سبب اختياره
          الاسمَ فقال «ابدأ وسنعدّل لاحقاً» — **فكُتب على معنى الكلمتين لا
          على قصّةٍ يرويها هو.** موضعُه المعجمُ (`site.homePage.nameP*`)
          فيُبدَّل بسطرٍ واحد. */}
      <Band tinted>
        <div id="name" className="mx-auto max-w-3xl scroll-mt-20 text-center">
          <BandHead title={withPlatform(H.nameTitle, name)} />
          <div className="flex flex-col gap-4 text-ink-muted">
            <p>{H.nameP1}</p>
            <p>{H.nameP2}</p>
            <p className="text-ink">{H.nameP3}</p>
          </div>
        </div>
      </Band>

      {/* **انضمّ إلينا** — الثلاثةُ أدوارٍ تقصد `/join` نفسَها. */}
      <Band>
        <BandHead title={H.joinTitle} lead={H.joinLead} />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {joins.map(([Icon, title, body]) => (
            <Tile key={title} Icon={Icon} title={title} body={body} />
          ))}
        </div>
        <div className="mt-8 text-center">
          <ButtonLink href="/join" size="lg">
            {H.joinCta}
          </ButtonLink>
        </div>
      </Band>

      {/* **وتواصلٌ مختصرٌ يوصل إلى صفحته** — **والصفحةُ فيها النموذجُ
          والأرقام**، ونسخُها هنا يجعل نصّاً واحداً في موضعين يفترقان. */}
      <Band tinted>
        <div className="mx-auto max-w-2xl text-center">
          <BandHead title={H.contactTitle} lead={H.contactLead} />
          <Link
            href="/contact"
            className="text-sm font-medium text-accent-text underline-offset-4 hover:underline"
          >
            {H.contactCta}
          </Link>
        </div>
      </Band>
    </>
  );
}
