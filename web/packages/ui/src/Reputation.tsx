"use client";

/**
 * لوحات السمعة المشتركة — نسخة واحدة لكل بوابات الموظفين (مندوب/متجر):
 * التقييمات والتعليقات المتلقّاة، والشكاوى والبلاغات. يُحقن لها api الخاص بالتطبيق.
 *
 * النصوص قابلة للتخصيص بحسب الدور: ما يراه **المتجر** تقييماً لخدمته يراه
 * **المندوب** تقييماً لمتاجره — والمصدر واحد لكنه لا يعني الشيء نفسه، فتسميته
 * "تقييمي" عند المندوب تضليل. كل بوابة تمرّر نصوصها من معجمها.
 */

import { useCallback } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import { Badge } from "./components";
import { useLiveData } from "./Notifications";
import { PageHeader, PageContainer, EmptyState, LoadingState, ListRow, StatGrid, StatCard, Stars } from "./layout";
import { IconStar, IconSupport } from "./icons";

const m = getMessages(defaultLocale);
const T = m.terms;
const R = m.shared.reputation;

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

/** نصوص قابلة للتخصيص — الافتراضي محايد من `shared.reputation`. */
export interface ReputationLabels {
  reviewsTitle?: string;
  reviewsHint?: string;
  reviewsEmpty?: string;
  avgLabel?: string;
  complaintsTitle?: string;
  complaintsHint?: string;
  complaintsEmpty?: string;
}

interface Review {
  order_number: number;
  merchant_name: string;
  stars: number;
  comment: string;
  created_at: string;
}
interface Complaint {
  number: number;
  order_number: number | null;
  subject: string;
  status: "open" | "in_progress" | "resolved";
  created_at: string;
}
interface Reputation {
  /** هل لهذا الدور نجومٌ أصلاً — يقولها الخادم ولا تُستنتج من العدد. */
  rated: boolean;
  rating: { avg: number; count: number; trend: "up" | "down" | "flat" };
  reviews: Review[];
  complaints: Complaint[];
}

function useReputation(api: ApiFn) {
  // حيّة: أي تقييم جديد أو شكوى يظهر فوراً بلا إعادة تحميل
  const { data } = useLiveData<Reputation>(
    useCallback(() => api<Reputation>("/api/v1/me/reputation"), [api]),
    ["rating", "ticket"],
  );
  return data;
}

export function ReputationReviews({ api, labels = {} }: { api: ApiFn; labels?: ReputationLabels }) {
  const data = useReputation(api);
  if (!data) return <LoadingState />;

  const trendText =
    data.rating.trend === "up" ? R.trendUp : data.rating.trend === "down" ? R.trendDown : R.trendFlat;

  // **من لا يُقيَّم لا تُعرض له بطاقةُ تقييم.**
  //
  // المتجرُ لم يعد له نجوم: الزبونُ لا يرى اسمَه ولا يختاره، **فما حكَم عليه
  // خدمتُنا كلُّها لا طعامُه وحده.** و«٠٫٠ من ٥» في شاشته أسوأُ من غياب
  // البطاقة — **يقرؤها حكماً عليه** فيسأل عمّا فعل، ولم يفعل شيئاً.
  //
  // **والخادمُ يقولها ولا تُستنتج من العدد**: صفرُ تقييماتٍ لمن يُقيَّم يعني
  // «لم يُقيَّم بعد»، **وصفرٌ لمن لا يُقيَّم يعني «لا يُقيَّم»** — ومعنيان
  // يفترقان في ما يُعرض.
  if (!data.rated) {
    return (
      <PageContainer>
        <PageHeader
          icon={IconStar}
          title={labels.reviewsTitle ?? T.ratings}
          subtitle={R.notRatedHint}
        />
        <EmptyState icon={IconStar} title={R.notRated} />
      </PageContainer>
    );
  }

  return (
    <PageContainer>
      <PageHeader
        icon={IconStar}
        title={labels.reviewsTitle ?? T.ratings}
        subtitle={labels.reviewsHint ?? R.reviewsHint}
      />

      <StatGrid>
        <StatCard
          label={labels.avgLabel ?? T.avgRating}
          value={data.rating.avg.toFixed(1)}
          icon={IconStar}
          tone="accent"
        />
        <StatCard label={T.ratingsCount} value={data.rating.count} />
        <StatCard label={T.trend} value={trendText} />
      </StatGrid>

      {data.reviews.length === 0 ? (
        <EmptyState icon={IconStar} title={labels.reviewsEmpty ?? R.reviewsEmpty} />
      ) : (
        <ul className="space-y-2">
          {data.reviews.map((rv, i) => (
            <li key={i} className="rounded-card border border-line bg-surface p-4">
              <div className="flex items-center justify-between gap-2">
                <Stars value={rv.stars} />
                <span className="text-xs text-ink-muted" dir="ltr">
                  {fmtDate(rv.created_at)}
                </span>
              </div>
              <p className="mt-1 text-sm text-ink-muted">
                {rv.merchant_name} — {T.order} #{fmtNum(rv.order_number)}
              </p>
              {rv.comment && <p className="mt-1 text-sm">{rv.comment}</p>}
            </li>
          ))}
        </ul>
      )}
    </PageContainer>
  );
}

const CVARIANT: Record<Complaint["status"], "warning" | "primary" | "success"> = {
  open: "warning",
  in_progress: "primary",
  resolved: "success",
};

export function ReputationComplaints({ api, labels = {} }: { api: ApiFn; labels?: ReputationLabels }) {
  const data = useReputation(api);
  if (!data) return <LoadingState />;

  return (
    <PageContainer>
      <PageHeader
        icon={IconSupport}
        title={labels.complaintsTitle ?? T.complaints}
        subtitle={labels.complaintsHint ?? R.complaintsHint}
      />

      {data.complaints.length === 0 ? (
        <EmptyState icon={IconSupport} title={labels.complaintsEmpty ?? R.complaintsEmpty} tone="success" />
      ) : (
        <ul className="space-y-2">
          {data.complaints.map((c) => (
            <li
              key={c.number}
              className="flex flex-wrap items-center gap-3 rounded-card border border-line bg-surface p-4"
            >
              <span className="font-bold">#{fmtNum(c.number)}</span>
              <span className="min-w-0 flex-1 text-sm">{c.subject}</span>
              {c.order_number != null && (
                <span className="text-xs text-ink-muted">
                  {T.order} #{fmtNum(c.order_number)}
                </span>
              )}
              <Badge variant={CVARIANT[c.status]}>{T.ticketStatus[c.status]}</Badge>
              <span className="text-xs text-ink-muted" dir="ltr">
                {fmtDate(c.created_at)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </PageContainer>
  );
}
