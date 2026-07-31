"use client";

/** طلباتي: السجل الكامل مع حالة كل طلب. */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  EntityCard,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  useLiveRefresh,
  IconOrder,
  IconStar,
  Stars,
  IconStore,
  IconWallet,
  IconCart,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
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
  merchant_logo_thumb_url: string | null;
  items_count: number;
  items_preview: string;
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
  merchant_stars: number;
  driver_stars: number | null;
  comment: string;
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
    <PageContainer>
      <PageHeader icon={IconOrder} title={m.terms.orders} />
      {orders.length === 0 ? (
        <EmptyState icon={IconOrder} title={m.site.orders.empty} />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {orders.map((o) => {
            const rate = rateMap[o.id];
            const canRate = o.status === "delivered" && rate && !rate.rated;
            const logo = mediaUrl(o.merchant_logo_thumb_url);
            const more = o.items_count - o.items_preview.split("، ").filter(Boolean).length;
            return (
              <EntityCard
                key={o.id}
                media={
                  logo ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={logo} alt="" className="h-12 w-12 rounded-control object-cover" />
                  ) : (
                    <span className="flex h-12 w-12 items-center justify-center rounded-control bg-primary-light">
                      <IconStore size={20} className="text-primary-dark" />
                    </span>
                  )
                }
                title={
                  <span className="flex items-center gap-2">
                    <span className="truncate">{o.merchant_name}</span>
                    <span className="shrink-0 text-xs font-normal text-ink-muted" dir="ltr">
                      #{fmtNum(o.number)}
                    </span>
                  </span>
                }
                /* «ماذا طلبتُ؟» أول سؤال يسأله صاحب الطلب — وكان يلزمه فتح
                   الطلب ليعرف. الأصناف هنا مباشرةً تحت اسم المتجر. */
                subtitle={
                  o.items_preview
                    ? more > 0
                      ? `${o.items_preview} ${m.site.orders.itemsMore.replace("{n}", fmtNum(more))}`
                      : o.items_preview
                    : m.site.orders.noItems
                }
                badge={
                  <Badge variant={VARIANT[o.status] ?? "primary"}>
                    {STATUS_LABELS[o.status] ?? o.status}
                  </Badge>
                }
                stats={[
                  {
                    label: m.site.orders.statItems,
                    value: fmtNum(o.items_count),
                    icon: <IconCart />,
                  },
                  {
                    label: `${m.site.orders.statTotal} (${m.common.currency})`,
                    value: fmtNum(o.total),
                    icon: <IconWallet />,
                  },
                ]}
                footer={<span dir="ltr">{fmtDateTime(o.created_at)}</span>}
                actions={
                  <>
                    <Link
                      href={`/orders/${o.id}`}
                      className="flex flex-1 items-center justify-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm font-medium text-ink hover:bg-page"
                    >
                      <IconOrder size={15} />
                      {m.site.orders.openOrder}
                    </Link>
                    {canRate && (
                      <Button
                        onClick={() => setRating(rate)}
                        className="flex flex-1 items-center justify-center gap-1.5 !py-1.5"
                      >
                        <IconStar size={15} />
                        {m.site.rating.rateOrder}
                      </Button>
                    )}
                    {o.status === "delivered" && rate?.rated && (
                      // نجومٌ لا شارة: «تقييماتي» تقول إنك قيّمت ولا تقول بكم
                      <span className="flex flex-1 items-center justify-center gap-2 rounded-control bg-page px-3 py-1.5">
                        <span className="text-xs text-ink-muted">{m.site.rating.merchant}</span>
                        <Stars value={rate.merchant_stars} size="sm" />
                      </span>
                    )}
                  </>
                }
              />
            );
          })}
        </div>
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
