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
  ReputationComplaints,
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
const R = m.customer.myReputation;

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
        <div className="grid gap-4 lg:grid-cols-2">
          {rows.map((t) => (
            /**
             * **كلُّ عنصرٍ في حقلٍ مستقلّ يُقرأ وحدَه.**
             *
             * كانت أسطراً متتابعةً بلا حدود: **العينُ لا تعرف أين ينتهي خبرٌ
             * ويبدأ آخر** — فيُقرأ التاريخُ جزءاً من السبب، والتعويضُ جزءاً من
             * الردّ. **وشكوى تُقرأ خطأً تُعاد.**
             */
            <Card key={t.id} className="flex flex-col gap-4">
              {/* ── الترويسة: السببُ · الرقمُ · الحالة ─────────────────── */}
              <div className="flex items-start gap-3">
                <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-control bg-primary-tint text-primary">
                  <IconSupport size={20} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="font-bold leading-tight">
                    {(m.site.complaint.reasons as Record<string, string>)[t.reason] || t.subject}
                  </p>
                  <p className="mt-0.5 text-xs text-ink-muted">{C.mineOne}</p>
                </div>
                <div className="flex shrink-0 flex-col items-end gap-1.5">
                  {/* **الرقمُ بحقلٍ خاصّ** — هو ما يقوله حين يتّصل يسأل. */}
                  <span
                    dir="ltr"
                    className="rounded-control bg-primary-tint px-2.5 py-1 text-sm font-bold tabular-nums text-primary-dark"
                  >
                    #{fmtRef(t.number)}
                  </span>
                  <Badge variant={TONE[t.status] ?? "neutral"}>
                    {(m.admin.tickets.status as Record<string, string>)[t.status] ?? t.status}
                  </Badge>
                </div>
              </div>

              {/* ── حقلان مستقلّان: متى · وعلى أيّ طلب ─────────────────── */}
              <div className="grid grid-cols-2 gap-2">
                <div className="rounded-control bg-field px-3 py-2">
                  <p className="text-2xs text-ink-muted">{C.fieldWhen}</p>
                  <p className="mt-0.5 text-sm font-medium tabular-nums" dir="ltr">
                    {fmtDateTime(t.created_at)}
                  </p>
                </div>
                <div className="rounded-control bg-field px-3 py-2">
                  <p className="text-2xs text-ink-muted">{C.fieldOrder}</p>
                  <p className="mt-0.5 text-sm font-medium tabular-nums" dir="ltr">
                    {t.order_number !== null ? `#${fmtRef(t.order_number)}` : "—"}
                  </p>
                </div>
              </div>

              {/* ── ردُّ المنصة — **حقلٌ مُعنوَنٌ لا سطرٌ عائم** ──────────── */}
              <div className="flex-1 surface-inset p-3">
                <p className="mb-1 flex items-center gap-1.5 text-2xs font-bold text-ink-muted">
                  <IconReply size={13} />
                  {C.fieldReply}
                </p>
                {t.resolution ? (
                  <p className="text-sm">{t.resolution}</p>
                ) : (
                  /* **ومفتوحةٌ بلا ردٍّ تقول ذلك** — الصمتُ في الشاشة يُقرأ
                     إهمالاً، **وجملةٌ واحدةٌ تحوّل الانتظارَ من قلقٍ إلى مهلة.** */
                  <p className="text-sm text-ink-muted">{C.waiting}</p>
                )}
              </div>

              {/* ── التعويضُ حقلٌ قائمٌ بذاته — **وهو ماله** ─────────────── */}
              {t.compensation > 0 && (
                <div className="flex items-center justify-between rounded-control border border-success-edge bg-success-tint px-3 py-2.5">
                  <span className="flex items-center gap-1.5 text-sm font-medium text-success">
                    <IconCheck size={15} strokeWidth={3} />
                    {C.compensated}
                  </span>
                  <span dir="ltr" className="text-base font-bold tabular-nums text-success">
                    {fmtNum(t.compensation)}{" "}
                    <span className="text-xs font-normal">{m.common.currency}</span>
                  </span>
                </div>
              )}
            </Card>
          ))}
        </div>
      )}
    
      {/* ══════════════════════════════════════════════════════════════
          **وما رُفع عليه يقع حيث يقع ما رفعه**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «يوجد قسمٌ خاصٌّ بالشكاوى والبلاغات».)

          **كان في صفحة الحساب** — وهي صفحةُ بياناتٍ وعناوين. **فمن أراد
          «الشكاوى والبلاغات» فتح هذا القسمَ فوجد نصفَه**، والنصفُ الآخرُ
          في صفحةٍ أخرى لا يدلّ عليها شيء.

          **وهما بابان لشيءٍ واحد**: ما رفعتَه، وما رُفع عليك. */}
      <ReputationComplaints
        api={api}
        labels={{
          complaintsTitle: R.title,
          complaintsHint: R.hint,
          complaintsEmpty: R.empty,
        }}
      />
    </PageContainer>
  );
}
