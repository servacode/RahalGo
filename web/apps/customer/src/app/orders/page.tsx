"use client";

/** طلباتي: السجل الكامل مع حالة كل طلب. */

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge, IconOrder } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");
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

export default function MyOrdersPage() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const [orders, setOrders] = useState<Order[] | null>(null);

  useEffect(() => {
    if (loading) return;
    if (!isLoggedIn(user)) {
      router.replace("/login?next=/orders");
      return;
    }
    api<{ orders: Order[] }>("/api/v1/my/orders?per_page=50")
      .then((d) => setOrders(d.orders))
      .catch(() => setOrders([]));
  }, [user, loading, router]);

  if (!orders) return <p className="py-10 text-center text-ink-muted">{m.common.loading}</p>;

  return (
    <div>
      <h1 className="mb-5 flex items-center gap-2 text-xl font-bold">
        <IconOrder className="text-primary" />
        {m.site.orders.title}
      </h1>
      {orders.length === 0 ? (
        <p className="py-10 text-center text-ink-muted">{m.site.orders.empty}</p>
      ) : (
        <ul className="space-y-2">
          {orders.map((o) => (
            <li key={o.id}>
              <Link
                href={`/orders/${o.id}`}
                className="flex flex-wrap items-center gap-3 rounded-card border border-line bg-surface p-4 hover:shadow-md"
              >
                <span className="font-bold">#{fmt.format(o.number)}</span>
                <span className="min-w-0 flex-1 truncate text-sm">{o.merchant_name}</span>
                <Badge variant={VARIANT[o.status] ?? "primary"}>
                  {STATUS_LABELS[o.status] ?? o.status}
                </Badge>
                <span className="text-sm font-bold text-primary-dark">
                  {fmt.format(o.total)} {m.common.currency}
                </span>
                <span className="text-xs text-ink-muted" dir="ltr">
                  {new Date(o.created_at).toLocaleString("ar-SY", {
                    dateStyle: "short",
                    timeStyle: "short",
                  })}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
