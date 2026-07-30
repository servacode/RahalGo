"use client";

/** التقييمات والتعليقات — ما تلقّاه المندوب (عبر أداء متاجره) من الزبائن. */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconStar } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const R = m.rep.reputation;
const fmt = new Intl.NumberFormat("ar-SY");

interface Review {
  order_number: number;
  merchant_name: string;
  stars: number;
  comment: string;
  created_at: string;
}
interface Reputation {
  rating: { avg: number; count: number; trend: "up" | "down" | "flat" };
  reviews: Review[];
}

function Stars({ n }: { n: number }) {
  return (
    <span dir="ltr" className="text-amber-400">
      {"★".repeat(n)}
      <span className="text-line">{"★".repeat(5 - n)}</span>
    </span>
  );
}

export default function ReviewsPage() {
  const [data, setData] = useState<Reputation | null>(null);

  useEffect(() => {
    api<Reputation>("/api/v1/me/reputation").then(setData).catch(() => undefined);
  }, []);

  if (!data) return <p className="py-12 text-center text-ink-muted">{m.common.loading}</p>;

  const trendText =
    data.rating.trend === "up" ? R.trendUp : data.rating.trend === "down" ? R.trendDown : R.trendFlat;

  return (
    <div className="space-y-5">
      <h1 className="flex items-center gap-2 text-lg font-bold">
        <IconStar size={20} className="text-primary" />
        {R.reviewsTitle}
      </h1>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <div className="rounded-card border border-line bg-surface p-4">
          <p className="text-2xl font-bold text-amber-500" dir="ltr">
            {data.rating.avg.toFixed(1)} ★
          </p>
          <p className="text-xs text-ink-muted">{R.avg}</p>
        </div>
        <div className="rounded-card border border-line bg-surface p-4">
          <p className="text-2xl font-bold">{fmt.format(data.rating.count)}</p>
          <p className="text-xs text-ink-muted">{R.count}</p>
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
              <p className="mt-1 text-sm">
                <span className="text-ink-muted">
                  {rv.merchant_name} — {R.order} #{fmt.format(rv.order_number)}
                </span>
              </p>
              {rv.comment && <p className="mt-1 text-sm">{rv.comment}</p>}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
