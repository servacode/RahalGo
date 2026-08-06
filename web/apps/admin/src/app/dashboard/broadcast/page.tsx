"use client";

/**
 * **إعلانُ المنصة — قسمٌ لحاله.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «بصفحة الإشعارات بلوحة الادمن يوجد إعلان
 *  المنصة، انقله إلى قسمٍ لحاله مبدئيّاً لبين ما ننتقل إلى لوحة الأدمن
 *  ونرتّبها بشكلٍ كاملٍ وصحيح».)
 *
 * # ولماذا خرج من صفحة الإشعارات
 *
 * **صفحةُ الإشعارات قراءة، والإعلانُ كتابة.** كان لوحُ الإرسال يعلو الأرشيفَ
 * في لوحة الإدارة وحدَها — **فصفحةٌ واحدةٌ في خمس واجهاتٍ لها في الإدارة
 * رأسٌ زائدٌ لا تعرفه أخواتُها.**
 *
 * **ومن فتحها ليقرأ ما فاته وجد نموذجَ إرسالٍ أوّلاً** — والأرشيفُ الذي جاء
 * من أجله تحته.
 *
 * **وهذا موضعٌ مبدئيٌّ** حتّى تُرتَّب لوحةُ الإدارة كاملةً — ولا يُبنى له
 * شيءٌ جديد: اللوحُ كما هو، وحوله عنوانُ صفحةٍ وحاويةٌ مركزيّان.
 */

import { PageContainer, PageHeader, IconWhatsApp } from "@rahalgo/ui";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import BroadcastPanel from "@/components/BroadcastPanel";

const m = getMessages(defaultLocale);

export default function Page() {
  return (
    <PageContainer>
      <PageHeader icon={IconWhatsApp} title={m.admin.broadcast.title} subtitle={m.admin.broadcast.hint} />
      <BroadcastPanel />
    </PageContainer>
  );
}
