"use client";

/**
 * صفحة الإشعارات الكاملة — **مكوّن واحد مركزي** ترثه كل اللوحات وموقع الزبون.
 *
 * الجرس يعرض آخر 30 في قائمة منسدلة ضيّقة: يكفي للحظة، ولا يكفي لمن غاب يومين
 * ويريد أن يعرف ما فاته. هنا الأرشيف كاملاً بترشيح بالنوع وتجميع بالتاريخ.
 */

import { useCallback, useMemo, useState } from "react";
import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime, fmtLongDate } from "@rahalgo/i18n";
import { PageContainer, PageHeader, EmptyState, LoadingState } from "./layout";
import { Chips } from "./navigation";
import { Alert } from "./feedback";
import { Button } from "./components";
import { useLiveData, emitLocal, READ_EVENT, type AppNotification } from "./Notifications";
import {
  IconBell,
  IconOrder,
  IconSupport,
  IconWallet,
  IconStar,
  IconLink,
  IconUser,
  IconPromos,
} from "./icons";

const m = getMessages(defaultLocale);
const N = m.shared.notifications;

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;
type LinkType = ComponentType<{
  href: string;
  className?: string;
  children: ReactNode;
  onClick?: () => void;
}>;

interface Feed {
  items: AppNotification[];
  unread: number;
  counts: Record<string, number>;
}

/** أيقونة ولون لكل نوع — مصدر واحد يخدم الصفحة والجرس. */
const KINDS: Record<string, { icon: ComponentType<{ size?: number; className?: string }>; tone: string }> = {
  // **والنغمةُ صبغةٌ ونصٌّ من الدلالة نفسِها** — كانت ثلاثةٌ خارجَ اللغة:
  // `bg-primary-light` تعبئةٌ مصمتة، و`text-accent-dark` درجةٌ لا دلالة،
  // و`bg-page` لونُ صفحةٍ لا لونُ نوع. **فتُقرأ الستّةُ خمسةً وواحداً غريباً.**
  order: { icon: IconOrder, tone: "text-primary bg-primary-tint" },
  ticket: { icon: IconSupport, tone: "text-danger bg-danger-tint" },
  wallet: { icon: IconWallet, tone: "text-success bg-success-tint" },
  rating: { icon: IconStar, tone: "text-accent bg-accent-tint" },
  lead: { icon: IconLink, tone: "text-info bg-info-tint" },
  account: { icon: IconUser, tone: "text-ink-muted bg-ink-faint" },
  // **والعرضُ له وجهُه** — إشعارٌ بلا أيقونةٍ خاصّةٍ يسقط على الافتراضيّ
  // فيختلط بما ليس منه في قائمةٍ تُمسح بالعين.
  offer: { icon: IconPromos, tone: "text-danger bg-danger-tint" },
};

/** يوم الإشعار بصيغة قابلة للقراءة — "اليوم" و"أمس" أوضح من تاريخ كامل. */
function dayLabel(iso: string): string {
  const d = new Date(iso);
  const today = new Date();
  const diff = Math.floor(
    (new Date(today.toDateString()).getTime() - new Date(d.toDateString()).getTime()) / 86_400_000,
  );
  if (diff <= 0) return N.today;
  if (diff === 1) return N.yesterday;
  return fmtLongDate(d);
}

/**
 * **ولا خيارَ للعرض** — الصفحةُ تملأ ما أُعطيت.
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «صفحةُ الإشعارات لازم ما فيها بادينك بشكلٍ
 *  مركزيّ مثل صفحة الزبون».)
 *
 * **كان مُعامِلاً**: الزبونُ يمرّر `full` والأربعُ تأخذ `wide` — **فتُحصر
 * باثني عشرَ وثمانين بكسلاً ويبقى الفراغُ يمينَها ويسارَها.** وكان تعليلُه
 * أنّ السايدبار يقتطع جانباً، **وهو تعليلُ صفحةٍ لا تُرى إلّا وحدَها.**
 *
 * **ومن فتح الزبونَ ثمّ اللوحةَ رأى صفحتين** — والمكوّنُ واحد.
 */
