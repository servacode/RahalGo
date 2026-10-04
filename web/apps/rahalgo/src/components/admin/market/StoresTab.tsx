"use client";

/**
 * **تبويبُ المتاجر — صفحةٌ موحّدة.**
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «المتاجرُ تُدار في اللوحة ولا تُعرض على الزبون —
 * واسمُها «المتاجر» لا «المورّدون»».)
 *
 * **بطاقةٌ لكلّ متجر**: أصنافُه وما يتوفّر منها وما أُضيف حديثاً وما بلا صورة،
 * **وعلامةٌ إن لم يضبط دوامَه** — وكلُّ بطاقةٍ تفتح صفحتَه.
 */

import { useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Badge,
  Input,
  Pagination,
  EmptyState,
  LoadingState,
  useLiveData,
  IconStore,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";

const m = getMessages(defaultLocale);
const T = m.admin.market.stores;

interface MarketStore {
  id: string;
  name: string;
  status: string;
  emergency_closed: boolean;
  logo_url: string | null;
  items: number;
  live_items: number;
  new_items: number;
  no_image_items: number;
  has_hours: boolean;
}

export default function StoresTab() {
  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);
  const query = new URLSearchParams({ q, page: String(page) }).toString();
  const { data } = useLiveData<{ stores: MarketStore[]; total: number; per_page: number }>(
    () => api(`/api/v1/admin/market/stores?${query}`),
    ["menu", "catalog"],
    [query],
  );

  return (
    <div className="space-y-3">
      <Input
        placeholder={T.search}
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          setPage(1);
        }}
      />
      {!data ? (
        <LoadingState />
      ) : data.stores.length === 0 ? (
        <EmptyState icon={IconStore} title={T.empty} />
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {data.stores.map((s) => (
            <Link
              key={s.id}
              href={`/dashboard/merchants/${s.id}`}
              className="surface flex items-center gap-3 p-3 transition-shadow hover:elev-2"
            >
              <div className="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-control bg-field">
                {s.logo_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={mediaUrl(s.logo_url) ?? ""} alt={s.name} className="h-full w-full object-cover" />
                ) : (
                  <IconStore size={20} />
                )}
              </div>
              <div className="min-w-0 flex-1 space-y-1">
                <p className="truncate font-bold">{s.name}</p>
                <p className="text-xs text-ink-muted">
                  {T.items.replace("{n}", fmtNum(s.items))}
                  {" · "}
                  {T.live.replace("{n}", fmtNum(s.live_items))}
                </p>
                <div className="flex flex-wrap gap-1">
                  <Badge variant={s.status === "active" ? "success" : "neutral"}>
                    {(m.admin.merchants as unknown as Record<string, string>)[s.status] ?? s.status}
                  </Badge>
                  {s.emergency_closed && <Badge variant="danger">{T.emergency}</Badge>}
                  {s.new_items > 0 && (
                    <Badge variant="accent">{T.newItems.replace("{n}", fmtNum(s.new_items))}</Badge>
                  )}
                  {s.no_image_items > 0 && (
                    <Badge variant="warning">
                      {T.noImageItems.replace("{n}", fmtNum(s.no_image_items))}
                    </Badge>
                  )}
                  {!s.has_hours && <Badge variant="warning">{T.noHours}</Badge>}
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
      {data && (
        <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
      )}
    </div>
  );
}
