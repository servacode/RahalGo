"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتي — الجاريةُ والمنتهية، وحالُها يتحدّث وحدَه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والتحديثُ بالنداء كلَّ خمسَ عشرةَ ثانية** ما دامت الصفحةُ ظاهرة —
 * **ولا بثَّ حيّاً للزبون على الويب بعد** (المرحلة الثانية). **ومن أخفى
 * التبويبَ لا يُستنزف رصيدُ بياناته.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, Chips, EmptyState, LoadingState, ReloadState, IconOrder } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { ENDED, type MyOrder, type MyOrdersPage } from "@/lib/customer/types";
import { OrderSummary } from "@/components/customer/OrderSummary";
import { NeedLogin } from "@/components/customer/NeedLogin";

const m = getMessages(defaultLocale);
const O = m.shop.orders;
const EVERY_MS = 15_000;
const PER_PAGE = 30;

export default function OrdersPage() {
  const { user, loading } = useAuth();
  const [tab, setTab] = useState<"open" | "history">("open");
  const [orders, setOrders] = useState<MyOrder[] | null>(null);
  const [total, setTotal] = useState(0);
  const [pages, setPages] = useState(1);
  const [error, setError] = useState("");
  const [more, setMore] = useState(false);

  const load = useCallback(
    (upTo: number) => {
      if (!user) return;
      Promise.all(
        Array.from({ length: upTo }, (_, i) =>
          api<MyOrdersPage>(`/api/v1/my/orders?open_only=false&page=${i + 1}&per_page=${PER_PAGE}`),
        ),
      )
        .then((all) => {
          const seen = new Set<string>();
          const list: MyOrder[] = [];
          for (const p of all) for (const o of p.orders ?? []) if (!seen.has(o.id) && seen.add(o.id)) list.push(o);
          setOrders(list);
          setTotal(all[0]?.total ?? list.length);
          setError("");
        })
        .catch((err) => setError(errorText(err)))
        .finally(() => setMore(false));
    },
    [user],
  );

  useEffect(() => {
    load(pages);
    const t = setInterval(() => {
      if (document.visibilityState === "visible") load(pages);
    }, EVERY_MS);
    return () => clearInterval(t);
  }, [load, pages]);

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;

  const open = (orders ?? []).filter((o) => !ENDED.has(o.status));
  const history = (orders ?? []).filter((o) => ENDED.has(o.status));
  const list = tab === "open" ? open : history;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="heading-page">{O.title}</h1>
        <span className="text-xs text-ink-muted">{O.live}</span>
      </div>
      <Chips<"open" | "history">
        items={[
          { id: "open", label: O.open, count: open.length },
          { id: "history", label: O.history },
        ]}
        value={tab}
        onChange={setTab}
      />
      {error && !orders ? (
        <ReloadState label={error} onRetry={() => load(pages)} />
      ) : orders === null ? (
        <LoadingState variant="list" rows={3} />
      ) : list.length === 0 ? (
        <EmptyState icon={IconOrder} title={tab === "open" ? O.noneOpen : O.noneHistory} />
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {list.map((o) => (
            <OrderSummary key={o.id} order={o} />
          ))}
        </div>
      )}
      {orders && orders.length < total && tab === "history" && (
        <Button
          variant="secondary"
          className="mx-auto"
          disabled={more}
          onClick={() => {
            setMore(true);
            setPages((p) => p + 1);
          }}
        >
          {O.loadMore}
        </Button>
      )}
    </div>
  );
}