export function NotificationsPage({
  api,
  Link,
}: {
  api: ApiFn;
  Link: LinkType;
}) {
  const [kind, setKind] = useState("");

  const { data, loading, error, reload } = useLiveData<Feed>(
    () => api(`/api/v1/me/notifications?limit=200${kind ? `&kind=${kind}` : ""}`),
    ["order", "ticket", "wallet", "rating", "lead", "account", READ_EVENT],
    // **والنوعُ يُعيد الجلب** — بدونه تُضيء الشريحةُ ولا تُنادى الشبكة.
    [kind],
  );

  const markAll = useCallback(async () => {
    await api("/api/v1/me/notifications/read", { method: "POST", body: JSON.stringify({ id: "" }) });
    emitLocal(READ_EVENT); // الجرس يلتقط الصفر فوراً بلا تحديث صفحة
    reload();
  }, [api, reload]);

  const markOne = useCallback(
    async (id: string) => {
      await api("/api/v1/me/notifications/read", { method: "POST", body: JSON.stringify({ id }) });
      emitLocal(READ_EVENT);
      reload();
    },
    [api, reload],
  );

  const items = useMemo(() => data?.items ?? [], [data]);

  // تجميع بالأيام — بلا هذا يصير الأرشيف جداراً من الأسطر
  const groups = useMemo(() => {
    const out: { day: string; rows: AppNotification[] }[] = [];
    for (const n of items) {
      const day = dayLabel(n.created_at);
      const last = out[out.length - 1];
      if (last && last.day === day) last.rows.push(n);
      else out.push({ day, rows: [n] });
    }
    return out;
  }, [items]);

  if (loading) return <LoadingState />;

  const counts = data?.counts ?? {};
  const filters = [
    { id: "", label: N.all },
    // **والنوعُ الذي لا إشعارَ له لا حبّةَ له** — مرشِّحٌ يُضغط فيُفرغ الشاشةَ
    // يُقرأ عطباً. (`counts` تبقى لهذا وحدَه بعد أن ذهبت الأرقام.)
    ...Object.keys(KINDS)
      .filter((k) => counts[k])
      .map((k) => ({ id: k, label: N.kinds[k as keyof typeof N.kinds] ?? k })),
  ];

  return (
    /* **ولا حشوةَ هنا** — صارت مركزيّةً في غلاف الموقع (`<main>`).
       **ومكوّنٌ مشتركٌ يحمل حشوةَ موقعٍ بعينه يفرضها على اللوحات الأربع.** */
    <PageContainer>
      <PageHeader
        icon={IconBell}
        title={N.title}
        subtitle={data && data.unread > 0 ? N.unreadCount.replace("{n}", fmtNum(data.unread)) : undefined}
        actions={
          data && data.unread > 0 ? (
            <Button variant="secondary" onClick={markAll}>
              {N.markAllRead}
            </Button>
          ) : undefined
        }
      />

      {filters.length > 1 && (
        /* **شريطٌ مقسّم لا أزرارٌ متناثرة**: المرشّحاتُ خياراتُ شيءٍ واحد،
           وحدٌّ يجمعها يقول ذلك قبل أن تُقرأ. */
        /* ══════════════════════════════════════════════════════════════
           **ولا هامشَ سالباً — وهو سببُ التمرير الأفقيّ**
           ══════════════════════════════════════════════════════════════

           (شهده المالك ٢٠٢٦-٠٨-٠٦: «هناك سكرول أفقيٌّ مزعج».)

           كان `-mx-1` هنا وفي عنوان اليوم — **هامشٌ سالبٌ بأربعة بكسلاتٍ من
           كلّ جهة** ليُبلّط حشوةَ الغلاف فيمتدّ الصفُّ إلى حافّتَي البطاقة.

           **وحُذفت تلك الحشوةُ من الموقع** (قاعدةُ المالك: لا حشوةَ يميناً
           ويساراً) — **فلم يبقَ ما يبتلع السالب**، فامتدّ العنصرُ ثمانيةَ
           بكسلاتٍ خارجَ النافذة **وظهر شريطُ تمريرٍ أفقيٌّ للصفحة كلِّها.**

           **وقِيس**: النافذةُ ١٦٠٠ و`scrollWidth` ١٦٠٤، والعنصران يمتدّان من
           ‎−٤ إلى ١٦٠٤.

           **وثمانيةُ بكسلاتٍ تكفي**: شريطُ التمرير لا يسأل عن المقدار.

           **والدرسُ أوسعُ من الإصلاح**: كلُّ `-mx-*` كُتب في زمنِ غلافٍ محشوٍّ
           **صار دَيناً حين حُذفت الحشوة.**

           ── وما بقي من الصياغة الأولى ──

           كانت `inline-flex flex-wrap` —
           **تحتضن الحافّةَ وتلتفّ سطرين على الجوّال**، فيزيد ارتفاعُ الرأس
           **ويُدفع أوّلُ إشعارٍ تحت الطيّة.** */
        /* **والصفُّ صار مركزيّاً** (`Chips`) — كان هنا بمقاسٍ، وفي حسابات
           الإدارة بمقاسين، وفي صفحة الصنف بثالث. **وأربعةُ مقاساتٍ لعنصرٍ
           واحدٍ تُقرأ أربعَ منصّات.** (طلبُ المالك ٢٠٢٦-٠٨-٠٧.)

           **ونصُّ المختارة صار داكناً**: كان أبيضَ على سماويٍّ فاتحٍ بتباين
           ١٫٧٢ مقيسٍ على الشاشة — **والحدُّ ٤٫٥.**

           **ولا رقمَ في الحبّة** (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «ظهورُ الأرقام
           مزعج — ألغِه»): **الرقمُ كان يقول ما تقوله القائمةُ تحته.** وعددُ
           غير المقروء يبقى في الرأس — **وهو الرقمُ الوحيدُ الذي يُفيد: يقول
           كم بقي لا كم يوجد.** */
        <Chips items={filters} value={kind} onChange={setKind} />
      )}

      {/* ══════════════════════════════════════════════════════════════
          **«لم أصل» غيرُ «وصلتُ فلم أجد»**
          ══════════════════════════════════════════════════════════════

          **`useLiveData` تُرجع `error` والصفحةُ كانت لا تقرؤها** — تأخذ
          `{ data, loading, reload }` وتترك الرابعة. **فيسقط النداءُ وتبقى
          `data` فارغةً، فتُعرض «لا إشعارات».**

          **وقِيس حيّاً** (٢٠٢٦-٠٨-٠٦): جلسةٌ منتهيةٌ ردّ نداؤها ٤٠١ **والشاشةُ
          قالت «لا إشعارات»** — لا خطأَ ولا زرَّ إعادة.

          **وأثرُه أنّ صاحبَ الحساب يظنّ إشعاراتِه مُحيت**: يقرأ «لا إشعارات»
          وعنده اثنان وثلاثون. **والفراغُ كذبٌ حين يكون سببُه انقطاعاً.**

          **وهي عائلةُ الخلل التي يحرسها `check-central` في نداءات `fetch`
          المباشرة** — **وهذه داخل خطّافٍ فلم يرَها الحارس.** */}
      {error ? (
        <Alert tone="warning" title={m.errors.offline}>
          {m.errors.offlineHint}
        </Alert>
      ) : items.length === 0 ? (
        <EmptyState icon={IconBell} title={N.empty} />
      ) : (
        <div className="space-y-6">
          {groups.map((g) => (
            <section key={g.day}>
              {/* **عنوانُ اليوم يلتصق عند التمرير.**

                  أرشيفٌ من مئتي سطرٍ يفقد صاحبَه: يمرّر فينسى أيَّ يومٍ يقرأ.
                  **والعنوانُ الذي يهرب مع التمرير عنوانٌ لا يُقرأ إلّا مرّة.** */}
              {/* **لافتةٌ لا خطٌّ يعبر الشاشة.** كان الاسمُ بين خطّين يمتدّان
                  إلى الطرفين — **وعلى ألفٍ وتسعمئة يصير خطّاً بطول الشاشة
                  وكلمةٌ في وسطه**، فيُقرأ فاصلاً لا عنواناً. */}
              {/* ══════════════════════════════════════════════════════
                  **واللافتةُ وحدَها تلتصق — لا شريطٌ يعبر الشاشة**
                  ══════════════════════════════════════════════════════

                  (شهده المالك ٢٠٢٦-٠٨-٠٧ في صورة.)

                  **كان الغلافُ يحمل خلفيّةً معتمة** (`bg-page`) — وقِيس:
                  **شريطٌ بعرض ١٣٦٨ من ١٤٠٠**، لوحٌ داكنٌ يعبر الشاشة فوق
                  خلفيّةٍ زجاجيّة. **ومن رآه قرأه فاصلاً مكسوراً لا عنواناً.**

                  **والتضبيبُ كان على الغلاف** فأخذ معه العرضَ كلَّه. **وصار
                  على اللافتة**: هي زجاجٌ (`surface`) تحمل تضبيبَها، **وما
                  حولها يمرّ تحته المحتوى ظاهراً** — وهو المطلوب أصلاً.

                  **والخطُّ الذي كان يتبع الاسمَ حُذف**: `flex-1` داخلَ
                  `inline-flex` لا يمتدّ، **فكان عنصراً بعرض صفرٍ في الشجرة**
                  يُقرأ في قارئ الشاشة ولا يُرى. (قِيس: عنصرٌ واحدٌ بعرض صفر.) */}
              <div className="sticky top-0 z-10 mb-3 py-2">
                <h2 className="surface inline-flex items-center gap-2 rounded-badge px-3 py-1 text-xs font-bold text-ink-muted">
                  <span className="h-1.5 w-1.5 rounded-badge bg-primary" />
                  <span className="shrink-0">{g.day}</span>
                </h2>
              </div>
              {/* ══════════════════════════════════════════════════════
                  **بطاقاتٌ في شبكةٍ لا أسطرٌ تمتدّ ألفَ بكسل**
                  ══════════════════════════════════════════════════════

                  (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «إعادةُ هيكلةٍ كاملةٍ لصفحة
                  الإشعارات».)

                  **كان عموداً واحداً يملأ عرضَ الشاشة**: العنوانُ في أقصى
                  اليمين والوقتُ في أقصى اليسار، **وبينهما ألفٌ وأربعمئة بكسلٍ
                  فارغة** — فتقفز العينُ من طرفٍ إلى طرفٍ لتربط خبراً بوقته.

                  **ولا يُحصر العرضُ** — قاعدةُ المالك: لا حشوةَ جانبيّةً
                  والمحتوى يملأ الشاشة. **والبطاقاتُ تملأ العرضَ وتبقى مقروءة**:
                  كلٌّ مغلقةٌ على نفسها بعنوانها ومتنِها ووقتِها.

                  **والترتيبُ الزمنيُّ يبقى مفهوماً** لأنّ اليومَ مجموعٌ فوقها
                  — ما داخل اليوم يُقرأ لوحةَ أخبارٍ لا سجلَّ ثوان.

                  # وستٌّ في السطر — مفروضةً لا محسوبة

                  (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «اجعلها ٦ بكلّ سطر».)

                  **جُرّب العرضُ المقصود أوّلاً** (`auto-fill · minmax`) — وهو
                  ما نجح في شبكة الأصناف. **وأعطى خمسةً لا ستّاً على شاشة
                  المالك**: الشبكةُ تحسب بالبكسل المنطقيّ، **وتحجيمُ ويندوز
                  يجعل شاشةً بألفٍ وتسعمئة تُقاس بألفٍ وسبعمئة.**

                  **ولا يُخمَّن عرضُ شاشةِ أحد.** فالعددُ يُفرض عند حدٍّ معلوم:

                      < ٦٤٠   → واحد   · ٣٧٥px
                      ٦٤٠     → اثنان  · ٣٧٠
                      ١٠٢٤    → ثلاثة  · ٣٢٨
                      ١٢٨٠    → أربعة  · ٣٠٧
                      ١٥٣٦    → **ستّة** · ٢٤٤–٣٠٨ حسب الشاشة

                  **والفرقُ عن شبكة الأصناف مقصود**: البطاقةُ هناك تحمل صورةً
                  لها نسبةٌ تنكسر إن ضاقت، **وهذه تحمل سطرين من نصّ** — تضيق
                  إلى مئتين وأربعين ولا تنكسر. */}
              <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-6">
                {g.rows.map((n) => {
                  const meta = KINDS[n.kind] ?? KINDS.account!;
                  const Icon = meta.icon;
                  const inner = (
                    /* **غيرُ المقروء بعمودٍ جانبيّ لا بغسلةِ لون.**

                       كانت خلفيةٌ زرقاء تغمر السطرَ كلَّه — فيبهت النصُّ فيها
                       ويصير الأحدثُ أصعبَ قراءةً من الأقدم. **وما يُميَّز
                       بإضعافه لم يُميَّز.** والعمودُ يقول الشيءَ نفسه بحرفٍ
                       واحد ولا يمسّ النصّ. */
                    <div
                      className={`flex h-full items-start gap-3 rounded-card border p-3.5 transition-colors ${
                        n.read
                          ? "border-line bg-surface hover:border-primary-edge"
                          : "border-primary-edge bg-primary-tint"
                      }`}
                    >
                      <span
                        className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-control ${meta.tone}`}
                      >
                        <Icon size={17} />
                      </span>
                      <div className="min-w-0 flex-1">
                        <p
                          className={`flex items-center gap-2 leading-snug ${
                            n.read ? "text-ink" : "font-bold text-ink"
                          }`}
                        >
                          {/* **ونقطةٌ صريحةٌ لغير المقروء.**

                              كان يُميَّز بحدٍّ جمريٍّ بشفافيّة ٥٠٪ وخلفيّةٍ
                              بأربعين — **ولا يكادان يُريان على أرضٍ ملوّنة**،
                              فتبدو البطاقاتُ كلُّها سواءً.

                              **والنقطةُ تُقرأ بلمحةٍ ولا تمسّ النصّ** — بخلاف
                              الغسلة اللونيّة التي تُضعف ما تحتها. */}
                          {!n.read && (
                            <span
                              aria-hidden
                              className="h-2 w-2 shrink-0 rounded-badge bg-accent"
                            />
                          )}
                          <span className="min-w-0 truncate">{n.title}</span>
                        </p>
                        {/* **الجسدُ سطران لا سطرٌ مقتطع**: «تعويض عن طلبٍ فشل —
                            المطعم مغلق» يُقصّ عند «طلبٍ» فيبقى السؤال. */}
                        {n.body && (
                          <p className="mt-0.5 line-clamp-2 text-sm leading-relaxed text-ink-muted">
                            {n.body}
                          </p>
                        )}
                        {/* **والوقتُ تحت المتن لا في طرف السطر.**

                            كان مثبَّتاً في أقصى الجهة الأخرى — **وعلى شاشةٍ
                            عريضةٍ يبعد عن عنوانه ألفاً وأربعمئة بكسل.**

                            **و`inline-block` لا `block`**: الثاني يأخذ العرضَ
                            كاملاً، **و`dir="ltr"` يُحاذي نصَّه لليسار** —
                            فينتهي ملتصقاً بالحافّة الأخرى من البطاقة، **وهي
                            الفجوةُ نفسُها بحجمٍ أصغر.** (وقع فعلاً وشهده
                            المالك ٢٠٢٦-٠٨-٠٦.)

                            **والتاريخُ مع الساعة يبقى**: العناوينُ تجمع
                            بالأيام، **وسطرٌ يقول «٣:٤٠ م» وحدَه يُقرأ خارجَ
                            عنوانه** حين يُنسخ أو يُذكر. (قرارُ المالك
                            ٢٠٢٦-٠٨-٠٥.) */}
                        <span
                          className="mt-2 inline-block text-2xs tabular-nums text-ink-muted"
                          dir="ltr"
                        >
                          {fmtDateTime(n.created_at)}
                        </span>
                      </div>
                    </div>
                  );
                  return (
                    <li key={n.id} className="min-w-0">
                      {n.href ? (
                        <Link
                          href={n.href}
                          onClick={() => !n.read && void markOne(n.id)}
                          className="block h-full"
                        >
                          {inner}
                        </Link>
                      ) : (
                        <button
                          type="button"
                          onClick={() => markOne(n.id)}
                          className="block w-full text-start transition-colors hover:bg-row-hover"
                        >
                          {inner}
                        </button>
                      )}
                    </li>
                  );
                })}
              </ul>
            </section>
          ))}
        </div>
      )}
    </PageContainer>
  );
}
