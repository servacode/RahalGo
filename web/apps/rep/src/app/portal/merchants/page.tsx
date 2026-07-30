"use client";

/** متاجر المندوب — المتاجر المنسوبة له وأداؤها. */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge, IconStore } from "@rahalgo/ui";
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

export default function MerchantsPage() {
  const [merchants, setMerchants] = useState<RepMerchant[] | null>(null);

  useEffect(() => {
    api<RepMerchant[]>("/api/v1/rep/merchants")
      .then(setMerchants)
      .catch(() => setMerchants([]));
  }, []);

  if (!merchants) {
    return <p className="py-12 text-center text-ink-muted">{m.common.loading}</p>;
  }

  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-lg font-bold">
        <IconStore size={20} className="text-primary" />
        {m.rep.merchantsTitle}
      </h1>

      {merchants.length === 0 ? (
        <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-ink-muted">
          {m.rep.merchantsEmpty}
        </p>
      ) : (
        <ul className="space-y-2">
          {merchants.map((mr) => {
            const logo = mediaUrl(mr.logo_thumb_url);
            return (
              <li
                key={mr.id}
                className="flex items-center gap-3 rounded-card border border-line bg-surface p-3"
              >
                {logo ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={logo} alt="" className="h-11 w-11 rounded-control object-cover" />
                ) : (
                  <span className="flex h-11 w-11 items-center justify-center rounded-control bg-primary-light">
                    {mr.category_icon}
                  </span>
                )}
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">{mr.name}</p>
                  <p className="text-xs text-ink-muted">
                    {m.rep.joinedAt} <span dir="ltr">{mr.joined_at}</span> —{" "}
                    {m.rep.deliveredCount.replace("{n}", fmt.format(mr.delivered_orders))}
                  </p>
                </div>
                <Badge variant={mr.status === "active" ? "success" : "danger"}>
                  {mr.status === "active" ? m.admin.merchants.active : m.admin.merchants.inactive}
                </Badge>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
