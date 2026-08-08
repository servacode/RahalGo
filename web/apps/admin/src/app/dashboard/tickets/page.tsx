"use client";

/**
 * **ما يقوله الناس — في بابٍ واحد.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٨: «نجمع النزاعات والشكاوى والتقييم بصفحةٍ واحدة
 *  بأزرار تبويب».)
 *
 * # لماذا الشكوى والتقييم معاً
 *
 * **جوابان لسؤالٍ واحد**: «ما رأيُ الناس بنا؟». والطريقُ الطبيعيّ أن يُرى
 * سائقٌ هبط تقييمُه **فتُقرأ شكاواه** — وكان ذلك رابطين في القائمة وصفحتين
 * لا يعرف الواحدةُ منهما بالأخرى.
 *
 * **والتقييماتُ بلا زرٍّ واحد** — جدولُ قراءةٍ يشغل باباً في قائمةٍ من عشرين
 * باباً. **وتبويبٌ أليقُ بها.**
 *
 * # ولماذا لم تدخل النزاعات
 *
 * **هي مالٌ لا رأي**: موضعُها بين الخزينة والخسائر حيث يعمل الماليّ، وصلاحيّتُها
 * أضيق. **وقد جُمعت مع الخسائر** — فالخسارةُ هي التي تُنشئ النزاع: المنصّةُ
 * تعوّض السائقَ ثمّ تطالب المتجر. (`/dashboard/losses`)
 *
 * # والعرضان يبقيان كما هما
 *
 * **لكلّ تبويبٍ عنوانُه وأزرارُه** — فالشكاوى تحمل «شكوى جديدة» ومبدّلَ
 * الجدول والبطاقات، والتقييماتُ ترشيحَها. **وصفحةٌ جامعةٌ تسلب الأزرارَ
 * أماكنها تُفسد بابين لتجمعهما.**
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { TabCards } from "@rahalgo/ui";
import { TicketsView } from "@/components/support/tickets";
import { RatingsView } from "@/components/support/ratings";

const m = getMessages(defaultLocale);

const TABS = [
  { key: "tickets", label: m.terms.complaints },
  { key: "ratings", label: m.admin.nav.ratings },
] as const;

export default function SupportPage() {
  const [tab, setTab] = useState<string>("tickets");
  return (
    <div className="space-y-5">
      <TabCards items={TABS.map((t) => ({ key: t.key, label: t.label }))} active={tab} onChange={setTab} />
      {tab === "tickets" ? <TicketsView /> : <RatingsView />}
    </div>
  );
}
