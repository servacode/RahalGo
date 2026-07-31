"use client";

/** تقييماتي: طلباتي المُسلَّمة — قيّم ما لم يُقيَّم، وشاهد تقييماتك السابقة. */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, IconStar } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";
import RatingModal from "@/components/RatingModal";

const m = getMessages(defaultLocale);
const R = m.site.rating;
const fmt = new Intl.NumberFormat("ar-SY");

interface RatedOrder {
  order_id: string;
  number: number;
  merchant_name: string;
  has_driver: boolean;
  merchant_stars: number;
  driver_stars: number | null;
  comment: string;
  rated: boolean;
  created_at: string;
}

function StarRow({ n }: { n: number }) {
  return (
    <span dir="ltr" className="text-amber-400">
      {"★".repeat(n)}
      <span className="text-line">{"★".repeat(5 - n)}</span>
    </span>
  );
}

export default function RatingsPage() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const [orders, setOrders] = useState<RatedOrder[] | null>(null);
  const [rating, setRating] = useState<RatedOrder | null>(null);

  const load = useCallback(() => {
    api<RatedOrder[]>("/api/v1/my/ratings")
      .then(setOrders)
      .catch(() => setOrders([]));
  }, []);

  useEffect(() => {
    if (loading) return;
    if (!isLoggedIn(user)) {
      router.replace("/login?next=/ratings");
      return;
    }
    load();
  }, [user, loading, router, load]);

  if (!orders) return <p className="py-10 text-center text-ink-muted">{m.common.loading}</p>;

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-5 flex items-center gap-2 text-xl font-bold">
        <IconStar className="text-primary" />
        {m.terms.ratings}
      </h1>

      {orders.length === 0 ? (
        <p className="py-10 text-center text-ink-muted">{R.empty}</p>
      ) : (
        <ul className="space-y-2">
          {orders.map((o) => (
            <li
              key={o.order_id}
              className="flex flex-wrap items-center gap-3 rounded-card border border-line bg-surface p-4"
            >
              <span className="font-bold">#{fmt.format(o.number)}</span>
              <span className="min-w-0 flex-1 truncate text-sm">{o.merchant_name}</span>
              {o.rated ? (
                <div className="flex flex-col items-end gap-0.5">
                  <StarRow n={o.merchant_stars} />
                  {o.comment && (
                    <span className="max-w-xs truncate text-xs text-ink-muted">{o.comment}</span>
                  )}
                </div>
              ) : (
                <Button onClick={() => setRating(o)} className="flex items-center gap-1.5">
                  <IconStar size={15} />
                  {R.rateOrder}
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}

      {rating && (
        <RatingModal
          order={rating}
          onClose={() => setRating(null)}
          onRated={() => {
            setRating(null);
            load();
          }}
        />
      )}
    </div>
  );
}
