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

import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  Badge,
  Card,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  useLiveData,
  IconSupport,
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
  const rows = data?.tickets ?? [];

  return (
    <PageContainer>
      <PageHeader icon={IconSupport} title={C.mine} />

      {rows.length === 0 ? (
        <EmptyState icon={IconSupport} title={C.noneTitle} />
      ) : (
        <div className="space-y-3">
          {rows.map((t) => (
            <Card key={t.id} className="space-y-3">
              <div className="flex items-start gap-3">
                <div className="min-w-0 flex-1">
                  <p className="font-bold leading-tight">
                    {(m.site.complaint.reasons as Record<string, string>)[t.reason] || t.subject}
                  </p>
                  <p className="mt-0.5 text-xs text-ink-muted" dir="ltr">
                    {fmtDateTime(t.created_at)}
                  </p>
                </div>
                <div className="flex shrink-0 flex-col items-end gap-1.5">
                  <span
                    dir="ltr"
                    className="rounded-control bg-primary-light px-2.5 py-1 text-sm font-bold tabular-nums text-primary-dark"
                  >
                    #{fmtNum(t.number)}
                  </span>
                  <Badge variant={TONE[t.status] ?? "neutral"}>
                    {(m.admin.tickets.status as Record<string, string>)[t.status] ?? t.status}
                  </Badge>
                </div>
              </div>

              {t.order_number !== null && (
                <p className="text-sm text-ink-muted">
                  {C.onOrder}{" "}
                  <span dir="ltr" className="font-medium text-ink">
                    #{fmtNum(t.order_number)}
                  </span>
                </p>
              )}

              {/* **وكلمةُ المنصة تُقرأ** — وشكوى تُغلق بلا كلمة تُقرأ تجاهلاً،
                  ولو كان القرارُ في صالحه. */}
              {t.resolution && (
                <p className="rounded-control bg-page px-3 py-2 text-sm">{t.resolution}</p>
              )}

              {t.compensation > 0 && (
                <p className="flex items-center justify-between rounded-control bg-success/10 px-3 py-2 text-sm font-medium text-success">
                  <span>{C.compensated}</span>
                  <span dir="ltr" className="tabular-nums">
                    {fmtNum(t.compensation)} {m.common.currency}
                  </span>
                </p>
              )}

              {/* **ومفتوحةٌ بلا ردٍّ تقول ذلك** — الصمتُ في الشاشة يُقرأ إهمالاً،
                  **وجملةٌ واحدةٌ تحوّل الانتظارَ من قلقٍ إلى مهلة.** */}
              {!t.resolution && !t.resolved_at && (
                <p className="text-sm text-ink-muted">{C.waiting}</p>
              )}
            </Card>
          ))}
        </div>
      )}
    </PageContainer>
  );
}
