"use client";

/**
 * **صفحاتُ الموقع — بيتُ ما يُضبط في كلّ صفحةٍ على حدة.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «تبويبٌ جديدٌ اسمه صفحات الموقع، بداخله تبويبات:
 *  الصفحة الرئيسية · تسوّق · المساعدة · تواصل معنا · شروط الاستخدام · سياسة
 *  الخصوصية — فقط اتركها عناوينَ، لا تضف أيَّ شيءٍ بداخلها في المرحلة الأولى،
 *  فقط تأسيسها».)
 *
 * # ولماذا تبويباتٌ لا صناديقُ متتالية
 *
 * **ستُّ صفحاتٍ متتاليةً في عمودٍ واحدٍ تُقرأ قائمةً لا اختياراً** — ومن أراد
 * «تواصل معنا» مرّ بخمسٍ قبلها. **والتبويبُ يُظهر واحدةً ويُخفي خمساً**، وهو
 * ما يُراد: **من يضبط صفحةً لا ينظر في غيرها.**
 *
 * # والقسمُ اسمٌ لا شيفرة
 *
 * **كلُّ صفحةٍ اسمُ قسمٍ فرعيّ**: `page.home` · `page.shop` … **فمفتاحٌ
 * يُضاف إلى الفهرس باسم قسمه يظهر في تبويبه بلا أن تُمسّ هذه الشاشة.**
 *
 * **ولا تُبنى شاشةٌ لكلّ صفحة**: ستُّ شاشاتٍ تفترق في السادسة، **وهذه واحدةٌ
 * تخدمهنّ.**
 *
 * # وفارغُها اليوم مقصود
 *
 * **التأسيسُ قبل المحتوى بطلب المالك** — ولكلّ تبويبٍ فارغٍ نصٌّ يقول إنّه
 * ينتظر، **لا بياضٌ يُقرأ عطباً.**
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Tabs, EmptyState, IconSettings, type TabDef } from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const S = m.admin.settings;

/** **ترتيبُ الصفحات كترتيب رحلة الزائر** — لا أبجديّاً.
 *
 * **و«الدخول والتسجيل» صفحةٌ واحدةٌ لثلاث شاشات**: الدخولُ وإنشاءُ الحساب
 * واستعادةُ كلمة المرور. (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «صفحةٌ وحدة تكون، مشان
 * نتحكّم بصورة الخلفيّة تبعهنّ».) **وهي شاشةٌ واحدةٌ في عين الزائر** —
 * وخلفيّةٌ تختلف بين الثلاث تُقرأ موقعين لا موقعاً.
 *
 * **و«خلفيّة الموقع» ليست صفحةً** — هي ما تحت كلّ صفحة، **فموضعُها أوّلَ
 * الشريط** لأنّها تعمّ ما بعدَها. (قرارُ المالك ٢٠٢٦-٠٨-٠٩.) */
const PAGES = [
  "background",
  "home",
  "shop",
  "auth",
  "help",
  "contact",
  "terms",
  "privacy",
] as const;
type Page = (typeof PAGES)[number];

const pageLabel = (p: string) => (S.pages as Record<string, string>)[p] ?? p;

export default function SitePagesPanel({
  /** **ما يُرسم في التبويب** — من ينادي يمرّر مفاتيحَ القسم وحدَها. */
  render,
  /** كم مفتاحاً في كلّ صفحة — **لِيُرى الممتلئُ من الفارغ قبل الفتح.** */
  counts = {},
}: {
  render?: (section: string) => React.ReactNode;
  counts?: Record<string, number>;
}) {
  const [page, setPage] = useState<Page>("home");

  const items: TabDef<Page>[] = PAGES.map((p) => ({
    key: p,
    label: pageLabel(p),
    ...(counts[`page.${p}`] ? { count: counts[`page.${p}`] } : {}),
  }));

  const body = render?.(`page.${page}`);

  return (
    <div className="space-y-3">
      <Tabs items={items} value={page} onChange={setPage} />
      {body ?? (
        <EmptyState
          icon={IconSettings}
          title={S.emptyGroup}
          action={<p className="text-xs text-ink-muted">{S.emptyGroupHint}</p>}
        />
      )}
    </div>
  );
}
