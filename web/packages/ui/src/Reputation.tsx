"use client";

/**
 * لوحات السمعة المشتركة — نسخة واحدة لكل بوابات الموظفين (مندوب/متجر):
 * التقييمات والتعليقات المتلقّاة، والشكاوى والبلاغات. يُحقن لها api الخاص بالتطبيق.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge } from "./components";
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

function Stars({ n }: { n: number }) {
  return (
    <span dir="ltr" className="text-amber-400">
      {"★".repeat(n)}
      <span className="text-line">{"★".repeat(Math.max(0, 5 - n))}</span>
    </span>
  );
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
  if (!data) return <p className="py-12 text-center text-ink-muted">{m.common.loading}</p>;

  const trendText =
    data.rating.trend === "up" ? R.trendUp : data.rating.trend === "down" ? R.trendDown : R.trendFlat;

  return (
    <div className="space-y-5">
      <h1 className="flex items-center gap-2 text-lg font-bold">
        <IconStar size={20} className="text-primary" />
        {T.ratings}
      </h1>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <div className="rounded-card border border-line bg-surface p-4">
          <p className="text-2xl font-bold text-amber-500" dir="ltr">
            {data.rating.avg.toFixed(1)} ★
          </p>
          <p className="text-xs text-ink-muted">{T.avgRating}</p>
        </div>
        <div className="rounded-card border border-line bg-surface p-4">
          <p className="text-2xl font-bold">{fmt.format(data.rating.count)}</p>
          <p className="text-xs text-ink-muted">{T.ratingsCount}</p>
        </div>
        <div className="col-span-2 flex items-center rounded-card border border-line bg-surface p-4 sm:col-span-1">
          <p className="text-sm font-medium">{trendText}</p>
        </div>
      </div>

      {data.reviews.length === 0 ? (
        <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-ink-muted">
          {R.reviewsEmpty}
        </p>
      ) : (
        <ul className="space-y-2">
          {data.reviews.map((rv, i) => (
            <li key={i} className="rounded-card border border-line bg-surface p-4">
              <div className="flex items-center justify-between gap-2">
                <Stars n={rv.stars} />
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
    </div>
  );
}

const CVARIANT: Record<Complaint["status"], "warning" | "primary" | "success"> = {
  open: "warning",
  in_progress: "primary",
  resolved: "success",
};

export function ReputationComplaints({ api }: { api: ApiFn }) {
  const data = useReputation(api);
  if (!data) return <p className="py-12 text-center text-ink-muted">{m.common.loading}</p>;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="flex items-center gap-2 text-lg font-bold">
          <IconSupport size={20} className="text-primary" />
          {T.complaints}
        </h1>
        <p className="mt-1 text-sm text-ink-muted">{R.complaintsHint}</p>
      </div>

      {data.complaints.length === 0 ? (
        <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-success">
          {R.complaintsEmpty}
        </p>
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
    </div>
  );
}
