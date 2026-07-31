"use client";

/** طلباتي: السجل الكامل مع حالة كل طلب. */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  useLiveRefresh,
  IconOrder,
  IconStar,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";
import RatingModal from "@/components/RatingModal";

const m = getMessages(defaultLocale);
const STATUS_LABELS: Record<string, string> = m.orders.status;

const VARIANT: Record<string, "warning" | "primary" | "success" | "danger" | "neutral"> = {
  pending: "warning",
  delivered: "success",
  rejected: "danger",
  cancelled: "danger",
  failed: "danger",
  refunded: "neutral",
};

interface Order {
  id: string;
  number: number;
  merchant_name: string;
  status: string;
  total: number;
  created_at: string;
}

interface RateInfo {
  order_id: string;
  number: number;
  merchant_name: string;
  has_driver: boolean;
  rated: boolean;
}

export default function MyOrdersPage() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const [orders, setOrders] = useState<Order[] | null>(null);
  const [rateMap, setRateMap] = useState<Record<string, RateInfo>>({});
  const [rating, setRating] = useState<RateInfo | null>(null);

  const loadRatings = useCallback(() => {
    api<RateInfo[]>("/api/v1/my/ratings")
      .then((rs) => setRateMap(Object.fromEntries(rs.map((r) => [r.order_id, r]))))
      .catch(() => undefined);
  }, []);

  const load = useCallback(() => {
    api<{ orders: Order[] }>("/api/v1/my/orders?per_page=50")
      .then((d) => setOrders(d.orders))
      .catch(() => setOrders([]));
    loadRatings();
  }, [loadRatings]);

  useEffect(() => {
    if (loading) return;
    if (!isLoggedIn(user)) {
      router.replace("/login?next=/orders");
      return;
    }
    load();
  }, [user, loading, router, load]);

  useLiveRefresh(["order", "rating"], load);

  if (!orders) return <LoadingState />;

  return (
    <PageContainer width="wide">
      <PageHeader icon={IconOrder} title={m.terms.orders} />
      {orders.length === 0 ? (
        <EmptyState icon={IconOrder} title={m.site.orders.empty} />
      ) : (
        <ul className="space-y-2">
          {orders.map((o) => {
            const rate = rateMap[o.id];
            const canRate = o.status === "delivered" && rate && !rate.rated;
            return (
              <li
                key={o.id}
                className="flex flex-wrap items-center gap-3 rounded-card border border-line bg-surface p-4"
              >
                <Link
                  href={`/orders/${o.id}`}
                  className="flex min-w-0 flex-1 flex-wrap items-center gap-3 hover:opacity-80"
                >
                  <span className="font-bold">#{fmtNum(o.number)}</span>
                  <span className="min-w-0 flex-1 truncate text-sm">{o.merchant_name}</span>
                  <Badge variant={VARIANT[o.status] ?? "primary"}>
                    {STATUS_LABELS[o.status] ?? o.status}
                  </Badge>
                  <span className="text-sm font-bold text-primary-dark">
                    {fmtNum(o.total)} {m.common.currency}
                  </span>
                  <span className="text-xs text-ink-muted" dir="ltr">
                    {fmtDateTime(o.created_at)}
                  </span>
                </Link>
                {canRate && (
                  <Button onClick={() => setRating(rate)} className="flex shrink-0 items-center gap-1.5">
                    <IconStar size={15} />
                    {m.site.rating.rateOrder}
                  </Button>
                )}
                {o.status === "delivered" && rate?.rated && (
                  <span className="flex shrink-0 items-center gap-1 text-xs text-success"><IconStar size={12} className="fill-success" />{m.site.rating.myTitle}</span>
                )}
              </li>
            );
          })}
        </ul>
      )}

      {rating && (
        <RatingModal
          order={rating}
          onClose={() => setRating(null)}
          onRated={() => {
            setRating(null);
            loadRatings();
          }}
        />
      )}
    </PageContainer>
  );
}
