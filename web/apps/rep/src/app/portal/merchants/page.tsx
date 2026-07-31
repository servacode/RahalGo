"use client";

/** عملائي — المتاجر التي جلبها المندوب وأداؤها. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Badge,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  ListRow,
  useLiveData,
  IconStore,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

interface RepMerchant {
  id: string;
  name: string;
  category_icon: string;
  logo_thumb_url: string | null;
  status: string;
  joined_at: string;
  delivered_orders: number;
}

export default function ClientsPage() {
  const { data, loading } = useLiveData<RepMerchant[]>(
    () => api("/api/v1/rep/merchants"),
    ["lead", "order", "account"],
  );

  if (loading) return <LoadingState />;
  const merchants = data ?? [];

  return (
    <PageContainer>
      <PageHeader icon={IconStore} title={m.terms.clients} />

      {merchants.length === 0 ? (
        <EmptyState icon={IconStore} title={m.rep.merchantsEmpty} />
      ) : (
        <ul className="space-y-2">
          {merchants.map((mr) => {
            const logo = mediaUrl(mr.logo_thumb_url);
            return (
              <ListRow
                key={mr.id}
                leading={
                  logo ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={logo} alt="" className="h-11 w-11 rounded-control object-cover" />
                  ) : (
                    <span className="flex h-11 w-11 items-center justify-center rounded-control bg-primary-light">
                      {mr.category_icon}
                    </span>
                  )
                }
                title={mr.name}
                subtitle={
                  <>
                    {m.rep.joinedAt} <span dir="ltr">{mr.joined_at}</span> —{" "}
                    {m.rep.deliveredCount.replace("{n}", fmt.format(mr.delivered_orders))}
                  </>
                }
                trailing={
                  <Badge variant={mr.status === "active" ? "success" : "danger"}>
                    {mr.status === "active" ? m.terms.active : m.terms.suspended}
                  </Badge>
                }
              />
            );
          })}
        </ul>
      )}
    </PageContainer>
  );
}
