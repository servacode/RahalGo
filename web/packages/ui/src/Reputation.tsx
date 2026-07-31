"use client";

/**
 * لوحات السمعة المشتركة — نسخة واحدة لكل بوابات الموظفين (مندوب/متجر):
 * التقييمات والتعليقات المتلقّاة، والشكاوى والبلاغات. يُحقن لها api الخاص بالتطبيق.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge } from "./components";
import { PageHeader, PageContainer, EmptyState, LoadingState, ListRow, StatGrid, StatCard, Stars } from "./layout";
import { IconStar, IconSupport } from "./icons";

const m = getMessages(defaultLocale);
const T = m.terms;
const R = m.shared.reputation;
const fmt = new Intl.NumberFormat("ar-SY");

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

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
  rating: { avg: number; count: number; trend: "up" | "down" | "flat" };
  reviews: Review[];
  complaints: Complaint[];
}

function useReputation(api: ApiFn) {
  const [data, setData] = useState<Reputation | null>(null);
  useEffect(() => {
    api<Reputation>("/api/v1/me/reputation").then(setData).catch(() => undefined);
  }, [api]);
  return data;
}

export function ReputationReviews({ api }: { api: ApiFn }) {
  const data = useReputation(api);
  if (!data) return <LoadingState />;

  const trendText =
    data.rating.trend === "up" ? R.trendUp : data.rating.trend === "down" ? R.trendDown : R.trendFlat;

  return (
    <PageContainer>
      <PageHeader icon={IconStar} title={T.ratings} subtitle={R.reviewsHint} />

      <StatGrid>
        <StatCard label={T.avgRating} value={`${data.rating.avg.toFixed(1)} ★`} tone="accent" />
        <StatCard label={T.ratingsCount} value={data.rating.count} />
        <StatCard label={T.myRating} value={trendText} />
      </StatGrid>

      {data.reviews.length === 0 ? (
        <EmptyState icon={IconStar} title={R.reviewsEmpty} />
      ) : (
        <ul className="space-y-2">
          {data.reviews.map((rv, i) => (
            <li key={i} className="rounded-card border border-line bg-surface p-4">
              <div className="flex items-center justify-between gap-2">
                <Stars value={rv.stars} />
                <span className="text-xs text-ink-muted" dir="ltr">
                  {new Date(rv.created_at).toLocaleDateString("ar-SY")}
                </span>
              </div>
              <p className="mt-1 text-sm text-ink-muted">
                {rv.merchant_name} — {T.order} #{fmt.format(rv.order_number)}
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

export function ReputationComplaints({ api }: { api: ApiFn }) {
  const data = useReputation(api);
  if (!data) return <LoadingState />;

  return (
    <PageContainer>
      <PageHeader icon={IconSupport} title={T.complaints} subtitle={R.complaintsHint} />

      {data.complaints.length === 0 ? (
        <EmptyState icon={IconSupport} title={R.complaintsEmpty} tone="success" />
      ) : (
        <ul className="space-y-2">
          {data.complaints.map((c) => (
            <li
              key={c.number}
              className="flex flex-wrap items-center gap-3 rounded-card border border-line bg-surface p-4"
            >
              <span className="font-bold">#{fmt.format(c.number)}</span>
              <span className="min-w-0 flex-1 text-sm">{c.subject}</span>
              {c.order_number != null && (
                <span className="text-xs text-ink-muted">
                  {T.order} #{fmt.format(c.order_number)}
                </span>
              )}
              <Badge variant={CVARIANT[c.status]}>{T.ticketStatus[c.status]}</Badge>
              <span className="text-xs text-ink-muted" dir="ltr">
                {new Date(c.created_at).toLocaleDateString("ar-SY")}
              </span>
            </li>
          ))}
        </ul>
      )}
    </PageContainer>
  );
}
