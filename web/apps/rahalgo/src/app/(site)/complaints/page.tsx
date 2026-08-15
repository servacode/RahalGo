"use client";

/**
 * **شكاواي — وأين وصلت.**
 *
 * كان الزبونُ يفتح شكوى ثمّ **لا يجد لها أثراً في أيّ شاشة**: التذاكرُ كلُّها
 * في لوحة المنصة، **وما رآه بعدها رقمٌ في بطاقة الطلب لا يقول حالَها.**
 *
 * **ومن اشتكى ولم يرَ جواباً ظنّ أنّ شكواه ضاعت** — فيشتكي ثانيةً، أو يتّصل،
 * **أو يسكت ويذهب.** والسكوتُ أسوأ: نخسر الزبونَ ولا نعرف لماذا.
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «الشكاوي ليس لها مكان بحساب المستخدم، أين يرى ما
 * حالة الشكوى؟».)
 */

import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  ComplaintCard,
  ComplaintGrid,
  Badge,
  Card,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  useLiveData,
  IconSupport,
  IconReply,
  IconCheck,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.site.complaint;

interface Ticket {
  id: string;
  number: number;
  order_number: number | null;
  subject: string;
  reason: string;
  status: string;
  compensation: number;
  resolution: string;
  created_at: string;
  resolved_at: string | null;
}

/** **مفتوحةٌ تُنتظر، ومحلولةٌ انتهت** — ولونٌ واحدٌ لهما يجعل القائمة جداراً. */
const TONE: Record<string, "warning" | "success" | "neutral"> = {
  open: "warning",
  in_progress: "warning",
  resolved: "success",
  closed: "neutral",
};

export default function ComplaintsPage() {
  const { data, loading } = useLiveData<{ tickets: Ticket[] }>(
    () => api("/api/v1/my/tickets"),
    ["ticket"],
  );

  if (loading) return <LoadingState />;
  // **والنوعُ يُكتب هنا صراحةً** — **وبناءٌ ينجح على الجهاز ويسقط في
  // الحاوية** كان يُحلّ نوعَ الصفوف إلى `any` (قِيس ٢٠٢٦-٠٨-١٥):
  // **واستنتاجٌ يتبدّل بتبدّل بيئة البناء ليس استنتاجاً يُعتمد عليه.**
  const rows: Ticket[] = data?.tickets ?? [];

  return (
    <PageContainer>
      <PageHeader icon={IconSupport} title={m.terms.complaints} />

      {rows.length === 0 ? (
        <EmptyState icon={IconSupport} title={C.noneTitle} />
      ) : (
        /* **والكرتُ من المركز** — كان مبنيّاً هنا، **وصياغتُه الثالثةُ في
           المشروع**: سطرٌ عارٍ عند السائق والمتجر وجدولٌ في الإدارة. (قرارُ
           المالك ٢٠٢٦-٠٨-٠٧: «طبّقوه على كلّ صفحات البلاغات بكلّ اللوحات».) */
        <ComplaintGrid>
          {rows.map((t) => (
            <ComplaintCard key={t.id} ticket={t} note={C.mineOne} />
          ))}
        </ComplaintGrid>
      )}
    
      {/* **ولا قسمَ ثانٍ لِما رُفع عليه.**

          (تصحيحُ المالك ٢٠٢٦-٠٨-٠٧: «أنت مكرّر البلاغ مرّتين وهو أصلاً
           موجود».)

          **نقلتُ «بلاغاتٌ على طلباتي» إلى هنا ظنّاً أنّها بياناتٌ أخرى.**
          وهي ليست: `‎/my/tickets` تُرجع كلَّ تذاكر الزبون —
          `WHERE t.customer_id = $1` بلا تمييزٍ لمن فتحها. **فبلاغُ السائق
          داخلٌ في القائمة أعلاه منذ البداية**، ويُعرف بعنوانه.

          **والدرسُ أنّي حكمتُ على مصدرين باسميهما لا باستعلامَيهما.** */}
    </PageContainer>
  );
}
