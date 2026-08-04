"use client";

/**
 * إنذاراتُ المتجر — **ما يُعدّ لا ما يُقرأ ويُنسى.**
 *
 * # لماذا حلّت محلّ التقييمات
 *
 * المتجرُ لم يعد له نجوم: الزبونُ لا يرى اسمَه ولا يختاره، **فما حكَم عليه
 * خدمتُنا كلُّها لا طعامُه وحده.**
 *
 * # ولماذا صفحةٌ لا إشعار
 *
 * كان الإنذارُ يُرسل إشعاراً ويمضي — **يُقرأ مرّةً ويُنسى.** ثمّ يُحظر المتجرُ
 * **ولم يكن يعلم أنّ عليه شيئاً.** والحظرُ الذي يفاجئ يُفقدنا متجراً كان
 * يصحّح لو عرف، **ويُفقده رزقاً لم يُنبَّه إليه.**
 *
 * **ومن رأى عدّادَه يقترب من حدّه صحّح قبل أن يبلغه** — وهذا كلُّ الغرض.
 */

import { useCallback } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDate } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  StatGrid,
  StatCard,
  IconStatus,
  IconWarning,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const W = m.merchant.warnings;

type Warning = {
  id: string;
  reason: string;
  note: string;
  order_number: number | null;
  created_at: string;
};

export default function WarningsPage() {
  const { data } = useLiveData<{ warnings: Warning[] }>(
    useCallback(() => api<{ warnings: Warning[] }>("/api/v1/merchant/warnings"), []),
    ["merchant", "order"],
  );
  if (!data) return <LoadingState />;
  const list = data.warnings ?? [];

  return (
    <PageContainer>
      <PageHeader icon={IconWarning} title={W.title} subtitle={W.hint} />

      <StatGrid>
        <StatCard
          label={W.count}
          value={fmtNum(list.length)}
          icon={IconWarning}
          tone={list.length > 0 ? "danger" : "default"}
        />
      </StatGrid>

      {list.length === 0 ? (
        <EmptyState icon={IconStatus} title={W.empty} />
      ) : (
        <ul className="space-y-2">
          {list.map((x) => (
            <li key={x.id} className="rounded-card border border-line bg-surface p-4">
              <div className="flex items-center justify-between gap-2">
                <span className="font-medium">
                  {W.reasons[x.reason as keyof typeof W.reasons] ?? x.reason}
                </span>
                <span className="text-xs text-ink-muted" dir="ltr">
                  {fmtDate(x.created_at)}
                </span>
              </div>
              {x.order_number !== null && (
                <p className="mt-1 text-sm text-ink-muted">
                  {m.terms.order} #{fmtRef(x.order_number)}
                </p>
              )}
              {x.note && <p className="mt-1 text-sm">{x.note}</p>}
            </li>
          ))}
        </ul>
      )}
    </PageContainer>
  );
}
